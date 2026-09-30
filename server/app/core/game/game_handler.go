package game

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"osdtyp/app/core/game/player"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type GameHandler struct {
	Duration  uint32 // Duration of game in seconds
	Player    []*player.Player
	Logger    *zap.SugaredLogger
	CommonOut chan player.OutGoing
	seed      uint32
	Codegen   *utils.CodeGen
	snippet   string // Later use tokens, and live generation
	signal    chan []entity.WPMRes
}

func NewGameHandler(cg *utils.CodeGen, player_conns []entity.PlayerItem, logger *zap.SugaredLogger, duration time.Duration, sig chan []entity.WPMRes) GameHandler {
	logger.Infoln("In the game handler")
	players := make([]*player.Player, 0, len(player_conns))

	seed := rand.Uint32()
	lang_choice := entity.Language(seed % 6)
	snippet := cg.Generate(context.Background(), lang_choice.String(), seed, 1000)
	common_out := make(chan player.OutGoing)
	for _, item := range player_conns {
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
			Name:      item.Name,
		}

		players = append(players, &player)
	}
	// Sending the seed over

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
	// If any message comes, loop through all the players and send this message
	for update := range g.CommonOut {
		for _, player := range g.Player {
			player.LocalOut <- update
		}
		if update.PlayerID == 0 {
			return
		}
	}
}
func (g *GameHandler) EndLiveStream() {
	// The destructor routine
	leaderboard := make([]entity.WPMRes, 0, len(g.Player))
	close(g.CommonOut)
	for _, player := range g.Player {
		leaderboard = append(leaderboard, player.CalculateScore())
	}
	g.Logger.Infoln("Leaderboard prepared: ", leaderboard)
	// Send the scores to all the players
	for _, player := range g.Player {
		go func() {
			utils.SafeSend(player.WebSocOut, leaderboard, g.Logger)
			g.Logger.Infoln("Sent out the leaderboard")
			utils.SafeSend(player.WebSocOut, nil, g.Logger) // Unsub message
			g.Logger.Infoln("Dropped the comm")
		}()
	}
	// The result channel is unbuffered, so this used to block forever whenever
	// nobody was still listening: the matchmaker gives up after two minutes and
	// the scheduler after ten, and a game that outlives its reader wedged the
	// handler goroutine. Deliver it without waiting.
	select {
	case g.signal <- leaderboard:
	default:
		g.Logger.Warn("nobody was waiting for the leaderboard, dropping it")
	}
	g.Logger.Info("Game Wrapped")
}
