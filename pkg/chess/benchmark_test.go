package chess

import (
	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/uci"
	"testing"
	"time"
)

// BenchmarkMoveGeneration benchmarks legal move generation
func BenchmarkMoveGeneration(b *testing.B) {
	game := NewGame()

	// Setup a complex middle game position
	moves := []string{
		"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5",
		"d2d3", "d7d6", "c1g5", "f7f6", "g5h4", "g8h6",
	}

	for _, moveStr := range moves {
		move, _ := uci.ParseMove(moveStr)
		game.Move(move)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate all legal moves for current position
		for x := 0; x < 8; x++ {
			for y := 0; y < 8; y++ {
				piece, _ := game.board.GetPiece(x, y)
				if piece != nil && piece.GetColor() == game.currentTurn {
					game.getLegalMovesForPiece(x, y)
				}
			}
		}
	}
}

// BenchmarkGameStateUpdate benchmarks game state updates
func BenchmarkGameStateUpdate(b *testing.B) {
	game := NewGame()

	// Setup a position near endgame
	moves := []string{
		"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5",
		"d2d3", "d7d6", "c1g5", "f7f6", "g5h4", "g8h6",
		"h4g3", "h6f7", "b1c3", "c6d4", "c4d5", "d4f3",
	}

	for _, moveStr := range moves {
		move, _ := uci.ParseMove(moveStr)
		game.Move(move)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.updateGameState()
	}
}

// BenchmarkFullGame benchmarks a complete game
func BenchmarkFullGame(b *testing.B) {
	moves := []string{
		"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5",
		"d2d3", "d7d6", "c1g5", "f7f6", "g5h4", "g8h6",
		"h4g3", "h6f7", "b1c3", "c6d4", "c4d5", "d4f3",
		"g2f3", "f7g5", "g3g5", "f6g5", "d1g4", "g5g4",
		"f3g4", "d8h4", "g4g5", "h4f2", "e1d1", "f2f1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewGame()
		for _, moveStr := range moves {
			move, err := uci.ParseMove(moveStr)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
			err = game.Move(move)
			if err != nil {
				break // Game ended
			}
		}
	}
}

// BenchmarkTimerOperations benchmarks timer operations
func BenchmarkTimerOperations(b *testing.B) {
	timeConfig := timer.TimeConfig{
		InitialTime: 10 * time.Minute,
		Increment:   5 * time.Second,
	}

	game := NewTimedGame(timeConfig, timer.SystemClock{})
	game.StartTimer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate timer operations that happen during a move
		game.GetRemainingTime(common.White)
		game.GetRemainingTime(common.Black)

		// Simulate move
		move, _ := uci.ParseMove("e2e4")
		game.Move(move)

		// Reset game for next iteration
		game = NewTimedGame(timeConfig, timer.SystemClock{})
		game.StartTimer()
	}
}

// BenchmarkBoardCopy benchmarks board copying (used for legal move validation)
func BenchmarkBoardCopy(b *testing.B) {
	game := NewGame()

	// Setup a complex position
	moves := []string{
		"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5",
		"d2d3", "d7d6", "c1g5", "f7f6", "g5h4", "g8h6",
	}

	for _, moveStr := range moves {
		move, _ := uci.ParseMove(moveStr)
		game.Move(move)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = game.board.Copy()
	}
}

// BenchmarkLegalMoveValidation benchmarks the legal move validation process
func BenchmarkLegalMoveValidation(b *testing.B) {
	game := NewGame()

	// Setup position where king is nearly in check
	moves := []string{
		"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5",
		"d2d3", "d7d6", "c1g5", "f7f6", "g5h4", "d8e7",
	}

	for _, moveStr := range moves {
		move, _ := uci.ParseMove(moveStr)
		game.Move(move)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test a move that requires extensive validation
		game.getLegalMovesForPiece(4, 7) // King position
	}
}

// BenchmarkEnPassantValidation benchmarks en passant move validation
func BenchmarkEnPassantValidation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewGame()

		// Setup for en passant
		moves := []string{
			"e2e4", "d7d6",
			"e4e5", "f7f5", // Creates en passant opportunity
		}

		for _, moveStr := range moves {
			move, _ := uci.ParseMove(moveStr)
			game.Move(move)
		}

		// Test en passant validation
		game.getLegalMovesForPiece(4, 3) // White pawn that can capture en passant
	}
}

// BenchmarkCastlingValidation benchmarks castling validation
func BenchmarkCastlingValidation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewGame()

		// Setup for castling
		moves := []string{
			"e2e4", "e7e5",
			"g1f3", "b8c6",
			"f1c4", "f8c5",
		}

		for _, moveStr := range moves {
			move, _ := uci.ParseMove(moveStr)
			game.Move(move)
		}

		// Test castling validation
		game.getLegalMovesForPiece(4, 7) // White king
	}
}
