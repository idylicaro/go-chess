package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"testing"
)

// attackCase asserts whether square is attacked by byColor on a board holding pieces.
type attackCase struct {
	name    string
	pieces  map[string]IPiece
	square  string
	byColor common.Color
	want    bool
}

func runAttackCases(t *testing.T, cases []attackCase) {
	t.Helper()
	converter := &CoordinateConverter{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			board := newGameFromSquares(t, tc.pieces).board
			x, y, err := converter.FromAlgebraic(tc.square)
			if err != nil {
				t.Fatalf("invalid square %q: %v", tc.square, err)
			}
			if got := board.IsSquareAttacked(x, y, tc.byColor); got != tc.want {
				t.Errorf("IsSquareAttacked(%s, %v) = %v, want %v", tc.square, tc.byColor, got, tc.want)
			}
		})
	}
}

func withKings(extra map[string]IPiece) map[string]IPiece {
	pieces := map[string]IPiece{"a1": NewKing(common.White), "h8": NewKing(common.Black)}
	for square, piece := range extra {
		pieces[square] = piece
	}
	return pieces
}

// Regression for #3: pawns attacked their forward square and ignored empty diagonals.
func TestIsSquareAttacked_Pawns(t *testing.T) {
	blackPawnE3 := withKings(map[string]IPiece{"e3": NewPawn(common.Black)})
	whitePawnE4 := withKings(map[string]IPiece{"e4": NewPawn(common.White)})
	runAttackCases(t, []attackCase{
		{"black pawn attacks empty left diagonal", blackPawnE3, "d2", common.Black, true},
		{"black pawn attacks empty right diagonal", blackPawnE3, "f2", common.Black, true},
		{"black pawn does not attack forward square", blackPawnE3, "e2", common.Black, false},
		{"white pawn attacks empty diagonals", whitePawnE4, "f5", common.White, true},
		{"white pawn does not attack forward square", whitePawnE4, "e5", common.White, false},
		{"white pawn does not attack backwards", whitePawnE4, "d3", common.White, false},
		{"edge pawn attacks only inward", withKings(map[string]IPiece{"h2": NewPawn(common.White)}), "g3", common.White, true},
	})
}

// Regression for #3: castling destinations were reported as attacked by the king.
func TestIsSquareAttacked_KingAndKnight(t *testing.T) {
	kingE1 := map[string]IPiece{"e1": NewKing(common.White), "a8": NewKing(common.Black)}
	knightB1 := withKings(map[string]IPiece{"b1": NewKnight(common.White)})
	runAttackCases(t, []attackCase{
		{"king attacks adjacent square", kingE1, "f1", common.White, true},
		{"king does not attack its castling square", kingE1, "g1", common.White, false},
		{"knight attacks L-shaped square", knightB1, "c3", common.White, true},
		{"knight attacks own piece square (defends it)", withKings(map[string]IPiece{
			"b1": NewKnight(common.White), "d2": NewPawn(common.White),
		}), "d2", common.White, true},
		{"knight does not attack straight ahead", knightB1, "b3", common.White, false},
	})
}

func TestIsSquareAttacked_Sliders(t *testing.T) {
	rookBlocked := withKings(map[string]IPiece{"d1": NewRook(common.White), "d4": NewPawn(common.Black)})
	bishopC1 := withKings(map[string]IPiece{"c1": NewBishop(common.White), "e3": NewPawn(common.White)})
	queenD8 := withKings(map[string]IPiece{"d8": NewQueen(common.Black)})
	runAttackCases(t, []attackCase{
		{"rook attacks up to blocker", rookBlocked, "d4", common.White, true},
		{"rook does not attack past blocker", rookBlocked, "d5", common.White, false},
		{"rook does not attack diagonally", rookBlocked, "e2", common.White, false},
		{"bishop attacks diagonal", bishopC1, "d2", common.White, true},
		{"bishop does not attack past own piece", bishopC1, "g5", common.White, false},
		{"queen attacks file", queenD8, "d1", common.Black, true},
		{"queen attacks diagonal", queenD8, "a5", common.Black, true},
		{"queen does not attack knight-jump square", queenD8, "e6", common.Black, false},
		{"attacks are color specific", queenD8, "d1", common.White, false},
	})
}
