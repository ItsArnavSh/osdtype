package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"golang.org/x/crypto/bcrypt"
)

// Handles everything related to friends and friendly matches.

// Errors the friend and lobby surface can report.
var (
	// ErrSelfFollow is returned when a player tries to follow themselves.
	ErrSelfFollow = errors.New("you cannot follow yourself")
	// ErrLobbyPassword is returned when a lobby password is wrong or missing.
	ErrLobbyPassword = errors.New("that lobby's password is not correct")
	// ErrLobbyClosed is returned when a lobby is already running.
	ErrLobbyClosed = errors.New("that lobby has already started")
)

// inviteTTL is how long an invitation stays actionable. A lobby is not a
// permanent invitation: the code is the durable part, and this is the nudge.
const inviteTTL = 2 * time.Minute

// FollowUser makes one player's friend list include another's.
//
// It is idempotent. The friendship table has a composite primary key of
// (a, b), so following someone twice raised a duplicate key violation and the
// endpoint answered 500 for what is a normal thing to click twice.
func (s *ServiceLayer) FollowUser(ctx context.Context, follower, following uint32) error {
	if follower == following {
		return ErrSelfFollow
	}
	return s.db.FollowUser(ctx, follower, following)
}

// UnfollowUser removes a friendship.
func (s *ServiceLayer) UnfollowUser(ctx context.Context, follower, following uint32) error {
	if follower == following {
		return ErrSelfFollow
	}
	return s.db.UnfollowUser(ctx, follower, following)
}

// Friends returns a player's friends, resolved in one query and annotated with
// who is online.
//
// The previous version did not exist as a route at all: GetFriends was
// implemented and never called, and SearchUsers had no route, so there was no
// way in the product to find anyone, let alone invite them.
func (s *ServiceLayer) Friends(ctx context.Context, userID uint32) ([]entity.FriendCard, error) {
	rows, err := s.db.ListFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]entity.FriendCard, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.friendCard(r))
	}
	return out, nil
}

// friendCard decorates a friendship row with what the session registry knows
// about the other person.
func (s *ServiceLayer) friendCard(r entity.Friendship) entity.FriendCard {
	status := s.core.Sessions.GetStatus(r.FriendID)
	return entity.FriendCard{
		UserID:    r.FriendID,
		Username:  r.Username,
		AvatarURL: r.AvatarURL,
		Rating:    r.Rank,
		Tier:      utils.TierOf(r.Rank).Name,
		Online:    status != entity.OFFLINE,
		Status:    status,
		// The table stores a mutual pair as two rows, so both directions
		// being present is what distinguishes a friend from someone merely
		// followed.
		Friends: r.Mutual,
	}
}

// SearchUsers finds players by name, annotated with who is online.
//
// Results are ranked so an exact name match comes first, because searching for
// a friend whose name you already know and having to scroll past a hundred
// people who merely start with the same two letters is a bad experience.
func (s *ServiceLayer) SearchUsers(ctx context.Context, query string) ([]entity.UserSearchResult, error) {
	query = strings.TrimSpace(query)
	if len(query) < minSearchLen {
		return []entity.UserSearchResult{}, nil
	}

	rows, err := s.db.SearchPeople(ctx, query, searchLimit)
	if err != nil {
		return nil, err
	}

	out := make([]entity.UserSearchResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, entity.UserSearchResult{
			UserID:    r.ID,
			Username:  r.Username,
			AvatarURL: r.AvatarURL,
			Rating:    r.CurrentRank,
			Tier:      utils.TierOf(r.CurrentRank).Name,
			Online:    s.core.Sessions.GetStatus(r.ID) != entity.OFFLINE,
		})
	}
	return out, nil
}

// Search parameters.
const (
	// minSearchLen is the shortest query that can return anything. One
	// character would match most of the user table.
	minSearchLen = 2
	// searchLimit is how many results a search returns.
	searchLimit = 20
)

// CreateLobby opens a private game lobby for the caller and returns it.
//
// The share code is the primary key of the lobby: six characters the invitees
// can read aloud, with an optional password as a separate lock. That is the
// Among Us model, and it is the right one here because the code alone is short
// enough to survive being read off a screen badly.
//
// The caller is made a mod, so they can start the game and invite people.
func (s *ServiceLayer) CreateLobby(ctx context.Context, from uint32, req LobbyRequest) (entity.Room, error) {
	if !s.core.Sessions.Online(from) {
		return entity.Room{}, ErrNoLiveSession
	}

	mode := req.Mode
	if _, ok := entity.LobbyTypeFromString(req.ModeName); ok {
		mode, _ = entity.LobbyTypeFromString(req.ModeName)
	}

	hash := ""
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return entity.Room{}, err
		}
		hash = string(hashed)
	}

	room := entity.Room{
		Name:        strings.TrimSpace(req.Name),
		Desc:        strings.TrimSpace(req.Desc),
		Public:      visibilityOf(req.Public),
		Capacity:    clampCapacity(req.Capacity),
		Duration:    mode,
		CreatorID:   from,
		PasswordSet: hash != "",
	}
	room.ID = s.int_gen.GenerateID()
	room.PasswordHash = hash

	if room.Name == "" {
		room.Name = defaultLobbyName
	}

	if err := s.db.CreateRoom(ctx, &room); err != nil {
		return entity.Room{}, err
	}
	// Capacity zero means the host is never turned away, for the same reason as
	// in the lobby package: the room is empty at this point, so applying the
	// limit here could only convert a successful creation into a failure.
	if _, err := s.db.AddRosterMember(ctx, room.ID, from, entity.MOD, 0); err != nil {
		return entity.Room{}, err
	}

	s.logger.Infow("lobby created", "room_id", room.ID, "code", room.Code, "host", from, "mode", mode.String())
	return room, nil
}

// defaultLobbyName is used when a host does not name their lobby, so the room
// list has something to show.
const defaultLobbyName = "Private Lobby"

// LobbyRequest is what a create-lobby call asks for.
type LobbyRequest struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Password string `json:"password"`
	Mode     entity.LobbyType
	ModeName string `json:"mode"`
	Capacity int    `json:"capacity"`
	// Public opens the lobby to anyone who finds it. A private lobby is the
	// default, because that is what a share code implies.
	Public bool `json:"public"`
}

func visibilityOf(public bool) entity.RoomVisibility {
	if public {
		return entity.PUBLIC
	}
	return entity.PRIVATE
}

// Lobby capacity bounds.
const (
	// defaultLobbyCapacity matches the matchmaker's maximum match size.
	defaultLobbyCapacity = 6
	// maxLobbyCapacity is the ceiling. A round is one shared snippet and one
	// shared countdown, so a very large lobby makes it unwinnable for everyone.
	maxLobbyCapacity = 12
)

func clampCapacity(n int) int {
	if n <= 0 {
		return defaultLobbyCapacity
	}
	if n > maxLobbyCapacity {
		return maxLobbyCapacity
	}
	return n
}

// JoinLobby puts a player into a lobby, by share code.
//
// The password is checked here rather than in the handler so that every path
// into a lobby, including the invitation's deep link, goes through the same
// check. An empty password field on a lobby that has no password is not an
// error, which is what lets a public lobby be joined with just a code.
func (s *ServiceLayer) JoinLobby(ctx context.Context, userID uint32, code, password string) (entity.Room, error) {
	if !s.core.Sessions.Online(userID) {
		return entity.Room{}, ErrNoLiveSession
	}

	room, err := s.db.RoomByCode(ctx, normalizeCode(code))
	if err != nil {
		return entity.Room{}, ErrRoomNotFound
	}

	if err := checkPassword(room, password); err != nil {
		return entity.Room{}, err
	}

	if room.Status != entity.LobbyOpen {
		return entity.Room{}, ErrLobbyClosed
	}

	if err := s.core.ManualLobby.Join(ctx, room, userID); err != nil {
		return entity.Room{}, err
	}

	s.logger.Infow("player joined lobby", "user_id", userID, "room_id", room.ID, "code", room.Code)
	return room, nil
}

// checkPassword validates a candidate password against a lobby's hash.
func checkPassword(room entity.Room, candidate string) error {
	if room.PasswordHash == "" {
		return nil
	}
	if candidate == "" {
		return ErrLobbyPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(room.PasswordHash), []byte(candidate)); err != nil {
		return ErrLobbyPassword
	}
	return nil
}

// normalizeCode upper-cases and trims a typed code.
//
// Share codes get retyped from screenshots and read out loud, so being strict
// about case would only produce support questions. The code is matched with
// UPPER() on the database side too, so this is a courtesy rather than the
// thing making it work.
func normalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// ErrRoomNotFound is returned when a share code does not resolve.
var ErrRoomNotFound = errors.New("no lobby with that code")

// LeaveLobby removes a player from a lobby.
func (s *ServiceLayer) LeaveLobby(ctx context.Context, userID, roomID uint32) error {
	if _, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: roomID, UserID: userID}); err != nil {
		return err
	}
	if err := s.core.ManualLobby.Leave(ctx, roomID, userID); err != nil {
		return err
	}
	s.logger.Infow("player left lobby", "user_id", userID, "room_id", roomID)
	return nil
}

// InviteToLobby pushes a lobby invitation to a player.
//
// The invitation is written to their inbox first and pushed second, so a player
// whose tab is closed still finds it waiting. That ordering matters here
// specifically: an invitation is the one notification a person is likely to
// miss, because they were not looking at the game when it was sent.
//
// The inviter's name and the lobby's share code both travel with it, so the
// invitee can join without asking anyone to repeat the code.
func (s *ServiceLayer) InviteToLobby(ctx context.Context, inviter, invitee, roomID uint32) error {
	from, err := s.db.GetUser(ctx, inviter)
	if err != nil {
		return err
	}
	room, err := s.db.RoomByID(ctx, roomID)
	if err != nil {
		return err
	}
	if invitee == inviter {
		return errors.New("you are already in it")
	}

	return s.core.ManualLobby.Invite(ctx, from, room, invitee, s.db, inviteTTL)
}

// Lobby is a lobby as the interface needs it: the room, its share code, the
// roster, and whether the caller may start a game.
type Lobby struct {
	Room    entity.Room         `json:"room"`
	Members []entity.RoomMember `json:"members"`
	// IsHost is whether the caller may start the game. Only a mod can.
	IsHost bool `json:"is_host"`
	// Joined is whether the caller is on the roster.
	Joined bool `json:"joined"`
	// CanStart is the same as IsHost and at least two players are connected.
	CanStart bool `json:"can_start"`
	// HasPassword is whether a joiner will be asked for a password. The hash
	// itself is never sent.
	HasPassword bool `json:"has_password"`
}

// LobbyView assembles the lobby page for one caller.
func (s *ServiceLayer) LobbyView(ctx context.Context, caller, roomID uint32) (Lobby, error) {
	room, err := s.db.RoomByID(ctx, roomID)
	if err != nil {
		return Lobby{}, err
	}
	return s.lobbyViewFrom(ctx, caller, room)
}

// LobbyByCodeView assembles the lobby page for a caller who arrived with a
// code rather than an id, which is the invitation's path.
func (s *ServiceLayer) LobbyByCodeView(ctx context.Context, caller uint32, code, password string) (Lobby, error) {
	room, err := s.db.RoomByCode(ctx, normalizeCode(code))
	if err != nil {
		return Lobby{}, ErrRoomNotFound
	}
	if err := checkPassword(room, password); err != nil {
		return Lobby{}, err
	}
	return s.lobbyViewFrom(ctx, caller, room)
}

func (s *ServiceLayer) lobbyViewFrom(ctx context.Context, caller uint32, room entity.Room) (Lobby, error) {
	members, err := s.core.ManualLobby.Members(ctx, room.ID)
	if err != nil {
		return Lobby{}, err
	}

	online := 0
	joined := false
	isHost := false
	for _, m := range members {
		if m.Online {
			online++
		}
		if m.UserID == caller {
			joined = true
			isHost = m.IsMod
		}
	}

	return Lobby{
		Room:        room,
		Members:     members,
		IsHost:      isHost,
		Joined:      joined,
		CanStart:    isHost && online >= 2 && room.Status == entity.LobbyOpen,
		HasPassword: room.PasswordSet,
	}, nil
}

// StartLobby runs a lobby as a game.
//
// It returns as soon as the round is under way rather than when it ends: the
// caller gets their seed and countdown over the socket, and a HTTP request
// that blocked for the length of a round would time out on every start.
//
// Only a mod can start, and only while the lobby is open. A second start while
// one is in flight is refused, which is what a host double-clicking produced.
func (s *ServiceLayer) StartLobby(ctx context.Context, caller, roomID uint32) error {
	room, err := s.db.RoomByID(ctx, roomID)
	if err != nil {
		return err
	}

	perm, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: roomID, UserID: caller})
	if err != nil {
		return errors.New("you are not in that lobby")
	}
	if perm.Perm != entity.MOD {
		return errors.New("only the host can start the game")
	}
	if room.Status != entity.LobbyOpen {
		return ErrLobbyClosed
	}

	// Mark the room before launching, so a second start sees a room that is no
	// longer open even if the round has not produced its first frame yet.
	if err := s.db.SetRoomStatus(ctx, roomID, entity.LobbyRunning); err != nil {
		return err
	}

	sig := make(chan []entity.WPMRes, 1)
	go s.finishLobby(ctx, roomID, sig)

	if err := s.core.ManualLobby.StartAsync(ctx, roomID, room.Duration.Duration(), sig); err != nil {
		_ = s.db.SetRoomStatus(ctx, roomID, entity.LobbyOpen)
		return err
	}

	s.logger.Infow("lobby started", "room_id", roomID, "host", caller, "mode", room.Duration.String())
	return nil
}
