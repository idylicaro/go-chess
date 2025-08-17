// internal/chess/board_test.go
package chess

import (
	"errors"
	"testing"
	// Verifique o caminho real do seu pacote common
)

func TestBoard_AddPiece(t *testing.T) {
	tests := []struct {
		name         string
		initialBoard *Board
		pieceToAdd   IPiece
		x, y         int
		expectedErr  error
		expectPiece  IPiece // Expected piece at (x,y) after operation, nil if none
	}{
		{
			name:         "Add pawn to empty square",
			initialBoard: NewBoard(nil).(*Board),
			pieceToAdd:   NewPawn(White),
			x:            0, y: 0,
			expectedErr: nil,
			expectPiece: NewPawn(White),
		},
		{
			name: "Add rook to occupied square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(White), 0, 0)
				return b.(*Board)
			}(),
			pieceToAdd: NewRook(Black),
			x:          0, y: 0,
			expectedErr: errors.New("there is already a piece"),
			expectPiece: NewPawn(White), // Should still be the pawn
		},
		{
			name:         "Add piece to invalid coordinates",
			initialBoard: NewBoard(nil).(*Board),
			pieceToAdd:   NewKing(White),
			x:            -1, y: 0,
			expectedErr: errors.New("invalid coordinates"),
			expectPiece: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initialBoard.AddPiece(tt.pieceToAdd, tt.x, tt.y)

			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error, got %v", err)
				}
				actualPiece, _ := tt.initialBoard.GetPiece(tt.x, tt.y)
				if tt.expectPiece == nil {
					if actualPiece != nil {
						t.Errorf("expected no piece at (%d,%d), got %v", tt.x, tt.y, actualPiece)
					}
				} else {
					if actualPiece == nil {
						t.Errorf("expected piece %v at (%d,%d), got nil", tt.expectPiece, tt.x, tt.y)
					} else if actualPiece.GetType() != tt.expectPiece.GetType() || actualPiece.GetColor() != tt.expectPiece.GetColor() {
						t.Errorf("expected piece %v at (%d,%d), got %v", tt.expectPiece, tt.x, tt.y, actualPiece)
					}
				}
			}
		})
	}
}

func TestBoard_RemovePiece(t *testing.T) {
	tests := []struct {
		name         string
		initialBoard *Board
		x, y         int
		expectedErr  error
	}{
		{
			name: "Remove pawn from occupied square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(White), 0, 0)
				return b.(*Board)
			}(),
			x: 0, y: 0,
			expectedErr: nil,
		},
		{
			name:         "Remove piece from empty square",
			initialBoard: NewBoard(nil).(*Board),
			x:            0, y: 0,
			expectedErr: nil, // Should not error, just make sure it's nil
		},
		{
			name:         "Remove piece from invalid coordinates",
			initialBoard: NewBoard(nil).(*Board),
			x:            -1, y: 0,
			expectedErr: errors.New("invalid coordinates"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initialBoard.RemovePiece(tt.x, tt.y)

			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error, got %v", err)
				}
				actualPiece, _ := tt.initialBoard.GetPiece(tt.x, tt.y)
				if actualPiece != nil {
					t.Errorf("expected no piece at (%d,%d) after removal, got %v", tt.x, tt.y, actualPiece)
				}
			}
		})
	}
}

func TestBoard_MovePiece(t *testing.T) {
	tests := []struct {
		name             string
		initialBoard     *Board
		fromX, fromY     int
		toX, toY         int
		expectedErr      error
		expectedHasMoved map[string]bool // map[pieceType_color]hasMoved
		expectedKingPos  map[Color]Position
	}{
		{
			name: "Move Pawn to empty square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(White), 0, 6)
				b.SetKingPosition(White, 4, 7) // Set king pos
				b.SetKingPosition(Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 6, toX: 0, toY: 5,
			expectedErr:      nil,
			expectedHasMoved: map[string]bool{"Pawn_White": true},
			expectedKingPos: map[Color]Position{
				White: {X: 4, Y: 7},
				Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move Rook to capture opponent",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewRook(White), 0, 7)
				b.AddPiece(NewPawn(Black), 0, 5)
				b.SetKingPosition(White, 4, 7) // Set king pos
				b.SetKingPosition(Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 7, toX: 0, toY: 5,
			expectedErr:      nil,
			expectedHasMoved: map[string]bool{"Rook_White": true},
			expectedKingPos: map[Color]Position{
				White: {X: 4, Y: 7},
				Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move King and update KingPosition",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewKing(White), 4, 7)
				b.SetKingPosition(White, 4, 7) // Initial king pos
				b.SetKingPosition(Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 4, fromY: 7, toX: 4, toY: 6,
			expectedErr:      nil,
			expectedHasMoved: map[string]bool{"King_White": true},
			expectedKingPos: map[Color]Position{
				White: {X: 4, Y: 6}, // King position should be updated
				Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move from empty square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.SetKingPosition(White, 4, 7)
				b.SetKingPosition(Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 0, toX: 0, toY: 1,
			expectedErr:      errors.New("could not get piece from (0, 0): no piece found at coordinates"), // Customize if your GetPiece returns 'no piece found' error
			expectedHasMoved: nil,
			expectedKingPos: map[Color]Position{
				White: {X: 4, Y: 7},
				Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move to invalid coordinates",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(White), 0, 6)
				b.SetKingPosition(White, 4, 7)
				b.SetKingPosition(Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 6, toX: 0, toY: 8, // Invalid Y coordinate
			expectedErr:      errors.New("invalid coordinates"),
			expectedHasMoved: nil,
			expectedKingPos: map[Color]Position{
				White: {X: 4, Y: 7},
				Black: {X: 4, Y: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the piece before moving to check hasMoved status later
			pieceBeforeMove, _ := tt.initialBoard.GetPiece(tt.fromX, tt.fromY)

			err := tt.initialBoard.MovePiece(tt.fromX, tt.fromY, tt.toX, tt.toY)

			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				// If error, board state shouldn't change, but we are not checking it strictly here
				// as the error check is the primary focus.
			} else {
				if err != nil {
					t.Errorf("did not expect an error, got %v", err)
					return
				}

				// Verify the piece moved
				movedPiece, _ := tt.initialBoard.GetPiece(tt.toX, tt.toY)
				if movedPiece == nil {
					t.Errorf("expected piece at (%d,%d) after move, got nil", tt.toX, tt.toY)
				} else if movedPiece.GetType() != pieceBeforeMove.GetType() || movedPiece.GetColor() != pieceBeforeMove.GetColor() {
					t.Errorf("expected piece %v at destination, got %v", pieceBeforeMove, movedPiece)
				}

				// Verify original square is empty
				originalSquarePiece, _ := tt.initialBoard.GetPiece(tt.fromX, tt.fromY)
				if originalSquarePiece != nil {
					t.Errorf("expected original square (%d,%d) to be empty, got %v", tt.fromX, tt.fromY, originalSquarePiece)
				}

				// Verify hasMoved status
				if tt.expectedHasMoved != nil && pieceBeforeMove != nil {
					switch p := pieceBeforeMove.(type) {
					case *Pawn:
						if p.hasMoved != tt.expectedHasMoved["Pawn_White"] { // Simplified for White pawn here
							t.Errorf("Pawn hasMoved status mismatch. Expected %t, got %t", tt.expectedHasMoved["Pawn_White"], p.hasMoved)
						}
					case *Rook:
						if p.hasMoved != tt.expectedHasMoved["Rook_White"] { // Simplified for White rook here
							t.Errorf("Rook hasMoved status mismatch. Expected %t, got %t", tt.expectedHasMoved["Rook_White"], p.hasMoved)
						}
					case *King:
						if p.hasMoved != tt.expectedHasMoved["King_White"] { // Simplified for White king here
							t.Errorf("King hasMoved status mismatch. Expected %t, got %t", tt.expectedHasMoved["King_White"], p.hasMoved)
						}
					}
				}

				// Verify King Position update
				if tt.expectedKingPos != nil {
					for color, expectedPos := range tt.expectedKingPos {
						actualX, actualY := tt.initialBoard.GetKingPosition(color)
						if actualX != expectedPos.X || actualY != expectedPos.Y {
							t.Errorf("King position mismatch for %v. Expected (%d,%d), got (%d,%d)", color, expectedPos.X, expectedPos.Y, actualX, actualY)
						}
					}
				}
			}
		})
	}
}
