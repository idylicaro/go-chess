package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// scriptedAnalyst reports the info its script has for the position and
// depth, as an engine would while deepening.
type scriptedAnalyst struct {
	reply func(fen string, depth int) engine.Info

	mu    sync.Mutex
	calls []string // "depth fen"
}

func (a *scriptedAnalyst) Search(_ context.Context, fen string, limits engine.Limits, onInfo func(engine.Info)) (engine.Result, error) {
	a.mu.Lock()
	a.calls = append(a.calls, fmt.Sprintf("%d %s", limits.Depth, fen))
	a.mu.Unlock()
	if info := a.reply(fen, limits.Depth); info.HasScore {
		onInfo(info)
	}
	return engine.Result{}, nil
}

func (a *scriptedAnalyst) searches() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.calls)
}

func opening(fen string, depth int) engine.Info {
	return engine.Info{Depth: depth, MultiPV: 1, HasScore: true, Score: engine.Score{Value: 34},
		PV: []string{"e2e4", "e7e5", "g1f3"}}
}

func analysisModel(t *testing.T, a *scriptedAnalyst) Model {
	t.Helper()
	m, err := New(Config{Engine: a, EngineName: "FakeFish", Analyze: true})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// runAnalysis runs the command that Update returned and gives the analysis it
// produced. The spinner and clock timers that may come along are ignored.
func runAnalysis(t *testing.T, cmd tea.Cmd) analysisMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if got, ok := c().(analysisMsg); ok {
				return got
			}
		}
		t.Fatal("no analysis in the batch")
	}
	got, ok := msg.(analysisMsg)
	if !ok {
		t.Fatalf("got %T, want analysisMsg", msg)
	}
	return got
}

// deepen feeds analysis results back until the engine has searched every depth.
func deepen(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for range analysisDepths {
		next, nextCmd := m.Update(runAnalysis(t, cmd))
		m, cmd = next.(Model), nextCmd
		if cmd == nil {
			return m
		}
	}
	t.Fatal("the analysis never ended")
	return m
}

func TestEnteringTheGameStartsTheAnalysis(t *testing.T) {
	a := &scriptedAnalyst{reply: opening}
	next, cmd := analysisModel(t, a).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := next.(Model)
	if !strings.Contains(m.View(), "Analyzing…") || !strings.Contains(m.View(), "analysis · FakeFish") {
		t.Errorf("a pending analysis should say so:\n%s", m.View())
	}
	m = deepen(t, m, cmd)
	v := m.View()
	for _, want := range []string{"Eval +0.34", "depth 22", "1. e4 e5 2. Nf3", "██"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
	if a.searches() != len(analysisDepths) {
		t.Errorf("%d searches, want one per depth (%d)", a.searches(), len(analysisDepths))
	}
	if strings.Contains(v, "Analyzing…") {
		t.Errorf("the analysis is done:\n%s", v)
	}
}

func TestScoreIsShownFromWhitesPointOfView(t *testing.T) {
	a := &scriptedAnalyst{reply: func(fen string, depth int) engine.Info {
		// +0.80 for the side to move, here Black.
		return engine.Info{Depth: depth, MultiPV: 1, HasScore: true, Score: engine.Score{Value: 80}, PV: []string{"e7e5"}}
	}}
	m, err := New(Config{Engine: a, Analyze: true, FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"})
	if err != nil {
		t.Fatal(err)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = deepen(t, next.(Model), cmd)
	if v := m.View(); !strings.Contains(v, "Eval -0.80") || !strings.Contains(v, "1... e5") {
		t.Errorf("Black is better, so White's eval is negative:\n%s", v)
	}
}

func TestMateScore(t *testing.T) {
	a := &scriptedAnalyst{reply: func(fen string, depth int) engine.Info {
		return engine.Info{Depth: depth, MultiPV: 1, HasScore: true, Score: engine.Score{Mate: true, Value: 3}, PV: []string{"e2e4"}}
	}}
	next, cmd := analysisModel(t, a).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := deepen(t, next.(Model), cmd)
	if v := m.View(); !strings.Contains(v, "Eval #3") || strings.Contains(v, "░") {
		t.Errorf("White mates in 3 and the bar is full:\n%s", v)
	}
}

func TestMovingRestartsTheAnalysisAndDropsTheOldOne(t *testing.T) {
	a := &scriptedAnalyst{reply: opening}
	next, firstCmd := analysisModel(t, a).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := next.(Model)
	stale := runAnalysis(t, firstCmd) // the answer for the starting position, still in flight

	var cmd tea.Cmd
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEnter}, {Type: tea.KeyUp}, {Type: tea.KeyUp}, {Type: tea.KeyEnter}} {
		next, cmd = m.Update(key)
		m = next.(Model)
	}
	if len(m.history) != 1 || m.game.GetCurrentTurn() != chess.Black {
		t.Fatalf("the player should be able to move: %v", m.history)
	}
	if m.thinking {
		t.Error("in analysis mode the engine does not play")
	}
	next, _ = m.Update(stale) // arrives after the move: ignored
	m = next.(Model)
	if m.hasAnalysis {
		t.Error("an answer about the old position was applied")
	}
	m = deepen(t, m, cmd)
	if !m.hasAnalysis || m.analysisFEN != m.game.FEN() {
		t.Errorf("the new position should be analyzed: %q vs %q", m.analysisFEN, m.game.FEN())
	}
	a.mu.Lock()
	last := a.calls[len(a.calls)-1]
	a.mu.Unlock()
	if !strings.Contains(last, "4P3") {
		t.Errorf("last search was of %q, want the position after e4", last)
	}
}

func TestUndoReanalyzesAndTheMenuStopsIt(t *testing.T) {
	a := &scriptedAnalyst{reply: opening}
	next, _ := analysisModel(t, a).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := next.(Model)
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEnter}, {Type: tea.KeyUp}, {Type: tea.KeyUp}, {Type: tea.KeyEnter}} {
		next, _ = m.Update(key)
		m = next.(Model)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(Model)
	if len(m.history) != 0 || cmd == nil || m.analysisFEN != chess.StartingFEN {
		t.Errorf("undo should take back one move and analyze the start again: %d moves, FEN %q", len(m.history), m.analysisFEN)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.screen != screenMenu || m.analysisFEN != "" {
		t.Error("leaving the game forgets the analysis")
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || next.(Model).analysisFEN == "" {
		t.Error("coming back analyzes again")
	}
}

func TestNoAnalysisOnceTheGameIsOver(t *testing.T) {
	a := &scriptedAnalyst{reply: opening}
	m, err := New(Config{Engine: a, Analyze: true, FEN: "7k/8/8/8/8/8/5PPP/r5K1 w - - 0 1"}) // White is checkmated
	if err != nil {
		t.Fatal(err)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil && a.searches() > 0 {
		t.Error("a finished game is not analyzed")
	}
	if v := next.View(); strings.Contains(v, "Eval") || strings.Contains(v, "Analyzing") {
		t.Errorf("no evaluation for a finished game:\n%s", v)
	}
}
