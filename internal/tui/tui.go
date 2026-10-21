// Package tui is the full-screen terminal client, built on bubbletea in the
// Elm style: a Model, an Update that turns messages into a new Model and
// commands, and a View that draws it.
package tui

import (
	"context"
	"errors"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

const defaultTickInterval = 200 * time.Millisecond

// Config is what a TUI session is set up with.
type Config struct {
	// FEN is the position the game starts from; empty means the standard one.
	FEN string
	// Style says what the terminal can draw.
	Style render.Style
	// TimeControl, if set, makes the game timed. The clock runs while the
	// game screen is showing.
	TimeControl *chess.TimeConfig
	// Clock is the time source of a timed game; nil means the wall clock.
	// Tests pass a fake one.
	Clock chess.Clock
	// Engine, if set, plays the side opposite HumanColor. It is already
	// started and configured; the caller closes it after Run returns.
	Engine Engine
	// EngineName is shown beside the board.
	EngineName string
	// HumanColor is the player's side against an Engine. The board faces it.
	HumanColor chess.Color
	// Analyze makes the engine analyze the position instead of playing: both
	// sides are moved by hand and the panel shows the evaluation and the main
	// line. It needs an Engine.
	Analyze bool
	// Context bounds engine searches. Run sets it to its own context when it
	// is nil.
	Context context.Context
	// TickInterval is how often the clocks are redrawn and checked for a
	// flag; zero means 200ms.
	TickInterval time.Duration
}

// Run shows the TUI on out, reading keys and the mouse from in, until the
// player quits. It returns ctx.Err() when ctx ends first.
//
//	err := tui.Run(ctx, tui.Config{Style: style}, os.Stdin, os.Stdout)
func Run(ctx context.Context, cfg Config, in io.Reader, out io.Writer) error {
	if cfg.Context == nil {
		cfg.Context = ctx
	}
	model, err := New(cfg)
	if err != nil {
		return err
	}
	program := tea.NewProgram(model,
		tea.WithContext(ctx),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}
