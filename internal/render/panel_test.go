package render

import (
	"strings"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func TestStatus(t *testing.T) {
	check := func(fen, want string) {
		t.Helper()
		if got := Status(mustGame(t, fen)); got != want {
			t.Errorf("Status(%q) = %q, want %q", fen, got, want)
		}
	}
	check(chess.StartingFEN, "White to move")
	check("rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3", "Checkmate, Black wins")
	check("3k4/8/8/8/8/8/5q2/7K w - - 0 1", "Stalemate, draw") // no moves for the king: f2 queen covers g1, g2, h2, g... verify below
	check("rnbqkbnr/ppp2ppp/8/3pp3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 3", "White to move")
	check("4k3/4R3/8/8/8/8/8/4K3 b - - 0 1", "Black to move, in check")
	check("8/8/8/8/8/5k2/8/5K1B w - - 0 1", "Draw (InsufficientMaterial)")

	game := chess.NewGame()
	_ = game.Resign(chess.White)
	if got := Status(game); got != "White resigned, Black wins" {
		t.Errorf("resign: %q", got)
	}
	game = chess.NewGame()
	_ = game.OfferDraw(chess.White)
	_ = game.AcceptDraw(chess.Black)
	if got := Status(game); got != "Draw (Agreement)" {
		t.Errorf("agreement: %q", got)
	}
}

func TestFormatClock(t *testing.T) {
	tests := map[time.Duration]string{
		0: "00:00", -time.Second: "00:00", 59 * time.Second: "00:59", 3*time.Minute + 2*time.Second: "03:02",
		90 * time.Minute: "1:30:00", 59*time.Minute + 59*time.Second: "59:59", 1999 * time.Millisecond: "00:01",
	}
	for d, want := range tests {
		if got := FormatClock(d); got != want {
			t.Errorf("FormatClock(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestPanelSnapshots(t *testing.T) {
	moves := []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "Nf6", "O-O", "Be7", "Re1", "b5", "Bb3", "d6", "c3", "O-O", "h3", "Nb8", "d4"}
	golden(t, "panel_timed", Panel{
		Clocks:          &Clocks{White: 4*time.Minute + 12*time.Second, Black: 3*time.Minute + 59*time.Second, Active: chess.Black},
		Moves:           moves,
		FirstMoveNumber: 1,
		Status:          "Black to move",
	}.String())
	golden(t, "panel_untimed_short", Panel{Moves: moves[:3], FirstMoveNumber: 1, Status: "Black to move"}.String())
	golden(t, "panel_black_first", Panel{Moves: []string{"Kd7", "e5", "Ke6"}, FirstMoveNumber: 17, BlackFirst: true, Status: "White to move"}.String())
	golden(t, "panel_stopped", Panel{
		Clocks: &Clocks{White: 90 * time.Minute, Black: -time.Second, Stopped: true}, Moves: moves[:2], FirstMoveNumber: 1, Status: "Black ran out of time, White wins",
	}.String())
}

func TestPanelKeepsItsHeight(t *testing.T) {
	short := Panel{Moves: []string{"e4"}, FirstMoveNumber: 1, Status: "x"}.String()
	long := Panel{Moves: make([]string, 80), FirstMoveNumber: 1, Status: "x"}.String()
	if strings.Count(short, "\n") != strings.Count(long, "\n") {
		t.Errorf("panel height changes with the game: %d vs %d lines", strings.Count(short, "\n"), strings.Count(long, "\n"))
	}
	scrolled := Panel{Moves: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}, FirstMoveNumber: 1, Rows: 3}.String()
	if !strings.Contains(scrolled, "9. q") || strings.Contains(scrolled, "1. a") {
		t.Errorf("only the latest rows should show:\n%s", scrolled)
	}
}

func TestSideBySide(t *testing.T) {
	left := "ab\n\x1b[31mcdef\x1b[0m\n"
	right := "1\n2\n3\n"
	got := SideBySide(left, right, 2)
	want := "ab    1\n\x1b[31mcdef\x1b[0m  2\n      3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := SideBySide("a\nb\n", "x\n", 1); got != "a x\nb\n" {
		t.Errorf("short right side: %q", got)
	}
}

func TestBesideAlignsThePanelWithTheFirstRank(t *testing.T) {
	game := chess.NewGame()
	panel := Panel{Status: "S", FirstMoveNumber: 1}.String()
	plain := Options{}
	withMaterial := Options{Material: true}

	first := func(out string) string { return strings.Split(out, "\n")[0] }
	if got := first(Beside(Board(game.GetBoard(), plain), plain, panel, 3)); !strings.HasPrefix(got, "8 ") {
		t.Errorf("without material lines the first line is rank 8: %q", got)
	}
	out := Beside(Board(game.GetBoard(), withMaterial), withMaterial, "A\nB\n", 3)
	lines := strings.Split(out, "\n")
	if lines[0] != "" || !strings.HasPrefix(lines[1], "8 ") || !strings.HasSuffix(lines[1], "A") {
		t.Errorf("the panel should start level with rank 8, below the material line:\n%s", out)
	}
}
