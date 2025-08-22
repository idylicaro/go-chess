package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/uci"
)

// AlgebraicNotation handles parsing and generation of Standard Algebraic Notation (SAN)
type AlgebraicNotation struct {
	board IBoard
	game  *Game
}

// NewAlgebraicNotation creates a new notation handler
func NewAlgebraicNotation(board IBoard, game *Game) *AlgebraicNotation {
	return &AlgebraicNotation{
		board: board,
		game:  game,
	}
}

// ParseMove parses a SAN string into a UCI Move
// Examples: "e4", "Nf3", "Qh5+", "O-O", "Nbd2"
func (an *AlgebraicNotation) ParseMove(sanMove string) (uci.Move, error) {
	return uci.Move{}, fmt.Errorf("not implemented yet")
}

// GenerateMove converts a UCI move to SAN notation
// This handles ambiguity resolution automatically
func (an *AlgebraicNotation) GenerateMove(move uci.Move) (string, error) {
	// TODO: Implement SAN generation
	// This is where you'll convert uci.Move{FromX: 1, FromY: 0, ToX: 5, ToY: 2} to "Nf3"
	return "", fmt.Errorf("not implemented yet")
}

// isAmbiguous checks if a move needs disambiguation
func (an *AlgebraicNotation) isAmbiguous(move uci.Move, piece IPiece) bool {
	// TODO: Check if other pieces of same type can reach the same destination
	return false
}

// getDisambiguation returns the disambiguation string (file, rank, or both)
func (an *AlgebraicNotation) getDisambiguation(move uci.Move, piece IPiece) string {
	// TODO: Return "b", "1", or "b1" for disambiguation
	return ""
}
