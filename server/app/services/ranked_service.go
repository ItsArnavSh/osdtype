package services

import (
	"context"
	"osdtyp/app/entity"
)

func (s *ServiceLayer) JoinRankedLobby(ctx context.Context, userid uint32, duration entity.LobbyType) error {
	rank, err := s.db.GetRank(ctx, userid)
	if err != nil {
		return err
	}
	return s.core.Matchmaker.AddToGlobalLobby(userid, rank, duration)
}

// The game will be handled automatically using za core/
func (s *ServiceLayer) GetRank(ctx context.Context, userid uint32) (uint16, error) {
	return s.db.GetRank(ctx, userid)
}
