package render

import (
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// pieceValues are the usual point values; the king is not counted.
var pieceValues = map[common.PieceType]int{
	common.Pawn: 1, common.Knight: 3, common.Bishop: 3, common.Rook: 5, common.Queen: 9,
}

// startingCount is how many of each piece a side starts with.
var startingCount = map[common.PieceType]int{
	common.Pawn: 8, common.Knight: 2, common.Bishop: 2, common.Rook: 2, common.Queen: 1,
}

// capturedOrder lists pieces from the most valuable down, the order in
// which captures are shown.
var capturedOrder = []common.PieceType{common.Queen, common.Rook, common.Bishop, common.Knight, common.Pawn}

// Material is the sum of the piece values of each side on the board.
func Material(b chess.IBoard) (white, black int) {
	counts := pieceCounts(b)
	for _, t := range capturedOrder {
		white += counts[chess.White][t] * pieceValues[t]
		black += counts[chess.Black][t] * pieceValues[t]
	}
	return white, black
}

// Balance is White's material minus Black's: positive when White is ahead.
func Balance(b chess.IBoard) int {
	white, black := Material(b)
	return white - black
}

// Captured lists the pieces each side has taken, most valuable first, taking
// the standard starting set as the reference. A promoted piece counts as a
// captured pawn, since the pawn is gone and no piece of that kind was taken.
// It is only meaningful for games that start from the standard position.
func Captured(b chess.IBoard) (byWhite, byBlack []common.PieceType) {
	counts := pieceCounts(b)
	return missing(counts[chess.Black]), missing(counts[chess.White])
}

// missing lists what a side lost, given the pieces it still has.
func missing(have map[common.PieceType]int) []common.PieceType {
	promoted := 0
	for _, t := range capturedOrder[:4] {
		promoted += max(0, have[t]-startingCount[t])
	}
	var lost []common.PieceType
	for _, t := range capturedOrder {
		n := startingCount[t] - have[t]
		if t == common.Pawn {
			n -= promoted
		}
		for ; n > 0; n-- {
			lost = append(lost, t)
		}
	}
	return lost
}

func pieceCounts(b chess.IBoard) map[chess.Color]map[common.PieceType]int {
	counts := map[chess.Color]map[common.PieceType]int{chess.White: {}, chess.Black: {}}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if piece, err := b.GetPiece(x, y); err == nil && piece != nil {
				counts[piece.GetColor()][piece.GetType()]++
			}
		}
	}
	return counts
}

// materialLine draws what a side has captured, which are pieces of the
// other color, followed by "+N" when that side is ahead. Either part may be
// switched off; the result is empty when nothing is left to say.
func materialLine(captured []common.PieceType, capturedAreWhite bool, ahead int, style Style) string {
	var line strings.Builder
	for _, t := range captured {
		line.WriteString(pieceGlyph(t, capturedAreWhite, Style{Unicode: style.Unicode, Color: false}))
	}
	if ahead > 0 {
		if line.Len() > 0 {
			line.WriteByte(' ')
		}
		fmt.Fprintf(&line, "+%d", ahead)
	}
	return line.String()
}
