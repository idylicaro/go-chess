package chess

import (
	"idylicaro/go-chess/internal/common"
	"testing"
)

func TestCoordinateConverter_GetPieceSymbol(t *testing.T) {
	cc := &CoordinateConverter{}

	tests := []struct {
		name      string
		pieceType common.PieceType
		expected  string
	}{
		{"Pawn", common.Pawn, ""},
		{"Knight", common.Knight, "N"},
		{"Bishop", common.Bishop, "B"},
		{"Rook", common.Rook, "R"},
		{"Queen", common.Queen, "Q"},
		{"King", common.King, "K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cc.GetPieceSymbol(tt.pieceType)
			if result != tt.expected {
				t.Errorf("GetPieceSymbol() = %v, want %v", result, tt.expected)
			}
		})
	}
}
