package uci

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
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

// A public function to convert a UCI string into a Move struct.
func ParseMove(uciMove string) (Move, error) {
	if len(uciMove) < 4 || len(uciMove) > 5 {
		return Move{}, fmt.Errorf("invalid move: %s", uciMove)
	}

	fromX, fromY, err := parseCoordinate(uciMove[0:2])
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
		promoChar := uciMove[4]
		switch promoChar {
		case 'q':
			move.Promotion = common.Queen
		case 'r':
			move.Promotion = common.Rook
		case 'b':
			move.Promotion = common.Bishop
		case 'n':
			move.Promotion = common.Knight
		default:
			return Move{}, fmt.Errorf("promotion char invalid: %c", promoChar)
		}
	}

	return move, nil

}

// A private function to convert a chess coordinate (e.g., "e2")
func parseCoordinate(uciCoord string) (x, y int, err error) {
	// We access a byte from the string, which is the ASCII representation of the character.
	// 'e' is 101, '2' is 50. We use `rune()` for conversion and
	// subtract '1' from the character value to get the matrix index.
	if len(uciCoord) != 2 {
		return 0, 0, fmt.Errorf("invalid coordinate: %s", uciCoord)
	}

	xChar, yChar := uciCoord[0], uciCoord[1]
	x, ok := col[rune(xChar)]
	if !ok {
		return 0, 0, fmt.Errorf("invalid column: %c", xChar)
	}
	y = int(yChar - '1')
	if y < 0 || y > 7 {
		return 0, 0, fmt.Errorf("invalid row: %c", yChar)
	}
	y = 7 - y
	return x, y, nil
}
