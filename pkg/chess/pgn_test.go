package chess

import (
	"strings"
	"testing"
)

func mustPGN(t *testing.T, game *Game, headers PGNHeaders) string {
	t.Helper()
	pgn, err := game.PGN(headers)
	if err != nil {
		t.Fatalf("PGN: %v", err)
	}
	return pgn
}

func TestPGN_ScholarsMate(t *testing.T) {
	game := NewGame()
	playUCIMoves(t, game, "e2e4", "e7e5", "d1h5", "b8c6", "f1c4", "g8f6", "h5f7")
	want := `[Event "Test"]
[Site "?"]
[Date "????.??.??"]
[Round "?"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0
`
	if got := mustPGN(t, game, PGNHeaders{Event: "Test", White: "Alice", Black: "Bob"}); got != want {
		t.Errorf("PGN() =\n%s\nwant\n%s", got, want)
	}
}

func TestPGN_FromCustomPositionWithBlackToMove(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	game := mustParseFEN(t, fen)
	playUCIMoves(t, game, "e7e5", "g1f3")
	pgn := mustPGN(t, game, PGNHeaders{})

	for _, fragment := range []string{`[SetUp "1"]`, `[FEN "` + fen + `"]`, `[Result "*"]`, "\n1... e5 2. Nf3 *\n"} {
		if !strings.Contains(pgn, fragment) {
			t.Errorf("PGN() missing %q:\n%s", fragment, pgn)
		}
	}
}

func TestPGN_Results(t *testing.T) {
	tests := []struct {
		name, fen string
		moves     []string
		want      string
	}{
		{"black mates", StartingFEN, []string{"f2f3", "e7e5", "g2g4", "d8h4"}, "0-1"},
		{"stalemate", "7k/5Q2/8/6K1/8/8/8/8 w - - 0 1", []string{"g5g6"}, "1/2-1/2"},
		{"in progress", StartingFEN, []string{"e2e4"}, "*"},
		{"no moves yet", StartingFEN, nil, "*"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			game := mustParseFEN(t, tc.fen)
			playUCIMoves(t, game, tc.moves...)
			if pgn := mustPGN(t, game, PGNHeaders{}); !strings.Contains(pgn, `[Result "`+tc.want+`"]`) || !strings.HasSuffix(pgn, tc.want+"\n") {
				t.Errorf("PGN() result want %q:\n%s", tc.want, pgn)
			}
		})
	}
}

func TestPGN_EscapesTagValues(t *testing.T) {
	pgn := mustPGN(t, NewGame(), PGNHeaders{Event: `Club "Open" \ 2025`})
	if want := `[Event "Club \"Open\" \\ 2025"]`; !strings.Contains(pgn, want) {
		t.Errorf("PGN() missing %s:\n%s", want, pgn)
	}
}

func TestPGN_WrapsMovetextAt80Columns(t *testing.T) {
	game := NewGame()
	for i := 0; i < 10; i++ { // knights out and back: 40 plies of movetext
		playUCIMoves(t, game, "g1f3", "g8f6", "f3g1", "f6g8")
		game.positionHistory = nil // keep threefold repetition from ending the game
	}
	pgn := mustPGN(t, game, PGNHeaders{})
	movetext := pgn[strings.Index(pgn, "\n\n")+2:]
	lines := strings.Split(strings.TrimSuffix(movetext, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected the movetext to wrap, got one line: %q", movetext)
	}
	for _, line := range lines {
		if len(line) > pgnLineWidth {
			t.Errorf("line longer than %d chars (%d): %q", pgnLineWidth, len(line), line)
		}
		if strings.HasSuffix(line, ".") {
			t.Errorf("line ends with a move number cut from its move: %q", line)
		}
	}
}

func TestPGN_TimeControlTagIsOptional(t *testing.T) {
	game := NewGame()
	plain, err := game.PGN(PGNHeaders{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "TimeControl") {
		t.Errorf("tag present without a time control:\n%s", plain)
	}
	timed, err := game.PGN(PGNHeaders{TimeControl: "300+3"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(timed, "[Result \"*\"]\n[TimeControl \"300+3\"]\n") {
		t.Errorf("tag missing or misplaced:\n%s", timed)
	}
}
