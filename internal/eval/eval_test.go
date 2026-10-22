package eval

import (
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// evalTestFENs mix openings, middlegames and endgames, with castling and en
// passant fields, for the symmetry checks.
var evalTestFENs = []string{
	chess.StartingFEN,
	"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	"rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPP1PPP/RNBQKBNR w KQkq f6 0 3",
	"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	"4k3/8/8/8/8/8/4P3/4K3 b - - 0 1",
	"6k1/5ppp/8/8/8/8/1Q3PPP/6K1 w - - 0 1",
}

func TestEvaluate_StartingPositionIsZero(t *testing.T) {
	if got := Evaluate(boardFromFEN(t, chess.StartingFEN)); got != 0 {
		t.Errorf("Evaluate(start) = %d, want 0", got)
	}
}

func TestEvaluate_MirroredPositionScoresTheSame(t *testing.T) {
	for _, fen := range evalTestFENs {
		original, mirrored := Evaluate(boardFromFEN(t, fen)), Evaluate(boardFromFEN(t, mirrorFEN(fen)))
		if original != mirrored {
			t.Errorf("Evaluate(%q) = %d but its color mirror scores %d", fen, original, mirrored)
		}
	}
}

func TestEvaluate_IsFromTheSideToMovesView(t *testing.T) {
	white := Evaluate(boardFromFEN(t, "6k1/5ppp/8/8/8/8/1Q3PPP/6K1 w - - 0 1"))
	black := Evaluate(boardFromFEN(t, "6k1/5ppp/8/8/8/8/1Q3PPP/6K1 b - - 0 1"))
	if white < 800 || black != -white {
		t.Errorf("an extra queen scores %d for White to move and %d for Black, want >= 800 and its negation", white, black)
	}
}

func TestEvaluate_EndgameKingPrefersTheCenter(t *testing.T) {
	center := Evaluate(boardFromFEN(t, "7k/8/8/8/3K4/8/8/8 w - - 0 1"))
	corner := Evaluate(boardFromFEN(t, "7k/8/8/8/8/8/8/K7 w - - 0 1"))
	if center <= corner {
		t.Errorf("king on d4 scores %d, on a1 %d; want the center higher in an endgame", center, corner)
	}
}

func TestPhase_RunsFromAllPiecesToBareKings(t *testing.T) {
	if got := Phase(boardFromFEN(t, chess.StartingFEN)); got != maxPhase {
		t.Errorf("Phase(start) = %d, want %d", got, maxPhase)
	}
	if got := Phase(boardFromFEN(t, "7k/8/8/8/8/8/8/K7 w - - 0 1")); got != 0 {
		t.Errorf("Phase(bare kings) = %d, want 0", got)
	}
	if got := Phase(boardFromFEN(t, "QQQQQQQk/8/8/8/8/8/8/K7 w - - 0 1")); got != maxPhase {
		t.Errorf("Phase(seven queens) = %d, want it capped at %d", got, maxPhase)
	}
}

func TestTaper_BlendsByPhase(t *testing.T) {
	cases := []struct{ phase, want int }{{maxPhase, 100}, {0, -100}, {maxPhase / 2, 0}, {maxPhase + 8, 100}}
	for _, tc := range cases {
		if got := taper(100, -100, tc.phase); got != tc.want {
			t.Errorf("taper(100, -100, %d) = %d, want %d", tc.phase, got, tc.want)
		}
	}
}

func boardFromFEN(t *testing.T, fen string) *chess.SearchBoard {
	t.Helper()
	game, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	return chess.NewSearchBoard(game)
}

// mirrorFEN swaps the colors: ranks are flipped, piece letters and the side
// to move are swapped, and so are the castling rights and en passant rank.
func mirrorFEN(fen string) string {
	fields := strings.Fields(fen)
	ranks := strings.Split(fields[0], "/")
	for i, j := 0, len(ranks)-1; i < j; i, j = i+1, j-1 {
		ranks[i], ranks[j] = ranks[j], ranks[i]
	}
	fields[0] = swapCase(strings.Join(ranks, "/"))
	fields[1] = map[string]string{"w": "b", "b": "w"}[fields[1]]
	fields[2] = swapCase(fields[2])
	if fields[3] != "-" {
		fields[3] = fields[3][:1] + map[byte]string{'3': "6", '6': "3"}[fields[3][1]]
	}
	return strings.Join(fields, " ")
}

func swapCase(text string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		if r >= 'A' && r <= 'Z' {
			return r - 'A' + 'a'
		}
		return r
	}, text)
}
