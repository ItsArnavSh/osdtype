package services

import (
	"context"
	"osdtyp/app/entity"
)

func (s *ServiceLayer) JoinRankedLobby(ctx context.Context, userid uint64, duration entity.LobbyType) error {
	rank, err := s.db.GetRank(ctx, userid)
	if err != nil {
		return err
	}
	return s.core.Matchmaker.AddToGlobalLobby(userid, rank, duration)
}

//The game will be handled automatically using za core/
