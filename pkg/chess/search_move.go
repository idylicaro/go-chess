package chess

// NullSearchMove is the zero SearchMove: no move. Engines use it for "no
// best move yet" and for an empty transposition-table slot.
var NullSearchMove = SearchMove{}

// From is the square the move leaves, in SearchBoard's numbering.
func (m SearchMove) From() int {
	return int(m.from)
}

// To is the square the move reaches, in SearchBoard's numbering.
func (m SearchMove) To() int {
	return int(m.to)
}

// Promotion is the piece a pawn becomes, or PieceType Pawn when the move
// does not promote.
func (m SearchMove) Promotion() PieceType {
	return m.promotion
}

// IsNull reports whether m is NullSearchMove; a real move never stays on
// its square.
func (m SearchMove) IsNull() bool {
	return m.from == m.to
}

// String is the move in UCI notation, such as "e2e4" or "e7e8q".
//
//	move.String() // "g1f3"
func (m SearchMove) String() string {
	if m.IsNull() {
		return "0000" // the UCI null move
	}
	return m.toUCI().String()
}

// Move is the move in the form Game.Move takes.
func (m SearchMove) Move() Move {
	return m.toUCI()
}
