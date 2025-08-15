package chess

import (
	common "idylicaro/go-chess/internal/common"
)

type IPiece interface {
	GetColor() Color
	GetType() common.PieceType
	GetValidMoves(fromX, fromY int, board IBoard) []Position
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
	hasMoved bool
}

func NewPawn(color Color) *Pawn {
	return &Pawn{
		Piece: Piece{
			Type:  common.Pawn,
			Color: color,
		},
		hasMoved: false,
	}
}

func (p *Pawn) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	// Calls the specialized method without enPassant
	return p.GetValidMovesWithEnPassant(fromX, fromY, board, nil)
}

func (p *Pawn) GetValidMovesWithEnPassant(fromX, fromY int, board IBoard, enPassantTarget *Position) []Position {
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
		if !p.hasMoved {
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

	if enPassantTarget != nil {
		// Check if the pawn is on the correct rank (rank 4 for white, rank 5 for black)
		// And if the diagonal capture position matches the en passant target
		if (p.Color == White && fromY == 3) || (p.Color == Black && fromY == 4) {
			// Diagonal attack position to the left
			if leftAttackX == enPassantTarget.X && fromY == enPassantTarget.Y {
				validMoves = append(validMoves, *enPassantTarget)
			}
			// Diagonal attack position to the right
			if rightAttackX == enPassantTarget.X && fromY == enPassantTarget.Y {
				validMoves = append(validMoves, *enPassantTarget)
			}
		}
	}

	return validMoves

}

type Knight struct {
	Piece
}

func NewKnight(color Color) *Knight {
	return &Knight{
		Piece: Piece{
			Type:  common.Knight,
			Color: color,
		},
	}
}

func (k *Knight) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	validMoves := []Position{}
	knightMoves := [][]int{
		{-2, -1}, {-1, -2}, {1, -2}, {2, -1},
		{2, 1}, {1, 2}, {-1, 2}, {-2, 1},
	}

	for _, move := range knightMoves {
		dx, dy := move[0], move[1]
		newX, newY := fromX+dx, fromY+dy
		if err := board.IsValidCoordinates(newX, newY); err == nil {
			if piece, err := board.GetPiece(newX, newY); err == nil && (piece == nil || piece.GetColor() != k.Color) {
				validMoves = append(validMoves, Position{X: newX, Y: newY})
			}
		}

	}

	return validMoves
}

type Bishop struct {
	Piece
}

func NewBishop(color Color) *Bishop {
	return &Bishop{
		Piece: Piece{
			Type:  common.Bishop,
			Color: color,
		},
	}
}

func (bi *Bishop) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	validMoves := []Position{}
	// Bishop moves in 4 diagonal directions
	directions := [][]int{
		{1, -1},  // right up
		{-1, -1}, // left up
		{1, 1},   // right down
		{-1, 1},  // left down
	}

	for _, dir := range directions {
		dx, dy := dir[0], dir[1]
		for i := 1; i <= 7; i++ {
			newX := fromX + dx*i
			newY := fromY + dy*i

			if err := board.IsValidCoordinates(newX, newY); err != nil {
				break
			}

			piece, err := board.GetPiece(newX, newY)
			if err != nil {
				break
			}

			if piece == nil {
				validMoves = append(validMoves, Position{X: newX, Y: newY})
			} else {
				if bi.Color != piece.GetColor() {
					validMoves = append(validMoves, Position{X: newX, Y: newY})
				}
				break
			}
		}
	}

	return validMoves
}

type Rook struct {
	Piece
	hasMoved bool
}

func NewRook(color Color) *Rook {
	return &Rook{
		Piece: Piece{
			Type:  common.Rook,
			Color: color,
		},
		hasMoved: false,
	}
}

func (r *Rook) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	validMoves := []Position{}
	// Rook moves in 4 straight directions
	directions := [][]int{
		{0, -1}, // up
		{-1, 0}, // left
		{1, 0},  // right
		{0, 1},  // down
	}

	for _, dir := range directions {
		dx, dy := dir[0], dir[1]
		for i := 1; i <= 7; i++ {
			newX := fromX + dx*i
			newY := fromY + dy*i

			if err := board.IsValidCoordinates(newX, newY); err != nil {
				break
			}

			piece, err := board.GetPiece(newX, newY)
			if err != nil {
				break
			}

			if piece == nil {
				validMoves = append(validMoves, Position{X: newX, Y: newY})
			} else {
				if r.Color != piece.GetColor() {
					validMoves = append(validMoves, Position{X: newX, Y: newY})
				}
				break
			}
		}
	}
	return validMoves
}

type Queen struct {
	Piece
}

func NewQueen(color Color) *Queen {
	return &Queen{
		Piece: Piece{
			Type:  common.Queen,
			Color: color,
		},
	}
}

func (q *Queen) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	validMoves := []Position{}
	// Queen moves in 8 directions (combination of Rook and Bishop)
	directions := [][]int{
		{0, -1},  // up
		{-1, 0},  // left
		{1, 0},   // right
		{0, 1},   // down
		{1, -1},  // right up
		{-1, -1}, // left up
		{1, 1},   // right down
		{-1, 1},  // left down
	}

	for _, dir := range directions {
		dx, dy := dir[0], dir[1]
		for i := 1; i <= 7; i++ {
			newX := fromX + dx*i
			newY := fromY + dy*i

			if err := board.IsValidCoordinates(newX, newY); err != nil {
				break
			}

			piece, err := board.GetPiece(newX, newY)
			if err != nil {
				break
			}

			if piece == nil {
				validMoves = append(validMoves, Position{X: newX, Y: newY})
			} else {
				if q.Color != piece.GetColor() {
					validMoves = append(validMoves, Position{X: newX, Y: newY})
				}
				break
			}
		}
	}
	return validMoves
}

type King struct {
	Piece
	hasMoved bool
}

func NewKing(color Color) *King {
	return &King{
		Piece: Piece{
			Type:  common.King,
			Color: color,
		},
		hasMoved: false,
	}
}

func (k *King) GetValidMoves(fromX, fromY int, board IBoard) []Position {
	validMoves := []Position{}
	// King moves in 8 possible directions
	directions := [][]int{
		{0, -1},  // up
		{-1, 0},  // left
		{1, 0},   // right
		{0, 1},   // down
		{1, -1},  // right up
		{-1, -1}, // left up
		{1, 1},   // right down
		{-1, 1},  // left down
	}

	for _, dir := range directions {
		dx, dy := dir[0], dir[1]
		newX := fromX + dx
		newY := fromY + dy

		if err := board.IsValidCoordinates(newX, newY); err != nil {
			continue
		}

		piece, err := board.GetPiece(newX, newY)
		if err != nil {
			continue
		}

		if piece == nil || piece.GetColor() != k.Color {
			validMoves = append(validMoves, Position{X: newX, Y: newY})
		}
	}

	// Castling check
	if !k.hasMoved {
		if k.GetColor() == White {
			// Left
			if piece, _ := board.GetPiece(0, 7); piece != nil && piece.GetType() == common.Rook && !piece.(*Rook).hasMoved {
				p1, _ := board.GetPiece(1, 7)
				p2, _ := board.GetPiece(2, 7)
				p3, _ := board.GetPiece(3, 7)
				if p1 == nil && p2 == nil && p3 == nil {
					validMoves = append(validMoves, Position{X: 2, Y: 7})
				}
			}
			// Right
			if piece, _ := board.GetPiece(7, 7); piece != nil && piece.GetType() == common.Rook && !piece.(*Rook).hasMoved {
				p5, _ := board.GetPiece(5, 7)
				p6, _ := board.GetPiece(6, 7)
				if p5 == nil && p6 == nil {
					validMoves = append(validMoves, Position{X: 6, Y: 7})
				}
			}
		} else {
			// left
			if piece, _ := board.GetPiece(0, 0); piece != nil && piece.GetType() == common.Rook && !piece.(*Rook).hasMoved {
				p1, _ := board.GetPiece(1, 0)
				p2, _ := board.GetPiece(2, 0)
				p3, _ := board.GetPiece(3, 0)
				if p1 == nil && p2 == nil && p3 == nil {
					validMoves = append(validMoves, Position{X: 2, Y: 0})
				}
			}
			// Right
			if piece, _ := board.GetPiece(7, 0); piece != nil && piece.GetType() == common.Rook && !piece.(*Rook).hasMoved {
				p5, _ := board.GetPiece(5, 0)
				p6, _ := board.GetPiece(6, 0)
				if p5 == nil && p6 == nil {
					validMoves = append(validMoves, Position{X: 6, Y: 0})
				}
			}
		}
	}
	return validMoves
}
