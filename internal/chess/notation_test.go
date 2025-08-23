package chess

import (
	"idylicaro/go-chess/internal/uci"
	"testing"
)

func TestAlgebraicNotation_ParseMove(t *testing.T) {
	// Setup a game in initial position
	game := NewGame()
	notation := NewAlgebraicNotation(game.board, game)

	tests := []struct {
		name        string
		sanMove     string
		expectedUCI uci.Move
		wantErr     bool
	}{
		{
			name:    "Simple pawn move",
			sanMove: "e4",
			expectedUCI: uci.Move{
				FromX: 4, FromY: 6, // e2 in internal coordinates
				ToX: 4, ToY: 4, // e4 in internal coordinates
			},
			wantErr: false,
		},
		{
			name:    "Knight move",
			sanMove: "Nf3",
			expectedUCI: uci.Move{
				FromX: 6, FromY: 7, // g1 in internal coordinates
				ToX: 5, ToY: 5, // f3 in internal coordinates
			},
			wantErr: false,
		},
		// - Captures: "exd5", "Nxf7"
		// - Check: "Qh5+"
		// - Checkmate: "Qf7#"
		// - Castling: "O-O", "O-O-O"
		// - Ambiguous moves: "Nbd2", "R1a3", "Qh4e1"
		// - Promotion: "e8=Q"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			move, err := notation.ParseMove(tt.sanMove)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && move != tt.expectedUCI {
				t.Errorf("ParseMove() = %v, want %v", move, tt.expectedUCI)
			}
		})
	}
}

func TestAlgebraicNotation_GenerateMove(t *testing.T) {
	// Setup a game in initial position
	game := NewGame()
	notation := NewAlgebraicNotation(game.board, game)

	tests := []struct {
		name        string
		move        uci.Move
		expectedSAN string
		wantErr     bool
	}{
		{
			name: "Simple pawn move",
			move: uci.Move{
				FromX: 4, FromY: 6, // e2
				ToX: 4, ToY: 4, // e4
			},
			expectedSAN: "e4",
			wantErr:     false,
		},
		{
			name: "Knight move",
			move: uci.Move{
				FromX: 6, FromY: 7, // g1
				ToX: 5, ToY: 5, // f3
			},
			expectedSAN: "Nf3",
			wantErr:     false,
		},
		// TODO: Add more test cases for:
		// - Ambiguous moves that need disambiguation
		// - Captures
		// - Check and checkmate
		// - Castling
		// - En passant
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			san, err := notation.GenerateMove(tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && san != tt.expectedSAN {
				t.Errorf("GenerateMove() = %v, want %v", san, tt.expectedSAN)
			}
		})
	}
}

func TestAlgebraicNotation_Disambiguation(t *testing.T) {
	// TODO: Test cases for ambiguity resolution
	// Examples to test:
	// - Two knights can reach same square: "Nbd2" vs "Nfd2"
	// - Two rooks on same file: "R1a3" vs "R8a3"
	// - Two pieces need full disambiguation: "Qh4e1"
}
