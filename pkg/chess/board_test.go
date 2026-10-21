// internal/chess/board_test.go
package chess

import (
	"errors"
	"idylicaro/go-chess/internal/common"
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
			pieceToAdd:   NewPawn(common.White),
			x:            0, y: 0,
			expectedErr: nil,
			expectPiece: NewPawn(common.White),
		},
		{
			name: "Add rook to occupied square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(common.White), 0, 0)
				return b.(*Board)
			}(),
			pieceToAdd: NewRook(common.Black),
			x:          0, y: 0,
			expectedErr: errors.New("cannot add a piece at (0, 0): the square is occupied"),
			expectPiece: NewPawn(common.White), // Should still be the pawn
		},
		{
			name:         "Add piece to invalid coordinates",
			initialBoard: NewBoard(nil).(*Board),
			pieceToAdd:   NewKing(common.White),
			x:            -1, y: 0,
			expectedErr: errors.New("invalid coordinates (-1, 0): expected both between 0 and 7"),
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
				b.AddPiece(NewPawn(common.White), 0, 0)
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
			expectedErr: errors.New("invalid coordinates (-1, 0): expected both between 0 and 7"),
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
		name            string
		initialBoard    *Board
		fromX, fromY    int
		toX, toY        int
		expectedErr     error
		expectedKingPos map[common.Color]Position
	}{
		{
			name: "Move Pawn to empty square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(common.White), 0, 6)
				b.SetKingPosition(common.White, 4, 7) // Set king pos
				b.SetKingPosition(common.Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 6, toX: 0, toY: 5,
			expectedErr: nil,
			expectedKingPos: map[common.Color]Position{
				common.White: {X: 4, Y: 7},
				common.Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move Rook to capture opponent",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewRook(common.White), 0, 7)
				b.AddPiece(NewPawn(common.Black), 0, 5)
				b.SetKingPosition(common.White, 4, 7) // Set king pos
				b.SetKingPosition(common.Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 7, toX: 0, toY: 5,
			expectedErr: nil,
			expectedKingPos: map[common.Color]Position{
				common.White: {X: 4, Y: 7},
				common.Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move King and update KingPosition",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewKing(common.White), 4, 7)
				b.SetKingPosition(common.White, 4, 7) // Initial king pos
				b.SetKingPosition(common.Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 4, fromY: 7, toX: 4, toY: 6,
			expectedErr: nil,
			expectedKingPos: map[common.Color]Position{
				common.White: {X: 4, Y: 6}, // King position should be updated
				common.Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move from empty square",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.SetKingPosition(common.White, 4, 7)
				b.SetKingPosition(common.Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 0, toX: 0, toY: 1,
			expectedErr: errors.New("cannot move from (0, 0): there is no piece there"), // Customize if your GetPiece returns 'no piece found' error
			expectedKingPos: map[common.Color]Position{
				common.White: {X: 4, Y: 7},
				common.Black: {X: 4, Y: 0},
			},
		},
		{
			name: "Move to invalid coordinates",
			initialBoard: func() *Board {
				b := NewBoard(nil)
				b.AddPiece(NewPawn(common.White), 0, 6)
				b.SetKingPosition(common.White, 4, 7)
				b.SetKingPosition(common.Black, 4, 0)
				return b.(*Board)
			}(),
			fromX: 0, fromY: 6, toX: 0, toY: 8, // Invalid Y coordinate
			expectedErr: errors.New("invalid coordinates (0, 8): expected both between 0 and 7"),
			expectedKingPos: map[common.Color]Position{
				common.White: {X: 4, Y: 7},
				common.Black: {X: 4, Y: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the piece before moving to compare with the destination later
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
