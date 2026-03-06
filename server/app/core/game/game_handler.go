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

	"go.uber.org/zap"
)

type GameHandler struct {
	Duration  uint32 //Duration of game in seconds
	Player    []player.Player
	Logger    *zap.SugaredLogger
	CommonOut chan player.OutGoing
	seed      uint32
	Codegen   *utils.CodeGen
	snippet   string //Later use tokens, and live generation
	signal    chan []entity.WPMRes
}

func NewGameHandler(cg *utils.CodeGen, player_conns []entity.PlayerItem, logger *zap.SugaredLogger, duration time.Duration, sig chan []entity.WPMRes) GameHandler {
	logger.Infoln("In the game handler")
	var players []player.Player
	var wg sync.WaitGroup

	seed := rand.Uint32()
	lang_choice := entity.Language(seed % 6)
	snippet := cg.Generate(lang_choice.String(), seed, 1000)
	common_out := make(chan player.OutGoing)
	for _, item := range player_conns {
		wg.Add(1)
		player := player.Player{
			State:     strings.Builder{},
			WebSocIn:  item.IN,
			WebSocOut: item.OUT,
			Out:       common_out,
			In:        make(chan entity.Keypress),
			ID:        item.ID,
			Rank:      item.Rank,
			LocalOut:  make(chan player.OutGoing, 10),
			Logger:    logger,
			Duration:  duration,
			Snippet:   snippet,
			WG:        &wg,
		}

		players = append(players, player)
		//go player.PlayerOutUpdate()
	}
	//Sending the seed over

	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, seed)

	for _, conn := range player_conns {
		conn.OUT <- fmt.Appendf(nil, "%d", seed)
	}

	return GameHandler{
		Player:    players,
		Logger:    logger,
		CommonOut: common_out,
		seed:      seed,
		snippet:   snippet,
		signal:    sig,
	}
}

func (g *GameHandler) GlobalBroadcaster() {
	//If any message comes, loop through all the players and send this message
	for update := range g.CommonOut {
		for _, player := range g.Player {
			player.LocalOut <- update

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
	g.Logger.Infoln("Leaderboard prepared: ", leaderboard)
	//Send the scores to all the players
	for _, player := range g.Player {
		go func() {
			utils.SafeSend(player.WebSocOut, leaderboard, g.Logger)
			g.Logger.Infoln("Sent out the leaderboard")
			utils.SafeSend(player.WebSocOut, nil, g.Logger) //Unsub message
			g.Logger.Infoln("Dropped the comm")
		}()
	}
	g.signal <- leaderboard //Permanent
	g.Logger.Info("Game Wrapped")
}
