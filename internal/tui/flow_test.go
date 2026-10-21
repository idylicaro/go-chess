package tui

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/internal/timer/timertest"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// These tests run the model in a real tea.Program through teatest: the
// messages, commands, ticks and rendering all go through bubbletea's loop.

const waitFor = 3 * time.Second

func start(t *testing.T, cfg Config) *teatest.TestModel {
	t.Helper()
	m, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return teatest.NewTestModel(t, m, teatest.WithInitialTermSize(90, 30))
}

func expect(t *testing.T, tm *teatest.TestModel, text string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool { return bytes.Contains(out, []byte(text)) },
		teatest.WithDuration(waitFor), teatest.WithCheckInterval(10*time.Millisecond))
}

func expectMatch(t *testing.T, tm *teatest.TestModel, pattern string) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool { return re.Match(out) },
		teatest.WithDuration(waitFor), teatest.WithCheckInterval(10*time.Millisecond))
}

func typeKeys(tm *teatest.TestModel, names ...string) {
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
		}
		tm.Send(msg)
	}
}

func finish(t *testing.T, tm *teatest.TestModel) Model {
	t.Helper()
	typeKeys(tm, "q")
	return tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(Model)
}

func TestFlowStartPlayAndQuit(t *testing.T) {
	tm := start(t, Config{})
	expect(t, tm, "start a game")
	typeKeys(tm, "enter")
	expect(t, tm, "White to move")
	typeKeys(tm, "enter", "up", "up", "enter") // e2-e4
	expect(t, tm, "1. e4")

	final := finish(t, tm)
	if !strings.HasPrefix(final.game.FEN(), "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b") {
		t.Errorf("FEN = %q", final.game.FEN())
	}
}

func TestFlowCtrlCQuitsFromTheMenu(t *testing.T) {
	tm := start(t, Config{})
	expect(t, tm, "start a game")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

func TestFlowMouse(t *testing.T) {
	tm := start(t, Config{})
	typeKeys(tm, "enter")
	expect(t, tm, "White to move")
	// Screen position of a square: left padding, rank label, three cells a file;
	// below the padding, the title, a blank line and the material line.
	click := func(file, rank int) {
		tm.Send(tea.MouseMsg{
			X:      gamePadX + boardLabelW + file*squareW + 1,
			Y:      gamePadY + headerLines + materialLines + (8 - rank),
			Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
		})
	}
	click(4, 2) // e2
	click(4, 4) // e4
	expect(t, tm, "1. e4")
	click(3, 7) // d7
	click(3, 5) // d5
	expect(t, tm, "d5")
	final := finish(t, tm)
	if len(final.history) != 2 || final.history[1].san != "d5" {
		t.Errorf("history = %+v", final.history)
	}
}

func TestFlowEngineAnswersWhileTheInterfaceStaysAlive(t *testing.T) {
	eng := newFakeEngine()
	tm := start(t, Config{Engine: eng, EngineName: "FakeFish", HumanColor: chess.White})
	typeKeys(tm, "enter", "enter", "up", "up", "enter") // e2-e4
	expect(t, tm, "FakeFish is thinking")
	// The engine has not answered, yet the interface still reacts.
	typeKeys(tm, "f")
	expect(t, tm, "h  g  f  e  d  c  b  a")
	eng.answers <- answer{move: "e7e5"}
	expectMatch(t, tm, `e4\s+e5`)
	final := finish(t, tm)
	if len(final.history) != 2 || final.thinking {
		t.Errorf("history %+v, thinking %v", final.history, final.thinking)
	}
}

func TestFlowQuitWhileTheEngineThinksCancelsTheSearch(t *testing.T) {
	eng := newFakeEngine()
	tm := start(t, Config{Engine: eng, EngineName: "FakeFish"})
	typeKeys(tm, "enter", "enter", "up", "up", "enter")
	expect(t, tm, "FakeFish is thinking")
	waitForSearch(t, eng, 1)
	ctx := eng.last().ctx
	typeKeys(tm, "q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Error("the engine's search was left running after quitting")
	}
}

func TestFlowFlagFalls(t *testing.T) {
	clock := timertest.NewFakeClock()
	config := chess.TimeConfig{InitialTime: 30 * time.Second}
	tm := start(t, Config{TimeControl: &config, Clock: clock, TickInterval: 5 * time.Millisecond})
	typeKeys(tm, "enter")
	expect(t, tm, "00:30")
	clock.Advance(10 * time.Second)
	expect(t, tm, "00:20")
	clock.Advance(25 * time.Second)
	expect(t, tm, "White ran out of time, Black wins")
	final := finish(t, tm)
	if final.game.GetState() != chess.TimeoutWhite {
		t.Errorf("state = %v", final.game.GetState())
	}
}

func TestFlowAnalysisDeepens(t *testing.T) {
	a := &scriptedAnalyst{reply: opening}
	tm := start(t, Config{Engine: a, EngineName: "FakeFish", Analyze: true})
	typeKeys(tm, "enter")
	expect(t, tm, "Eval +0.34")
	expect(t, tm, "depth 22")
	final := finish(t, tm)
	if a.searches() != len(analysisDepths) || !final.hasAnalysis {
		t.Errorf("%d searches, analysis %v", a.searches(), final.hasAnalysis)
	}
}

// With a real Stockfish, if there is one: the same flows through the real
// UCI client and process.
func stockfish(t *testing.T) *engine.Engine {
	t.Helper()
	if testing.Short() {
		t.Skip("skipped with -short")
	}
	path, err := exec.LookPath("stockfish")
	if err != nil {
		t.Skip("stockfish is not installed")
	}
	eng, err := engine.Start(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { eng.Close() })
	return eng
}

func TestFlowAgainstStockfish(t *testing.T) {
	eng := stockfish(t)
	if err := eng.SetOption("Skill Level", "0"); err != nil {
		t.Fatal(err)
	}
	tm := start(t, Config{Engine: eng, EngineName: eng.Name})
	typeKeys(tm, "enter", "enter", "up", "up", "enter")
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool { return regexp.MustCompile(`1\. e4 +[a-hNBRQKO]\S*`).Match(out) },
		teatest.WithDuration(10*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
	final := finish(t, tm)
	if len(final.history) != 2 {
		t.Errorf("Stockfish should have replied: %+v", final.history)
	}
}

func TestFlowAnalysisWithStockfish(t *testing.T) {
	eng := stockfish(t)
	tm := start(t, Config{Engine: eng, EngineName: eng.Name, Analyze: true})
	typeKeys(tm, "enter")
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool { return regexp.MustCompile(`Eval [+-]\d\.\d\d\s+depth \d+`).Match(out) },
		teatest.WithDuration(10*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
	final := finish(t, tm)
	if !final.hasAnalysis || final.analysis.Depth < 6 {
		t.Errorf("analysis = %+v", final.analysis)
	}
}

func TestFlowQuitWhileStockfishThinks(t *testing.T) {
	eng := stockfish(t)
	tm := start(t, Config{Engine: eng, EngineName: eng.Name})
	typeKeys(tm, "enter", "enter", "up", "up", "enter")
	expect(t, tm, "is thinking")
	typeKeys(tm, "q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	// The engine is free again: the stop was honored and nothing is stuck.
	done := make(chan error, 1)
	go func() {
		_, err := eng.Search(context.Background(), chess.StartingFEN, engine.Limits{Depth: 1}, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("search after the quit: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the engine is still busy after the program quit")
	}
}
