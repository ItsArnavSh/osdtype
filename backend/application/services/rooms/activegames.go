package room

import (
	"context"
	"encoding/json"
	"osdtype/application/entity"
)

// The main state for all the games is being maintained here
type ActiveGames struct {
	games      map[string]GameHandler //Single instance for all games
	essentials entity.Essentials
}

func NewActiveGames(ess entity.Essentials) ActiveGames {
	return ActiveGames{
		games:      make(map[string]GameHandler),
		essentials: ess,
	}
}

func (a *ActiveGames) NewGame(ctx context.Context, request []byte) (GameHandler, error) {
	var conf entity.GameConf
	err := json.Unmarshal(request, &conf)
	if err != nil {
		return GameHandler{}, err
	}
	gameHandler, err := NewGameHandler(ctx, conf.Room, a.essentials, conf)
	if err != nil {
		return GameHandler{}, err
	}
	//Now we will save the gameHandler state
	a.games[conf.Room] = gameHandler
	//Todo: Add a purge function to purge inactive functions from here
	return gameHandler, nil
}
func (a *ActiveGames) RemoveGame(ctx context.Context, roomID string) {
	delete(a.games, roomID)
}
