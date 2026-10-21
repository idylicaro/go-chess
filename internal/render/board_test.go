package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// golden compares got with testdata/<name>.golden, or rewrites it with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Errorf("%s differs from the golden file.\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func mustGame(t *testing.T, fen string) *chess.Game {
	t.Helper()
	game, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	return game
}

var styles = map[string]Style{
	"color_unicode": {Color: true, Unicode: true},
	"color_ascii":   {Color: true},
	"plain_unicode": {Unicode: true},
	"plain_ascii":   {},
}

func TestBoardSnapshots(t *testing.T) {
	game := mustGame(t, "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			golden(t, "board_"+name, Board(game.GetBoard(), Options{Style: style}))
		})
	}
}

func TestBoardFlipped(t *testing.T) {
	game := chess.NewGame()
	golden(t, "board_flipped", Board(game.GetBoard(), Options{Flipped: true}))
}

func TestBoardHasEightRanksAndFileLabels(t *testing.T) {
	out := Board(chess.NewGame().GetBoard(), Options{})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 9 {
		t.Fatalf("got %d lines, want 9:\n%s", len(lines), out)
	}
	if strings.Fields(lines[8])[0] != "a" || strings.Join(strings.Fields(lines[8]), "") != "abcdefgh" {
		t.Errorf("file labels = %q", lines[8])
	}
	if lines[0] != "8  r  n  b  q  k  b  n  r  8" || lines[7] != "1  R  N  B  Q  K  B  N  R  1" {
		t.Errorf("first and last ranks:\n%s", out)
	}
}

func TestColorSquaresAlternate(t *testing.T) {
	out := Board(chess.NewGame().GetBoard(), Options{Style: Style{Color: true}})
	if !strings.Contains(out, "48;5;187") || !strings.Contains(out, "48;5;107") {
		t.Errorf("missing light or dark square colors:\n%q", out)
	}
	// a1 is a dark square, h1 a light one: the last rank's first cell is dark.
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[7], "1 \x1b[48;5;107") {
		t.Errorf("a1 should be dark: %q", lines[7])
	}
}

func TestBoardWithoutColorHasNoEscapes(t *testing.T) {
	for _, style := range []Style{{}, {Unicode: true}} {
		if out := Board(chess.NewGame().GetBoard(), Options{Style: style}); strings.Contains(out, "\x1b") {
			t.Errorf("%+v: output has escapes:\n%q", style, out)
		}
	}
}
