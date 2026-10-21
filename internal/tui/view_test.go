package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/muesli/termenv"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/internal/timer/timertest"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func TestMain(m *testing.M) {
	// Without a terminal lipgloss would drop its styling anyway; saying so
	// keeps the snapshots the same wherever the tests run.
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

// Snapshots of View() as golden files. Run `go test ./internal/tui -update`
// after changing the layout on purpose, and read the diff of testdata/.

// keys sends the named keys to the model, in order.
func keys(t *testing.T, m tea.Model, names ...string) Model {
	t.Helper()
	for _, name := range names {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
		switch name {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "left":
			msg = tea.KeyMsg{Type: tea.KeyLeft}
		case "right":
			msg = tea.KeyMsg{Type: tea.KeyRight}
		}
		m, _ = m.Update(msg)
	}
	return m.(Model)
}

// goTo moves the keyboard cursor to a square with arrow keys.
func goTo(t *testing.T, m Model, square string) Model {
	t.Helper()
	to, ok := render.ParseSquare(square)
	if !ok {
		t.Fatalf("bad square %q", square)
	}
	for m.cursor.X != to.X || m.cursor.Y != to.Y {
		key := ""
		switch {
		case m.cursor.X < to.X:
			key = "right"
		case m.cursor.X > to.X:
			key = "left"
		case m.cursor.Y < to.Y:
			key = "down"
		default:
			key = "up"
		}
		m = keys(t, m, key)
	}
	return m
}

// moves plays UCI moves (e2e4) by picking up a piece and dropping it.
func moves(t *testing.T, m Model, uci ...string) Model {
	t.Helper()
	for _, move := range uci {
		m = goTo(t, m, move[:2])
		m = keys(t, m, "enter")
		m = goTo(t, m, move[2:4])
		m = keys(t, m, "enter")
	}
	return m
}

func snapshot(t *testing.T, m Model) {
	t.Helper()
	golden.RequireEqual(t, []byte(m.View()))
}

func TestViewMenu(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, m)
}

func TestViewMenuCentered(t *testing.T) {
	m, _ := New(Config{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 14})
	snapshot(t, next.(Model))
}

func TestViewGameStart(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, keys(t, m, "enter"))
}

func TestViewGameStartUnicode(t *testing.T) {
	m, _ := New(Config{Style: render.Style{Unicode: true}})
	snapshot(t, keys(t, m, "enter"))
}

func TestViewGameStartColor(t *testing.T) {
	m, _ := New(Config{Style: render.Style{Color: true, Unicode: true}})
	snapshot(t, keys(t, m, "enter"))
}

func TestViewAfterMoves(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, moves(t, keys(t, m, "enter"), "e2e4", "e7e5", "g1f3"))
}

func TestViewSelection(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, keys(t, goTo(t, keys(t, m, "enter"), "b1"), "enter"))
}

func TestViewFlipped(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, keys(t, m, "enter", "f"))
}

func TestViewRefusedPick(t *testing.T) {
	m, _ := New(Config{})
	snapshot(t, keys(t, m, "enter", "k", "k", "k", "k", "k", "enter"))
}

func TestViewPromotionDialog(t *testing.T) {
	m, _ := New(Config{FEN: "7k/4P3/8/8/8/8/8/4K3 w - - 0 1"})
	snapshot(t, moves(t, keys(t, m, "enter"), "e7e8")) // the dialog opens and waits
}

func TestViewCheckmate(t *testing.T) {
	m, _ := New(Config{})
	// Fool's mate.
	snapshot(t, moves(t, keys(t, m, "enter"), "f2f3", "e7e5", "g2g4", "d8h4"))
}

func TestViewTimedGame(t *testing.T) {
	clock := timertest.NewFakeClock()
	config := chess.TimeConfig{InitialTime: 5 * time.Minute, Increment: 3 * time.Second}
	m, _ := New(Config{TimeControl: &config, Clock: clock})
	m = keys(t, m, "enter")
	clock.Advance(42 * time.Second)
	snapshot(t, m)
}

func TestViewTimedGameFlagged(t *testing.T) {
	clock := timertest.NewFakeClock()
	config := chess.TimeConfig{InitialTime: 10 * time.Second}
	m, _ := New(Config{TimeControl: &config, Clock: clock})
	m = keys(t, m, "enter")
	clock.Advance(11 * time.Second)
	next, _ := m.Update(tickMsg{m.tickGen})
	snapshot(t, next.(Model))
}

func TestViewEngineThinking(t *testing.T) {
	m, _ := New(Config{Engine: newFakeEngine(), EngineName: "Stockfish 17", HumanColor: chess.White})
	m = keys(t, m, "enter", "enter", "up", "up", "enter")
	snapshot(t, m)
}

func TestViewAnalysis(t *testing.T) {
	a := &scriptedAnalyst{reply: func(fen string, depth int) engine.Info {
		return engine.Info{Depth: depth, MultiPV: 1, HasScore: true, Score: engine.Score{Value: 34},
			PV: []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1b5", "a7a6", "b5a4", "g8f6", "e1g1", "f8e7"}}
	}}
	m, _ := New(Config{Engine: a, EngineName: "Stockfish 17", Analyze: true})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	snapshot(t, deepen(t, next.(Model), cmd))
}

func TestViewAnalysisPending(t *testing.T) {
	m, _ := New(Config{Engine: &scriptedAnalyst{reply: opening}, EngineName: "Stockfish 17", Analyze: true})
	snapshot(t, keys(t, m, "enter"))
}
