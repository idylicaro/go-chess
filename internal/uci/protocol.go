package uci

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
)

type Move struct {
	FromX, FromY int
	ToX, ToY     int
	Promotion    common.PieceType
}

var col = map[rune]int{
	'a': 0,
	'b': 1,
	'c': 2,
	'd': 3,
	'e': 4,
	'f': 5,
	'g': 6,
	'h': 7,
}

var promotionPieceMap = map[rune]common.PieceType{
	'q': common.Queen,
	'r': common.Rook,
	'b': common.Bishop,
	'n': common.Knight,
}

// A public function to convert a UCI string into a Move struct.
func ParseMove(uciMove string) (Move, error) {
	if len(uciMove) < 4 || len(uciMove) > 5 {
		return Move{}, fmt.Errorf("invalid UCI move %q: expected 4 characters like \"e2e4\", or 5 with a promotion like \"e7e8q\"", uciMove)
	}

	fromX, fromY, err := parseCoordinate(uciMove[:2])
	if err != nil {
		return Move{}, err
	}
	toX, toY, err := parseCoordinate(uciMove[2:4])
	if err != nil {
		return Move{}, err
	}

	move := Move{
		FromX: fromX,
		FromY: fromY,
		ToX:   toX,
		ToY:   toY,
	}

	if len(uciMove) == 5 {
		piece, ok := promotionPieceMap[rune(uciMove[4])]
		if !ok {
			return Move{}, fmt.Errorf("invalid promotion piece %q in %q: expected one of q, r, b, n", uciMove[4], uciMove)
		}

		// Only allow promotion for pawn moves to last rank
		// White pawn: fromY == 1, toY == 0
		// Black pawn: fromY == 6, toY == 7
		isWhitePromotion := fromY == 1 && toY == 0
		isBlackPromotion := fromY == 6 && toY == 7

		if !(isWhitePromotion || isBlackPromotion) {
			return Move{}, fmt.Errorf("invalid move %q: a promotion piece is only allowed on a move to the last rank (e.g. \"e7e8q\")", uciMove)
		}

		move.Promotion = piece
	}

	return move, nil

}

// A private function to convert a chess coordinate (e.g., "e2")
func parseCoordinate(uciCoord string) (x, y int, err error) {
	// We access a byte from the string, which is the ASCII representation of the character.
	// 'e' is 101, '2' is 50. We use `rune()` for conversion and
	// subtract '1' from the character value to get the matrix index.
	if len(uciCoord) != 2 {
		return 0, 0, fmt.Errorf("invalid square %q: expected a file a-h and a rank 1-8, like \"e2\"", uciCoord)
	}

	xChar, yChar := uciCoord[0], uciCoord[1]
	x, ok := col[rune(xChar)]
	if !ok {
		return 0, 0, fmt.Errorf("invalid file %q in square %q: expected a letter from a to h", xChar, uciCoord)
	}
	y = int(yChar - '1')
	if y < 0 || y > 7 {
		return 0, 0, fmt.Errorf("invalid rank %q in square %q: expected a digit from 1 to 8", yChar, uciCoord)
	}
	y = 7 - y
	return x, y, nil
}

// promotionLetterByPiece is the inverse of promotionPieceMap.
var promotionLetterByPiece = map[common.PieceType]byte{
	common.Queen: 'q', common.Rook: 'r', common.Bishop: 'b', common.Knight: 'n',
}

// String formats the move in UCI long algebraic notation, the inverse of
// ParseMove. The zero-value Promotion (Pawn) means "no promotion".
//
//	Move{FromX: 4, FromY: 6, ToX: 4, ToY: 4}.String() // "e2e4"
func (m Move) String() string {
	notation := formatCoordinate(m.FromX, m.FromY) + formatCoordinate(m.ToX, m.ToY)
	if letter, ok := promotionLetterByPiece[m.Promotion]; ok {
		notation += string(letter)
	}
	return notation
}

// formatCoordinate is the inverse of parseCoordinate: internal row 0 is rank 8.
func formatCoordinate(x, y int) string {
	return string([]byte{byte('a' + x), byte('8' - y)})
}
