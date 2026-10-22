package services

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"osdtyp/app/core/bots"
	gamecore "osdtyp/app/core/game"
	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const roomAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type CreateRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	Password    string `json:"password"`
	MaxPlayers  int    `json:"max_players"`
	Duration    int    `json:"duration"`
	Language    string `json:"language"`
}

type UpdateRoomRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
	Password    *string `json:"password"`
	MaxPlayers  *int    `json:"max_players"`
	Duration    *int    `json:"duration"`
	Language    *string `json:"language"`
}

type StartGameRequest struct {
	Duration int    `json:"duration"`
	Language string `json:"language"`
}

func validLanguage(lang string) bool {
	for _, l := range entity.Languages {
		if l == lang {
			return true
		}
	}
	return lang == "" || lang == "random"
}

func (s *ServiceLayer) mustLeaveCurrent(uid uint32) error {
	if _, ok := s.g().IsInGame(uid); ok {
		return errf(ErrConflict, "finish or leave your current game first")
	}
	if _, ok, _ := s.db.ActiveRoomForUser(context.Background(), uid); ok {
		return errf(ErrConflict, "leave your current room first")
	}
	return nil
}

func newRoomCode() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("R%05d", time.Now().UnixNano()%100000)
	}
	out := make([]byte, 6)
	for i, b := range buf {
		out[i] = roomAlphabet[int(b)%len(roomAlphabet)]
	}
	return string(out)
}

func randomSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint64(time.Now().UnixNano()) | 1
	}
	// Postgres and sqlite BIGINT are signed; the server-side seed therefore
	// uses 63 bits. Browser-generated seeds may still be full u64 values when
	// they are sent to the generator.
	seed := binary.BigEndian.Uint64(b[:]) & ((1 << 63) - 1)
	if seed == 0 {
		seed = 1
	}
	return seed
}

func (s *ServiceLayer) CreateRoom(ctx context.Context, uid uint32, req CreateRoomRequest) (entity.RoomView, error) {
	if err := s.mustLeaveCurrent(uid); err != nil {
		return entity.RoomView{}, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 64 {
		return entity.RoomView{}, errf(ErrBadRequest, "room name must be 1-64 characters")
	}
	if len(req.Description) > 280 {
		return entity.RoomView{}, errf(ErrBadRequest, "description is too long")
	}
	if req.MaxPlayers == 0 {
		req.MaxPlayers = 8
	}
	if req.MaxPlayers < 2 || req.MaxPlayers > 16 {
		return entity.RoomView{}, errf(ErrBadRequest, "max players must be between 2 and 16")
	}
	if req.Duration == 0 {
		req.Duration = 90
	}
	if _, ok := entity.LobbyTypeFromSeconds(req.Duration); !ok {
		return entity.RoomView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300 seconds")
	}
	if req.Language == "" {
		req.Language = "go"
	}
	if !validLanguage(req.Language) || req.Language == "random" {
		req.Language = "go"
	}
	if len(req.Password) > 72 {
		return entity.RoomView{}, errf(ErrBadRequest, "password is too long")
	}

	hash := ""
	if req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return entity.RoomView{}, err
		}
		hash = string(h)
	}

	var code string
	for i := 0; i < 8; i++ {
		code = newRoomCode()
		if _, err := s.db.GetRoomByCode(ctx, code); errors.Is(err, gorm.ErrRecordNotFound) {
			break
		}
	}

	room := entity.Room{
		ID:           s.intGen.GenerateID(),
		Code:         code,
		Name:         req.Name,
		Description:  req.Description,
		IsPublic:     req.IsPublic,
		PasswordHash: hash,
		MaxPlayers:   req.MaxPlayers,
		HostID:       uid,
		Status:       entity.RoomLobby,
		Duration:     req.Duration,
		Language:     req.Language,
	}
	if err := s.db.CreateRoom(ctx, &room); err != nil {
		return entity.RoomView{}, err
	}
	if err := s.db.AddMember(ctx, &entity.RoomMember{
		RoomID: room.ID, UserID: uid, Role: entity.RoleOwner, JoinedAt: time.Now().UTC(),
	}); err != nil {
		_ = s.db.CloseRoom(ctx, room.ID)
		return entity.RoomView{}, err
	}
	s.state().SetRoom(ctx, uid, room.Code)
	return s.roomView(ctx, room, uid, true), nil
}

func (s *ServiceLayer) GetRoom(ctx context.Context, uid uint32, code string) (entity.RoomView, error) {
	room, err := s.db.GetRoomByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.RoomView{}, errf(ErrNotFound, "room not found")
		}
		return entity.RoomView{}, err
	}
	banned, _ := s.db.IsBanned(ctx, room.ID, uid)
	if banned {
		return entity.RoomView{}, errf(ErrBanned, "you are banned from this room")
	}
	if room.Status == entity.RoomClosed {
		return entity.RoomView{}, errf(ErrNotFound, "room is closed")
	}
	_, memberErr := s.db.GetMember(ctx, room.ID, uid)
	include := memberErr == nil || room.IsPublic
	return s.roomView(ctx, room, uid, include), nil
}

func (s *ServiceLayer) ListRooms(ctx context.Context, uid uint32, mine bool, offset, limit int) ([]entity.RoomView, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var rooms []entity.Room
	var err error
	if mine {
		rooms, err = s.db.ListRoomsForUser(ctx, uid, offset, limit)
	} else {
		rooms, err = s.db.ListPublicRooms(ctx, offset, limit)
	}
	if err != nil {
		return nil, err
	}
	out := make([]entity.RoomView, 0, len(rooms))
	for _, room := range rooms {
		out = append(out, s.roomView(ctx, room, uid, false))
	}
	return out, nil
}

func (s *ServiceLayer) JoinRoom(ctx context.Context, uid uint32, code, password string) (entity.RoomView, error) {
	room, err := s.db.GetRoomByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.RoomView{}, errf(ErrNotFound, "room not found")
		}
		return entity.RoomView{}, err
	}
	if room.Status == entity.RoomClosed {
		return entity.RoomView{}, errf(ErrNotFound, "room is closed")
	}
	if member, err := s.db.GetMember(ctx, room.ID, uid); err == nil && member.Role != "" {
		s.state().SetRoom(ctx, uid, room.Code)
		return s.roomView(ctx, room, uid, true), nil
	}
	banned, _ := s.db.IsBanned(ctx, room.ID, uid)
	if banned {
		return entity.RoomView{}, errf(ErrBanned, "you are banned from this room")
	}
	if room.Status == entity.RoomPlaying {
		return entity.RoomView{}, errf(ErrConflict, "a game is already in progress")
	}
	if err := s.mustLeaveCurrent(uid); err != nil {
		return entity.RoomView{}, err
	}

	invited := false
	var inviteID uint32
	if inv, err := s.db.PendingInviteFor(ctx, room.ID, uid); err == nil {
		invited = true
		inviteID = inv.ID
	}

	if room.PasswordHash != "" && !invited {
		if password == "" {
			return entity.RoomView{}, errf(ErrPasswordRequired, "this room requires a password")
		}
		if bcrypt.CompareHashAndPassword([]byte(room.PasswordHash), []byte(password)) != nil {
			return entity.RoomView{}, errf(ErrWrongPassword, "incorrect room password")
		}
	}

	count, err := s.db.CountMembers(ctx, room.ID)
	if err != nil {
		return entity.RoomView{}, err
	}
	if int(count) >= room.MaxPlayers {
		return entity.RoomView{}, errf(ErrFull, "room is full")
	}

	if err := s.db.AddMember(ctx, &entity.RoomMember{
		RoomID: room.ID, UserID: uid, Role: entity.RoleMember, JoinedAt: time.Now().UTC(),
	}); err != nil {
		return entity.RoomView{}, err
	}
	if inviteID != 0 {
		_ = s.db.UpdateInviteStatus(ctx, inviteID, entity.InviteAccepted)
	}
	s.state().SetRoom(ctx, uid, room.Code)
	s.broadcastRoom(ctx, room.ID, "room_update")
	return s.roomView(ctx, room, uid, true), nil
}

func (s *ServiceLayer) LeaveRoom(ctx context.Context, uid uint32, code string) error {
	room, err := s.db.GetRoomByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	member, err := s.db.GetMember(ctx, room.ID, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.state().Clear(ctx, uid)
			return nil
		}
		return err
	}

	if _, inGame := s.g().IsInGame(uid); inGame {
		_, _ = s.g().Leave(ctx, uid)
	}
	if err := s.db.RemoveMember(ctx, room.ID, uid); err != nil {
		return err
	}

	if member.Role == entity.RoleOwner {
		members, _ := s.db.ListMembers(ctx, room.ID)
		humans := make([]entity.RoomMember, 0, len(members))
		for _, m := range members {
			if m.User != nil && !m.User.IsBot {
				humans = append(humans, m)
			}
		}
		if len(humans) == 0 {
			s.abandonRoomGame(ctx, room)
			for _, m := range members {
				_ = s.db.RemoveMember(ctx, room.ID, m.UserID)
				s.state().Clear(ctx, m.UserID)
			}
			_ = s.db.CloseRoom(ctx, room.ID)
		} else {
			next := humans[0]
			for _, m := range humans {
				if m.Role == entity.RoleMod {
					next = m
					break
				}
			}
			_ = s.db.TransferHost(ctx, room.ID, next.UserID)
		}
	}
	s.state().Clear(ctx, uid)
	s.hub().SendToUser(uid, hub.Ev("room_left", map[string]any{"room_code": room.Code}))
	s.broadcastRoom(ctx, room.ID, "room_update")
	return nil
}

// abandonRoomGame closes a runtime and its durable row without producing
// results. Finished games are left untouched.
func (s *ServiceLayer) abandonRoomGame(ctx context.Context, room entity.Room) {
	if room.GameID == 0 {
		return
	}
	game, err := s.db.GetGame(ctx, room.GameID)
	if err == nil && (game.Status == entity.GameCountdown || game.Status == entity.GameActive) {
		_ = s.db.AbandonGame(ctx, room.GameID)
		if game.ContestID != 0 {
			if contest, err := s.db.GetContest(ctx, game.ContestID); err == nil {
				contest.Status = entity.ContestEnded
				_ = s.db.SaveContest(ctx, &contest)
			}
		}
	}
	s.g().Abandon(room.GameID)
}

func (s *ServiceLayer) UpdateRoom(ctx context.Context, uid uint32, code string, req UpdateRoomRequest) (entity.RoomView, error) {
	room, member, err := s.requireRoomRole(ctx, uid, code, entity.RoleOwner)
	if err != nil {
		return entity.RoomView{}, err
	}
	if room.Status == entity.RoomPlaying {
		return entity.RoomView{}, errf(ErrConflict, "cannot change settings during a game")
	}
	fields := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 64 {
			return entity.RoomView{}, errf(ErrBadRequest, "invalid room name")
		}
		fields["name"] = name
	}
	if req.Description != nil {
		if len(*req.Description) > 280 {
			return entity.RoomView{}, errf(ErrBadRequest, "description is too long")
		}
		fields["description"] = *req.Description
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}
	if req.Password != nil {
		if len(*req.Password) > 72 {
			return entity.RoomView{}, errf(ErrBadRequest, "password is too long")
		}
		if *req.Password == "" {
			fields["password_hash"] = ""
		} else {
			h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				return entity.RoomView{}, err
			}
			fields["password_hash"] = string(h)
		}
	}
	if req.MaxPlayers != nil {
		if *req.MaxPlayers < 2 || *req.MaxPlayers > 16 {
			return entity.RoomView{}, errf(ErrBadRequest, "max players must be between 2 and 16")
		}
		count, _ := s.db.CountMembers(ctx, room.ID)
		if *req.MaxPlayers < int(count) {
			return entity.RoomView{}, errf(ErrBadRequest, "max players is below the current member count")
		}
		fields["max_players"] = *req.MaxPlayers
	}
	if req.Duration != nil {
		if _, ok := entity.LobbyTypeFromSeconds(*req.Duration); !ok {
			return entity.RoomView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300")
		}
		fields["duration"] = *req.Duration
	}
	if req.Language != nil {
		if !validLanguage(*req.Language) || *req.Language == "random" {
			return entity.RoomView{}, errf(ErrBadRequest, "unknown language")
		}
		if *req.Language != "" {
			fields["language"] = *req.Language
		}
	}
	if len(fields) > 0 {
		if err := s.db.UpdateRoomFields(ctx, room.ID, fields); err != nil {
			return entity.RoomView{}, err
		}
	}
	_ = member
	updated, _ := s.db.GetRoomByID(ctx, room.ID)
	s.broadcastRoom(ctx, room.ID, "room_update")
	return s.roomView(ctx, updated, uid, true), nil
}

func (s *ServiceLayer) requireRoomRole(ctx context.Context, uid uint32, code string, min entity.RoomRole) (entity.Room, entity.RoomMember, error) {
	room, err := s.db.GetRoomByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return entity.Room{}, entity.RoomMember{}, errf(ErrNotFound, "room not found")
	}
	member, err := s.db.GetMember(ctx, room.ID, uid)
	if err != nil {
		return entity.Room{}, entity.RoomMember{}, errf(ErrForbidden, "you are not a member of this room")
	}
	if min == entity.RoleOwner && member.Role != entity.RoleOwner {
		return entity.Room{}, entity.RoomMember{}, errf(ErrForbidden, "only the room owner can do that")
	}
	if min == entity.RoleMod && member.Role != entity.RoleOwner && member.Role != entity.RoleMod {
		return entity.Room{}, entity.RoomMember{}, errf(ErrForbidden, "moderator permission required")
	}
	return room, member, nil
}

func (s *ServiceLayer) KickUser(ctx context.Context, uid uint32, code string, target uint32) error {
	room, requester, err := s.requireRoomRole(ctx, uid, code, entity.RoleMod)
	if err != nil {
		return err
	}
	targetMember, err := s.db.GetMember(ctx, room.ID, target)
	if err != nil {
		return errf(ErrNotFound, "user is not in this room")
	}
	if target == room.HostID {
		return errf(ErrForbidden, "the room owner cannot be kicked")
	}
	if requester.Role == entity.RoleMod && targetMember.Role == entity.RoleMod {
		return errf(ErrForbidden, "moderators cannot kick each other")
	}
	if _, inGame := s.g().IsInGame(target); inGame {
		_, _ = s.g().Leave(ctx, target)
	}
	if err := s.db.RemoveMember(ctx, room.ID, target); err != nil {
		return err
	}
	s.state().Clear(ctx, target)
	s.hub().SendToUser(target, hub.Ev("room_left", map[string]any{"room_code": room.Code, "reason": "kicked"}))
	s.broadcastRoom(ctx, room.ID, "room_update")
	return nil
}

func (s *ServiceLayer) BanUser(ctx context.Context, uid uint32, code string, target uint32) error {
	room, requester, err := s.requireRoomRole(ctx, uid, code, entity.RoleMod)
	if err != nil {
		return err
	}
	if target == room.HostID {
		return errf(ErrForbidden, "the room owner cannot be banned")
	}
	targetMember, err := s.db.GetMember(ctx, room.ID, target)
	if err == nil && requester.Role == entity.RoleMod && targetMember.Role == entity.RoleMod {
		return errf(ErrForbidden, "moderators cannot ban each other")
	}
	if err := s.db.AddBan(ctx, &entity.RoomBan{RoomID: room.ID, UserID: target, BannedBy: uid, CreatedAt: time.Now().UTC()}); err != nil {
		return err
	}
	_ = s.db.RemoveMember(ctx, room.ID, target)
	if _, inGame := s.g().IsInGame(target); inGame {
		_, _ = s.g().Leave(ctx, target)
	}
	s.state().Clear(ctx, target)
	s.hub().SendToUser(target, hub.Ev("room_left", map[string]any{"room_code": room.Code, "reason": "banned"}))
	s.broadcastRoom(ctx, room.ID, "room_update")
	return nil
}

func (s *ServiceLayer) UnbanUser(ctx context.Context, uid uint32, code string, target uint32) error {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleMod)
	if err != nil {
		return err
	}
	return s.db.RemoveBan(ctx, room.ID, target)
}

func (s *ServiceLayer) PromoteUser(ctx context.Context, uid uint32, code string, target uint32) error {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleOwner)
	if err != nil {
		return err
	}
	if _, err := s.db.GetMember(ctx, room.ID, target); err != nil {
		return errf(ErrNotFound, "user is not in this room")
	}
	if err := s.db.UpdateMemberRole(ctx, room.ID, target, entity.RoleMod); err != nil {
		return err
	}
	s.broadcastRoom(ctx, room.ID, "room_update")
	return nil
}

func (s *ServiceLayer) DemoteUser(ctx context.Context, uid uint32, code string, target uint32) error {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleOwner)
	if err != nil {
		return err
	}
	if target == room.HostID {
		return errf(ErrForbidden, "the owner is always a moderator")
	}
	if err := s.db.UpdateMemberRole(ctx, room.ID, target, entity.RoleMember); err != nil {
		return err
	}
	s.broadcastRoom(ctx, room.ID, "room_update")
	return nil
}

func (s *ServiceLayer) AddBot(ctx context.Context, uid uint32, code string, persona entity.Persona) (entity.RoomView, error) {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleMod)
	if err != nil {
		return entity.RoomView{}, err
	}
	if room.Status == entity.RoomPlaying {
		return entity.RoomView{}, errf(ErrConflict, "a game is already in progress")
	}
	cfg, ok := s.bots().Config(persona)
	if !ok {
		return entity.RoomView{}, errf(ErrBadRequest, "unknown bot persona")
	}
	name := bots.Username(persona)
	bot, err := s.db.GetUserFromName(ctx, name)
	if err != nil {
		bot = s.newUser(name, cfg.Rating, true)
		if err := s.db.AddUser(ctx, bot); err != nil {
			return entity.RoomView{}, err
		}
	}
	if _, err := s.db.GetMember(ctx, room.ID, bot.ID); err == nil {
		return s.roomView(ctx, room, uid, true), nil
	}
	count, _ := s.db.CountMembers(ctx, room.ID)
	if int(count) >= room.MaxPlayers {
		return entity.RoomView{}, errf(ErrFull, "room is full")
	}
	if err := s.db.AddMember(ctx, &entity.RoomMember{
		RoomID: room.ID, UserID: bot.ID, Role: entity.RoleMember, JoinedAt: time.Now().UTC(),
	}); err != nil {
		return entity.RoomView{}, err
	}
	_ = s.rdb().SetOnline(ctx, bot.ID, 24*time.Hour)
	s.broadcastRoom(ctx, room.ID, "room_update")
	return s.roomView(ctx, room, uid, true), nil
}

func (s *ServiceLayer) StartRoomGame(ctx context.Context, uid uint32, code string, req StartGameRequest) (entity.GameView, error) {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleMod)
	if err != nil {
		return entity.GameView{}, err
	}
	if room.Status == entity.RoomPlaying && room.GameID != 0 {
		return s.GetGameView(ctx, uid, room.GameID)
	}
	if room.Status == entity.RoomClosed {
		return entity.GameView{}, errf(ErrConflict, "room is closed")
	}
	if req.Duration != 0 {
		if _, ok := entity.LobbyTypeFromSeconds(req.Duration); !ok {
			return entity.GameView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300")
		}
		room.Duration = req.Duration
	}
	if req.Language != "" {
		if !validLanguage(req.Language) || req.Language == "random" {
			return entity.GameView{}, errf(ErrBadRequest, "unknown language")
		}
		room.Language = req.Language
	}
	game, err := s.createRoomGame(ctx, room, entity.GameRoom, 0)
	if err != nil {
		return entity.GameView{}, err
	}
	if view, ok := s.g().Snapshot(uid); ok {
		view.RoomCode = room.Code
		return view, nil
	}
	return s.gameViewFromDB(ctx, uid, game)
}

func (s *ServiceLayer) createRoomGame(ctx context.Context, room entity.Room, mode entity.GameMode, contestID uint32) (entity.Game, error) {
	members, err := s.db.ListMembers(ctx, room.ID)
	if err != nil {
		return entity.Game{}, err
	}
	if len(members) < 2 {
		return entity.Game{}, errf(ErrConflict, "at least two players are needed to start")
	}
	users := make(map[uint32]entity.User, len(members))
	players := make([]entity.GamePlayer, 0, len(members))
	botConfigs := map[uint32]entity.BotConfig{}
	for _, m := range members {
		if m.User == nil {
			continue
		}
		users[m.UserID] = *m.User
		players = append(players, entity.GamePlayer{UserID: m.UserID, RatingBefore: m.User.Rating, IsBot: m.User.IsBot, User: m.User})
		if m.User.IsBot {
			if _, cfg, ok := s.bots().PersonaForUsername(m.User.Username); ok {
				botConfigs[m.UserID] = cfg
			}
		}
	}
	if len(players) < 2 {
		return entity.Game{}, errf(ErrConflict, "at least two players are needed to start")
	}

	seed := randomSeed()
	language := room.Language
	if language == "" || language == "random" {
		language = entity.LanguageForSeed(seed)
	}
	if !validLanguage(language) {
		return entity.Game{}, errf(ErrBadRequest, "unknown language")
	}
	genCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	snippet, _ := s.codegen.GenerateSnippet(genCtx, language, seed)
	cancel()
	startOffset := utils.SnippetStartOffset(seed, snippet)

	now := time.Now().UTC()
	starts := now.Add(s.countdown)
	ends := starts.Add(time.Duration(room.Duration) * time.Second)
	newGame := entity.Game{
		ID:          s.intGen.GenerateID(),
		Mode:        mode,
		RoomID:      room.ID,
		ContestID:   contestID,
		Duration:    room.Duration,
		Language:    language,
		Seed:        seed,
		Snippet:     snippet,
		StartOffset: startOffset,
		Status:      entity.GameCountdown,
		StartsAt:    &starts,
		EndsAt:      &ends,
		CreatedAt:   now,
	}
	if err := s.db.CreateGame(ctx, &newGame, players); err != nil {
		return entity.Game{}, err
	}
	_ = s.db.UpdateRoomFields(ctx, room.ID, map[string]any{
		"status":  entity.RoomPlaying,
		"game_id": newGame.ID,
	})
	for uid := range users {
		s.state().SetGame(ctx, uid, newGame.ID, room.Code)
	}
	s.broadcastRoom(ctx, room.ID, "room_update")
	opts := gamecore.StartOptions{Game: newGame, Players: players, Users: users, Bots: botConfigs}
	if err := s.g().Start(ctx, opts); err != nil {
		return entity.Game{}, err
	}
	return newGame, nil
}

type InviteRequest struct {
	Username string `json:"username"`
	UserID   uint32 `json:"user_id"`
}

func (s *ServiceLayer) InviteUser(ctx context.Context, uid uint32, code string, req InviteRequest) (entity.InviteView, error) {
	room, member, err := s.requireRoomRole(ctx, uid, code, entity.RoleMember)
	if err != nil {
		return entity.InviteView{}, err
	}
	if room.Status == entity.RoomPlaying {
		return entity.InviteView{}, errf(ErrConflict, "a game is already in progress")
	}
	var target entity.User
	if req.UserID != 0 {
		target, err = s.db.GetUser(req.UserID)
	} else {
		target, err = s.db.GetUserFromName(ctx, strings.TrimSpace(req.Username))
	}
	if err != nil {
		return entity.InviteView{}, errf(ErrNotFound, "user not found")
	}
	if target.ID == uid {
		return entity.InviteView{}, errf(ErrBadRequest, "you are already in this room")
	}
	if _, err := s.db.GetMember(ctx, room.ID, target.ID); err == nil {
		return entity.InviteView{}, errf(ErrConflict, "user is already in this room")
	}
	if existing, err := s.db.PendingInviteFor(ctx, room.ID, target.ID); err == nil {
		return s.inviteView(ctx, existing), nil
	}
	invite := entity.RoomInvite{
		ID:        s.intGen.GenerateID(),
		RoomID:    room.ID,
		InviterID: uid,
		InviteeID: target.ID,
		Status:    entity.InvitePending,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := s.db.CreateInvite(ctx, &invite); err != nil {
		return entity.InviteView{}, err
	}
	invite.Room = &room
	invite.Inviter = &entity.User{ID: member.UserID}
	if inviter, err := s.db.GetUser(uid); err == nil {
		invite.Inviter = &inviter
	}
	view := s.inviteView(ctx, invite)
	s.hub().SendToUser(target.ID, hub.Ev("invite", view))
	return view, nil
}

func (s *ServiceLayer) ListInvites(ctx context.Context, uid uint32) ([]entity.InviteView, error) {
	return s.pendingInvites(ctx, uid)
}

func (s *ServiceLayer) AcceptInvite(ctx context.Context, uid, inviteID uint32) (entity.RoomView, error) {
	inv, err := s.db.GetInvite(ctx, inviteID)
	if err != nil {
		return entity.RoomView{}, errf(ErrNotFound, "invite not found")
	}
	if inv.InviteeID != uid {
		return entity.RoomView{}, errf(ErrForbidden, "this invite is not for you")
	}
	if inv.Status != entity.InvitePending {
		return entity.RoomView{}, errf(ErrConflict, "invite is no longer valid")
	}
	if time.Now().UTC().After(inv.ExpiresAt) {
		_ = s.db.UpdateInviteStatus(ctx, inv.ID, entity.InviteExpired)
		return entity.RoomView{}, errf(ErrConflict, "invite has expired")
	}
	if inv.Room == nil {
		return entity.RoomView{}, errf(ErrNotFound, "room not found")
	}
	view, err := s.JoinRoom(ctx, uid, inv.Room.Code, "")
	if err != nil {
		return entity.RoomView{}, err
	}
	_ = s.db.UpdateInviteStatus(ctx, inv.ID, entity.InviteAccepted)
	return view, nil
}

func (s *ServiceLayer) DeclineInvite(ctx context.Context, uid, inviteID uint32) error {
	inv, err := s.db.GetInvite(ctx, inviteID)
	if err != nil {
		return errf(ErrNotFound, "invite not found")
	}
	if inv.InviteeID != uid {
		return errf(ErrForbidden, "this invite is not for you")
	}
	return s.db.UpdateInviteStatus(ctx, inv.ID, entity.InviteDeclined)
}

func (s *ServiceLayer) inviteView(ctx context.Context, inv entity.RoomInvite) entity.InviteView {
	view := entity.InviteView{
		ID: inv.ID, Status: inv.Status, CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
	}
	if inv.Room != nil {
		view.RoomCode, view.RoomName = inv.Room.Code, inv.Room.Name
	}
	if inv.Inviter != nil {
		user, _ := s.db.GetUser(inv.InviterID)
		view.From, view.FromID = user.Username, user.ID
	}
	return view
}

func (s *ServiceLayer) CloseRoom(ctx context.Context, uid uint32, code string) error {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleOwner)
	if err != nil {
		return err
	}
	members, _ := s.db.ListMembers(ctx, room.ID)
	s.abandonRoomGame(ctx, room)
	for _, m := range members {
		_ = s.db.RemoveMember(ctx, room.ID, m.UserID)
		s.state().Clear(ctx, m.UserID)
		if m.User != nil && !m.User.IsBot {
			s.hub().SendToUser(m.UserID, hub.Ev("room_closed", map[string]any{"room_code": room.Code}))
		}
	}
	return s.db.CloseRoom(ctx, room.ID)
}

// broadcastRoom pushes a personalised room view to every human member.
func (s *ServiceLayer) broadcastRoom(ctx context.Context, roomID uint32, typ string) {
	room, err := s.db.GetRoomByID(ctx, roomID)
	if err != nil {
		return
	}
	members, err := s.db.ListMembers(ctx, roomID)
	if err != nil {
		return
	}
	for _, m := range members {
		if m.User == nil || m.User.IsBot {
			continue
		}
		view := s.roomView(ctx, room, m.UserID, true)
		s.hub().SendToUser(m.UserID, hub.Ev(typ, view))
	}
}
