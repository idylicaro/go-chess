package match

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Clock is what a player is told about the time when asked for a move.
type Clock struct {
	WTime, BTime, WInc, BInc time.Duration
}

// Player is one side of a game: an engine process, or a fake in tests.
type Player interface {
	// NewGame says the next position belongs to a different game.
	NewGame() error
	// BestMove returns the move, in UCI notation, after moves from the
	// starting position.
	BestMove(ctx context.Context, moves []string, clock Clock) (string, error)
	Close() error
}

// EngineSpec says how to start one engine of a match.
type EngineSpec struct {
	Name    string            // label in reports and PGN; the engine's id name when empty
	Path    string            // executable, looked up in PATH without a slash
	Options map[string]string // UCI options, sent with setoption
}

// PlayerFactory starts a player for spec; Tournament calls it once per
// engine and concurrent game.
type PlayerFactory func(spec EngineSpec) (Player, error)

// uciPlayer is an engine process driven through the project's UCI client.
type uciPlayer struct {
	engine *engine.Engine
}

// StartUCIPlayer is the PlayerFactory for real engines.
//
//	player, err := match.StartUCIPlayer(match.EngineSpec{Path: "bin/chess-engine"})
func StartUCIPlayer(spec EngineSpec) (Player, error) {
	eng, err := engine.Start(spec.Path)
	if err != nil {
		return nil, err
	}
	for name, value := range spec.Options {
		if err := eng.SetOption(name, value); err != nil {
			_ = eng.Close()
			return nil, fmt.Errorf("engine %q option %s=%s: %w", spec.Path, name, value, err)
		}
	}
	return &uciPlayer{engine: eng}, nil
}

func (p *uciPlayer) NewGame() error {
	return p.engine.NewGame()
}

func (p *uciPlayer) BestMove(ctx context.Context, moves []string, clock Clock) (string, error) {
	limits := engine.Limits{WTime: clock.WTime, BTime: clock.BTime, WInc: clock.WInc, BInc: clock.BInc}
	result, err := p.engine.SearchGame(ctx, chess.StartingFEN, moves, limits, nil)
	if err != nil {
		return "", err
	}
	return strings.ToLower(result.Best), nil
}

func (p *uciPlayer) Close() error {
	return p.engine.Close()
}
