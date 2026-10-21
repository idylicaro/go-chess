package render

import (
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// ParseSquare reads a square name such as "e4".
func ParseSquare(name string) (chess.Position, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if len(name) != 2 || name[0] < 'a' || name[0] > 'h' || name[1] < '1' || name[1] > '8' {
		return chess.Position{}, false
	}
	return chess.Position{X: int(name[0] - 'a'), Y: int('8' - name[1])}, true
}

// SquareName is the inverse of ParseSquare.
func SquareName(p chess.Position) string {
	return string([]byte{byte('a' + p.X), byte('8' - p.Y)})
}

// CheckSquare is the square of the king in check, or nil when the side to
// move is not in check. A checkmated king counts as in check.
func CheckSquare(game *chess.Game) *chess.Position {
	switch game.GetState() {
	case chess.Check, chess.Checkmate:
		x, y := game.GetBoard().GetKingPosition(game.GetCurrentTurn())
		return &chess.Position{X: x, Y: y}
	}
	return nil
}

// Targets lists the squares the piece on from can legally move to; promotions
// count once per square. It is empty when there is no piece or it cannot move.
//
//	options.Selected, options.Targets = &from, render.Targets(game, from)
func Targets(game *chess.Game, from chess.Position) []chess.Position {
	var targets []chess.Position
	seen := map[chess.Position]bool{}
	for _, move := range game.LegalMoves() {
		to := chess.Position{X: move.ToX, Y: move.ToY}
		if move.FromX == from.X && move.FromY == from.Y && !seen[to] {
			seen[to] = true
			targets = append(targets, to)
		}
	}
	return targets
}
