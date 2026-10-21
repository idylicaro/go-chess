package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/internal/common"
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

func play(t *testing.T, game *chess.Game, sans ...string) chess.Move {
	t.Helper()
	var move chess.Move
	for _, san := range sans {
		var err error
		if move, err = game.ParseSAN(san); err != nil {
			t.Fatal(err)
		}
		if err := game.Move(move); err != nil {
			t.Fatal(err)
		}
	}
	return move
}

func TestHighlightsLastMoveAndCheck(t *testing.T) {
	game := chess.NewGame()
	last := play(t, game, "e4", "e5", "Qh5", "Nc6", "Qxf7+")
	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			golden(t, "highlight_check_"+name, Board(game.GetBoard(), Options{Style: style, LastMove: &last, Check: CheckSquare(game)}))
		})
	}
}

func TestHighlightsSelectionAndTargets(t *testing.T) {
	game := mustGame(t, "rnbqkbnr/ppp1pppp/8/3p4/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 0 3")
	from, _ := ParseSquare("e4")
	targets := Targets(game, from)
	if len(targets) != 2 { // e5 and the capture on d5
		t.Fatalf("targets = %v", targets)
	}
	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			golden(t, "highlight_select_"+name, Board(game.GetBoard(), Options{Style: style, Selected: &from, Targets: targets}))
		})
	}
}

func TestMarksDoNotChangeThePieces(t *testing.T) {
	game := chess.NewGame()
	last := play(t, game, "e4")
	from, _ := ParseSquare("g8")
	plain := Board(game.GetBoard(), Options{})
	marked := Board(game.GetBoard(), Options{LastMove: &last, Selected: &from, Targets: Targets(game, from)})
	if strings.Count(plain, "\n") != strings.Count(marked, "\n") {
		t.Error("highlights changed the number of lines")
	}
}

func TestCheckSquareOnlyWhileInCheck(t *testing.T) {
	game := chess.NewGame()
	if CheckSquare(game) != nil {
		t.Error("check square at the start")
	}
	play(t, game, "f3", "e5", "g4", "Qh4#")
	if sq := CheckSquare(game); sq == nil || SquareName(*sq) != "e1" {
		t.Errorf("checkmated king square = %v", sq)
	}
}

func TestTargets(t *testing.T) {
	game := chess.NewGame()
	names := func(from string) string {
		p, _ := ParseSquare(from)
		var out []string
		for _, target := range Targets(game, p) {
			out = append(out, SquareName(target))
		}
		return strings.Join(out, " ")
	}
	if got := names("g1"); got != "f3 h3" && got != "h3 f3" {
		t.Errorf("g1 targets = %q", got)
	}
	if got := names("e4"); got != "" { // empty square
		t.Errorf("empty square targets = %q", got)
	}
	if got := names("e7"); got != "" { // Black's piece on White's turn
		t.Errorf("opponent's piece targets = %q", got)
	}
	promo := mustGame(t, "7k/4P3/8/8/8/8/8/4K3 w - - 0 1")
	from, _ := ParseSquare("e7")
	if got := Targets(promo, from); len(got) != 1 {
		t.Errorf("promotion square counted %d times: %v", len(got), got)
	}
}

func TestParseSquare(t *testing.T) {
	for _, name := range []string{"a1", "h8", "E4", " e4 "} {
		p, ok := ParseSquare(name)
		if !ok || SquareName(p) != strings.ToLower(strings.TrimSpace(name)) {
			t.Errorf("ParseSquare(%q) = %v, %v", name, p, ok)
		}
	}
	for _, name := range []string{"", "e", "e9", "i1", "e44", "e0"} {
		if _, ok := ParseSquare(name); ok {
			t.Errorf("ParseSquare(%q) succeeded", name)
		}
	}
}

func TestMaterialAndCaptured(t *testing.T) {
	game := chess.NewGame()
	if w, b := Material(game.GetBoard()); w != 39 || b != 39 {
		t.Errorf("starting material = %d, %d", w, b)
	}
	if byWhite, byBlack := Captured(game.GetBoard()); len(byWhite) != 0 || len(byBlack) != 0 {
		t.Errorf("captures at the start: %v %v", byWhite, byBlack)
	}
	play(t, game, "e4", "d5", "exd5", "Qxd5", "Nc3", "Qxg2", "Nf3", "Qxh1")
	// White took a pawn; Black took pawns g2 and a rook, and the queen took d5's pawn.
	byWhite, byBlack := Captured(game.GetBoard())
	if len(byWhite) != 1 || byWhite[0] != common.Pawn {
		t.Errorf("White captured %v, want one pawn", byWhite)
	}
	want := []common.PieceType{common.Rook, common.Pawn, common.Pawn}
	if len(byBlack) != 3 || byBlack[0] != want[0] || byBlack[1] != want[1] || byBlack[2] != want[2] {
		t.Errorf("Black captured %v, want %v", byBlack, want)
	}
	if got := Balance(game.GetBoard()); got != -6 { // white lost R+2P (7), black lost P (1)
		t.Errorf("balance = %d, want -6", got)
	}
}

func TestCapturedWithPromotion(t *testing.T) {
	// White has promoted a pawn to a second queen and Black has lost no piece.
	game := mustGame(t, "rnbqkbnr/pppppppp/8/8/8/8/1PPPPPPP/RNBQKBNR w KQkq - 0 1")
	byWhite, byBlack := Captured(game.GetBoard())
	if len(byWhite) != 0 || len(byBlack) != 1 || byBlack[0] != common.Pawn {
		t.Errorf("one missing pawn: %v %v", byWhite, byBlack)
	}
	// A second queen and seven pawns: the pawn promoted, nobody lost a piece.
	promoted := mustGame(t, "rnbqkbnr/pppppppp/8/8/8/Q7/1PPPPPPP/RNBQKBNR w KQkq - 0 1")
	_, byBlack = Captured(promoted.GetBoard())
	if len(byBlack) != 0 {
		t.Errorf("a promoted piece must not count as captured pawn twice: %v", byBlack)
	}
}

func TestMaterialSnapshots(t *testing.T) {
	game := chess.NewGame()
	play(t, game, "e4", "d5", "exd5", "Qxd5", "Nc3", "Qxg2", "Nf3", "Qxh1")
	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			golden(t, "material_"+name, Board(game.GetBoard(), Options{Style: style, Material: true, Captured: true}))
		})
	}
	golden(t, "material_flipped", Board(game.GetBoard(), Options{Flipped: true, Material: true, Captured: true}))
	golden(t, "material_balance_only", Board(game.GetBoard(), Options{Material: true}))
}

func TestNoMaterialLinesByDefault(t *testing.T) {
	out := Board(chess.NewGame().GetBoard(), Options{})
	if strings.Count(out, "\n") != 9 {
		t.Errorf("board has %d lines, want 9", strings.Count(out, "\n"))
	}
	out = Board(chess.NewGame().GetBoard(), Options{Material: true, Captured: true})
	if strings.Count(out, "\n") != 11 || !strings.HasPrefix(out, "\n") {
		t.Errorf("with material enabled the board keeps a line above and below:\n%q", out)
	}
}

func TestCursorBeatsOtherMarksButKeepsTheDot(t *testing.T) {
	game := chess.NewGame()
	e2, _ := ParseSquare("e2")
	e3, _ := ParseSquare("e3")
	opts := Options{Selected: &e2, Targets: Targets(game, e2), Cursor: &e3}
	plain := Board(game.GetBoard(), opts)
	if !strings.Contains(plain, "|*|") || !strings.Contains(plain, "{P}") {
		t.Errorf("cursor on an empty target should show |*|, with the selection still marked:\n%s", plain)
	}
	opts.Cursor = &e2
	if out := Board(game.GetBoard(), opts); !strings.Contains(out, "|P|") || strings.Contains(out, "{P}") {
		t.Errorf("cursor on the selected piece wins:\n%s", out)
	}
	opts.Style = Style{Color: true}
	if out := Board(game.GetBoard(), opts); !strings.Contains(out, "48;5;215") {
		t.Errorf("cursor color missing:\n%q", out)
	}
}
