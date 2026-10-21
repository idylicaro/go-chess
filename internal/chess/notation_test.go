package chess

import (
	common "idylicaro/go-chess/internal/common"
	"strings"
	"testing"
)

// sanCase is a position, a move in UCI and its expected SAN.
type sanCase struct {
	name, fen, uci, san string
}

var sanCases = []sanCase{
	{"pawn push", StartingFEN, "e2e4", "e4"},
	{"knight move", StartingFEN, "g1f3", "Nf3"},
	{"pawn capture", "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2", "e4d5", "exd5"},
	{"en passant", "rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPP1PPP/RNBQKBNR w KQkq f6 0 3", "e5f6", "exf6"},
	{"short castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1g1", "O-O"},
	{"long castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1c1", "O-O-O"},
	{"promotion with check", "8/P7/8/8/8/8/8/k6K w - - 0 1", "a7a8q", "a8=Q+"},
	{"underpromotion", "8/P7/8/8/8/8/8/k6K w - - 0 1", "a7a8n", "a8=N"},
	{"promotion capture", "1r6/P7/8/8/8/8/8/k6K w - - 0 1", "a7b8r", "axb8=R"},
	{"check", "4k3/8/8/8/8/8/8/R3K3 w Q - 0 1", "a1a8", "Ra8+"},
	{"checkmate", "rnbqkbnr/pppp1ppp/8/4p3/6P1/5P2/PPPPP2P/RNBQKBNR b KQkq g3 0 2", "d8h4", "Qh4#"},
	{"piece capture", "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3", "f3e5", "Nxe5"},
	{"file disambiguation", "4k3/8/8/8/8/8/8/1N2KN2 w - - 0 1", "b1d2", "Nbd2"},
	{"file disambiguation, other knight", "4k3/8/8/8/8/8/8/1N2KN2 w - - 0 1", "f1d2", "Nfd2"},
	{"rank disambiguation", "4k3/8/8/R7/8/8/8/R3K3 w - - 0 1", "a1a3", "R1a3"},
	{"full-square disambiguation", "8/2k5/8/8/4Q2Q/8/8/K6Q w - - 0 1", "h4e1", "Qh4e1"},
	{"pinned twin needs no disambiguation", "4k3/4r3/8/8/8/8/4N3/1N2K3 w - - 0 1", "b1c3", "Nc3"},
}

func TestSAN_FormatsMoves(t *testing.T) {
	for _, tc := range sanCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustParseFEN(t, tc.fen).SAN(mustParseUCI(t, tc.uci))
			if err != nil {
				t.Fatalf("SAN(%s): %v", tc.uci, err)
			}
			if got != tc.san {
				t.Errorf("SAN(%s) = %q, want %q", tc.uci, got, tc.san)
			}
		})
	}
}

func TestSAN_PromotionWithoutPieceIsQueen(t *testing.T) {
	game := mustParseFEN(t, "8/P7/8/8/8/8/8/k6K w - - 0 1")
	move := mustParseUCI(t, "a7a8") // Promotion is the zero value
	if got, err := game.SAN(move); err != nil || got != "a8=Q+" {
		t.Errorf("SAN(a7a8) = %q, %v; want \"a8=Q+\"", got, err)
	}
}

func TestSAN_RejectsIllegalMove(t *testing.T) {
	_, err := NewGame().SAN(mustParseUCI(t, "e2e5"))
	if err == nil || !strings.Contains(err.Error(), "e2e5") {
		t.Errorf("SAN(e2e5) error = %v, want it to name the move", err)
	}
}

func TestParseSAN_ResolvesMoves(t *testing.T) {
	for _, tc := range sanCases {
		t.Run(tc.name, func(t *testing.T) {
			assertParsesTo(t, tc.fen, tc.san, tc.uci)
		})
	}
}

func TestParseSAN_IsLenient(t *testing.T) {
	tests := []struct{ name, fen, san, uci string }{
		{"annotations", StartingFEN, "e4!?", "e2e4"},
		{"redundant disambiguation", StartingFEN, "Ngf3", "g1f3"},
		{"capture marker on a quiet move", StartingFEN, "Nxf3", "g1f3"},
		{"missing capture marker", "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2", "ed5", "e4d5"},
		{"promotion without '='", "8/P7/8/8/8/8/8/k6K w - - 0 1", "a8N", "a7a8n"},
		{"zeros for castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "0-0-0", "e1c1"},
		{"black to move", "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", "Nc6", "b8c6"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertParsesTo(t, tc.fen, tc.san, tc.uci)
		})
	}
}

func assertParsesTo(t *testing.T, fen, san, wantUCI string) {
	t.Helper()
	move, err := mustParseFEN(t, fen).ParseSAN(san)
	if err != nil {
		t.Fatalf("ParseSAN(%q): %v", san, err)
	}
	if move.String() != wantUCI {
		t.Errorf("ParseSAN(%q) = %s, want %s", san, move, wantUCI)
	}
}

func TestParseSAN_RejectsInvalidInput(t *testing.T) {
	tests := []struct{ name, fen, san, wantFragment string }{
		{"ambiguous knights", "4k3/8/8/8/8/8/8/1N2KN2 w - - 0 1", "Nd2", "ambiguous"},
		{"promotion needs a piece", "8/P7/8/8/8/8/8/k6K w - - 0 1", "a8", "matches no legal move"},
		{"illegal king move", StartingFEN, "Ke2", "matches no legal move"},
		{"not SAN at all", StartingFEN, "Zz9", "invalid SAN"},
		{"king letter does not mean castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "Kg1", "matches no legal move"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mustParseFEN(t, tc.fen).ParseSAN(tc.san)
			if err == nil || !strings.Contains(err.Error(), tc.wantFragment) || !strings.Contains(err.Error(), tc.san) {
				t.Errorf("ParseSAN(%q) error = %v, want it to mention %q and the input", tc.san, err, tc.wantFragment)
			}
		})
	}
}

// Every legal move in positions rich in castling, en passant, promotions and
// pins must survive SAN formatting and parsing unchanged.
func TestSAN_RoundTripsEveryLegalMove(t *testing.T) {
	for _, position := range []string{"kiwipete", "position4", "position5"} {
		game := mustParseFEN(t, perftReferenceFENs[position])
		for _, move := range game.legalUCIMoves() {
			san, err := game.SAN(move)
			if err != nil {
				t.Fatalf("%s: SAN(%s): %v", position, move, err)
			}
			parsed, err := game.ParseSAN(san)
			if err != nil || parsed != move {
				t.Errorf("%s: ParseSAN(SAN(%s) = %q) = %s, %v", position, move, san, parsed, err)
			}
		}
	}
}

func TestIsSideToMoveInCheck(t *testing.T) {
	if NewGame().isSideToMoveInCheck() {
		t.Error("initial position reported as check")
	}
	game := mustParseFEN(t, "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1")
	if !game.isSideToMoveInCheck() || game.currentTurn != common.White {
		t.Error("white king attacked by rook on e2 not reported as check")
	}
}
