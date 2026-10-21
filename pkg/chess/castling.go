package chess

import (
	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/uci"
)

// CastlingRights is the set of castles still available, as in the FEN
// castling field (#7). It is game state: a king that never moved can still
// have lost its rights, which piece flags cannot express.
type CastlingRights uint8

const (
	WhiteKingSide CastlingRights = 1 << iota
	WhiteQueenSide
	BlackKingSide
	BlackQueenSide

	NoCastlingRights  CastlingRights = 0
	AllCastlingRights                = WhiteKingSide | WhiteQueenSide | BlackKingSide | BlackQueenSide
)

// castlingSide describes one of the four castles in internal coordinates.
type castlingSide struct {
	right        CastlingRights
	fenLetter    byte
	color        common.Color
	kingFrom     Position
	kingTo       Position
	rookFrom     Position
	mustBeEmpty  []Position // between king and rook
	kingPassesBy Position   // square the king crosses; must not be attacked
}

// castlingSides is ordered as FEN writes the castling field: KQkq.
var castlingSides = []castlingSide{
	{WhiteKingSide, 'K', common.White, Position{4, 7}, Position{6, 7}, Position{7, 7}, []Position{{5, 7}, {6, 7}}, Position{5, 7}},
	{WhiteQueenSide, 'Q', common.White, Position{4, 7}, Position{2, 7}, Position{0, 7}, []Position{{1, 7}, {2, 7}, {3, 7}}, Position{3, 7}},
	{BlackKingSide, 'k', common.Black, Position{4, 0}, Position{6, 0}, Position{7, 0}, []Position{{5, 0}, {6, 0}}, Position{5, 0}},
	{BlackQueenSide, 'q', common.Black, Position{4, 0}, Position{2, 0}, Position{0, 0}, []Position{{1, 0}, {2, 0}, {3, 0}}, Position{3, 0}},
}

// Has reports whether every right in other is still available.
//
//	AllCastlingRights.Has(WhiteKingSide) // true
func (r CastlingRights) Has(other CastlingRights) bool {
	return r&other == other
}

// afterMove drops the rights a move destroys: any king move loses both of its
// sides, and any move from or to a rook's home corner loses that side (this
// covers the rook moving away and the rook being captured).
func (r CastlingRights) afterMove(piece IPiece, move uci.Move) CastlingRights {
	from := Position{X: move.FromX, Y: move.FromY}
	to := Position{X: move.ToX, Y: move.ToY}
	for _, side := range castlingSides {
		kingMoved := piece.GetType() == common.King && piece.GetColor() == side.color
		if kingMoved || from == side.rookFrom || to == side.rookFrom {
			r &^= side.right
		}
	}
	return r
}

// castlingMoves returns the king destinations of every castle currently
// legal for the side to move, except the final check on the destination
// square, which the regular legality simulation performs.
func (g *Game) castlingMoves(kingFrom Position) []Position {
	var destinations []Position
	for _, side := range castlingSides {
		if side.color == g.currentTurn && side.kingFrom == kingFrom && g.canCastle(side) {
			destinations = append(destinations, side.kingTo)
		}
	}
	return destinations
}

// canCastle checks rights, piece placement, empty squares between king and
// rook, and that the king is neither in check nor passing through an attack.
func (g *Game) canCastle(side castlingSide) bool {
	rook, _ := g.board.GetPiece(side.rookFrom.X, side.rookFrom.Y)
	hasOwnRook := rook != nil && rook.GetType() == common.Rook && rook.GetColor() == side.color
	if !g.castlingRights.Has(side.right) || !hasOwnRook {
		return false
	}
	for _, square := range side.mustBeEmpty {
		if piece, _ := g.board.GetPiece(square.X, square.Y); piece != nil {
			return false
		}
	}
	opponent := g.getOpponentColor()
	return !g.board.IsSquareAttacked(side.kingFrom.X, side.kingFrom.Y, opponent) &&
		!g.board.IsSquareAttacked(side.kingPassesBy.X, side.kingPassesBy.Y, opponent)
}
