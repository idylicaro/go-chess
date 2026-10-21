package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"testing"
)

func TestPieceCode_RoundTripsEveryPiece(t *testing.T) {
	for _, color := range []common.Color{common.White, common.Black} {
		for pieceType := common.Pawn; pieceType <= common.King; pieceType++ {
			code := codeOf(pieceType, color)
			if code == noPiece || code.pieceType() != pieceType || code.color() != color || !code.isColor(color) {
				t.Errorf("codeOf(%v, %v) = %d decodes to (%v, %v)", pieceType, color, code, code.pieceType(), code.color())
			}
			view := pieceViews[code]
			if view.GetType() != pieceType || view.GetColor() != color || codeOfPiece(view) != code {
				t.Errorf("pieceViews[%d] is %v %v, want %v %v", code, view.GetColor(), view.GetType(), color, pieceType)
			}
		}
	}
}

func TestPieceCode_EmptySquare(t *testing.T) {
	if codeOfPiece(nil) != noPiece || noPiece.isColor(common.White) || pieceViews[noPiece] != nil {
		t.Error("an empty square must pack to noPiece, belong to no color and read back as a nil IPiece")
	}
}

func TestBoard_CopyIsIndependent(t *testing.T) {
	original := newInitialBoard()
	copied := original.Copy()
	_ = copied.MovePiece(4, 6, 4, 4)
	if piece, _ := original.GetPiece(4, 6); piece == nil {
		t.Error("moving a piece on the copy emptied e2 on the original")
	}
}
