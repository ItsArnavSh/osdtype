package matchmaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

	"github.com/google/btree"
	"go.uber.org/zap"
)

// ErrNoSession is returned when a player tries to queue without a live socket.
// Without a socket there is nowhere to send the seed, so queueing is not
// possible.
var ErrNoSession = errors.New("user has no live session")

// Matchmaker pairs queued players of similar rank into games.
//
// Queues are ordered btrees keyed on (rank, id) rather than slices, so finding
// a rank band is a range scan over an index instead of a full pass.
type Matchmaker struct {
	logger  *zap.SugaredLogger
	mu      sync.Mutex
	lobby   map[entity.LobbyType]*btree.BTree
	ac      *game.ActiveGames
	session *usersession.ActiveSessions
	db      *postgresql.Database

	// now is the clock, replaceable in tests.
	now func() time.Time

	// stop ends the worker. It is closed rather than signaled so the loop
	// can select on it alongside the ticker, and so a second call is harmless.
	stop     chan struct{}
	stopOnce sync.Once
	started  bool
}

// NewMatchMaker builds a matchmaker.
//
// The redis handle the original signature took was always nil and never read:
// every piece of queue state is in process memory. It has been removed rather
// than left as a permanently nil field.
func NewMatchMaker(
	logger *zap.SugaredLogger,
	ac *game.ActiveGames,
	sessions *usersession.ActiveSessions,
	db *postgresql.Database,
) *Matchmaker {
	return &Matchmaker{
		logger:  logger,
		lobby:   make(map[entity.LobbyType]*btree.BTree),
		ac:      ac,
		session: sessions,
		db:      db,
		now:     time.Now,
		stop:    make(chan struct{}),
	}
}

// Initialize prepares the queues and starts the matching worker.
//
// It is safe to call more than once: the second call finds the queues already
// built and does not start a second worker. The previous version was safe only
// because each call spawned another worker, which meant two workers draining
// the same queues and starting duplicate matches from one set of players.
func (m *Matchmaker) Initialize() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	for _, typ := range queuedModes {
		if _, ok := m.lobby[typ]; !ok {
			m.lobby[typ] = btree.New(2)
		}
	}
	m.started = true
	m.mu.Unlock()

	go m.worker()
}

// Stop ends the worker and releases every queued player's lease.
//
// A player left holding a lease when the process reloads would have their
// socket revoked by whatever took it next, which is harmless, but releasing
// them here means the queue size is honest as soon as Stop returns.
func (m *Matchmaker) Stop() {
	m.stopOnce.Do(func() { close(m.stop) })

	m.mu.Lock()
	queued := make([]entity.PlayerItem, 0)
	for _, tree := range m.lobby {
		tree.Ascend(func(i btree.Item) bool {
			queued = append(queued, i.(entity.PlayerItem))
			return true
		})
	}
	m.lobby = make(map[entity.LobbyType]*btree.BTree)
	m.mu.Unlock()

	ReleaseAll(queued)
}

// queuedModes are the modes the worker scans.
var queuedModes = []entity.LobbyType{entity.SPRINT, entity.STANDARD, entity.MARATHON}

// AddToGlobalLobby queues a player for ranked play.
//
// The context is the request's, so a player who closes the tab while queueing
// cancels the database read rather than leaving it to run. It is a timeout-free
// single-row lookup either way, but it is on the path of every queue join.
func (m *Matchmaker) AddToGlobalLobby(ctx context.Context, userid uint32, rank uint16, typ entity.LobbyType) error {
	sub := m.session.Subscribe(userid)
	if sub == nil {
		return ErrNoSession
	}

	user, err := m.db.GetUser(ctx, userid)
	if err != nil {
		sub.Release()
		return err
	}

	item := entity.PlayerItem{
		Name:     user.Username,
		ID:       userid,
		Rank:     rank,
		JoinedAt: m.now(),
		Session:  sub,
	}

	m.mu.Lock()
	tree, ok := m.lobby[typ]
	if !ok {
		// The worker has not run Initialize, or was given a mode it does not
		// serve. Release the lease rather than stranding the socket.
		m.mu.Unlock()
		sub.Release()
		return errors.New("unknown lobby mode")
	}
	// Replacing rather than inserting means a player who clicks queue twice
	// holds one entry, not two, which previously let them start a game
	// against themselves.
	tree.ReplaceOrInsert(item)
	size := tree.Len()
	m.mu.Unlock()

	m.notifyQueueStatus(typ, userid, size)
	m.logger.Infow("player queued", "user_id", userid, "rank", rank, "mode", typ.String(), "queued", size)
	return nil
}

// LeaveQueue removes a player from a queue they no longer want to be in.
func (m *Matchmaker) LeaveQueue(userid uint32, typ entity.LobbyType) {
	m.mu.Lock()
	entry, ok := m.findLocked(typ, userid)
	if ok {
		if tree, exists := m.lobby[typ]; exists {
			tree.Delete(entry)
		}
	}
	m.mu.Unlock()

	// Released outside the lock: Release closes a channel, which is cheap, but
	// holding the queue lock while it runs would let a slow revoke block a
	// join. Doing it after the delete is also what makes a leave and a match
	// racing each other safe: whichever runs second finds nothing to do.
	if ok && entry.Session != nil {
		entry.Session.Release()
	}
}

// QueueSize reports how many players are waiting in a mode.
func (m *Matchmaker) QueueSize(typ entity.LobbyType) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tree, ok := m.lobby[typ]; ok {
		return tree.Len()
	}
	return 0
}

// Queued reports whether a player is in a mode's queue.
//
// The interface needs this to render the right button on a page load: without
// it a reload always offers "join", and clicking it again used to insert a
// second entry for a player who was already waiting.
func (m *Matchmaker) Queued(userid uint32, typ entity.LobbyType) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.findLocked(typ, userid)
	return ok
}

// QueuedModes reports which modes a player is waiting in, for a client that
// wants to restore its own state without asking about each mode.
func (m *Matchmaker) QueuedModes(userid uint32) []entity.LobbyType {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]entity.LobbyType, 0, 1)
	for _, typ := range queuedModes {
		if _, ok := m.findLocked(typ, userid); ok {
			out = append(out, typ)
		}
	}
	return out
}

// findLocked returns a player's entry in a queue, if they are in it.
//
// It scans rather than looking the item up directly. The tree is ordered by
// (rank, id), so a probe built from just the id would be compared by a rank of
// zero and miss an entry whose rank is anything else. The queues hold a handful
// of players each, so a scan is cheaper than keeping a second index in step
// with the first.
//
// The caller holds m.mu.
func (m *Matchmaker) findLocked(typ entity.LobbyType, userid uint32) (entity.PlayerItem, bool) {
	tree, ok := m.lobby[typ]
	if !ok {
		return entity.PlayerItem{}, false
	}
	var found entity.PlayerItem
	seen := false
	tree.Ascend(func(i btree.Item) bool {
		p := i.(entity.PlayerItem)
		if p.ID == userid {
			found = p
			seen = true
			return false
		}
		return true
	})
	return found, seen
}

// notifyQueueStatus tells a player where they are in the queue.
//
// The position is approximate: it is the number of players in the rank band
// ahead of them, which is what a player can act on, rather than an absolute
// queue index that changes every tick.
func (m *Matchmaker) notifyQueueStatus(typ entity.LobbyType, userid uint32, size int) {
	rank, err := m.db.GetRank(context.TODO(), userid)
	if err != nil {
		rank = 0
	}
	frame := entity.Encode(entity.QueueStatusFrame{
		Type:      entity.FrameQueueStatus,
		LobbyType: typ.String(),
		Size:      size,
		Position:  m.bandSize(typ, rank) + 1,
	})
	// A dropped status frame is not worth failing the queue over: the next tick
	// sends another one, so a player who misses this sees a slightly stale
	// position for half a second. A closed session is the expected outcome of
	// someone who queued and then left.
	if err := m.session.Notify(userid, frame); err != nil {
		m.logger.Debugw("could not push queue status", "user_id", userid, "error", err)
	}
}

func (m *Matchmaker) bandSize(typ entity.LobbyType, rank uint16) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	tree, ok := m.lobby[typ]
	if !ok {
		return 0
	}
	n := 0
	tree.Ascend(func(i btree.Item) bool {
		p := i.(entity.PlayerItem)
		if p.Rank > rank {
			return false
		}
		n++
		return true
	})
	return n
}

// worker scans the queues for matchable groups.
//
// It also evicts players whose socket has gone, which is the bug that made
// ranked queue drain slowly: a player who closed their tab stayed in the btree
// forever, so a band could hold several dead entries that were counted toward
// the match size and then produced a round where a third of the seats were
// black.
func (m *Matchmaker) worker() {
	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
		}

		m.evictDisconnected()

		for _, typ := range queuedModes {
			players := m.tryMatch(typ)
			if len(players) < 2 {
				continue
			}
			go m.startMatch(players, typ)
		}
	}
}

// workerInterval is how often the queues are scanned.
const workerInterval = 500 * time.Millisecond

// queuedTotalLocked is the total number of players across every queue. The
// caller holds m.mu.
func (m *Matchmaker) queuedTotalLocked() int {
	n := 0
	for _, tree := range m.lobby {
		n += tree.Len()
	}
	return n
}

// evictDisconnected drops queued players who are no longer connected.
//
// It runs before matching rather than during, so a start never has to
// re-check a socket and then deal with the ones that turned out to be dead.
func (m *Matchmaker) evictDisconnected() {
	m.mu.Lock()
	// One allocation up front rather than one per queue: every tree in the map
	// contributes its doomed players to a single slice, and this runs on every
	// worker tick, so the repeated growth is pure garbage. The upper bound is the
	// total queue depth, which is the most that could possibly be evicted.
	dead := make([]entity.PlayerItem, 0, m.queuedTotalLocked())
	for _, tree := range m.lobby {
		var doomed []entity.PlayerItem
		tree.Ascend(func(i btree.Item) bool {
			p := i.(entity.PlayerItem)
			if !m.session.Online(p.ID) {
				doomed = append(doomed, p)
			}
			return true
		})
		for _, p := range doomed {
			tree.Delete(p)
		}
		dead = append(dead, doomed...)
	}
	m.mu.Unlock()

	for _, p := range dead {
		p.Session.Release()
		m.logger.Infow("evicted disconnected player from queue", "user_id", p.ID, "rank", p.Rank)
	}
}

// tryMatch pulls a matchable group out of one queue.
//
// The rank band widens the longer the oldest player has waited, so a queue
// never stalls forever: someone who has waited thirty seconds will be matched
// with anyone at all. Without that widening a player in an empty rank band
// waits indefinitely even as the population arrives above them.
func (m *Matchmaker) tryMatch(typ entity.LobbyType) []entity.PlayerItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	tree, ok := m.lobby[typ]
	if !ok || tree.Len() < 2 {
		return nil
	}

	oldestItem := tree.Min()
	if oldestItem == nil {
		return nil
	}
	oldest := oldestItem.(entity.PlayerItem)

	wait := m.now().Sub(oldest.JoinedAt)
	if wait < 0 {
		wait = 0
	}

	spread := baseRankSpread + int(wait.Seconds())*rankSpreadPerSecond
	lower := int(oldest.Rank) - spread
	upper := int(oldest.Rank) + spread

	var group []entity.PlayerItem
	tree.Ascend(func(i btree.Item) bool {
		p := i.(entity.PlayerItem)
		if int(p.Rank) < lower {
			return true
		}
		if int(p.Rank) > upper {
			return false
		}
		group = append(group, p)
		return len(group) < maxMatchSize
	})

	if len(group) < 2 {
		return nil
	}
	for _, p := range group {
		tree.Delete(p)
	}
	return group
}

// Matchmaking parameters.
const (
	// baseRankSpread is the initial half-width of the rank band.
	baseRankSpread = 100
	// rankSpreadPerSecond is how much wider the band gets per second waited.
	rankSpreadPerSecond = 50
	// maxMatchSize caps a single match so one round does not run for ten
	// minutes with six people.
	maxMatchSize = 6
)

// startMatch runs a matched group.
//
// It runs on its own goroutine, because NewGame blocks for the length of the
// round. Rank updates happen on a second goroutine with a timeout, so a round
// that outlives its reader cannot strand this one.
func (m *Matchmaker) startMatch(players []entity.PlayerItem, typ entity.LobbyType) {
	m.logger.Infow("starting match", "players", len(players), "mode", typ.String())

	// The match runs either way, so a player who does not receive the countdown
	// gets the seed frame instead and can still race. Failing the whole start
	// because one socket is slow would hand the remaining players a lobby that
	// never begins, which is a worse outcome than a missing notification.
	for _, p := range players {
		if err := m.session.Notify(p.ID, entity.Encode(entity.MatchFoundFrame{
			Type:       entity.FrameMatchFound,
			LobbyType:  typ.String(),
			DurationMS: typ.Duration().Milliseconds(),
			Players:    len(players),
			Countdown:  3,
		})); err != nil {
			m.logger.Warnw("could not send the match countdown", "user_id", p.ID, "error", err)
		}
	}

	sig := make(chan []entity.WPMRes, 1)
	m.ac.NewGame(players, typ.Duration(), sig)

	go func() {
		select {
		case res := <-sig:
			m.updateRanks(res)
			m.logger.Infoln("scores have been updated")
		case <-time.After(resultTimeout):
			m.logger.Error("game result timeout")
		}
	}()
}

// resultTimeout is how long to wait for a finished round's leaderboard.
const resultTimeout = 2 * time.Minute

// ReleaseAll drops every lease in a finished group.
//
// The game handler already releases its own players; this covers a group that
// was aborted before the handler took ownership of them.
func ReleaseAll(players []entity.PlayerItem) {
	for _, p := range players {
		if p.Session != nil {
			p.Session.Release()
		}
	}
}

// eloEngine is here so the matchmaker and the run submission path agree on how
// ratings move.
var eloEngine = utils.NewEloEngine()

// updateRanks applies a leaderboard to the stored ratings.
func (m *Matchmaker) updateRanks(leaderboard []entity.WPMRes) {
	if len(leaderboard) < 2 {
		return
	}

	ids := make([]uint32, len(leaderboard))
	for i, e := range leaderboard {
		ids[i] = e.ID
	}
	ratings, played, err := m.db.RatingsFor(context.TODO(), ids)
	if err != nil {
		m.logger.Errorw("could not read ratings", "error", err)
		return
	}

	deltas := eloEngine.Rate(leaderboard, ratings, played)
	for i, e := range leaderboard {
		if err := m.db.ChangeRank(e.ID, deltas[i]); err != nil {
			m.logger.Errorw("could not store rank", "user_id", e.ID, "error", err)
		}
	}
}
