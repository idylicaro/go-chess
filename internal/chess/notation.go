package chess

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/uci"
	"strings"
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
	// remove check/checkmate symbols
	cleanMove := strings.TrimSuffix(strings.TrimSuffix(sanMove, "+"), "#")

	// Castling moves
	if cleanMove == "O-O" {
		// Kingside castling
		if an.game.currentTurn == common.White {
			return uci.Move{FromX: 4, FromY: 7, ToX: 6, ToY: 7}, nil // e1-g1
		} else {
			return uci.Move{FromX: 4, FromY: 0, ToX: 6, ToY: 0}, nil // e8-g8
		}
	}
	if cleanMove == "O-O-O" {
		// Queenside castling
		if an.game.currentTurn == common.White {
			return uci.Move{FromX: 4, FromY: 7, ToX: 2, ToY: 7}, nil // e1-c1
		} else {
			return uci.Move{FromX: 4, FromY: 0, ToX: 2, ToY: 0}, nil // e8-c8
		}
	}

	// Parse destination square
	if len(cleanMove) < 2 {
		return uci.Move{}, fmt.Errorf("invalid move format: %s", sanMove)
	}

	destSquare := cleanMove[len(cleanMove)-2:]
	converter := &CoordinateConverter{}
	destX, destY, err := converter.FromAlgebraic(destSquare)
	if err != nil {
		return uci.Move{}, fmt.Errorf("invalid destination square: %v", err)
	}

	// Determine piece type and parse disambiguation
	var pieceType common.PieceType
	var disambiguation string
	var isCapture bool

	if len(cleanMove) == 2 {
		pieceType = common.Pawn
	} else {
		remainder := cleanMove[:len(cleanMove)-2]

		firstChar := remainder[0]
		switch firstChar {
		case 'N':
			pieceType = common.Knight
		case 'B':
			pieceType = common.Bishop
		case 'R':
			pieceType = common.Rook
		case 'Q':
			pieceType = common.Queen
		case 'K':
			pieceType = common.King
		default:
			pieceType = common.Pawn
			disambiguation = string(firstChar)
		}

		// Remove piece symbol if it was a named piece
		if firstChar >= 'A' && firstChar <= 'Z' && pieceType != common.Pawn {
			remainder = remainder[1:]
		}

		if strings.Contains(remainder, "x") {
			isCapture = true
			remainder = strings.Replace(remainder, "x", "", 1)
		}

		// What's left is disambiguation
		if remainder != "" {
			disambiguation = remainder
		}
	}

	sourceX, sourceY, err := an.findSourceSquare(pieceType, destX, destY, disambiguation, isCapture)
	if err != nil {
		return uci.Move{}, fmt.Errorf("could not find source square: %v", err)
	}

	return uci.Move{
		FromX: sourceX,
		FromY: sourceY,
		ToX:   destX,
		ToY:   destY,
	}, nil
}

// GenerateMove converts a UCI move to SAN notation
// This handles ambiguity resolution automatically
func (an *AlgebraicNotation) GenerateMove(move uci.Move) (string, error) {
	piece, err := an.board.GetPiece(move.FromX, move.FromY)
	if err != nil {
		return "", fmt.Errorf("no piece at source position: %v", err)
	}
	if piece == nil {
		return "", fmt.Errorf("no piece at position (%d,%d)", move.FromX, move.FromY)
	}

	// Get piece symbol (empty for pawns, N/B/R/Q/K for others)
	converter := &CoordinateConverter{}
	symbol := converter.GetPieceSymbol(piece.GetType())

	// Get destination square in algebraic notation
	destSquare, err := converter.ToAlgebraic(move.ToX, move.ToY)
	if err != nil {
		return "", fmt.Errorf("invalid destination coordinates: %v", err)
	}

	destPiece, destErr := an.board.GetPiece(move.ToX, move.ToY)
	isCapture := destErr == nil && destPiece != nil && destPiece.GetColor() != piece.GetColor()

	if piece.GetType().String() == "Pawn" {
		if isCapture {
			// Pawn capture: show source file + x + destination (e.g., "exd5")
			sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)
			return sourceSquare[0:1] + "x" + destSquare, nil
		} else {
			// Simple pawn move: just destination (e.g., "e4")
			return destSquare, nil
		}
	}

	// Handle piece moves (Knight, Bishop, Rook, Queen, King)
	var result string = symbol

	// Check if we need disambiguation (multiple pieces can reach same destination)
	if an.isAmbiguous(move, piece) {
		disambiguation := an.getDisambiguation(move, piece)
		result += disambiguation
	}

	if isCapture {
		result += "x"
	}

	result += destSquare

	// TODO: Add check (+) and checkmate (#) symbols later

	return result, nil
}

// isAmbiguous checks if a move needs disambiguation
func (an *AlgebraicNotation) isAmbiguous(move uci.Move, piece IPiece) bool {
	// Find all pieces of same type and color that can reach the destination
	candidateCount := 0

	// Scan the entire board
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			candidate, err := an.board.GetPiece(x, y)
			if err != nil || candidate == nil {
				continue
			}

			// Skip if not same type and color
			if candidate.GetType() != piece.GetType() || candidate.GetColor() != piece.GetColor() {
				continue
			}

			// Check if this piece can reach the destination
			validMoves := candidate.GetValidMoves(x, y, an.board)
			for _, validMove := range validMoves {
				if validMove.X == move.ToX && validMove.Y == move.ToY {
					candidateCount++
					break // Found one valid move to destination, that's enough
				}
			}
		}
	}

	// If more than one piece can reach the destination, it's ambiguous
	return candidateCount > 1
}

// getDisambiguation returns the disambiguation string (file, rank, or both)
func (an *AlgebraicNotation) getDisambiguation(move uci.Move, piece IPiece) string {
	// Find all other pieces of same type that can reach the destination
	conflictingMoves := []uci.Move{}

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			// Skip the piece we're moving
			if x == move.FromX && y == move.FromY {
				continue
			}

			candidate, err := an.board.GetPiece(x, y)
			if err != nil || candidate == nil {
				continue
			}

			// Skip if not same type and color
			if candidate.GetType() != piece.GetType() || candidate.GetColor() != piece.GetColor() {
				continue
			}

			// Check if this piece can reach the destination
			validMoves := candidate.GetValidMoves(x, y, an.board)
			for _, validMove := range validMoves {
				if validMove.X == move.ToX && validMove.Y == move.ToY {
					conflictingMoves = append(conflictingMoves, uci.Move{
						FromX: x, FromY: y,
						ToX: move.ToX, ToY: move.ToY,
					})
					break
				}
			}
		}
	}

	// If no conflicts, no disambiguation needed
	if len(conflictingMoves) == 0 {
		return ""
	}

	converter := &CoordinateConverter{}
	sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)

	// Check if file disambiguation is sufficient
	// (no other conflicting piece is on the same file)
	fileConflict := false
	for _, conflict := range conflictingMoves {
		if conflict.FromX == move.FromX {
			fileConflict = true
			break
		}
	}

	if !fileConflict {
		// File disambiguation is sufficient (e.g., "Nbd2")
		return sourceSquare[0:1] // return just the file (a-h)
	}

	// Check if rank disambiguation is sufficient
	// (no other conflicting piece is on the same rank)
	rankConflict := false
	for _, conflict := range conflictingMoves {
		if conflict.FromY == move.FromY {
			rankConflict = true
			break
		}
	}

	if !rankConflict {
		// Rank disambiguation is sufficient (e.g., "R1a3")
		return sourceSquare[1:2] // return just the rank (1-8)
	}

	// Need full square disambiguation (e.g., "Qh4e1")
	return sourceSquare
}

// findSourceSquare finds the source coordinates for a move given the piece type,
// destination, and any disambiguation information
func (an *AlgebraicNotation) findSourceSquare(pieceType common.PieceType, destX, destY int, disambiguation string, isCapture bool) (int, int, error) {
	candidates := []Position{}

	// Search the board for pieces of the specified type and color
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, err := an.board.GetPiece(x, y)
			if err != nil || piece == nil {
				continue
			}

			// Check if this piece matches our criteria
			if piece.GetType() != pieceType || piece.GetColor() != an.game.currentTurn {
				continue
			}

			// Check if this piece can reach the destination
			validMoves := piece.GetValidMoves(x, y, an.board)
			canReachDest := false
			for _, move := range validMoves {
				if move.X == destX && move.Y == destY {
					canReachDest = true
					break
				}
			}

			if !canReachDest {
				continue
			}

			// If we have disambiguation, check if this piece matches
			if disambiguation != "" {
				converter := &CoordinateConverter{}
				sourceSquare, _ := converter.ToAlgebraic(x, y)

				// Check different types of disambiguation
				if len(disambiguation) == 1 {
					char := disambiguation[0]
					if char >= 'a' && char <= 'h' {
						// File disambiguation (e.g., "b" in "Nbd2")
						if sourceSquare[0] != char {
							continue
						}
					} else if char >= '1' && char <= '8' {
						// Rank disambiguation (e.g., "1" in "R1a3")
						if sourceSquare[1] != char {
							continue
						}
					}
				} else if len(disambiguation) == 2 {
					// Full square disambiguation (e.g., "h4" in "Qh4e1")
					if sourceSquare != disambiguation {
						continue
					}
				}
			}

			candidates = append(candidates, Position{X: x, Y: y})
		}
	}

	// Validate results
	if len(candidates) == 0 {
		return 0, 0, fmt.Errorf("no valid piece found for move")
	}

	if len(candidates) > 1 {
		return 0, 0, fmt.Errorf("ambiguous move - multiple pieces can make this move")
	}

	return candidates[0].X, candidates[0].Y, nil
}
