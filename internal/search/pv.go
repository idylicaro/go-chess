package search

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// pvTable is the triangular principal-variation table: row ply holds the
// best line found from that ply, built from the row below whenever a move
// raises alpha.
type pvTable struct {
	moves  [maxPly + 1][maxPly + 1]chess.SearchMove
	length [maxPly + 1]int
}

// clear empties the line at ply before its node is searched.
func (t *pvTable) clear(ply int) {
	t.length[ply] = 0
}

// update makes move followed by the child's line the best line at ply.
func (t *pvTable) update(ply int, move chess.SearchMove) {
	t.moves[ply][0] = move
	child := t.length[ply+1]
	copy(t.moves[ply][1:1+child], t.moves[ply+1][:child])
	t.length[ply] = child + 1
}

// line returns a copy of the root's principal variation.
func (t *pvTable) line() []chess.SearchMove {
	return append([]chess.SearchMove(nil), t.moves[0][:t.length[0]]...)
}
