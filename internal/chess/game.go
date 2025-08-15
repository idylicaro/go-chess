package chess

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/uci"
)

type Game struct {
	board           IBoard
	currentTurn     Color
	state           GameState
	enPassantTarget *Position
}

type Color int

const (
	White Color = iota
	Black
)

type GameState int

const (
	InProgress GameState = iota
	Check
	Checkmate
	Stalemate
	Draw
)

func (g *Game) GetCurrentTurn() Color {
	return g.currentTurn
}

func (g *Game) GetState() GameState {
	return g.state
}

func (g *Game) Move(move uci.Move) error {
	pieceToMove, _ := g.board.GetPiece(move.FromX, move.FromY)

	if pieceToMove.GetColor() != g.currentTurn {
		return fmt.Errorf("not %v's turn", pieceToMove.GetColor())
	}

	// Clear en passant target of last move
	g.enPassantTarget = nil

	// en passant check
	if pieceToMove.GetType() == common.Pawn {
		deltaY := move.ToY - move.FromY
		if (pieceToMove.GetColor() == White && deltaY == -2) || (pieceToMove.GetColor() == Black && deltaY == 2) {
			// Calculate the en passant target (the square the pawn passed over)
			targetY := move.FromY + deltaY/2
			g.enPassantTarget = &Position{X: move.ToX, Y: targetY}
		}
	}

	// 1. Validate the move
	// 1.1 ?Validate if check board state?
	// 2. Call the appropriate move method on the board
	// 2. Update the board state
	// 3. Switch turns
	return nil
}

func (g *Game) GetValidMovesForPiece(fromX, fromY int) ([]Position, error) {
	piece, err := g.board.GetPiece(fromX, fromY)
	if err != nil {
		return nil, err
	}
	if pawn, ok := piece.(*Pawn); ok {
		return pawn.GetValidMovesWithEnPassant(fromX, fromY, g.board, g.enPassantTarget), nil
	}
	return piece.GetValidMoves(fromX, fromY, g.board), nil
}
