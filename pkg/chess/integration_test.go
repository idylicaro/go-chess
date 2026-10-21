package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/timer"
	"github.com/idylicaro/go-chess/internal/timer/timertest"
	"github.com/idylicaro/go-chess/internal/uci"
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

// TestGameWithTimerIntegration tests game with timer functionality, driving
// time with a fake clock so every assertion is exact (#6).
func TestGameWithTimerIntegration(t *testing.T) {
	t.Run("Game with timer - increment credited per move", func(t *testing.T) {
		clock := timertest.NewFakeClock()
		game := newStartedTimedGame(t, timer.TimeConfig{InitialTime: 30 * time.Second, Increment: time.Second}, clock)

		playTimedMoves(t, game, clock, 100*time.Millisecond, "e2e4", "e7e5", "g1f3", "b8c6")

		// Each side thought 2 x 100ms and earned 2 x 1s.
		assertClock(t, game, common.White, 31*time.Second+800*time.Millisecond)
		assertClock(t, game, common.Black, 31*time.Second+800*time.Millisecond)
		if game.GetState() != InProgress {
			t.Errorf("Expected game to be in progress, got %v", game.GetState())
		}
	})

	t.Run("Game with timer - no increment", func(t *testing.T) {
		clock := timertest.NewFakeClock()
		game := newStartedTimedGame(t, timer.TimeConfig{InitialTime: 30 * time.Second}, clock)

		playTimedMoves(t, game, clock, 100*time.Millisecond, "e2e4", "e7e5", "g1f3", "b8c6")

		assertClock(t, game, common.White, 29*time.Second+800*time.Millisecond)
		assertClock(t, game, common.Black, 29*time.Second+800*time.Millisecond)
	})

	// Regression for #6: the move was applied and the turn passed even though
	// the side to move had already run out of time.
	t.Run("Game timeout scenario - move rejected", func(t *testing.T) {
		clock := timertest.NewFakeClock()
		game := newStartedTimedGame(t, timer.TimeConfig{InitialTime: 100 * time.Millisecond}, clock)
		clock.Advance(200 * time.Millisecond)

		if err := game.Move(mustParseUCI(t, "e2e4")); err == nil {
			t.Fatal("Move after running out of time succeeded, want an error")
		}
		if game.GetState() != TimeoutWhite {
			t.Errorf("Expected state %v, got %v", TimeoutWhite, game.GetState())
		}
		if game.GetCurrentTurn() != common.White {
			t.Errorf("turn passed to %v after a rejected move", game.GetCurrentTurn())
		}
		if pawn, _ := game.GetBoard().GetPiece(4, 6); pawn == nil {
			t.Error("e2 pawn moved although the move was rejected")
		}
	})

	t.Run("Game timer - pause freezes both clocks", func(t *testing.T) {
		clock := timertest.NewFakeClock()
		game := newStartedTimedGame(t, timer.TimeConfig{InitialTime: 10 * time.Second}, clock)
		clock.Advance(time.Second)

		if err := game.PauseTimer(); err != nil {
			t.Fatalf("PauseTimer: %v", err)
		}
		clock.Advance(time.Hour)
		assertClock(t, game, common.White, 9*time.Second)
		if err := game.Move(mustParseUCI(t, "e2e4")); err != nil {
			t.Fatalf("move while paused should not time out: %v", err)
		}
	})
}

func TestGameState_StringCoversTimeouts(t *testing.T) {
	for state, want := range map[GameState]string{TimeoutWhite: "TimeoutWhite", TimeoutBlack: "TimeoutBlack"} {
		if got := state.String(); got != want {
			t.Errorf("GameState(%d).String() = %q, want %q", int(state), got, want)
		}
	}
}

func newStartedTimedGame(t *testing.T, config timer.TimeConfig, clock timer.Clock) *Game {
	t.Helper()
	game := NewTimedGame(config, clock)
	if err := game.StartTimer(); err != nil {
		t.Fatalf("Failed to start timer: %v", err)
	}
	return game
}

// playTimedMoves plays each UCI move after the side to move thinks for thinkTime.
func playTimedMoves(t *testing.T, game *Game, clock *timertest.FakeClock, thinkTime time.Duration, moves ...string) {
	t.Helper()
	for _, notation := range moves {
		clock.Advance(thinkTime)
		if err := game.Move(mustParseUCI(t, notation)); err != nil {
			t.Fatalf("Failed to make move %s: %v", notation, err)
		}
	}
}

func assertClock(t *testing.T, game *Game, color common.Color, want time.Duration) {
	t.Helper()
	got, err := game.GetRemainingTime(color)
	if err != nil {
		t.Fatalf("GetRemainingTime(%v): %v", color, err)
	}
	if got != want {
		t.Errorf("%v clock = %v, want %v", color, got, want)
	}
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
