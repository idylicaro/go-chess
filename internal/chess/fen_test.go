package chess

import (
	"strings"
	"testing"
)

// perftReferenceFENs are the chessprogramming.org perft positions; they
// exercise castling rights, en passant and promotions.
var perftReferenceFENs = map[string]string{
	"initial":   StartingFEN,
	"kiwipete":  "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	"position3": "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	"position4": "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
	"position5": "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
	"position6": "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
}

func mustParseFEN(t *testing.T, fen string) *Game {
	t.Helper()
	game, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN: %v", err)
	}
	return game
}

func playUCIMoves(t *testing.T, game *Game, moves ...string) {
	t.Helper()
	for _, notation := range moves {
		if err := game.Move(mustParseUCI(t, notation)); err != nil {
			t.Fatalf("Move(%s): %v", notation, err)
		}
	}
}

func TestFEN_RoundTrip(t *testing.T) {
	positions := map[string]string{"black to move with en passant": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"}
	for name, fen := range perftReferenceFENs {
		positions[name] = fen
	}
	for name, fen := range positions {
		t.Run(name, func(t *testing.T) {
			if got := mustParseFEN(t, fen).FEN(); got != fen {
				t.Errorf("FEN() = %q, want %q", got, fen)
			}
		})
	}
}

func TestFEN_NewGameMatchesStartingFEN(t *testing.T) {
	if got := NewGame().FEN(); got != StartingFEN {
		t.Errorf("NewGame().FEN() = %q, want %q", got, StartingFEN)
	}
}

func TestFEN_TracksCountersAndEnPassant(t *testing.T) {
	game := NewGame()
	playUCIMoves(t, game, "e2e4")
	assertFEN(t, game, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	playUCIMoves(t, game, "g8f6", "g1f3")
	assertFEN(t, game, "rnbqkb1r/pppppppp/5n2/8/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 2 2")
}

func TestFEN_FourFieldsDefaultCounters(t *testing.T) {
	game := mustParseFEN(t, "4k3/8/8/8/8/8/8/4K3 b - -")
	assertFEN(t, game, "4k3/8/8/8/8/8/8/4K3 b - - 0 1")
}

func TestFEN_ParseDerivesGameState(t *testing.T) {
	tests := map[string]GameState{
		"rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3": Checkmate, // fool's mate
		"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1":                                Stalemate,
		"4k3/8/8/8/8/8/4r3/4K3 w - - 0 1":                               Check,
	}
	for fen, want := range tests {
		if got := mustParseFEN(t, fen).GetState(); got != want {
			t.Errorf("ParseFEN(%q) state = %v, want %v", fen, got, want)
		}
	}
}

func assertFEN(t *testing.T, game *Game, want string) {
	t.Helper()
	if got := game.FEN(); got != want {
		t.Errorf("FEN() = %q\n                  want %q", got, want)
	}
}

func TestParseFEN_RejectsMalformedInput(t *testing.T) {
	tests := map[string]string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP w KQkq - 0 1":                        "7 ranks",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNRR w KQkq - 0 1":              "describes 9 squares",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBN w KQkq - 0 1":                "describes 7 squares",
		"rnbqkbnr/ppppxppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1":               "piece letter 'x'",
		"rnbq1bnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1":               "0 Black kings",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR x KQkq - 0 1":               "side to move \"x\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkk - 0 1":               "castling rights \"KQkk\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq e3 0 1":              "en passant square \"e3\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - -1 1":              "halfmove clock \"-1\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 0":               "fullmove number \"0\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0":                 "got 5 fields",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1 extra":         "got 7 fields",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - zero 1":            "halfmove clock \"zero\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq e6x 0 1":             "en passant square \"e6x\"",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1 trailing junk": "got 8 fields",
	}
	for fen, wantFragment := range tests {
		_, err := ParseFEN(fen)
		if err == nil || !strings.Contains(err.Error(), wantFragment) {
			t.Errorf("ParseFEN(%q) error = %v, want it to mention %q", fen, err, wantFragment)
		}
	}
}
