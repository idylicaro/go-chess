package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/render"
)

func newModel(t *testing.T) Model {
	t.Helper()
	m, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func press(m tea.Model, keys ...string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		switch key {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "ctrl+c":
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		}
		m, cmd = m.Update(msg)
	}
	return m, cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestNewRejectsABadFEN(t *testing.T) {
	if _, err := New(Config{FEN: "junk"}); err == nil {
		t.Error("expected an error for a bad FEN")
	}
}

func TestStartsOnTheMenuAndEnterStartsAGame(t *testing.T) {
	m := newModel(t)
	if m.screen != screenMenu || !strings.Contains(m.View(), "enter   start a game") {
		t.Errorf("first screen should be the menu:\n%s", m.View())
	}
	next, _ := press(m, "enter")
	if next.(Model).screen != screenGame {
		t.Error("enter should start the game")
	}
	view := next.View()
	if !strings.Contains(view, "White to move") || !strings.Contains(view, "a  b  c  d  e  f  g  h") {
		t.Errorf("game screen lacks the board or status:\n%s", view)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, key := range []string{"q", "ctrl+c"} {
		for _, start := range [][]string{nil, {"enter"}} {
			m, _ := press(newModel(t), start...)
			if _, cmd := press(m, key); !isQuit(cmd) {
				t.Errorf("%q after %v should quit", key, start)
			}
		}
	}
	if _, cmd := press(newModel(t), "esc"); !isQuit(cmd) {
		t.Error("esc on the menu should quit")
	}
	m, cmd := press(newModel(t), "enter", "esc")
	if isQuit(cmd) || m.(Model).screen != screenMenu {
		t.Error("esc in a game should go back to the menu")
	}
}

func TestFlipKey(t *testing.T) {
	m, _ := press(newModel(t), "enter")
	before := m.View()
	m, _ = press(m, "f")
	after := m.View()
	if before == after || !strings.Contains(after, "h  g  f  e  d  c  b  a") {
		t.Errorf("f should turn the board around:\n%s", after)
	}
	m, _ = press(m, "f")
	if m.View() != before {
		t.Error("f twice should restore the board")
	}
}

func TestWindowSizeCentersTheView(t *testing.T) {
	m, _ := newModel(t).Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 30 {
		t.Errorf("view has %d lines, want the terminal height of 30", len(lines))
	}
}

func TestViewUsesTheConfiguredStyle(t *testing.T) {
	m, _ := New(Config{Style: render.Style{Unicode: true}})
	next, _ := press(m, "enter")
	if !strings.Contains(next.View(), "♜") {
		t.Errorf("Unicode style should draw chess symbols:\n%s", next.View())
	}
}
