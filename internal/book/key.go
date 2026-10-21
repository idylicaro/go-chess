package book

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Offsets of each kind of key inside random64.
const (
	castleKeys    = 768
	enPassantKeys = 772
	turnKey       = 780
)

// polyglotCastles lists the rights in Polyglot's key order.
var polyglotCastles = [4]chess.CastlingRights{chess.WhiteKingSide, chess.WhiteQueenSide, chess.BlackKingSide, chess.BlackQueenSide}

// Key is the Polyglot hash of board, the key a book is sorted by. It is
// not the engine's own Zobrist key: the format fixes its random numbers.
//
//	book.Key(chess.NewSearchBoard(chess.NewGame())) // 0x463b96181691fc9c
func Key(board *chess.SearchBoard) uint64 {
	key := pieceKeys(board)
	for i, right := range polyglotCastles {
		if board.CastlingRights().Has(right) {
			key ^= random64[castleKeys+i]
		}
	}
	if file, ok := capturableEnPassantFile(board); ok {
		key ^= random64[enPassantKeys+file]
	}
	if board.SideToMove() == chess.White {
		key ^= random64[turnKey]
	}
	return key
}

// pieceKeys: kind is 2 per piece type, black first (black pawn 0, white
// pawn 1 ... white king 11); squares count from a1 by rank.
func pieceKeys(board *chess.SearchBoard) uint64 {
	var key uint64
	for square := range 64 {
		pieceType, color, occupied := board.PieceOn(square)
		if !occupied {
			continue
		}
		kind := 2 * int(pieceType)
		if color == chess.White {
			kind++
		}
		key ^= random64[64*kind+polyglotSquare(square)]
	}
	return key
}

// polyglotSquare maps chess.SearchBoard's square (0 is a8) to Polyglot's
// (0 is a1).
func polyglotSquare(square int) int {
	return (7-square/8)*8 + square%8
}

// capturableEnPassantFile counts the en passant square only when a pawn of
// the side to move stands next to the pawn that just advanced, as the
// format requires (unlike FEN).
func capturableEnPassantFile(board *chess.SearchBoard) (int, bool) {
	target, ok := board.EnPassantSquare()
	if !ok {
		return 0, false
	}
	side := board.SideToMove()
	pushed := target + 8 // the advanced pawn is one row behind, seen from the capturer
	if side == chess.Black {
		pushed = target - 8
	}
	file := target % 8
	for _, neighbour := range []int{file - 1, file + 1} {
		if neighbour < 0 || neighbour > 7 {
			continue
		}
		if pieceType, color, occupied := board.PieceOn(pushed - file + neighbour); occupied && pieceType == chess.Pawn && color == side {
			return file, true
		}
	}
	return 0, false
}
