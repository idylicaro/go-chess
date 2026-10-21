package search

import (
	"math/bits"
	"sync/atomic"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// The transposition table remembers searched positions by Zobrist key (the
// same key as threefold repetition, #12), so a position reached by another
// move order is not searched again and its best move is tried first (#40).

// bound says how a stored score relates to the true one.
type bound uint8

const (
	boundNone  bound = iota
	boundExact       // the score is exact
	boundLower       // the search failed high: the true score is at least this
	boundUpper       // the search failed low: the true score is at most this
)

// ttEntryBytes is the size of one ttSlot, for sizing the table in megabytes.
const ttEntryBytes = 16

// DefaultHashMB is the table size when Options.HashMB is zero.
const DefaultHashMB = 16

// ttSlot is lockless: check holds key XOR data, so a slot torn by two
// goroutines writing at once fails the key check instead of returning a
// mix of two positions (Hyatt and Mann's scheme). The atomics keep the race
// detector quiet and cost nothing on amd64 loads.
type ttSlot struct {
	check atomic.Uint64
	data  atomic.Uint64
}

// ttEntry is a slot's data unpacked.
type ttEntry struct {
	move  chess.SearchMove
	score int
	depth int
	bound bound
}

type transpositionTable struct {
	slots []ttSlot
	mask  uint64
}

// newTranspositionTable allocates the largest power-of-two number of slots
// that fits in megabytes (at least one).
func newTranspositionTable(megabytes int) *transpositionTable {
	slots := max(1, megabytes) << 20 / ttEntryBytes
	size := uint64(1) << (bits.Len64(uint64(slots)) - 1)
	return &transpositionTable{slots: make([]ttSlot, size), mask: size - 1}
}

// probe returns the entry stored for key, if any.
func (t *transpositionTable) probe(key uint64) (ttEntry, bool) {
	slot := &t.slots[key&t.mask]
	data := slot.data.Load()
	if slot.check.Load()^data != key || data == 0 {
		return ttEntry{}, false
	}
	return unpackEntry(data), true
}

// store always replaces the slot of key: the simplest scheme, and it keeps
// the newest result, which is the most likely to be probed again.
func (t *transpositionTable) store(key uint64, entry ttEntry) {
	data := packEntry(entry)
	slot := &t.slots[key&t.mask]
	slot.data.Store(data)
	slot.check.Store(key ^ data)
}

// clear forgets every position, for a new game.
func (t *transpositionTable) clear() {
	for i := range t.slots {
		t.slots[i].check.Store(0)
		t.slots[i].data.Store(0)
	}
}

// Data layout: move in bits 0-15, score (offset to be positive) in 16-31,
// depth in 32-39 and bound in 40-41.
func packEntry(entry ttEntry) uint64 {
	return uint64(entry.move.Pack()) | uint64(uint16(entry.score+infinity))<<16 |
		uint64(uint8(entry.depth))<<32 | uint64(entry.bound)<<40
}

func unpackEntry(data uint64) ttEntry {
	return ttEntry{
		move:  chess.UnpackSearchMove(uint16(data)),
		score: int(uint16(data>>16)) - infinity,
		depth: int(uint8(data >> 32)),
		bound: bound(data >> 40 & 3),
	}
}

// scoreToTT stores mate scores as distance from this node rather than from
// the root, so they stay right when the position is reached at another ply.
func scoreToTT(score, ply int) int {
	switch {
	case score >= mateBound:
		return score + ply
	case score <= -mateBound:
		return score - ply
	}
	return score
}

func scoreFromTT(score, ply int) int {
	switch {
	case score >= mateBound:
		return score - ply
	case score <= -mateBound:
		return score + ply
	}
	return score
}

// cutoffScore returns the stored score when it settles the node: deep
// enough, and exact or a bound outside the window.
func (e ttEntry) cutoffScore(depth, ply, alpha, beta int) (int, bool) {
	if e.depth < depth {
		return 0, false
	}
	score := scoreFromTT(e.score, ply)
	switch {
	case e.bound == boundExact,
		e.bound == boundLower && score >= beta,
		e.bound == boundUpper && score <= alpha:
		return score, true
	}
	return 0, false
}

// boundFor classifies a node's result against its original window.
func boundFor(score, alpha, beta int) bound {
	switch {
	case score <= alpha:
		return boundUpper
	case score >= beta:
		return boundLower
	}
	return boundExact
}
