package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// Zobrist hashing gives each position a 64-bit key that makeMove updates
// with a few XORs, replacing the repetition key that was rebuilt by string
// concatenation after every move (#12). Equal positions get equal keys; a
// collision between different ones is possible but vanishingly rare.

// zobristSeed is fixed so keys, and so any stored hash, are reproducible.
const zobristSeed = 0x9E3779B97F4A7C15

type zobristKeys struct {
	piece    [16][64]uint64 // indexed by pieceCode; noPiece stays zero
	castling [16]uint64     // indexed by CastlingRights
	epFile   [8]uint64
	black    uint64 // toggled when Black is to move
}

var zobrist = buildZobristKeys()

func buildZobristKeys() *zobristKeys {
	keys := &zobristKeys{}
	state := uint64(zobristSeed)
	for code := range pieceViews {
		if pieceCode(code) == noPiece {
			continue
		}
		for square := 0; square < 64; square++ {
			keys.piece[code][square] = splitMix64(&state)
		}
	}
	for rights := range keys.castling {
		keys.castling[rights] = splitMix64(&state)
	}
	for file := range keys.epFile {
		keys.epFile[file] = splitMix64(&state)
	}
	keys.black = splitMix64(&state)
	return keys
}

// splitMix64 is a small, well-mixed generator; good enough for hash keys.
func splitMix64(state *uint64) uint64 {
	*state += 0x9E3779B97F4A7C15
	z := *state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// enPassantKey is zero when there is no en passant target. As before, the
// target counts whenever it is set, even if no pawn can take it.
func enPassantKey(target *Position) uint64 {
	if target == nil {
		return 0
	}
	return zobrist.epFile[target.X]
}

// computeHash builds the key from scratch; makeMove keeps it current after.
//
//	game.hash == game.computeHash() // holds after every move
func (g *Game) computeHash() uint64 {
	var hash uint64
	for square, code := range g.board.squares {
		hash ^= zobrist.piece[code][square]
	}
	hash ^= zobrist.castling[g.castlingRights] ^ enPassantKey(g.enPassantTarget)
	if g.currentTurn == common.Black {
		hash ^= zobrist.black
	}
	return hash
}

// putCode writes a square and keeps the hash in step.
func (g *Game) putCode(square int, code pieceCode) {
	old := g.board.squares[square]
	g.hash ^= zobrist.piece[old][square] ^ zobrist.piece[code][square]
	g.board.squares[square] = code
}

// isThreefoldRepetition records the current position and reports whether it
// has now occurred three times. Only positions since the last capture or
// pawn move can repeat, so the scan stops there.
func (g *Game) isThreefoldRepetition() bool {
	g.positionHistory = append(g.positionHistory, g.hash)
	occurrences := 0
	oldest := max(0, len(g.positionHistory)-1-g.halfMoveClock)
	for _, hash := range g.positionHistory[oldest:] {
		if hash == g.hash {
			occurrences++
		}
	}
	return occurrences >= 3
}
