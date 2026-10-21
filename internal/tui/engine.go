package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Engine is what the TUI needs from a UCI engine; *engine.Engine fits.
type Engine interface {
	Search(ctx context.Context, fen string, limits engine.Limits, onInfo func(engine.Info)) (engine.Result, error)
}

// thinkTime is how long the engine thinks per move on an untimed game.
const thinkTime = time.Second

// spinnerInterval is how often the "thinking" indicator moves.
const spinnerInterval = 120 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// engineMoveMsg is the engine's answer to the search with the same gen.
type engineMoveMsg struct {
	gen  int
	best string
	err  error
}

// spinMsg advances the "thinking" indicator.
type spinMsg struct{ gen int }

func (m Model) engineColor() chess.Color {
	if m.cfg.HumanColor == chess.White {
		return chess.Black
	}
	return chess.White
}

// engineToMove says whether it is the engine's turn in a game still on.
func (m Model) engineToMove() bool {
	return m.cfg.Engine != nil && !m.cfg.Analyze && !m.engineDown && !m.game.IsOver() && m.game.GetCurrentTurn() == m.engineColor()
}

// limits are the search limits for the engine's turn: its clock when the
// game is timed, a fixed think time otherwise.
func (m Model) limits() engine.Limits {
	if !m.game.HasTimer() {
		return engine.Limits{MoveTime: thinkTime}
	}
	white, _ := m.game.GetRemainingTime(chess.White)
	black, _ := m.game.GetRemainingTime(chess.Black)
	return engine.Limits{WTime: white, BTime: black, WInc: m.cfg.TimeControl.Increment, BInc: m.cfg.TimeControl.Increment}
}

// startSearch asks the engine for a move in a command, so the UI keeps
// answering keys and redrawing while it thinks. It does nothing unless it is
// the engine's turn.
func (m Model) startSearch() (Model, tea.Cmd) {
	if !m.engineToMove() || m.thinking {
		return m, nil
	}
	m.searchGen++
	m.thinking, m.frame = true, 0
	ctx, cancel := context.WithCancel(m.baseContext())
	m.cancelSearch = cancel
	gen, fen, limits, eng := m.searchGen, m.game.FEN(), m.limits(), m.cfg.Engine
	search := func() tea.Msg {
		res, err := eng.Search(ctx, fen, limits, nil)
		return engineMoveMsg{gen: gen, best: res.Best, err: err}
	}
	return m, tea.Batch(search, m.spin())
}

func (m Model) baseContext() context.Context {
	if m.cfg.Context != nil {
		return m.cfg.Context
	}
	return context.Background()
}

func (m Model) spin() tea.Cmd {
	gen := m.searchGen
	return tea.Tick(spinnerInterval, func(time.Time) tea.Msg { return spinMsg{gen} })
}

// stopSearch abandons the search in progress; its answer, if it still
// arrives, is ignored.
func (m Model) stopSearch() Model {
	if m.cancelSearch != nil {
		m.cancelSearch()
	}
	m.cancelSearch = nil
	m.thinking = false
	m.searchGen++
	return m
}

func (m Model) updateSpin(msg spinMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.searchGen || !m.thinking {
		return m, nil
	}
	m.frame = (m.frame + 1) % len(spinnerFrames)
	return m, m.spin()
}

// updateEngineMove plays the engine's answer.
func (m Model) updateEngineMove(msg engineMoveMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.searchGen {
		return m, nil // from a search that was abandoned
	}
	m.thinking, m.cancelSearch = false, nil
	if msg.err != nil {
		m.engineDown = true
		if !errors.Is(msg.err, context.Canceled) {
			m.message = fmt.Sprintf("The engine failed: %v. Press u to take a move back and try again.", msg.err)
		}
		return m, nil
	}
	move, err := chess.ParseMove(msg.best)
	if err == nil && !m.isLegal(move) {
		err = fmt.Errorf("%w: %s", chess.ErrIllegalMove, msg.best)
	}
	if err != nil {
		m.engineDown = true
		m.message = fmt.Sprintf("The engine answered %q, which is not a legal move. Press u to take a move back and try again.", msg.best)
		return m, nil
	}
	return m.play(move), nil
}

func (m Model) isLegal(move chess.Move) bool {
	for _, legal := range m.game.LegalMoves() {
		if legal.FromX == move.FromX && legal.FromY == move.FromY && legal.ToX == move.ToX && legal.ToY == move.ToY &&
			(legal.Promotion == move.Promotion || move.Promotion == 0) {
			return true
		}
	}
	return false
}

// thinkingLine is the indicator under the board while the engine thinks.
func (m Model) thinkingLine() string {
	name := m.cfg.EngineName
	if name == "" {
		name = "The engine"
	}
	return fmt.Sprintf("%s %s is thinking…", spinnerFrames[m.frame], name)
}
