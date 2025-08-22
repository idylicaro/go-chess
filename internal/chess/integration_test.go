package chess

import (
	common "idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/uci"
	"testing"
	"time"
)

// TestGameIntegration tests a complete game scenario
func TestGameIntegration(t *testing.T) {
	tests := []struct {
		name     string
		moves    []string
		expected GameState
	}{
		{
			name:     "Scholar's Mate",
			moves:    []string{"e2e4", "e7e5", "d1h5", "b8c6", "f1c4", "g8f6", "h5f7"},
			expected: Checkmate,
		},
		{
			name:     "Basic Opening",
			moves:    []string{"e2e4", "e7e5", "g1f3", "b8c6"},
			expected: InProgress,
		},
		{
			name:     "Simple Promotion Setup",
			moves:    []string{"d2d4", "b7b5", "d4d5", "b5b4", "d5d6", "b4b3", "d6c7", "b3c2", "c7b8q"},
			expected: InProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()

			for i, moveStr := range tt.moves {
				move, err := uci.ParseMove(moveStr)
				if err != nil {
					t.Fatalf("Failed to parse move %d (%s): %v", i+1, moveStr, err)
				}

				err = game.Move(move)
				if err != nil {
					t.Fatalf("Failed to make move %d (%s): %v", i+1, moveStr, err)
				}
			}

			if game.GetState() != tt.expected {
				t.Errorf("Expected game state %v, got %v", tt.expected, game.GetState())
			}
		})
	}
}

// TestGameWithTimerIntegration tests game with timer functionality
func TestGameWithTimerIntegration(t *testing.T) {
	t.Run("Game with timer - normal play", func(t *testing.T) {
		timeConfig := timer.TimeConfig{
			InitialTime: 30 * time.Second,
			Increment:   1 * time.Second,
		}

		game := NewGameWithTimer(timeConfig)

		// Start the timer
		err := game.StartTimer()
		if err != nil {
			t.Fatalf("Failed to start timer: %v", err)
		}

		// Make some moves
		moves := []string{"e2e4", "e7e5", "g1f3", "b8c6"}
		for _, moveStr := range moves {
			move, err := uci.ParseMove(moveStr)
			if err != nil {
				t.Fatalf("Failed to parse move %s: %v", moveStr, err)
			}

			err = game.Move(move)
			if err != nil {
				t.Fatalf("Failed to make move %s: %v", moveStr, err)
			}

			// Small delay to simulate thinking time
			time.Sleep(100 * time.Millisecond)
		}

		// Check that game is still in progress
		if game.GetState() != InProgress {
			t.Errorf("Expected game to be in progress, got %v", game.GetState())
		}

		// Check remaining times
		whiteTime, err := game.GetRemainingTime(common.White)
		if err != nil {
			t.Fatalf("Failed to get white time: %v", err)
		}

		blackTime, err := game.GetRemainingTime(common.Black)
		if err != nil {
			t.Fatalf("Failed to get black time: %v", err)
		}

		// Both players should have close to initial time + increments
		// Each player made 2 moves, so should have ~30s + 2s - small time consumed
		// Allow range of 29s to 33s to account for increment and small time consumption
		if whiteTime < 29*time.Second || whiteTime > 33*time.Second {
			t.Errorf("White time should be between 29-33s (with increment), got %v", whiteTime)
		}

		if blackTime < 29*time.Second || blackTime > 33*time.Second {
			t.Errorf("Black time should be between 29-33s (with increment), got %v", blackTime)
		}
	})

	t.Run("Game with timer - no increment", func(t *testing.T) {
		timeConfig := timer.TimeConfig{
			InitialTime: 30 * time.Second,
			Increment:   0, // No increment
		}

		game := NewGameWithTimer(timeConfig)

		// Start the timer
		err := game.StartTimer()
		if err != nil {
			t.Fatalf("Failed to start timer: %v", err)
		}

		// Make some moves
		moves := []string{"e2e4", "e7e5", "g1f3", "b8c6"}
		for _, moveStr := range moves {
			move, err := uci.ParseMove(moveStr)
			if err != nil {
				t.Fatalf("Failed to parse move %s: %v", moveStr, err)
			}

			err = game.Move(move)
			if err != nil {
				t.Fatalf("Failed to make move %s: %v", moveStr, err)
			}

			// Small delay to simulate thinking time
			time.Sleep(100 * time.Millisecond)
		}

		// Check remaining times - should be less than initial (no increment)
		whiteTime, err := game.GetRemainingTime(common.White)
		if err != nil {
			t.Fatalf("Failed to get white time: %v", err)
		}

		blackTime, err := game.GetRemainingTime(common.Black)
		if err != nil {
			t.Fatalf("Failed to get black time: %v", err)
		}

		// Without increment, time should decrease
		if whiteTime >= 30*time.Second {
			t.Errorf("White time should be less than 30s (no increment), got %v", whiteTime)
		}

		if blackTime >= 30*time.Second {
			t.Errorf("Black time should be less than 30s (no increment), got %v", blackTime)
		}
	})

	t.Run("Game timeout scenario", func(t *testing.T) {
		timeConfig := timer.TimeConfig{
			InitialTime: 100 * time.Millisecond, // Very short time
			Increment:   0,
		}

		game := NewGameWithTimer(timeConfig)
		err := game.StartTimer()
		if err != nil {
			t.Fatalf("Failed to start timer: %v", err)
		}

		// Wait for timeout
		time.Sleep(200 * time.Millisecond)

		// Try to make a move - should detect timeout
		move, _ := uci.ParseMove("e2e4")
		err = game.Move(move)

		// Should either get timeout error or game state should show timeout
		if err == nil {
			// Check if game state reflects timeout
			state := game.GetState()
			if state != TimeoutWhite && state != TimeoutBlack {
				t.Errorf("Expected timeout state, got %v", state)
			}
		}
	})
}

// TestCastlingIntegration tests castling scenarios
func TestCastlingIntegration(t *testing.T) {
	tests := []struct {
		name          string
		setupMoves    []string
		castlingMove  string
		shouldSucceed bool
	}{
		{
			name:          "White kingside castling",
			setupMoves:    []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5"},
			castlingMove:  "e1g1",
			shouldSucceed: true,
		},
		{
			name:          "White queenside castling",
			setupMoves:    []string{"d2d4", "d7d5", "c1f4", "c8f5", "d1d3", "d8d6", "b1c3", "b8c6"},
			castlingMove:  "e1c1",
			shouldSucceed: true,
		},
		{
			name:          "Castling through check (should fail)",
			setupMoves:    []string{"e2e4", "e7e5", "g1f3", "d8h4", "f1c4"},
			castlingMove:  "e1g1",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()

			// Setup position
			for _, moveStr := range tt.setupMoves {
				move, err := uci.ParseMove(moveStr)
				if err != nil {
					t.Fatalf("Failed to parse setup move %s: %v", moveStr, err)
				}

				err = game.Move(move)
				if err != nil {
					t.Fatalf("Failed to make setup move %s: %v", moveStr, err)
				}
			}

			// Try castling
			castlingMove, err := uci.ParseMove(tt.castlingMove)
			if err != nil {
				t.Fatalf("Failed to parse castling move: %v", err)
			}

			err = game.Move(castlingMove)

			if tt.shouldSucceed && err != nil {
				t.Errorf("Castling should have succeeded but failed: %v", err)
			} else if !tt.shouldSucceed && err == nil {
				t.Errorf("Castling should have failed but succeeded")
			}
		})
	}
}

// TestEnPassantIntegration tests en passant scenarios
func TestEnPassantIntegration(t *testing.T) {
	game := NewGame()

	// Setup for en passant
	moves := []string{
		"e2e4", "d7d6", // Open lines
		"e4e5", "f7f5", // Black pawn moves 2 squares
	}

	for _, moveStr := range moves {
		move, err := uci.ParseMove(moveStr)
		if err != nil {
			t.Fatalf("Failed to parse move %s: %v", moveStr, err)
		}

		err = game.Move(move)
		if err != nil {
			t.Fatalf("Failed to make move %s: %v", moveStr, err)
		}
	}

	// Now white should be able to capture en passant
	enPassantMove, err := uci.ParseMove("e5f6")
	if err != nil {
		t.Fatalf("Failed to parse en passant move: %v", err)
	}

	err = game.Move(enPassantMove)
	if err != nil {
		t.Fatalf("En passant should have succeeded: %v", err)
	}

	// Check that the pawn on f5 was captured
	capturedPawn, _ := game.GetBoard().GetPiece(5, 4) // f5
	if capturedPawn != nil {
		t.Errorf("En passant should have captured the pawn on f5")
	}
}
