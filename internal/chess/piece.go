package chess

import common "idylicaro/go-chess/internal/common"

type IPiece interface {
	GetValidMoves(fromX, fromY int, board *Board) []Position
	GetColor() Color
	GetType() common.PieceType
}

type Piece struct {
	Type  common.PieceType
	Color Color
}

func (p *Piece) GetColor() Color {
	return p.Color
}

// GetType implementa o outro método da interface IPiece.
func (p *Piece) GetType() common.PieceType {
	return p.Type
}
