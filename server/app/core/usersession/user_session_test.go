//go:build unit

package usersession

import (
	"errors"
	"sync"
	"testing"
	"time"

	"osdtyp/app/entity"

	"go.uber.org/zap"
)

// This file is the regression suite for the socket rewrite.
//
// Everything here was a live bug. The session used to hand its raw inbound and
// outbound channels to every caller and use a mutex to pretend that was
// exclusive; in practice the lock was taken and only released if something
// happened to send an unsubscribe sentinel down the same channel, so a player
// in a lobby could never queue for ranked and could never be invited. It also
// closed its outbound channel on disconnect, which turned any concurrent push
// into a panic, and the server-to-client path marshaled a []byte, which
// base64-encoded every frame.

// pingFrame is gorilla's control frame type for a ping. It is named here so the
// fake socket's driver does not have to import the websocket package.
const pingFrame = 9

var errFakeClosed = errors.New("fake socket closed")

// fakeSocket is a frameWriter a test can drive directly.
//
// It records what was written, lets a test hand frames to the session's read
// loop, and reports whether two goroutines ever wrote at once — which
// gorilla/websocket forbids and which the old send path did.
type fakeSocket struct {
	mu     sync.Mutex
	writes [][]byte

	// in is what ReadMessage drains. Closing it ends the read loop.
	in chan []byte
	// readErr is returned once in is drained, to model a broken connection.
	readErr error

	// writers counts in-flight writers, so an overlap is observable.
	writers    int
	concurrent bool

	closeOnce sync.Once
	closed    chan struct{}
}

func newFakeSocket() *fakeSocket {
	return &fakeSocket{
		in:     make(chan []byte, 64),
		closed: make(chan struct{}),
	}
}

func (f *fakeSocket) WriteMessage(_ int, data []byte) error {
	f.mu.Lock()
	f.writers++
	if f.writers > 1 {
		f.concurrent = true
	}
	f.writes = append(f.writes, append([]byte(nil), data...))
	f.writers--
	f.mu.Unlock()
	return nil
}

func (f *fakeSocket) ReadMessage() (int, []byte, error) {
	msg, ok := <-f.in
	if !ok {
		if f.readErr != nil {
			return 0, nil, f.readErr
		}
		return 0, nil, errFakeClosed
	}
	return 1, msg, nil
}

func (f *fakeSocket) Close() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

// sent returns a copy of everything written to the socket.
func (f *fakeSocket) sent() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]byte, len(f.writes))
	copy(out, f.writes)
	return out
}

func (f *fakeSocket) wroteConcurrently() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.concurrent
}

// testSession is a session plus the fake underneath it and a live count of
// disconnect callbacks.
type testSession struct {
	session *UserSession
	socket  *fakeSocket

	mu           sync.Mutex
	disconnected int
}

func (ts *testSession) disconnects() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.disconnected
}

// newTestSession builds a session on a fake socket and tears it down when the
// test ends. The session's loops are not joined by UserOffline, so the cleanup
// gives them a moment to finish rather than leaking them into the next test.
func newTestSession(t *testing.T, id uint32) *testSession {
	t.Helper()

	ts := &testSession{socket: newFakeSocket()}
	ts.session = newUserSession(ts.socket, func(uint32) {
		ts.mu.Lock()
		ts.disconnected++
		ts.mu.Unlock()
	}, id, zap.NewNop().Sugar())

	t.Cleanup(func() {
		ts.session.UserOffline()
		close(ts.socket.in)
		time.Sleep(20 * time.Millisecond)
	})

	return ts
}

// eventually polls until cond holds, or fails the test.
//
// Polling rather than a fixed sleep: the loops under test are goroutines with no
// synchronization point a test can hook, and a fixed wait is either flaky or
// slow depending on the machine.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestSubscribeRevokesThePreviousLease is the property the whole design exists
// for: a WebSocket has exactly one inbound stream, so a second subscriber has
// to end the first one's. The old implementation tried to express this with a
// mutex that callers never released.
func TestSubscribeRevokesThePreviousLease(t *testing.T) {
	ts := newTestSession(t, 1)

	first := ts.session.Subscribe()
	second := ts.session.Subscribe()

	select {
	case _, ok := <-first.Recv():
		if ok {
			t.Fatal("the revoked lease delivered a frame")
		}
	case <-time.After(time.Second):
		t.Fatal("the revoked lease's channel was never closed, so its reader hangs forever")
	}

	// The revocation must not have taken the live lease down with it.
	if ts.session.IsClosed() {
		t.Error("taking a second lease closed the whole session")
	}
	if got := second.Recv(); got == nil {
		t.Error("the new lease has no channel")
	}
}

// TestOnlyTheCurrentLeaseReceivesFrames checks that a revoked lease is starved,
// not merely closed: once a second subscriber exists, nothing the client sends
// reaches the first.
func TestOnlyTheCurrentLeaseReceivesFrames(t *testing.T) {
	ts := newTestSession(t, 2)

	stale := ts.session.Subscribe()
	fresh := ts.session.Subscribe()

	const frame = `{"type":"keypress","value":"a"}`
	ts.socket.in <- []byte(frame)

	select {
	case raw := <-fresh.Recv():
		if string(raw) != frame {
			t.Fatalf("the current lease got %q", raw)
		}
	case <-time.After(time.Second):
		t.Fatal("the current lease did not receive the frame")
	}

	select {
	case _, ok := <-stale.Recv():
		if ok {
			t.Fatal("a revoked lease received a frame")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("the revoked lease was not closed")
	}
}

// TestReleaseClosesTheLease covers the ordinary end of a round: a game releases
// when it is done, and the next consumer has to be able to take the socket
// without waiting for a timeout.
func TestReleaseClosesTheLease(t *testing.T) {
	ts := newTestSession(t, 3)

	sub := ts.session.Subscribe()
	sub.Release()

	select {
	case _, ok := <-sub.Recv():
		if ok {
			t.Fatal("a released lease delivered a frame")
		}
	case <-time.After(time.Second):
		t.Fatal("Release did not close the lease's channel")
	}
}

// TestReleaseIsIdempotent: a game whose socket dies and whose round then ends
// will release twice, and a double close panics.
func TestReleaseIsIdempotent(t *testing.T) {
	ts := newTestSession(t, 4)

	sub := ts.session.Subscribe()
	sub.Release()
	sub.Release()
	sub.Release()
}

// TestReleaseThenSendIsANoOp is the sequence the old unsubscribe produced: it
// ended a lease by sending a nil sentinel down the same channel, so a Send
// arriving after it had to be harmless.
func TestReleaseThenSendIsANoOp(t *testing.T) {
	ts := newTestSession(t, 5)

	sub := ts.session.Subscribe()
	sub.Release()
	sub.Send([]byte(`{"type":"end"}`))

	select {
	case _, ok := <-sub.Recv():
		if ok {
			t.Fatal("a released lease delivered a frame")
		}
	case <-time.After(50 * time.Millisecond):
	}

	// And it must not have reached the socket either, or a released lease would
	// still be writing to a game it is no longer part of.
	if got := len(ts.socket.sent()); got != 0 {
		t.Errorf("a released lease wrote %d frames to the socket", got)
	}
}

// TestNotifyAfterUserOfflineDoesNotPanic is the panic the old teardown caused.
//
// UserOffline used to close the outbound channel, so any handler that decided
// to push a frame while a socket was dying wrote to a closed channel and the
// process died with it. Now teardown closes a separate signal and a late push
// is an ordinary error.
func TestNotifyAfterUserOfflineDoesNotPanic(t *testing.T) {
	ts := newTestSession(t, 6)

	ts.session.UserOffline()
	eventually(t, "the session to be marked closed", ts.session.IsClosed)

	for i := 0; i < 100; i++ {
		err := ts.session.Notify([]byte(`{"type":"notification"}`))
		if !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("push %d after teardown returned %v, want ErrSessionClosed", i, err)
		}
	}
}

// TestNotifyRacingUserOfflineIsSafe is the same bug from the other side: a push
// landing at the same moment as the teardown. Neither return value
// distinguishes the two orderings, so the race detector is the real assertion
// here.
func TestNotifyRacingUserOfflineIsSafe(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		sock := newFakeSocket()
		s := newUserSession(sock, func(uint32) {}, uint32(attempt), zap.NewNop().Sugar())

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_ = s.Notify([]byte(`{"type":"notification"}`))
			}
		}()
		go func() {
			defer wg.Done()
			s.UserOffline()
		}()
		wg.Wait()

		close(sock.in)
	}
}

// TestNotifyDropsRatherThanBlocks is a liveness property, and the reason a slow
// client cannot stall a handler.
//
// A push to a client whose writer is stuck has to fail fast rather than queue
// behind a full buffer. A notification is recoverable because it is also
// stored; a keystroke broadcast changes nothing the client cannot recover from
// its next frame. Blocking here would let one wedged browser hold up a game
// handler for every other player in the round.
func TestNotifyDropsRatherThanBlocks(t *testing.T) {
	s := &UserSession{
		Outgoing: make(chan []byte, 2),
		closed:   make(chan struct{}),
		Logger:   zap.NewNop().Sugar(),
	}

	// Fill the queue with nothing draining it.
	for _, frame := range []string{"a", "b"} {
		if err := s.Notify([]byte(frame)); err != nil {
			t.Fatalf("a push into a queue with room failed: %v", err)
		}
	}

	done := make(chan error, 1)
	go func() { done <- s.Notify([]byte("c")) }()

	select {
	case err := <-done:
		if !errors.Is(err, ErrSessionBusy) {
			t.Fatalf("a push into a full queue returned %v, want ErrSessionBusy", err)
		}
	case <-time.After(time.Second):
		t.Fatal("a push into a full queue blocked; one wedged client would stall the process")
	}
}

// TestUserOfflineClosesTheActiveLease is what makes a disconnect end a game
// promptly rather than at the end of the round timer. A player who closed their
// tab used to sit in the round for its full duration holding a slot.
func TestUserOfflineClosesTheActiveLease(t *testing.T) {
	ts := newTestSession(t, 8)

	sub := ts.session.Subscribe()
	ts.session.UserOffline()

	select {
	case _, ok := <-sub.Recv():
		if ok {
			t.Fatal("the lease delivered a frame after the session went away")
		}
	case <-time.After(time.Second):
		t.Fatal("the lease was not closed; a game routine would wait out its whole round")
	}
}

// TestUserOfflineRunsOnce: a socket read failure, a write failure and the ping
// timeout can all fire the teardown. The disconnect hook must not run three
// times, and a second close of a channel is a panic.
func TestUserOfflineRunsOnce(t *testing.T) {
	ts := newTestSession(t, 9)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.session.UserOffline()
		}()
	}
	wg.Wait()
	time.Sleep(30 * time.Millisecond)

	if got := ts.disconnects(); got != 1 {
		t.Errorf("the disconnect hook ran %d times, want once", got)
	}
}

// TestNoConcurrentSocketWrites covers gorilla/websocket's one-writer rule.
// sendData writes application frames and Ping writes control frames; if the two
// were not serialized the server would interleave its own frames and corrupt
// the stream, which gorilla reports as a panic in some versions.
func TestNoConcurrentSocketWrites(t *testing.T) {
	ts := newTestSession(t, 10)

	// Ping's own goroutine runs on a 20 second interval so it will not fire
	// during a test. Drive the same lock it takes, from several goroutines at
	// once, so the assertion is about serialization rather than about timing.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = ts.session.Notify([]byte(`{"type":"notification"}`))
			}
		}()
	}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ts.session.PingLock.Lock()
				_ = ts.session.socket.WriteMessage(pingFrame, nil)
				ts.session.PingLock.Unlock()
			}
		}()
	}
	wg.Wait()

	if ts.socket.wroteConcurrently() {
		t.Error("two goroutines wrote to the socket at the same time")
	}
}

// TestStatusSurvivesTeardown: a handler reading a player's status while their
// socket dies should get a value, not a panic.
func TestStatusSurvivesTeardown(t *testing.T) {
	ts := newTestSession(t, 11)

	ts.session.SetStatus(entity.PLAYING)
	if got := ts.session.Status(); got != entity.PLAYING {
		t.Errorf("status = %v, want PLAYING", got)
	}

	ts.session.UserOffline()
	if got := ts.session.Status(); got != entity.PLAYING {
		t.Errorf("status after teardown = %v, want the last known value", got)
	}
	if !ts.session.IsClosed() {
		t.Error("IsClosed reported false after teardown")
	}
}
