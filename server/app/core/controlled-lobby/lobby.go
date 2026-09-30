package controlledlobby

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// The roster lives in the database, not in memory.
//
// It used to be a map here, which meant a share-link room vanished on restart
// and, more importantly, that nothing could show who was in it: a player
// joining a room had no way to find out whether anyone was already waiting.
// Room membership is already modeled by room_users, so the roster is a query.
//
// The split is: the database owns identity, and this package owns sockets.
// A game starts by reading the membership and resolving each member to their
// live session at that moment.

// Errors returned by lobby operations.
var (
	// ErrNoSession means the caller has no live socket, so there is nowhere to
	// send a seed and they cannot take part.
	ErrNoSession = errors.New("no live session for this user")
	// ErrBadCode means the share code did not resolve to a room.
	ErrBadCode = errors.New("no room with that code")
	// ErrBadPassword means a password was required and was not supplied.
	ErrBadPassword = errors.New("incorrect lobby password")
)

// ErrRoomFull and ErrRoomBlocked are the store's sentinels rather than this
// package's, because the transaction that raises them is in the persistence
// layer. They are aliased so errors.Is matches either spelling and a caller
// does not have to know which package happened to produce it.
var (
	ErrRoomFull    = entity.ErrRoomFull
	ErrRoomBlocked = entity.ErrRoomBlocked
)

// defaultCapacity is how many players a lobby holds unless the host says
// otherwise. It matches the matchmaker's maximum match size.
const defaultCapacity = 6

// maxCapacity is the ceiling. A round is one shared snippet and a shared
// countdown, so a very large room makes the game unwinnable for everyone.
const maxCapacity = 12

// Store is the slice of the database a lobby needs. Declaring it here keeps
// this package independent of the persistence layer, which would otherwise be
// an import cycle through core.
type Store interface {
	// Roster returns a room's active members: anyone who is a member and
	// neither blocked nor left.
	Roster(ctx context.Context, roomID uint32) ([]entity.RosterMember, error)
	// AddRosterMember adds a member, reporting whether the roster changed.
	// The capacity check belongs inside it, atomically with the insert; a
	// caller that checks first and inserts second gets a room of unbounded
	// size the moment two joins overlap.
	AddRosterMember(ctx context.Context, roomID, userID uint32, perm entity.RoomPerm, capacity int) (added bool, err error)
	// RemoveRosterMember removes a member entirely.
	RemoveRosterMember(ctx context.Context, roomID, userID uint32) error
	// RoomMembers returns the members along with whether each is online and
	// what their rank is.
	RoomMembers(ctx context.Context, roomID uint32) ([]entity.RoomMember, error)
	// CreateRoom stores a room and fills in the id and share code it mints.
	// It takes a pointer because the room comes back changed; a by-value
	// signature would silently hand the caller an empty code.
	CreateRoom(ctx context.Context, room *entity.Room) error
}

// ControlledLobby starts games for a room: private lobbies that friends join,
// and contests that the scheduler runs.
type ControlledLobby struct {
	logger  *zap.SugaredLogger
	ac      *game.ActiveGames
	session *usersession.ActiveSessions
	store   Store

	// starting guards against the same room being started twice, which
	// happened whenever a host pressed start while a contest's scheduled start
	// was also firing.
	starting sync.Mutex
	inFlight map[uint32]bool
}

// NewControlledLobby builds a lobby manager.
func NewControlledLobby(logger *zap.SugaredLogger, ac *game.ActiveGames, session *usersession.ActiveSessions, store Store) *ControlledLobby {
	return &ControlledLobby{
		logger:   logger,
		ac:       ac,
		session:  session,
		store:    store,
		inFlight: make(map[uint32]bool),
	}
}

// CreateGame opens a new lobby and returns it with its share code filled in.
//
// The share code is minted by the database layer inside CreateRoom, so this
// does not have to retry a collision itself.
func (c *ControlledLobby) CreateGame(ctx context.Context, room entity.Room) (entity.Room, error) {
	if room.Public != entity.PUBLIC {
		room.Public = entity.PRIVATE
	}
	if room.Capacity <= 0 {
		room.Capacity = defaultCapacity
	}
	if room.Capacity > maxCapacity {
		room.Capacity = maxCapacity
	}
	room.Status = entity.LobbyOpen

	if err := c.store.CreateRoom(ctx, &room); err != nil {
		return entity.Room{}, err
	}

	// The creator is a member and a mod, so they can invite and start.
	//
	// The capacity passed here is zero, meaning unlimited, and deliberately so:
	// the host is inserted into a room that was empty a moment ago, so the check
	// could only ever turn a successful creation into a failure if a stale or
	// hand-edited capacity row said the room held nobody. A host who creates a
	// one-player room has to end up in it.
	if _, err := c.store.AddRosterMember(ctx, room.ID, room.CreatorID, entity.MOD, 0); err != nil {
		return entity.Room{}, err
	}

	c.logger.Infow("lobby created", "room_id", room.ID, "code", room.Code, "creator", room.CreatorID)
	return room, nil
}

// Join puts a player into a room's roster.
//
// The capacity check lives in the store, not here. Reading the roster here,
// comparing it to the capacity, and then calling in to insert is three
// operations, and forty simultaneous join requests all read the same size
// before any of them wrote — so all forty passed and a six-player lobby took
// ten. The store counts and inserts under a lock on the room row, which makes
// "at most capacity" true of the data rather than true of the timing.
func (c *ControlledLobby) Join(ctx context.Context, room entity.Room, userID uint32) error {
	// Joining needs a live socket, but not exclusive ownership of it: the game
	// does not start until someone presses start, and taking a lease here
	// would revoke whatever the player is currently doing elsewhere.
	if !c.session.Online(userID) {
		c.logger.Warnw("player tried to join without a live session", "user_id", userID, "room_id", room.ID)
		return ErrNoSession
	}

	added, err := c.store.AddRosterMember(ctx, room.ID, userID, entity.MEMBER, effectiveCapacity(room))
	if err != nil {
		return err
	}
	if !added {
		// Already on the roster. Not an error, and deliberately not a
		// re-broadcast: a client that retried after a timeout should not make
		// everybody else's roster frame arrive again.
		return nil
	}

	c.session.UpdateSession(userID, entity.PLAYING)
	c.broadcastRoster(ctx, room.ID)
	return nil
}

// effectiveCapacity is the room's capacity with the server's bounds applied, so
// a stored value from a bug or a hand-edited row cannot produce a room of four
// hundred or one of nobody.
func effectiveCapacity(room entity.Room) int {
	switch {
	case room.Capacity <= 0:
		return defaultCapacity
	case room.Capacity > maxCapacity:
		return maxCapacity
	default:
		return room.Capacity
	}
}

// Leave removes a player from a room.
func (c *ControlledLobby) Leave(ctx context.Context, roomID, userID uint32) error {
	if err := c.store.RemoveRosterMember(ctx, roomID, userID); err != nil {
		return err
	}
	c.session.UpdateSession(userID, entity.AVAILABLE)
	c.broadcastRoster(ctx, roomID)
	return nil
}

// Members returns a room's roster, ordered for display.
//
// The online flag comes from the session registry rather than the database,
// because presence is the one thing the database cannot answer. The ordering is
// mods first, then by rating, so a host sees themselves at the top of their
// own lobby.
func (c *ControlledLobby) Members(ctx context.Context, roomID uint32) ([]entity.RoomMember, error) {
	members, err := c.store.RoomMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	c.markOnline(members)
	return sortedMembers(members), nil
}

// markOnline fills in each member's presence from the session registry.
//
// The database cannot answer this, so the flag is applied here rather than in
// the query. Doing it in one place means the roster endpoint, the live roster
// broadcast and the start check all agree on who is connected, which is what
// stops "you need two players" disagreeing with the player list.
func (c *ControlledLobby) markOnline(members []entity.RoomMember) {
	for i := range members {
		members[i].Online = c.session.Online(members[i].UserID)
	}
}

// Start runs a room as a game and returns the leaderboard.
//
// It resolves the roster to live sessions here, which is the bridge between
// the durable membership and the sockets. Members who are not connected are
// skipped rather than failing the start, because a friend closing their laptop
// should not cancel the game for everyone else.
func (c *ControlledLobby) Start(ctx context.Context, roomID uint32, duration time.Duration, sig chan []entity.WPMRes) ([]entity.WPMRes, error) {
	c.starting.Lock()
	if c.inFlight[roomID] {
		c.starting.Unlock()
		return nil, errors.New("this room is already starting")
	}
	c.inFlight[roomID] = true
	c.starting.Unlock()

	defer func() {
		c.starting.Lock()
		delete(c.inFlight, roomID)
		c.starting.Unlock()
	}()

	items, err := c.players(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if len(items) < 2 {
		return nil, errors.New("a game needs at least two connected players")
	}

	c.ac.NewGame(items, duration, sig)
	return nil, nil
}

// StartAsync runs a room on its own goroutine and hands the leaderboard to
// sig. This is what the scheduler and the HTTP start route both use, because
// a round blocks for its whole duration.
func (c *ControlledLobby) StartAsync(ctx context.Context, roomID uint32, duration time.Duration, sig chan []entity.WPMRes) error {
	go func() {
		if _, err := c.Start(ctx, roomID, duration, sig); err != nil {
			c.logger.Errorw("could not start room", "room_id", roomID, "error", err)
			select {
			case sig <- nil:
			default:
			}
		}
	}()
	return nil
}

// players resolves a room's membership into matchable players.
func (c *ControlledLobby) players(ctx context.Context, roomID uint32) ([]entity.PlayerItem, error) {
	members, err := c.store.Roster(ctx, roomID)
	if err != nil {
		return nil, err
	}

	items := make([]entity.PlayerItem, 0, len(members))
	for _, m := range members {
		sub := c.session.Subscribe(m.UserID)
		if sub == nil {
			c.logger.Infow("skipping offline member", "user_id", m.UserID, "room_id", roomID)
			continue
		}
		items = append(items, entity.PlayerItem{
			ID:      m.UserID,
			Name:    m.Username,
			Rank:    m.Rank,
			Session: sub,
		})
	}
	return items, nil
}

// Invite pushes a lobby invitation to a player, over the socket if they are
// connected and into their inbox either way.
func (c *ControlledLobby) Invite(ctx context.Context, from entity.User, room entity.Room, invitee uint32, store usersession.NotificationStore, ttl time.Duration) error {
	invite := entity.LobbyInvite{
		From:      from.Username,
		FromID:    from.ID,
		RoomID:    room.ID,
		RoomCode:  room.Code,
		RoomName:  room.Name,
		ExpiresAt: time.Now().Add(ttl).UnixMilli(),
	}

	if err := c.session.Deliver(ctx, store, invitee, entity.KindLobbyInvite, invite); err != nil {
		return err
	}
	c.logger.Infow("invitation delivered", "from", from.ID, "to", invitee, "room_id", room.ID)
	return nil
}

// WarnContest tells a room's members a contest is about to run.
func (c *ControlledLobby) WarnContest(ctx context.Context, roomID, jobID uint32, seconds int) {
	members, err := c.store.RoomMembers(ctx, roomID)
	if err != nil {
		c.logger.Errorw("could not read roster to warn it", "room_id", roomID, "error", err)
		return
	}
	frame := entity.Encode(entity.ContestStartingFrame{
		Type:    entity.FrameContestStarting,
		JobID:   jobID,
		Seconds: seconds,
	})
	c.notifyEach(members, frame)
}

// notifyEach pushes one frame to every online member of a room.
//
// A member whose socket has gone is skipped rather than treated as a failure of
// the whole broadcast, and the error is logged at debug because it is the
// expected outcome of someone closing their laptop mid-round. ErrSessionBusy is
// the one that matters: it means the writer is behind and a frame was genuinely
// dropped, and the client recovers on its next roster frame.
func (c *ControlledLobby) notifyEach(members []entity.RoomMember, frame []byte) {
	for _, m := range members {
		if !m.Online {
			continue
		}
		if err := c.session.Notify(m.UserID, frame); err != nil {
			c.logger.Debugw("could not push a room frame", "user_id", m.UserID, "error", err)
		}
	}
}

// broadcastRoster pushes the current roster to everyone in the room, so the
// member list updates live instead of only on a page reload.
func (c *ControlledLobby) broadcastRoster(ctx context.Context, roomID uint32) {
	members, err := c.Members(ctx, roomID)
	if err != nil {
		c.logger.Errorw("could not read roster", "room_id", roomID, "error", err)
		return
	}

	frame := entity.Encode(entity.RosterFrame{
		Type:    entity.FrameRoster,
		RoomID:  roomID,
		Members: members,
	})
	c.notifyEach(members, frame)
}

// NormalizeCode upper-cases a share code and trims whitespace, so a code
// typed in lowercase or with a stray space still resolves.
//
// Among Us style codes are case insensitive in practice because people retype
// them from a screenshot, and being strict about it just produces support
// questions.
func NormalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// ValidateCode reports whether a string has the shape of a share code.
func ValidateCode(code string) bool {
	if len(code) != utils.CodeLength {
		return false
	}
	for _, r := range code {
		if !utils.CodeAlphabet[r] {
			return false
		}
	}
	return true
}

// ParseRoomID parses a room id from a query parameter.
func ParseRoomID(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// sortedMembers orders a roster for display: mods first, then by rank.
func sortedMembers(members []entity.RoomMember) []entity.RoomMember {
	out := make([]entity.RoomMember, len(members))
	copy(out, members)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsMod != out[j].IsMod {
			return out[i].IsMod
		}
		if out[i].Rank != out[j].Rank {
			return out[i].Rank > out[j].Rank
		}
		return out[i].UserID < out[j].UserID
	})
	return out
}
