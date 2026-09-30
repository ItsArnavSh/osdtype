package game

import (
	"sync"
	"time"

	"osdtyp/app/core/game/player"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// tokensPerRound is how long a generated snippet is. 1000 characters is a
// little over three minutes of steady typing, which covers the longest mode
// with room to spare.
const tokensPerRound = 1000

// countdownBefore is how long players get to get ready before a round. The
// clock on each player only starts at their first keystroke, so the countdown
// is a courtesy rather than a cost.
const countdownBefore = 3 * time.Second

// GameHandler runs one round: it picks the snippet, counts players in,
// broadcasts live input and produces a leaderboard.
type GameHandler struct {
	Logger *zap.SugaredLogger
	Player []*player.Player

	seed    uint32
	lang    entity.Language
	snippet string
	signal  chan []entity.WPMRes

	broadcastCh chan entity.KeystrokeFrame
	done        chan struct{}
	stopOnce    sync.Once
}

// NewGameHandler prepares a round.
//
// The snippet and the seed are chosen together: the language is derived from
// the seed, so one number identifies the whole round. The seed is sent to
// every client, which lets a client regenerate the identical snippet locally
// and keeps the code off the wire entirely.
func NewGameHandler(cg *utils.CodeGen, items []entity.PlayerItem, logger *zap.SugaredLogger, duration time.Duration, sig chan []entity.WPMRes) *GameHandler {
	logger.Infow("preparing round", "players", len(items), "duration", duration)

	seed := entropy()
	lang := entity.Language(seed % uint32(len(entity.KnownLanguages)))
	snippet := cg.Generate(ctxForGame(), lang.String(), seed, tokensPerRound)

	// A generator that is down must not produce an unplayable round, so fall
	// back to the smallest grammar rather than handing everyone an empty
	// snippet and a zero score.
	if snippet == "" {
		logger.Warnw("snippet service returned nothing, falling back to a built-in snippet", "lang", lang.String())
		snippet = fallbackSnippet(lang)
	}

	bcast := make(chan entity.KeystrokeFrame, broadcastBuffer)
	players := make([]*player.Player, 0, len(items))
	for _, item := range items {
		players = append(players, player.New(item, snippet, bcast, logger, duration))
	}

	return &GameHandler{
		Logger:      logger,
		Player:      players,
		seed:        seed,
		lang:        lang,
		snippet:     snippet,
		signal:      sig,
		broadcastCh: bcast,
		done:        make(chan struct{}),
	}
}

// broadcastBuffer is how many live keystrokes can queue before the fan-out
// reader has to catch up. Live keystrokes are cosmetic, so this only needs to
// be big enough to absorb a burst.
const broadcastBuffer = 256

// Run plays the round to completion and returns the leaderboard.
//
// This blocks for the length of the round. Callers that need to keep doing
// other work must call it on its own goroutine: the scheduler used to call it
// inline, which stalled the whole scheduling loop for every minute of every
// contest.
func (g *GameHandler) Run() []entity.WPMRes {
	g.announce()

	var players sync.WaitGroup
	for _, p := range g.Player {
		players.Add(1)
		go p.PlayerInRoutine(&players)
	}

	// The broadcaster is stopped and joined before the round is scored, so the
	// last keystrokes of a round are still delivered and no broadcast
	// goroutine outlives the handler.
	broadcastDone := make(chan struct{})
	go g.broadcast(broadcastDone)

	g.countdown()

	players.Wait()
	g.stop()
	<-broadcastDone

	return g.conclude(entity.EndFinished)
}

// announce tells every client what round they are in and seeds them.
func (g *GameHandler) announce() {
	frame := entity.Encode(entity.NewSeedFrame(g.seed, g.lang, tokensPerRound, g.Player[0].Duration))
	for _, p := range g.Player {
		p.Session.Send(frame)
	}
}

// countdown gives players a moment to get their hands on the keyboard.
func (g *GameHandler) countdown() {
	remaining := int(countdownBefore / time.Second)
	for remaining > 0 {
		frame := entity.Encode(entity.CountdownFrame{Type: entity.FrameCountdown, Seconds: remaining})
		for _, p := range g.Player {
			p.Session.Send(frame)
		}
		time.Sleep(time.Second)
		remaining--
	}

	start := entity.Encode(entity.CountdownFrame{Type: entity.FrameStart})
	for _, p := range g.Player {
		p.Session.Send(start)
	}
}

// broadcast fans live keystrokes out to everyone in the round.
//
// This is the pipeline that used to have no writer at all: Player.Send was
// commented out at its only call site, so the fan-out queue was never fed and
// no player ever saw another player type. It exits when the round's player
// routines have all finished.
func (g *GameHandler) broadcast(done chan<- struct{}) {
	defer close(done)

	for {
		select {
		case <-g.done:
			return
		case frame := <-g.broadcastCh:
			raw := entity.Encode(frame)
			for _, p := range g.Player {
				p.Session.Send(raw)
			}
		}
	}
}

// stop ends the fan-out, at most once.
func (g *GameHandler) stop() { g.stopOnce.Do(func() { close(g.done) }) }

// conclude scores the round, tells everyone the result and releases the leases.
func (g *GameHandler) conclude(reason string) []entity.WPMRes {
	leaderboard := make([]entity.WPMRes, 0, len(g.Player))
	for _, p := range g.Player {
		leaderboard = append(leaderboard, p.CalculateScore())
	}
	sortLeaderboard(leaderboard)

	// Stop the fan-out before the result goes out, so a late keystroke cannot
	// arrive after the leaderboard the player is looking at. Run also closes
	// it, so this is guarded.
	g.stop()

	g.Logger.Infoln("leaderboard prepared", "players", len(leaderboard))
	end := entity.Encode(entity.EndFrame{
		Type:        entity.FrameEnd,
		Reason:      reason,
		Leaderboard: leaderboard,
	})

	for _, p := range g.Player {
		p.Session.Send(end)
		// Hand the socket back so the player can queue for another game or
		// receive a notification without waiting for their lease to be
		// revoked.
		p.Session.Release()
	}

	// The result channel is unbuffered, so delivering without waiting is the
	// only way to avoid wedging the handler when nobody is listening. A
	// dropped leaderboard costs the rank update, which is logged.
	select {
	case g.signal <- leaderboard:
	default:
		g.Logger.Warn("nobody was waiting for the leaderboard, dropping it")
	}

	g.Logger.Infoln("round wrapped")
	return leaderboard
}

// Abort ends a round early, for instance when too many players left.
func (g *GameHandler) Abort() { g.conclude(entity.EndAborted) }

// PlayerCount reports how many players are in the round.
func (g *GameHandler) PlayerCount() int { return len(g.Player) }

// Seed reports the round's seed, for tests and diagnostics.
func (g *GameHandler) Seed() uint32 { return g.seed }

// Snippet reports the round's snippet.
func (g *GameHandler) Snippet() string { return g.snippet }
