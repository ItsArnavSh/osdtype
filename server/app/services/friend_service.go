package services

import (
	"context"

	"osdtyp/app/entity"
)

// Handles everything related to friends and friendly matches

func (s *ServiceLayer) FollowUser(ctx context.Context, follower, following uint32) error {
	return s.db.FollowUser(ctx, follower, following)
}
func (s *ServiceLayer) UnfollowUser(ctx context.Context, follower, following uint32) error {
	return s.db.UnfollowUser(ctx, follower, following)
}

// JoinNewLobby creates a lobby and drops the caller straight into it.
//
// The error path used to call JoinControlledLobby again, which is the call that
// had just failed: it could not undo anything, and had it ever succeeded it
// would have joined the user to the very lobby the comment meant to destroy.
// The lobby is now dropped with RemoveLobby instead.
func (s *ServiceLayer) JoinNewLobby(userid uint32) (uint32, error) {
	lobby_id := s.core.ManualLobby.CreateNewLobby()
	if err := s.core.ManualLobby.JoinControlledLobby(userid, lobby_id); err != nil {
		s.core.ManualLobby.RemoveLobby(lobby_id)
		return 0, err
	}
	s.core.Sessions.UpdateSession(userid, entity.PLAYING)
	return lobby_id, nil
}
func (s *ServiceLayer) JoinControlledLobby(userid uint32, lobbyid uint32) error {
	err := s.core.ManualLobby.JoinControlledLobby(userid, lobbyid)
	if err != nil {
		return err
	}
	s.core.Sessions.UpdateSession(userid, entity.PLAYING)
	return nil
}

func (s *ServiceLayer) InvitePlayerToLobby(invitor string, invitee uint32, lobbyid uint32) {
	if s.core.Sessions.GetStatus(invitee) == entity.AVAILABLE {
		_, send := s.core.Sessions.GetSession(invitee).Subscribe()
		invitation := entity.Invite{
			From:    invitor,
			LobbyID: lobbyid,
		}
		send <- invitation
		send <- nil // Unsubscribe
	}
}
func (s *ServiceLayer) SearchUsers(ctx context.Context, username string) ([]entity.User, error) {
	return s.db.SearchPeople(ctx, username, 10)
}
