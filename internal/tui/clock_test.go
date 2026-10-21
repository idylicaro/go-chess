package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/timer/timertest"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func timedModel(t *testing.T, config chess.TimeConfig) (Model, *timertest.FakeClock) {
	t.Helper()
	clock := timertest.NewFakeClock()
	m, err := New(Config{TimeControl: &config, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	return m, clock
}

// enter presses enter on the menu and returns the model and the first tick.
func enter(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model), cmd
}

func TestUntimedGameHasNoClocksAndNoTicks(t *testing.T) {
	m, cmd := enter(t, newModel(t))
	if cmd != nil || strings.Contains(m.View(), "▶") {
		t.Errorf("an untimed game must not tick or show clocks:\n%s", m.View())
	}
}

func TestClocksShowAndTickDown(t *testing.T) {
	m, clock := timedModel(t, chess.TimeConfig{InitialTime: 5 * time.Minute})
	m, cmd := enter(t, m)
	if cmd == nil {
		t.Fatal("entering a timed game must start the ticks")
	}
	if v := m.View(); !strings.Contains(v, "▶ White") || !strings.Contains(v, "05:00") {
		t.Errorf("clocks missing:\n%s", v)
	}
	clock.Advance(3 * time.Second)
	next, cmd := m.Update(tickMsg{m.tickGen})
	m = next.(Model)
	if cmd == nil {
		t.Error("the tick must schedule the next one")
	}
	if v := m.View(); !strings.Contains(v, "04:57") {
		t.Errorf("White's clock should read 04:57:\n%s", v)
	}
}

func TestTickCommandFiresATickMessage(t *testing.T) {
	m, _ := New(Config{TimeControl: &chess.TimeConfig{InitialTime: time.Minute}, TickInterval: time.Millisecond})
	_, cmd := enter(t, m)
	msg := cmd()
	if tick, ok := msg.(tickMsg); !ok || tick.gen != 1 {
		t.Errorf("msg = %#v, want tickMsg{gen: 1}", msg)
	}
}

func TestIncrementAndTheOtherClockAfterAMove(t *testing.T) {
	m, clock := timedModel(t, chess.TimeConfig{InitialTime: time.Minute, Increment: 2 * time.Second})
	m, _ = enter(t, m)
	clock.Advance(5 * time.Second)
	next, _ := press(m, "enter", "up", "up", "enter") // e2-e4
	v := next.View()
	if !strings.Contains(v, "▶ Black") || !strings.Contains(v, "00:57") || !strings.Contains(v, "01:00") {
		t.Errorf("White spent 5s and earned 2s; Black's clock now runs:\n%s", v)
	}
}

func TestFlagEndsTheGame(t *testing.T) {
	m, clock := timedModel(t, chess.TimeConfig{InitialTime: 10 * time.Second})
	m, _ = enter(t, m)
	next, _ := press(m, "enter") // e2 picked up, about to move
	m = next.(Model)
	clock.Advance(11 * time.Second)
	updated, cmd := m.Update(tickMsg{m.tickGen})
	m = updated.(Model)
	if cmd != nil {
		t.Error("no more ticks after the game ended")
	}
	if !m.game.IsOver() || m.game.GetState() != chess.TimeoutWhite {
		t.Fatalf("state = %v", m.game.GetState())
	}
	if m.selected != nil || m.message != "White ran out of time, Black wins" {
		t.Errorf("selection %v, message %q", m.selected, m.message)
	}
	if v := m.View(); !strings.Contains(v, "White ran out of time, Black wins") || strings.Contains(v, "▶") {
		t.Errorf("view:\n%s", v)
	}
	next, _ = press(m, "enter") // nothing can be picked up now
	if next.(Model).selected != nil {
		t.Error("a finished game takes no moves")
	}
}

func TestMoveAfterTheFlagBeforeTheTickIsRefused(t *testing.T) {
	m, clock := timedModel(t, chess.TimeConfig{InitialTime: 10 * time.Second})
	m, _ = enter(t, m)
	next, _ := press(m, "enter", "up", "up") // pawn in hand, cursor on e4
	clock.Advance(11 * time.Second)          // the flag falls before any tick
	next, _ = press(next, "enter")
	got := next.(Model)
	if len(got.history) != 0 || got.message != "White ran out of time, Black wins" {
		t.Errorf("history %v, message %q", got.history, got.message)
	}
}

func TestMenuPausesTheClock(t *testing.T) {
	m, clock := timedModel(t, chess.TimeConfig{InitialTime: time.Minute})
	m, _ = enter(t, m)
	clock.Advance(10 * time.Second)
	next, _ := press(m, "esc") // back to the menu: the clock stops
	m = next.(Model)
	clock.Advance(time.Hour)
	if updated, cmd := m.Update(tickMsg{m.tickGen}); cmd != nil || updated.(Model).game.IsOver() {
		t.Error("a tick on the menu must neither flag nor reschedule")
	}
	m, cmd := enter(t, m)
	if cmd == nil {
		t.Error("returning to the game restarts the ticks")
	}
	if v := m.View(); !strings.Contains(v, "00:50") {
		t.Errorf("the hour on the menu must not count:\n%s", v)
	}
}

func TestStaleTicksAreIgnored(t *testing.T) {
	m, _ := timedModel(t, chess.TimeConfig{InitialTime: time.Minute})
	m, _ = enter(t, m)
	oldGen := m.tickGen
	next, _ := press(m, "esc")
	m, _ = enter(t, next.(Model))
	if m.tickGen == oldGen {
		t.Fatal("each visit to the game starts a new chain of ticks")
	}
	if _, cmd := m.Update(tickMsg{oldGen}); cmd != nil {
		t.Error("a tick from an earlier chain must not reschedule, or the ticks would double up")
	}
}

func TestNewTimedGameFromFEN(t *testing.T) {
	config := chess.TimeConfig{InitialTime: time.Minute}
	m, err := New(Config{FEN: "4k3/8/8/8/8/8/4P3/4K3 b - - 0 1", TimeControl: &config})
	if err != nil {
		t.Fatal(err)
	}
	m, _ = enter(t, m)
	if v := m.View(); !strings.Contains(v, "▶ Black") {
		t.Errorf("Black to move first, so Black's clock runs:\n%s", v)
	}
}
