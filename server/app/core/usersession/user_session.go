package usersession

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"osdtyp/app/entity"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// A UserSession owns exactly one WebSocket for the lifetime of a
// connection.
//
// The rule the whole design rests on: only sendData and Ping ever touch the
// socket, and they are serialized by PingLock. Everything else in the process
// reaches the socket through Notify, which queues a frame, or through
// Subscribe, which hands out a revocable lease on the inbound stream.
//
// The previous version handed the raw inbound and outbound channels to every
// caller and used a mutex to pretend that was exclusive. In practice the lock
// was taken and never released, which deadlocked any player who was in a
// lobby and then tried to queue for ranked, and any attempt to invite them.

var (
	// ErrSessionClosed is returned when a session has already been torn down.
	ErrSessionClosed = errors.New("session is closed")
	// ErrSessionBusy is returned when a frame could not be queued because the
	// writer is behind.
	ErrSessionBusy = errors.New("session writer is busy")
)

// Subscriber is a lease on a session's inbound stream.
//
// Send and Release are the only things a consumer of a game needs; Recv is
// the frame stream. A lease is invalidated either by the session going away
// or by another caller taking a newer lease, and in both cases Recv is closed
// so a reader blocked in a select wakes up instead of hanging.
type Subscriber struct {
	s    *UserSession
	recv chan []byte

	// closeOnce owns the single close of recv. Every path that ends a lease —
	// Release, a newer Subscribe, and the session's own teardown — goes
	// through closeRecv, so the channel is closed exactly once.
	//
	// It is deliberately not the same Once a Release would nest inside.
	// sync.Once is not reentrant: an earlier version had Release call revoke
	// and then had revoke reach back for the very Once Release was already
	// inside, which deadlocked on itself. Nothing caught it, because nothing
	// called Release, so it would have frozen the process at the end of the
	// first round anybody played.
	closeOnce sync.Once
}

// Recv is the inbound frame stream. It closes when the lease is revoked.
func (sub *Subscriber) Recv() <-chan []byte { return sub.recv }

// Send queues an outbound frame.
//
// The return of Notify is deliberately dropped. entity.Socket, which this
// implements, is declared as Send([]byte) with no error, so the outcome cannot
// be handed to the caller; the game loop is not the place that decides whether a
// dropped notification is worth a retry, because notifications are also stored
// in the database. The error is logged instead of swallowed, so a session whose
// queue is full is visible in the logs rather than being silently lossy.
func (sub *Subscriber) Send(raw []byte) {
	if err := sub.s.Notify(raw); err != nil {
		sub.s.Logger.Debugw("game frame dropped",
			"user_id", sub.s.UserID, "error", err)
	}
}

// Release gives up the lease so another consumer can take the socket.
//
// It is deliberately not required. A game that ends without releasing still
// costs nothing: the next Subscribe revokes it anyway.
func (sub *Subscriber) Release() { sub.s.revoke(sub) }

// closeRecv ends the lease's inbound stream. Safe from any path, any number of
// times.
func (sub *Subscriber) closeRecv() { sub.closeOnce.Do(func() { close(sub.recv) }) }

// UserSession is one user's live connection.
type UserSession struct {
	status atomic.Int32
	UserID uint32

	// Outgoing carries frames already marshaled. sendData is its only
	// reader, so a producer never writes to the socket itself and cannot race
	// the ping loop.
	Outgoing chan []byte

	// OnDisconnect runs once, when the session ends.
	OnDisconnect func(uint32)
	Logger       *zap.SugaredLogger

	// leaseMu guards current. Taking a lease is a pointer swap, so Subscribe
	// never blocks and two callers cannot both believe they hold the socket.
	leaseMu sync.Mutex
	current *Subscriber

	// PingLock serializes the two writers to the socket: sendData's frames and
	// Ping's control frames. gorilla/websocket permits exactly one concurrent
	// writer and one concurrent reader.
	PingLock sync.Mutex

	// socket is unexported so nothing outside this package can write to it.
	// Every previous leak of a raw channel or a bare *websocket.Conn came from
	// this field being reachable.
	socket frameWriter

	offlineOnce sync.Once
	closed      chan struct{}
}

// frameWriter is the half of a WebSocket a session needs.
//
// Naming it means a session does not depend on the concrete websocket type, so
// the write path can be exercised without a real handshake — and so the single
// place that is allowed to touch the socket is a small, obvious interface
// rather than a field every package can reach for.
type frameWriter interface {
	// WriteMessage writes one frame.
	WriteMessage(messageType int, data []byte) error
	// ReadMessage blocks for the next frame.
	ReadMessage() (messageType int, p []byte, err error)
	// Close ends the connection.
	Close() error
}

// nopSocket is a socket that accepts writes and never produces a read.
//
// It backs a session with no transport behind it, which is what lets code that
// needs a *live* session — the lobby resolving a roster, the matchmaker
// admitting a player — be exercised without a WebSocket handshake.
type nopSocket struct {
	closed chan struct{}
	once   sync.Once
}

func newNopSocket() *nopSocket { return &nopSocket{closed: make(chan struct{})} }

func (*nopSocket) WriteMessage(int, []byte) error { return nil }

func (s *nopSocket) ReadMessage() (int, []byte, error) {
	<-s.closed
	return 0, nil, ErrSessionClosed
}

func (s *nopSocket) Close() error {
	s.once.Do(func() { close(s.closed) })
	return nil
}

// NewUserSession wires a live socket up to a session and starts its loops.
func NewUserSession(ws *websocket.Conn, disc func(uint32), id uint32, logger *zap.SugaredLogger) *UserSession {
	return newUserSession(ws, disc, id, logger)
}

func newUserSession(socket frameWriter, disc func(uint32), id uint32, logger *zap.SugaredLogger) *UserSession {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	user := &UserSession{
		// A small buffer absorbs a burst of pushes, such as a friend coming
		// online while a notification is being delivered to them.
		Outgoing:     make(chan []byte, outgoingBuffer),
		OnDisconnect: disc,
		UserID:       id,
		Logger:       logger,
		socket:       socket,
		closed:       make(chan struct{}),
	}
	user.status.Store(int32(entity.AVAILABLE))

	logger.Infow("making new session", "user_id", id)
	go user.sendData()
	go user.receiveData()
	go user.Ping()
	return user
}

// Status reports the user's current availability.
func (u *UserSession) Status() entity.UserStatus { return entity.UserStatus(u.status.Load()) }

// SetStatus records the user's current availability.
func (u *UserSession) SetStatus(s entity.UserStatus) { u.status.Store(int32(s)) }

// IsClosed reports whether the session has already been torn down.
func (u *UserSession) IsClosed() bool {
	select {
	case <-u.closed:
		return true
	default:
		return false
	}
}

// Notify queues a frame for delivery.
//
// It never blocks and never panics. A full queue means the writer is behind,
// so the frame is dropped and the caller is told: a dropped notification is
// recoverable because notifications are also stored in the database, and a
// dropped keystroke changes nothing the client cannot recover on its next
// frame. Blocking here instead would stall whichever HTTP handler or game
// routine happened to be pushing.
func (u *UserSession) Notify(raw []byte) error {
	select {
	case <-u.closed:
		return ErrSessionClosed
	default:
	}
	select {
	case u.Outgoing <- raw:
		return nil
	case <-u.closed:
		return ErrSessionClosed
	default:
		return ErrSessionBusy
	}
}

// Subscribe takes a revocable lease on the inbound stream.
//
// Taking a lease revokes the previous one. This is the answer to "only one
// process can have the socket at a time": instead of a mutex that callers
// forget to unlock, the newest caller wins and the old one is told its stream
// has ended.
func (u *UserSession) Subscribe() *Subscriber {
	sub := &Subscriber{s: u, recv: make(chan []byte, recvBuffer)}

	u.leaseMu.Lock()
	prev := u.current
	u.current = sub
	u.leaseMu.Unlock()

	if prev != nil {
		prev.closeRecv()
	}
	return sub
}

// revoke ends a lease without taking a newer one.
//
// It clears the session's pointer only if the lease being ended is still the
// active one: a game that finishes late must not unseat the round that started
// after it.
func (u *UserSession) revoke(sub *Subscriber) {
	u.leaseMu.Lock()
	if u.current == sub {
		u.current = nil
	}
	u.leaseMu.Unlock()

	sub.closeRecv()
}

// sendData is the only writer of application frames.
func (u *UserSession) sendData() {
	for {
		select {
		case <-u.closed:
			return
		case raw := <-u.Outgoing:
			u.PingLock.Lock()
			err := u.socket.WriteMessage(websocket.TextMessage, raw)
			u.PingLock.Unlock()
			if err != nil {
				u.UserOffline()
				return
			}
		}
	}
}

// receiveData is the only reader of the socket, and the only writer to the
// active lease.
func (u *UserSession) receiveData() {
	for {
		_, message, err := u.socket.ReadMessage()
		if err != nil {
			u.Logger.Debugw("session read ended", "user_id", u.UserID, "error", err)
			u.UserOffline()
			return
		}
		if len(message) == 0 {
			continue
		}

		u.leaseMu.Lock()
		cur := u.current
		u.leaseMu.Unlock()

		if cur == nil {
			// Nobody is playing this session right now. Frames that arrive
			// outside a game have nowhere to go.
			u.Logger.Debugw("frame received with no active lease", "user_id", u.UserID)
			continue
		}

		select {
		case cur.recv <- message:
		case <-u.closed:
			return
		default:
			// The reader is not keeping up. Dropping a keystroke is better
			// than stalling the socket reader, which would stop pings and take
			// the whole session down.
			u.Logger.Warnw("dropping frame, lease receiver is behind",
				"user_id", u.UserID, "queued", len(cur.recv))
		}
	}
}

// recvBuffer is how many client frames a lease buffers. Typing bursts are far
// smaller than this, so it only fills if the consumer has actually stalled.
const recvBuffer = 256

// outgoingBuffer is how many outbound frames can queue before a push is
// dropped. A burst here is a roster broadcast plus a notification, so this is
// generous; it exists to absorb a fan-out, not to buffer a backlog.
const outgoingBuffer = 64

// Ping keeps intermediaries from idling the connection out.
//
// It selects on closed so the loop ends with the session. The previous
// version ranged over a ticker with no exit, so every disconnect left a
// goroutine writing to a dead socket every five seconds for the life of the
// process.
func (u *UserSession) Ping() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-u.closed:
			return
		case <-ticker.C:
			u.PingLock.Lock()
			err := u.socket.WriteMessage(websocket.PingMessage, nil)
			u.PingLock.Unlock()
			if err != nil {
				u.UserOffline()
				return
			}
		}
	}
}

// pingInterval is how often a keepalive goes out on an otherwise idle socket.
const pingInterval = 20 * time.Second

// UserOffline tears the session down. It runs at most once.
//
// The outbound channel is deliberately not closed. Closing it made every
// concurrent Notify panic on a closed channel, which is reachable from any
// handler that decided to push a frame while the socket was dying. Closing
// the closed signal instead makes sendData return and turns a late Notify
// into an ordinary error.
func (u *UserSession) UserOffline() {
	u.offlineOnce.Do(func() {
		u.Logger.Infow("ending session", "user_id", u.UserID)

		close(u.closed)

		// Closing the transport releases the read and write goroutines
		// immediately rather than leaving them parked on a socket nobody is
		// going to answer.
		_ = u.socket.Close()

		// End any active lease so a game routine blocked on Recv wakes up and
		// returns rather than waiting out its whole round timer.
		u.leaseMu.Lock()
		cur := u.current
		u.current = nil
		u.leaseMu.Unlock()
		if cur != nil {
			cur.closeRecv()
		}

		if u.OnDisconnect != nil {
			u.OnDisconnect(u.UserID)
		}
	})
}
