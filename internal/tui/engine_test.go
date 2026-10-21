package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/internal/timer/timertest"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// fakeEngine answers each search from a channel, so a test decides when the
// engine "finishes thinking". A search ends early if its context does.
type fakeEngine struct {
	answers chan answer

	mu       sync.Mutex
	searches []search
}

type answer struct {
	move string
	err  error
}

type search struct {
	fen    string
	limits engine.Limits
	ctx    context.Context
}

func newFakeEngine() *fakeEngine { return &fakeEngine{answers: make(chan answer, 8)} }

func (f *fakeEngine) Search(ctx context.Context, fen string, limits engine.Limits, _ func(engine.Info)) (engine.Result, error) {
	f.mu.Lock()
	f.searches = append(f.searches, search{fen, limits, ctx})
	f.mu.Unlock()
	select {
	case a := <-f.answers:
		return engine.Result{Best: a.move}, a.err
	case <-ctx.Done():
		return engine.Result{}, ctx.Err()
	}
}

func (f *fakeEngine) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.searches)
}

func (f *fakeEngine) last() search {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.searches[len(f.searches)-1]
}

// searchOf finds the command that runs the engine in what Update returned,
// runs it and returns its message. Other commands, such as the spinner's
// timer, are left alone.
func searchResult(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	results := make(chan tea.Msg, 4)
	var cmds []tea.Cmd
	if batch, ok := cmd().(tea.BatchMsg); ok { // the spinner's tea.Tick sleeps, so it is run too but ignored
		cmds = batch
	} else {
		t.Fatal("expected a batch of commands")
	}
	for _, c := range cmds {
		if c == nil {
			continue
		}
		go func() { results <- c() }()
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case msg := <-results:
			if _, isMove := msg.(engineMoveMsg); isMove {
				return msg
			}
		case <-deadline:
			t.Fatal("the engine's answer never came")
		}
	}
}

func engineModel(t *testing.T, human chess.Color, eng *fakeEngine, opts ...func(*Config)) Model {
	t.Helper()
	cfg := Config{Engine: eng, EngineName: "FakeFish", HumanColor: human}
	for _, opt := range opts {
		opt(&cfg)
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// playE4 is the player's e2-e4 on the game screen, returning the commands of
// the last key.
func playE4(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // menu → game
	m = next.(Model)
	var cmd tea.Cmd
	for _, key := range []string{"enter", "up", "up", "enter"} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		switch key {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		}
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

func TestEngineRepliesWithoutBlockingTheUI(t *testing.T) {
	eng := newFakeEngine()
	m, cmd := playE4(t, engineModel(t, chess.White, eng))
	if !m.thinking || !strings.Contains(m.View(), "FakeFish is thinking…") {
		t.Fatalf("the indicator should show while the engine thinks:\n%s", m.View())
	}

	// The command has not finished, but the UI keeps answering keys.
	done := make(chan tea.Msg, 1)
	go func() { done <- searchResult(t, cmd) }()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = next.(Model)
	if !m.flipped {
		t.Error("f should flip the board while the engine thinks")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // try to pick up a piece
	m = next.(Model)
	if m.selected != nil || !strings.Contains(m.message, "Wait for the engine") {
		t.Errorf("no moving while the engine thinks: selected %v, message %q", m.selected, m.message)
	}
	waitForSearch(t, eng, 1)
	if got := eng.last(); !strings.HasPrefix(got.fen, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b") || got.limits.MoveTime != thinkTime {
		t.Errorf("search = %+v", got)
	}

	eng.answers <- answer{move: "e7e5"}
	reply := <-done
	next, _ = m.Update(reply)
	m = next.(Model)
	if m.thinking {
		t.Error("the engine's reply ends the thinking")
	}
	if len(m.history) != 2 || m.history[1].san != "e5" {
		t.Errorf("history = %+v, want e4 e5", m.history)
	}
	if v := m.View(); strings.Contains(v, "thinking") || !strings.Contains(v, "White to move") {
		t.Errorf("view after the reply:\n%s", v)
	}
	if eng.count() != 1 {
		t.Errorf("the engine searched %d times, want once", eng.count())
	}
}

func TestEngineMovesFirstWhenThePlayerIsBlack(t *testing.T) {
	eng := newFakeEngine()
	m := engineModel(t, chess.Black, eng)
	if !m.flipped {
		t.Error("the board should face Black")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !m.thinking || cmd == nil {
		t.Fatal("entering the game should start the engine's first move")
	}
	eng.answers <- answer{move: "e2e4"}
	next, _ = m.Update(searchResult(t, cmd))
	m = next.(Model)
	if len(m.history) != 1 || m.history[0].san != "e4" || m.game.GetCurrentTurn() != chess.Black {
		t.Errorf("history %+v, turn %v", m.history, m.game.GetCurrentTurn())
	}
	if m.cursor.Y != 1 {
		t.Errorf("the cursor should start on Black's side, at e7: %+v", m.cursor)
	}
}

// waitForSearch blocks until the engine has been asked n times.
func waitForSearch(t *testing.T, eng *fakeEngine, n int) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if eng.count() >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the engine was asked %d times, want %d", eng.count(), n)
}

func TestQuitCancelsTheSearch(t *testing.T) {
	eng := newFakeEngine()
	m, cmd := playE4(t, engineModel(t, chess.White, eng))
	done := make(chan tea.Msg, 1)
	go func() { done <- searchResult(t, cmd) }()
	waitForSearch(t, eng, 1)
	ctx := eng.last().ctx
	if ctx.Err() != nil {
		t.Fatal("the search should be running")
	}

	_, quit := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !isQuit(quit) {
		t.Error("q should quit")
	}
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("quitting must cancel the search that is running")
	}
	if msg := (<-done).(engineMoveMsg); !errors.Is(msg.err, context.Canceled) {
		t.Errorf("the cancelled search answered %+v", msg)
	}
}

func TestUndoWhileThinkingAbandonsTheSearch(t *testing.T) {
	eng := newFakeEngine()
	m, cmd := playE4(t, engineModel(t, chess.White, eng))
	done := make(chan tea.Msg, 1)
	go func() { done <- searchResult(t, cmd) }()
	waitForSearch(t, eng, 1)
	ctx := eng.last().ctx

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(Model)
	if ctx.Err() == nil {
		t.Error("undo must cancel the search")
	}
	if m.thinking || len(m.history) != 0 || m.game.FEN() != chess.StartingFEN {
		t.Errorf("thinking %v, history %v, FEN %s", m.thinking, m.history, m.game.FEN())
	}
	// The cancelled search answers anyway; the stale answer changes nothing.
	stale := <-done
	next, _ = m.Update(stale)
	if got := next.(Model); len(got.history) != 0 || got.game.FEN() != chess.StartingFEN {
		t.Errorf("a stale answer was applied: %v", got.history)
	}
}

func TestUndoTakesBackBothMoves(t *testing.T) {
	eng := newFakeEngine()
	m, cmd := playE4(t, engineModel(t, chess.White, eng))
	eng.answers <- answer{move: "e7e5"}
	next, _ := m.Update(searchResult(t, cmd))
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(Model)
	if len(m.history) != 0 || m.game.FEN() != chess.StartingFEN || m.thinking {
		t.Errorf("undo should take back the engine's reply and the player's move: %v", m.history)
	}
}

func TestEngineFailureIsReportedOnceAndNotRetried(t *testing.T) {
	eng := newFakeEngine()
	m, cmd := playE4(t, engineModel(t, chess.White, eng))
	eng.answers <- answer{err: errors.New("engine exited")}
	next, again := m.Update(searchResult(t, cmd))
	m = next.(Model)
	if again != nil {
		if _, isBatch := again().(tea.BatchMsg); isBatch {
			t.Error("a failed engine must not be asked again")
		}
	}
	if !strings.Contains(m.message, "The engine failed: engine exited") || m.thinking {
		t.Errorf("message %q, thinking %v", m.message, m.thinking)
	}
	if eng.count() != 1 {
		t.Errorf("%d searches, want 1", eng.count())
	}
	// Taking the move back lets the player try again.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if got := next.(Model); got.engineDown {
		t.Error("undo should clear the failure")
	}
}

func TestIllegalEngineMoveIsRefused(t *testing.T) {
	for _, bad := range []string{"e7e4", "(none)", "zz"} {
		eng := newFakeEngine()
		m, cmd := playE4(t, engineModel(t, chess.White, eng))
		eng.answers <- answer{move: bad}
		next, _ := m.Update(searchResult(t, cmd))
		m = next.(Model)
		if len(m.history) != 1 || !strings.Contains(m.message, "not a legal move") {
			t.Errorf("%q: history %v, message %q", bad, m.history, m.message)
		}
	}
}

func TestTimedGameGivesTheEngineItsClock(t *testing.T) {
	eng := newFakeEngine()
	config := chess.TimeConfig{InitialTime: 3 * time.Minute, Increment: 2 * time.Second}
	m, _ := playE4(t, engineModel(t, chess.White, eng, func(c *Config) {
		c.TimeControl, c.Clock = &config, timertest.NewFakeClock()
	}))
	limits := m.limits()
	if limits.WTime != 3*time.Minute+2*time.Second || limits.BTime != 3*time.Minute || limits.WInc != 2*time.Second || limits.MoveTime != 0 {
		t.Errorf("limits = %+v", limits)
	}
}

func TestMenuStopsTheSearchAndComingBackRestartsIt(t *testing.T) {
	eng := newFakeEngine()
	m, _ := playE4(t, engineModel(t, chess.White, eng))
	if !m.thinking {
		t.Fatal("setup: the engine should be thinking")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // esc with nothing in hand: to the menu
	m = next.(Model)
	if m.screen != screenMenu || m.thinking {
		t.Errorf("screen %v, thinking %v", m.screen, m.thinking)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !m.thinking || cmd == nil {
		t.Error("returning to the game should restart the search")
	}
}
