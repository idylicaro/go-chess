package search

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Move ordering decides how much alpha-beta prunes: the sooner the best
// move is tried, the sooner the rest are cut off (#38). The order is the
// hash move, then captures and queen promotions by MVV-LVA (most valuable
// victim, least valuable attacker), then the two killer moves of the ply,
// then quiet moves by their history, and underpromotions last.
const (
	hashMoveScore     = 1 << 30
	tacticalBase      = 1 << 20
	firstKillerScore  = tacticalBase - 1
	secondKillerScore = tacticalBase - 2
	// historyLimit keeps every history score below the killers; reaching
	// it halves the table, so old cutoffs fade.
	historyLimit       = tacticalBase / 2
	underpromotionBase = -1
)

// orderingValues ranks the pieces for MVV-LVA, indexed by chess.PieceType.
var orderingValues = [6]int{100, 320, 330, 500, 900, 2000}

// moveOrderer holds what the search learns about quiet moves: killers are
// quiet moves that caused a cutoff at the same ply, history counts the
// cutoffs of each from-to pair over the whole search.
type moveOrderer struct {
	killers [maxPly + 1][2]chess.SearchMove
	history [2][64][64]int
}

// score ranks move for the ordering at ply; higher is tried first.
func (o *moveOrderer) score(board *chess.SearchBoard, move, hashMove chess.SearchMove, ply int) int {
	if move == hashMove {
		return hashMoveScore
	}
	if score, tactical := tacticalScore(board, move); tactical {
		return score
	}
	if move.Promotion() != chess.Pawn {
		return underpromotionBase - int(chess.Queen-move.Promotion())
	}
	return o.quietScore(board.SideToMove(), move, ply)
}

// tacticalScore is the MVV-LVA score of a capture or queen promotion.
func tacticalScore(board *chess.SearchBoard, move chess.SearchMove) (int, bool) {
	victim, captures := board.CapturedPiece(move)
	queens := move.Promotion() == chess.Queen
	if !captures && !queens {
		return 0, false
	}
	gain := 0
	if captures {
		gain += orderingValues[victim]
	}
	if queens {
		gain += orderingValues[chess.Queen]
	}
	return tacticalBase + gain*8 - int(board.MovedPiece(move)), true
}

func (o *moveOrderer) quietScore(side chess.Color, move chess.SearchMove, ply int) int {
	switch move {
	case o.killers[ply][0]:
		return firstKillerScore
	case o.killers[ply][1]:
		return secondKillerScore
	}
	return o.history[side][move.From()][move.To()]
}

// recordCutoff learns from a quiet move that failed high at ply.
func (o *moveOrderer) recordCutoff(side chess.Color, move chess.SearchMove, depth, ply int) {
	if o.killers[ply][0] != move {
		o.killers[ply][1] = o.killers[ply][0]
		o.killers[ply][0] = move
	}
	entry := &o.history[side][move.From()][move.To()]
	*entry += depth * depth
	if *entry >= historyLimit {
		o.ageHistory()
	}
}

func (o *moveOrderer) ageHistory() {
	for side := range o.history {
		for from := range o.history[side] {
			for to := range o.history[side][from] {
				o.history[side][from][to] /= 2
			}
		}
	}
}

// movePicker hands out moves best first. Selection sort is cheaper than a
// full sort here: most nodes cut off after the first few moves.
type movePicker struct {
	moves  []chess.SearchMove
	scores [chess.MaxSearchMoves]int
	next   int
}

// newMovePicker scores moves for the node at ply.
func (o *moveOrderer) newMovePicker(board *chess.SearchBoard, moves []chess.SearchMove, hashMove chess.SearchMove, ply int) movePicker {
	picker := movePicker{moves: moves}
	for i, move := range moves {
		picker.scores[i] = o.score(board, move, hashMove, ply)
	}
	return picker
}

// pick returns the best move not yet handed out.
func (p *movePicker) pick() (chess.SearchMove, bool) {
	if p.next >= len(p.moves) {
		return chess.NullSearchMove, false
	}
	best := p.next
	for i := p.next + 1; i < len(p.moves); i++ {
		if p.scores[i] > p.scores[best] {
			best = i
		}
	}
	p.swap(p.next, best)
	p.next++
	return p.moves[p.next-1], true
}

func (p *movePicker) swap(i, j int) {
	p.moves[i], p.moves[j] = p.moves[j], p.moves[i]
	p.scores[i], p.scores[j] = p.scores[j], p.scores[i]
}
