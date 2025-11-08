package services

import (
	"context"
	"osdtyp/app/entity"
)

//Handles everything related to friends and friendly matches

func (s *ServiceLayer) FollowUser(ctx context.Context, follower, following uint64) error {
	return s.db.FollowUser(ctx, follower, following)
}
func (s *ServiceLayer) UnfollowUser(ctx context.Context, follower, following uint64) error {
	return s.db.UnfollowUser(ctx, follower, following)
}

// Will return the lobby id of this lobby
func (s *ServiceLayer) JoinNewLobby(userid uint64) (uint64, error) {
	lobby_id := s.core.ManualLobby.CreateNewLobby()
	err := s.core.ManualLobby.JoinControlledLobby(userid, lobby_id)
	if err != nil {
		//NUke da lobby
		s.JoinControlledLobby(userid, lobby_id)
		return 0, err
	}
	return lobby_id, nil
}
func (s *ServiceLayer) JoinControlledLobby(userid uint64, lobbyid uint64) error {

	err := s.core.ManualLobby.JoinControlledLobby(userid, lobbyid)
	if err != nil {
		return err
	}
	s.core.Sessions.UpdateSession(userid, entity.PLAYING)
	return nil
}

func (s *ServiceLayer) InvitePlayerToLobby(invitor string, invitee uint64, lobbyid uint64) {
	if s.core.Sessions.GetStatus(invitee) == entity.AVAILABLE {
		_, send := s.core.Sessions.GetSession(invitee).Subscribe()
		invitation := entity.Invite{
			From:    invitor,
			LobbyID: lobbyid,
		}
		send <- invitation
		send <- nil //Unsubscribe
	}
}
