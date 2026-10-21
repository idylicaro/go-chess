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

// game starts a fresh game on the game screen.
func startedGame(t *testing.T) tea.Model {
	t.Helper()
	m, _ := press(newModel(t), "enter")
	return m
}

func at(m tea.Model) Model { return m.(Model) }

func TestMoveByKeyboard(t *testing.T) {
	m := startedGame(t)
	// The cursor starts on e2: pick the pawn up, go up two squares, drop it.
	m, _ = press(m, "enter")
	if at(m).selected == nil || len(at(m).targets) != 2 {
		t.Fatalf("e2 pawn should be selected with two targets: %+v", at(m))
	}
	if !strings.Contains(m.View(), "|P|") || strings.Count(m.View(), "*") != 2 { // the cursor is on the pawn
		t.Errorf("selection and targets should be drawn:\n%s", m.View())
	}
	m, _ = press(m, "up", "up", "enter")
	if got := at(m).game.FEN(); !strings.HasPrefix(got, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b") {
		t.Errorf("FEN = %q", got)
	}
	if !strings.Contains(m.View(), "1. e4") || !strings.Contains(m.View(), "Black to move") {
		t.Errorf("panel should list the move:\n%s", m.View())
	}
}

func TestVimKeysAndCursorStopsAtTheEdge(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "k", "k", "k", "k", "k", "k", "k", "k", "h", "h", "h", "h", "h", "h")
	if c := at(m).cursor; c.X != 0 || c.Y != 0 {
		t.Errorf("cursor = %+v, want the a8 corner", c)
	}
	m, _ = press(m, "j", "l")
	if c := at(m).cursor; c.X != 1 || c.Y != 1 {
		t.Errorf("cursor = %+v, want b7", c)
	}
}

func TestCursorFollowsTheOrientation(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "f", "up") // flipped: up on screen is toward rank 1
	if c := at(m).cursor; c.X != 4 || c.Y != 7 {
		t.Errorf("flipped cursor after up = %+v, want e1", c)
	}
}

func TestRefusedPicksExplainThemselves(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "k", "k", "k", "k", "k", "enter") // e7: Black's pawn on White's turn
	if at(m).selected != nil || !strings.Contains(at(m).message, "Black's piece") {
		t.Errorf("message = %q", at(m).message)
	}
	m, _ = press(m, "j", "j", "j", "j", "j", "j", "j", "j", "enter") // e1: the king is boxed in
	if at(m).selected != nil || !strings.Contains(at(m).message, "has no legal move") {
		t.Errorf("message = %q", at(m).message)
	}
	m, _ = press(m, "k", "k", "k", "k", "enter") // e3: empty
	if at(m).selected != nil || at(m).message != "" {
		t.Errorf("empty square: selected %v, message %q", at(m).selected, at(m).message)
	}
}

func TestDropOnAnIllegalSquareReselects(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "enter", "k", "k", "k", "enter") // e2 selected, e5 is not a target
	if at(m).selected != nil || len(at(m).history) != 0 {
		t.Errorf("selection should be dropped without a move: %+v", at(m))
	}
	m, _ = press(m, "j", "j", "j", "enter", "esc") // back on e2: pick up, then esc
	if at(m).selected != nil || at(m).screen != screenGame {
		t.Error("esc with a piece in hand should only put it down")
	}
	m, _ = press(m, "esc")
	if at(m).screen != screenMenu {
		t.Error("esc again goes back to the menu")
	}
}

func TestReselectingSwitchesPiece(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "enter", "l", "enter") // pick e2, then f2: the pawn in hand changes
	if s := at(m).selected; s == nil || s.X != 5 {
		t.Errorf("selected = %v, want the f2 pawn", s)
	}
}

// clickAt builds a left click on the cell of the rendered view where the
// glyph of the first piece in the line starting with rank is, shifted by files.
func clickOn(t *testing.T, m tea.Model, rank string, file int) tea.MouseMsg {
	t.Helper()
	for y, line := range strings.Split(m.View(), "\n") {
		if trimmed := strings.TrimLeft(line, " "); strings.HasPrefix(trimmed, rank+" ") {
			x := strings.IndexAny(line, "PRNBQKpnbrqk.") // first piece or empty square: file a
			return tea.MouseMsg{X: x + 3*file, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
		}
	}
	t.Fatalf("no rank %s in the view:\n%s", rank, m.View())
	return tea.MouseMsg{}
}

func TestMoveByMouse(t *testing.T) {
	m := startedGame(t)
	m, _ = m.Update(clickOn(t, m, "2", 4)) // e2
	if at(m).selected == nil || at(m).cursor.X != 4 || at(m).cursor.Y != 6 {
		t.Fatalf("click on e2: %+v", at(m))
	}
	m, _ = m.Update(clickOn(t, m, "4", 4)) // e4
	if !strings.HasPrefix(at(m).game.FEN(), "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b") {
		t.Errorf("FEN = %q", at(m).game.FEN())
	}
}

func TestMouseOnAFlippedBoard(t *testing.T) {
	m, _ := press(startedGame(t), "f")
	// Flipped, the h-file is on the left: the first cell of rank 2 is h2.
	m, _ = m.Update(clickOn(t, m, "2", 0))
	if s := at(m).selected; s == nil || s.X != 7 || s.Y != 6 {
		t.Errorf("selected = %v, want h2", s)
	}
}

func TestMouseIgnoresEverythingButLeftPressOnTheBoard(t *testing.T) {
	m := startedGame(t)
	for _, msg := range []tea.MouseMsg{
		{X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},   // outside the board
		{X: 40, Y: 40, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, // far outside
		{X: 8, Y: 9, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}, // release
		{X: 8, Y: 9, Action: tea.MouseActionPress, Button: tea.MouseButtonRight},  // other button
		{X: 8, Y: 9, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone},  // motion
	} {
		next, _ := m.Update(msg)
		if at(next).selected != nil || at(next).cursor != at(m).cursor {
			t.Errorf("%+v changed the state", msg)
		}
	}
	menu := newModel(t)
	next, _ := menu.Update(tea.MouseMsg{X: 8, Y: 9, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if at(next).screen != screenMenu {
		t.Error("a click on the menu must not start anything")
	}
}

// readyToPromote is a game with White's pawn about to move from e7 to e8. A
// Model shares its game with its copies, so each case builds its own.
func readyToPromote(t *testing.T) tea.Model {
	t.Helper()
	m, err := New(Config{FEN: "7k/4P3/8/8/8/8/8/4K3 w - - 0 1"})
	if err != nil {
		t.Fatal(err)
	}
	next, _ := press(m, "enter")
	next, _ = press(next, "k", "k", "k", "k", "k", "enter") // cursor e2 → e7: pick the pawn up
	if at(next).selected == nil {
		t.Fatalf("pawn not selected: %+v", at(next))
	}
	next, _ = press(next, "k", "enter") // e8
	return next
}

func TestPromotionDialog(t *testing.T) {
	m := readyToPromote(t)
	if at(m).promoting == nil || !strings.Contains(m.View(), "Promote to:") {
		t.Fatalf("promotion dialog expected:\n%s", m.View())
	}
	if len(at(m).history) != 0 {
		t.Error("no move before the piece is chosen")
	}
}

func TestPromotionPicksThePiece(t *testing.T) {
	// q is the queen here and must not quit.
	for key, want := range map[string]string{"q": "e8=Q+", "r": "e8=R+", "b": "e8=B", "n": "e8=N"} {
		done, cmd := press(readyToPromote(t), key)
		if isQuit(cmd) {
			t.Fatalf("%s quit the program instead of picking a piece", key)
		}
		if h := at(done).history; len(h) != 1 || h[0].san != want {
			t.Errorf("%s: history = %+v, want %s", key, h, want)
		}
		if at(done).promoting != nil {
			t.Errorf("%s: the dialog should close", key)
		}
	}
}

func TestPromotionWithArrowsAndEnter(t *testing.T) {
	// The dialog opens on the Queen; left wraps around to the Knight.
	done, _ := press(readyToPromote(t), "left", "enter")
	if h := at(done).history; len(h) != 1 || h[0].san != "e8=N" {
		t.Errorf("left+enter: %+v", h)
	}
	done, _ = press(readyToPromote(t), "right", "enter")
	if h := at(done).history; len(h) != 1 || h[0].san != "e8=R+" {
		t.Errorf("right+enter: %+v", h)
	}
	done, _ = press(readyToPromote(t), "enter")
	if h := at(done).history; len(h) != 1 || h[0].san != "e8=Q+" {
		t.Errorf("enter: %+v", h)
	}
}

func TestPromotionCanBeCancelled(t *testing.T) {
	cancelled, cmd := press(readyToPromote(t), "esc")
	if isQuit(cmd) || at(cancelled).promoting != nil || len(at(cancelled).history) != 0 || at(cancelled).selected != nil {
		t.Errorf("esc should cancel the promotion: %+v", at(cancelled))
	}
	// Other keys do nothing while the dialog is open.
	same, cmd := press(readyToPromote(t), "x", "f", "u")
	if isQuit(cmd) || at(same).promoting == nil || at(same).flipped {
		t.Error("unrelated keys must not act while the dialog is open")
	}
}

func TestUndoKeyAndMessageAtTheStart(t *testing.T) {
	m := startedGame(t)
	m, _ = press(m, "u")
	if at(m).message != "Nothing to take back." {
		t.Errorf("message = %q", at(m).message)
	}
	m, _ = press(m, "enter", "up", "up", "enter", "u")
	if at(m).game.FEN() != chessStart || len(at(m).history) != 0 || at(m).message != "" {
		t.Errorf("undo: %q, %d moves, message %q", at(m).game.FEN(), len(at(m).history), at(m).message)
	}
}

const chessStart = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func TestGameOverStopsMoves(t *testing.T) {
	m, err := New(Config{FEN: "7k/8/8/8/8/8/5PPP/r5K1 w - - 0 1"}) // White is mated? back-rank: Ra1+ ... king g1, pawns f2 g2 h2
	if err != nil {
		t.Fatal(err)
	}
	next, _ := press(m, "enter")
	if !at(next).game.IsOver() {
		t.Fatalf("setup should be checkmate: %s", at(next).game.FEN())
	}
	next, _ = press(next, "enter")
	if at(next).selected != nil {
		t.Error("nothing can be picked up once the game is over")
	}
	if !strings.Contains(next.View(), "Checkmate, Black wins") {
		t.Errorf("status:\n%s", next.View())
	}
}

func TestMateEndsTheGameWithAMessage(t *testing.T) {
	m, err := New(Config{FEN: "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1"})
	if err != nil {
		t.Fatal(err)
	}
	next, _ := press(m, "enter")
	// Cursor e2 → a1 (left 4, down 1): pick the rook up and mate on a8.
	next, _ = press(next, "h", "h", "h", "h", "j", "enter", "k", "k", "k", "k", "k", "k", "k", "enter")
	if !at(next).game.IsOver() || !strings.Contains(at(next).message, "Checkmate, White wins") {
		t.Errorf("message %q, FEN %s", at(next).message, at(next).game.FEN())
	}
}
