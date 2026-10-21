package book

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// polyglotPromotions are the pieces of the move's promotion field, 1 to 4.
var polyglotPromotions = [5]string{"", "n", "b", "r", "q"}

// decodeMove turns a book move into UCI notation. Polyglot writes castling
// as the king taking its own rook (e1h1); UCI moves the king two squares.
func decodeMove(board *chess.SearchBoard, move uint16) string {
	toFile, toRow := int(move&7), int(move>>3&7)
	fromFile, fromRow := int(move>>6&7), int(move>>9&7)
	promotion := polyglotPromotions[min(int(move>>12&7), 4)]
	if isCastlingAsRookCapture(board, fromFile, fromRow, toFile, toRow) {
		toFile = map[int]int{7: 6, 0: 2}[toFile]
	}
	return squareName(fromFile, fromRow) + squareName(toFile, toRow) + promotion
}

// isCastlingAsRookCapture: the king on e1/e8 "moving" to its rook's corner.
func isCastlingAsRookCapture(board *chess.SearchBoard, fromFile, fromRow, toFile, toRow int) bool {
	if fromFile != 4 || fromRow != toRow || (fromRow != 0 && fromRow != 7) || (toFile != 0 && toFile != 7) {
		return false
	}
	pieceType, _, occupied := board.PieceOn((7-fromRow)*8 + fromFile)
	return occupied && pieceType == chess.King
}

// squareName names a square by Polyglot file and row (row 0 is rank 1).
func squareName(file, row int) string {
	return string([]byte{byte('a' + file), byte('1' + row)})
}
