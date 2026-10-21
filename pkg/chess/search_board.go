package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// MaxSearchMoves is the size of a move buffer that never grows: no position
// has more pseudo-legal moves.
const MaxSearchMoves = maxMovesPerPosition

// SearchBoard is a position for engines: moves are made and taken back in
// place without allocating, and only what a search needs is tracked
// (legality, the Zobrist key, repetitions and the fifty-move clock). It
// starts as a copy of a Game, which is left untouched (#37). A SearchBoard is
// not safe for concurrent use; give each goroutine its own Clone.
//
// Squares are indexes 0..63 row by row from rank 8: 0 is a8, 7 is h8, 56 is
// a1 and 63 is h1.
//
//	board := chess.NewSearchBoard(chess.NewGame())
//	var buffer [chess.MaxSearchMoves]chess.SearchMove
//	for _, move := range board.GenerateMoves(buffer[:0]) {
//		if undo, legal := board.MakeMove(move); legal {
//			// search the position after move
//			board.UnmakeMove(move, undo)
//		}
//	}
type SearchBoard struct {
	game *Game
	// history holds the keys of the positions before the current one, the
	// game's and the searched line's, for repetition detection.
	history []uint64
}

// SearchUndo is what UnmakeMove and UnmakeNullMove need to take a move back.
type SearchUndo struct {
	undo moveUndo
}

// NewSearchBoard copies the position of game, with its past positions so a
// search sees repetitions of moves already played.
//
//	board := chess.NewSearchBoard(game)
func NewSearchBoard(game *Game) *SearchBoard {
	past := game.positionHistory
	if len(past) > 0 {
		past = past[:len(past)-1] // the last entry is the current position
	}
	return &SearchBoard{game: game.clone(), history: append([]uint64(nil), past...)}
}

// Clone returns an independent copy, for a search in another goroutine.
func (b *SearchBoard) Clone() *SearchBoard {
	return &SearchBoard{game: b.game.clone(), history: append([]uint64(nil), b.history...)}
}

// SideToMove is the color whose move it is.
func (b *SearchBoard) SideToMove() Color {
	return b.game.currentTurn
}

// Hash is the Zobrist key of the position, the same one Game uses for
// threefold repetition.
func (b *SearchBoard) Hash() uint64 {
	return b.game.hash
}

// HalfMoveClock counts the plies since the last capture or pawn move.
func (b *SearchBoard) HalfMoveClock() int {
	return b.game.halfMoveClock
}

// CastlingRights are the castles still available.
func (b *SearchBoard) CastlingRights() CastlingRights {
	return b.game.castlingRights
}

// EnPassantSquare is the square a pawn skipped on the last move, if any,
// whether or not a pawn can take there (as in FEN).
func (b *SearchBoard) EnPassantSquare() (square int, ok bool) {
	target := b.game.enPassantTarget
	if target == nil {
		return 0, false
	}
	return squareIndex(target.X, target.Y), true
}

// FEN describes the position. The move number is not kept across null moves.
func (b *SearchBoard) FEN() string {
	return b.game.FEN()
}

// PieceOn returns the piece on square, or occupied false for an empty one.
//
//	pieceType, color, occupied := board.PieceOn(60) // the e1 square
func (b *SearchBoard) PieceOn(square int) (pieceType PieceType, color Color, occupied bool) {
	code := b.game.board.squares[square]
	return code.pieceType(), code.color(), code != noPiece
}

// InCheck reports whether the side to move is in check.
func (b *SearchBoard) InCheck() bool {
	return b.game.isSideToMoveInCheck()
}

// IsRepetition reports whether the current position already occurred since
// the last capture or pawn move. A search scores it as a draw: the side that
// can repeat once can repeat again.
func (b *SearchBoard) IsRepetition() bool {
	oldest := max(0, len(b.history)-b.game.halfMoveClock)
	for i := len(b.history) - 2; i >= oldest; i -= 2 { // same side to move only
		if b.history[i] == b.game.hash {
			return true
		}
	}
	return false
}

// GenerateMoves appends the pseudo-legal moves of the side to move: they may
// leave the own king in check, which MakeMove then rejects. Pass a slice of
// a stack array to generate without allocating.
func (b *SearchBoard) GenerateMoves(moves []SearchMove) []SearchMove {
	return b.game.appendPseudoLegalMoves(moves)
}

// GenerateCaptures appends the pseudo-legal captures and promotions, the
// moves a quiescence search looks at.
func (b *SearchBoard) GenerateCaptures(moves []SearchMove) []SearchMove {
	start := len(moves)
	moves = b.game.appendPseudoLegalMoves(moves)
	tactical := moves[:start]
	// Filtering in place is safe: tactical never grows past the move being read.
	for _, move := range moves[start:] {
		if move.promotion != common.Pawn || b.IsCapture(move) {
			tactical = append(tactical, move)
		}
	}
	return tactical
}

// IsCapture reports whether move takes a piece, en passant included.
func (b *SearchBoard) IsCapture(move SearchMove) bool {
	_, captures := b.CapturedPiece(move)
	return captures
}

// MovedPiece is the type of the piece move picks up.
func (b *SearchBoard) MovedPiece(move SearchMove) PieceType {
	return b.game.board.squares[move.from].pieceType()
}

// CapturedPiece is the type of the piece move takes, if any. A pawn moving
// diagonally to an empty square takes en passant.
func (b *SearchBoard) CapturedPiece(move SearchMove) (PieceType, bool) {
	squares := &b.game.board.squares
	if target := squares[move.to]; target != noPiece {
		return target.pieceType(), true
	}
	isDiagonalPawnMove := squares[move.from].pieceType() == common.Pawn && move.from%8 != move.to%8
	return common.Pawn, isDiagonalPawnMove
}

// MakeMove plays a pseudo-legal move from GenerateMoves. It returns legal
// false, with the position unchanged, if the move leaves the own king in
// check; otherwise take it back with UnmakeMove.
func (b *SearchBoard) MakeMove(move SearchMove) (SearchUndo, bool) {
	g := b.game
	mover := g.currentTurn
	b.history = append(b.history, g.hash)
	undo := g.makeMove(move)
	if g.isKingAttacked(mover) {
		g.unmakeMove(move, undo)
		b.history = b.history[:len(b.history)-1]
		return SearchUndo{}, false
	}
	return SearchUndo{undo: undo}, true
}

// UnmakeMove takes back a move made with MakeMove.
func (b *SearchBoard) UnmakeMove(move SearchMove, undo SearchUndo) {
	b.game.unmakeMove(move, undo.undo)
	b.history = b.history[:len(b.history)-1]
}

// MakeNullMove passes the turn without moving, for null-move pruning. Do
// not call it while in check: the position after it would be illegal.
func (b *SearchBoard) MakeNullMove() SearchUndo {
	g := b.game
	b.history = append(b.history, g.hash)
	undo := moveUndo{castlingRights: g.castlingRights, enPassantTarget: g.enPassantTarget, halfMoveClock: g.halfMoveClock, hash: g.hash}
	g.hash ^= enPassantKey(g.enPassantTarget) ^ zobrist.black
	g.enPassantTarget = nil
	g.halfMoveClock++
	g.currentTurn = opposite(g.currentTurn)
	return SearchUndo{undo: undo}
}

// UnmakeNullMove takes back MakeNullMove.
func (b *SearchBoard) UnmakeNullMove(undo SearchUndo) {
	g := b.game
	g.currentTurn = opposite(g.currentTurn)
	g.enPassantTarget, g.halfMoveClock, g.hash = undo.undo.enPassantTarget, undo.undo.halfMoveClock, undo.undo.hash
	b.history = b.history[:len(b.history)-1]
}
