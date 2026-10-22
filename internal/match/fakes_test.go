package match

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// firstLegalMovePlayer plays the alphabetically first legal move: a
// deterministic, instant and weak opponent.
type firstLegalMovePlayer struct {
	newGames atomic.Int32
	closed   atomic.Bool
}

func (p *firstLegalMovePlayer) NewGame() error {
	p.newGames.Add(1)
	return nil
}

func (p *firstLegalMovePlayer) BestMove(_ context.Context, moves []string, _ Clock) (string, error) {
	game := chess.NewGame()
	for _, move := range moves {
		if err := playUCI(game, move); err != nil {
			return "", err
		}
	}
	var legal []string
	for _, move := range game.LegalMoves() {
		legal = append(legal, move.String())
	}
	sort.Strings(legal)
	return legal[0], nil
}

func (p *firstLegalMovePlayer) Close() error {
	p.closed.Store(true)
	return nil
}

// illegalMovePlayer always answers a move no position allows.
type illegalMovePlayer struct{ firstLegalMovePlayer }

func (p *illegalMovePlayer) BestMove(context.Context, []string, Clock) (string, error) {
	return "a1a1", nil
}

// scriptedPlayer plays its moves in order, whatever the position.
type scriptedPlayer struct {
	firstLegalMovePlayer
	script []string
	next   int
}

func (p *scriptedPlayer) BestMove(context.Context, []string, Clock) (string, error) {
	move := p.script[p.next]
	p.next++
	return move, nil
}

// slowPlayer thinks for a fixed time on a fake clock before moving.
type slowPlayer struct {
	firstLegalMovePlayer
	clock     *manualClock
	thinking  time.Duration
	lastClock Clock
}

func (p *slowPlayer) BestMove(ctx context.Context, moves []string, clock Clock) (string, error) {
	p.lastClock = clock
	p.clock.advance(p.thinking)
	return p.firstLegalMovePlayer.BestMove(ctx, moves, clock)
}

// manualClock is a fake time source that only moves when told to.
type manualClock struct{ current time.Time }

func (c *manualClock) now() time.Time { return c.current }

func (c *manualClock) advance(by time.Duration) { c.current = c.current.Add(by) }

func (p *firstLegalMovePlayer) isClosed() bool { return p.closed.Load() }
