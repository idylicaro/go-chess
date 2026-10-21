package moveinput

import (
	"errors"
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func game(t *testing.T, fen string) *chess.Game {
	t.Helper()
	g, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestParseAcceptsSANAndUCI(t *testing.T) {
	g := chess.NewGame()
	for _, input := range []string{"e4", "e2e4", "  e2e4 ", "E2E4", "Nf3", "g1f3", "Nf3+"} {
		if _, err := Parse(g, input); err != nil {
			t.Errorf("Parse(%q): %v", input, err)
		}
	}
	a, _ := Parse(g, "Nf3")
	b, _ := Parse(g, "g1f3")
	if a != b {
		t.Errorf("SAN and UCI differ: %v vs %v", a, b)
	}
}

func TestParseUCIPromotion(t *testing.T) {
	g := game(t, "8/4P2k/8/8/8/8/8/4K3 w - - 0 1")
	move, err := Parse(g, "e7e8n")
	if err != nil || move.Promotion != chess.PromoteToKnight {
		t.Errorf("e7e8n: %+v, %v", move, err)
	}
	if _, err := Parse(g, "e7e8"); err != nil { // defaults to a queen
		t.Errorf("e7e8: %v", err)
	}
	if move, err := Parse(g, "e8=N"); err != nil || move.Promotion != chess.PromoteToKnight {
		t.Errorf("e8=N: %+v, %v", move, err)
	}
}

func TestParseClassifiesErrors(t *testing.T) {
	check := game(t, "4k3/8/8/8/8/8/4r3/R3K2R w - - 0 1")
	ambiguous := game(t, "4k3/8/8/8/8/8/4K3/R6R w - - 0 1")
	promo := game(t, "7k/4P3/8/8/8/8/8/4K3 w - - 0 1")
	tests := []struct {
		name  string
		game  *chess.Game
		input string
		kind  error
		text  string
	}{
		{"empty", chess.NewGame(), "  ", chess.ErrInvalidNotation, "type a move"},
		{"gibberish", chess.NewGame(), "hello", chess.ErrInvalidNotation, "not a move"},
		{"bad square", chess.NewGame(), "e2e9", chess.ErrInvalidNotation, "invalid rank"},
		{"bad promotion letter", chess.NewGame(), "e7e8x", chess.ErrInvalidNotation, "promotion"},
		{"no piece", chess.NewGame(), "e4e5", chess.ErrIllegalMove, "no piece on e4"},
		{"opponent piece", chess.NewGame(), "e7e5", chess.ErrIllegalMove, "Black's, but it is White's turn"},
		{"cannot reach", chess.NewGame(), "g1g4", chess.ErrIllegalMove, "knight on g1 cannot move to g4"},
		{"SAN illegal", chess.NewGame(), "Nf6", chess.ErrIllegalMove, "no piece can make that move"},
		{"in check", check, "Ra2", chess.ErrIllegalMove, "in check"},
		{"missing promotion", promo, "e8", chess.ErrIllegalMove, "must name its piece"},
		{"ambiguous", ambiguous, "Rd1", chess.ErrAmbiguousMove, "Rad1 or Rhd1"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.game, tt.input)
		if err == nil {
			t.Errorf("%s: Parse(%q) succeeded", tt.name, tt.input)
			continue
		}
		var moveErr *Error
		if !errors.As(err, &moveErr) || !errors.Is(err, tt.kind) {
			t.Errorf("%s: err = %v, want kind %v", tt.name, err, tt.kind)
		}
		if !strings.Contains(err.Error(), tt.text) {
			t.Errorf("%s: message %q lacks %q", tt.name, err.Error(), tt.text)
		}
		for _, other := range []error{chess.ErrInvalidNotation, chess.ErrIllegalMove, chess.ErrAmbiguousMove} {
			if other != tt.kind && errors.Is(err, other) {
				t.Errorf("%s: err is also %v", tt.name, other)
			}
		}
	}
}

func TestParsePinnedPieceLeavesKingInCheck(t *testing.T) {
	g := game(t, "4r1k1/8/8/8/8/8/4B3/4K3 w - - 0 1") // bishop pinned on the e-file
	_, err := Parse(g, "e2d3")
	if err == nil || !strings.Contains(err.Error(), "leave your king in check") {
		t.Errorf("err = %v", err)
	}
}
