package chess

import "testing"

// Regression for #5: halfMoveClock counts plies, but the draw fired at 50
// (25 moves per side) instead of 100 plies (50 moves per side).
func TestFiftyMoveRule(t *testing.T) {
	tests := []struct {
		name       string
		clockStart int
		move       string
		wantClock  int
		wantState  GameState
	}{
		{"99th quiet ply is not a draw", 98, "g1f3", 99, InProgress},
		{"100th quiet ply is a draw", 99, "g1f3", 100, Draw},
		{"pawn move on the 100th ply resets the clock", 99, "e2e4", 0, InProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()
			game.halfMoveClock = tt.clockStart
			if err := game.Move(mustParseUCI(t, tt.move)); err != nil {
				t.Fatalf("Move(%s): %v", tt.move, err)
			}
			if game.halfMoveClock != tt.wantClock {
				t.Errorf("halfMoveClock = %d, want %d", game.halfMoveClock, tt.wantClock)
			}
			if game.GetState() != tt.wantState {
				t.Errorf("state = %v, want %v", game.GetState(), tt.wantState)
			}
		})
	}
}
