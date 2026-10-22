package search

// Scores are centipawns from the side to move's view. A mate is scored
// mateScore minus its distance in plies, so the search prefers the shortest
// mate and the longest defence.
const (
	mateScore = 32000
	infinity  = mateScore + 1
	// mateBound is the lowest score that still means a forced mate: no
	// evaluation gets near it, and no mate is longer than maxPly.
	mateBound = mateScore - maxPly
	drawScore = 0
)

// maxPly bounds how deep a line may go, extensions included.
const maxPly = 128

// matedIn is the score of being checkmated at ply.
func matedIn(ply int) int {
	return -mateScore + ply
}

// isMateScore reports whether score announces a forced mate either way.
func isMateScore(score int) bool {
	return score >= mateBound || score <= -mateBound
}

// MateInMoves turns a mate score into UCI's "score mate N": moves (not plies)
// to the mate, negative when the side to move is the one being mated. It
// returns ok false for an ordinary score.
//
//	MateInMoves(mateScore - 3) // 2, true: mate on the second move
func MateInMoves(score int) (moves int, ok bool) {
	if !isMateScore(score) {
		return 0, false
	}
	if score > 0 {
		return (mateScore - score + 1) / 2, true
	}
	return -(mateScore + score) / 2, true
}
