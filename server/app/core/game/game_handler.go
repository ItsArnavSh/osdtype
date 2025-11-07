package game

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"osdtyp/app/core/game/player"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type GameHandler struct {
	Duration  uint64 //Duration of game in seconds
	Player    []player.Player
	Logger    *zap.SugaredLogger
	CommonOut chan player.OutGoing
	seed      uint64
	Codegen   *utils.CodeGen
	snippet   string //Later use tokens, and live generation
	wg        *sync.WaitGroup
	//Need to upgrade resrap for supporting that
}

func NewGameHandler(cg *utils.CodeGen, player_conns []entity.PlayerItem, logger *zap.SugaredLogger, duration time.Duration) GameHandler {
	logger.Infof("In the game handler")
	var players []player.Player
	wg := sync.WaitGroup{}

	seed := rand.Uint64()
	snippet := cg.Generate("c", seed, 1000)
	common_out := make(chan player.OutGoing)
	for _, item := range player_conns {
		wg.Add(1)
		player := player.Player{
			State:    strings.Builder{},
			Conn:     item.Websock,
			Out:      common_out,
			In:       make(chan entity.Keypress),
			ID:       item.ID,
			Rank:     item.Rank,
			LocalOut: make(chan player.OutGoing, 10),
			Logger:   logger,
			Duration: duration,
			Snippet:  snippet,
			WG:       &wg,
		}

		players = append(players, player)
		go player.PlayerInRoutine()
		go player.PlayerOutUpdate()
	}
	//Sending the seed over

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, seed)

	for _, conn := range player_conns {
		conn.Websock.WriteMessage(websocket.TextMessage, fmt.Appendf(nil, "%d", seed))
	}

	return GameHandler{
		Player:    players,
		Logger:    logger,
		CommonOut: common_out,
		seed:      seed,
		snippet:   snippet,
		wg:        &wg,
	}
}

func (g *GameHandler) GlobalBroadcaster() {
	for update := range g.CommonOut {
		for _, player := range g.Player {
			select {
			case player.LocalOut <- update:

			default:
				// player’s queue is full; skip to avoid blocking
				g.Logger.Warnf("Dropping update: slow or disconnected")
			}

		}
		if update.PlayerID == 0 {
			close(g.CommonOut)
			return
		}

	}

}
func (g *GameHandler) EndLiveStream() {
	//The destructor routine
	var leaderboard []entity.WPMRes
	g.CommonOut <- player.OutGoing{PlayerID: 0, CurrentPoints: 0} //Means its done

	for _, player := range g.Player {
		leaderboard = append(leaderboard, player.CalculateScore())
	}
	//Wait before all websockets are resolved
	g.Logger.Info("Waiting for processes to end")
	g.wg.Wait()
	//Send the scores to all the players
	for _, player := range g.Player {
		go func() {
			player.Conn.WriteJSON(leaderboard)
		}()
	}
}
