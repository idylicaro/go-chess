// Package eval scores chess positions for the engine's search: material and
// piece-square tables, blended between middlegame and endgame by how much
// material is left (#36).
package eval

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// scoreTables holds value plus square bonus for every color, piece and
// square, so Evaluate is one lookup per piece.
type scoreTables [2][6]pieceTable

var (
	middlegameScores = buildScoreTables(middlegameValues, middlegameSquares)
	endgameScores    = buildScoreTables(endgameValues, endgameSquares)
)

// buildScoreTables mirrors White's tables vertically for Black.
func buildScoreTables(values [6]int, squares [6]pieceTable) *scoreTables {
	tables := &scoreTables{}
	for piece := range squares {
		for square := range 64 {
			tables[chess.White][piece][square] = values[piece] + squares[piece][square]
			tables[chess.Black][piece][square] = values[piece] + squares[piece][mirrorSquare(square)]
		}
	}
	return tables
}

// mirrorSquare flips the rank: a8 (0) becomes a1 (56).
func mirrorSquare(square int) int {
	return square ^ 56
}

// Evaluate returns the static score of board in centipawns, from the side to
// move's point of view: positive is good for the side to move.
//
//	eval.Evaluate(chess.NewSearchBoard(chess.NewGame())) // 0
func Evaluate(board *chess.SearchBoard) int {
	var middlegame, endgame [2]int
	phase := 0
	for square := range 64 {
		pieceType, color, occupied := board.PieceOn(square)
		if !occupied {
			continue
		}
		middlegame[color] += middlegameScores[color][pieceType][square]
		endgame[color] += endgameScores[color][pieceType][square]
		phase += phaseWeights[pieceType]
	}
	us, them := board.SideToMove(), opponent(board.SideToMove())
	return taper(middlegame[us]-middlegame[them], endgame[us]-endgame[them], phase)
}

// taper blends the two scores: all middlegame at the starting material, all
// endgame with bare kings. Promotions can push phase past maxPhase.
func taper(middlegame, endgame, phase int) int {
	phase = min(phase, maxPhase)
	return (middlegame*phase + endgame*(maxPhase-phase)) / maxPhase
}

// Phase is the game phase of board, from maxPhase (all pieces on) to 0.
func Phase(board *chess.SearchBoard) int {
	phase := 0
	for square := range 64 {
		if pieceType, _, occupied := board.PieceOn(square); occupied {
			phase += phaseWeights[pieceType]
		}
	}
	return min(phase, maxPhase)
}

func opponent(color chess.Color) chess.Color {
	if color == chess.White {
		return chess.Black
	}
	return chess.White
}
