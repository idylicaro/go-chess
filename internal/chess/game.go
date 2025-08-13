package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/uci"
)

type Game struct {
	board       IBoard
	currentTurn Color
	state       GameState
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
	pieceToMove, err := g.board.GetPiece(move.FromX, move.FromY)
	if err != nil {
		return err
	}

	if pieceToMove.Color != g.currentTurn {
		return fmt.Errorf("not %v's turn", pieceToMove.Color)
	}

	// 1. Validate the move
	// 1.1 ?Validate if check board state?
	// 2. Call the appropriate move method on the board
	// 2. Update the board state
	// 3. Switch turns
	return nil
}
