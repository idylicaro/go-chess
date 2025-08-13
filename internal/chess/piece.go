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

func (p *Piece) GetType() common.PieceType {
	return p.Type
}

type Pawn struct {
	Piece
}

func NewPawn(color Color) *Pawn {
	return &Pawn{
		Piece: Piece{
			Type:  common.Pawn,
			Color: color,
		},
	}
}

func (p *Pawn) GetValidMoves(fromX, fromY int, board *Board) []Position {
	// 3 - has avaliable enemy piece to attack ?
	validMoves := []Position{}
	var direction int

	if p.Color == White {
		direction = -1
	} else {
		direction = 1
	}

	oneStepY := fromY + direction

	if piece, err := board.GetPiece(fromX, oneStepY); err == nil && piece == nil {
		validMoves = append(validMoves, Position{X: fromX, Y: oneStepY})

		// intial step verification
		if (p.Color == White && fromY == 6) || (p.Color == Black && fromY == 1) {
			twoStepsY := fromY + 2*direction
			if piece, err := board.GetPiece(fromX, twoStepsY); err == nil && piece == nil {
				validMoves = append(validMoves, Position{X: fromX, Y: twoStepsY})
			}
		}
	}

	leftAttackX := fromX - 1
	rightAttackX := fromX + 1
	attackY := fromY + direction

	if leftAttackX >= 0 {
		if piece, err := board.GetPiece(leftAttackX, attackY); err == nil && piece != nil && p.Color != piece.GetColor() {
			validMoves = append(validMoves, Position{X: leftAttackX, Y: attackY})
		}
	}

	if rightAttackX <= 7 {
		if piece, err := board.GetPiece(rightAttackX, attackY); err == nil && piece != nil && p.Color != piece.GetColor() {
			validMoves = append(validMoves, Position{X: rightAttackX, Y: attackY})
		}
	}

	return validMoves

}
