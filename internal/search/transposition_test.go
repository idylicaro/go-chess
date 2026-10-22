package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func TestTranspositionTable_StoresAndProbesAnEntry(t *testing.T) {
	table := newTranspositionTable(1)
	move := findMove(t, boardFromFEN(t, "4k3/P7/8/8/8/8/8/4K3 w - - 0 1"), "a7a8n")
	want := ttEntry{move: move, score: -1234, depth: 9, bound: boundLower}
	table.store(0xDEADBEEF, want)
	got, found := table.probe(0xDEADBEEF)
	if !found || got != want {
		t.Errorf("probe = %+v, %v, want %+v", got, found, want)
	}
}

func TestTranspositionTable_MissesAnotherKeyInTheSameSlot(t *testing.T) {
	table := newTranspositionTable(1)
	table.store(5, ttEntry{score: 10, depth: 1, bound: boundExact})
	if _, found := table.probe(5 + uint64(len(table.slots))); found {
		t.Error("probe found an entry stored under another key")
	}
	if _, found := table.probe(6); found {
		t.Error("probe found an entry in an empty slot")
	}
}

func TestTranspositionTable_RejectsATornSlot(t *testing.T) {
	table := newTranspositionTable(1)
	table.store(7, ttEntry{score: 10, depth: 1, bound: boundExact})
	// Another goroutine's write lands in data but not yet in check.
	table.slots[7].data.Store(packEntry(ttEntry{score: 99, depth: 3, bound: boundExact}))
	if _, found := table.probe(7); found {
		t.Error("probe accepted a slot whose data does not match its key")
	}
}

func TestTranspositionTable_SizeIsAPowerOfTwoWithinTheBudget(t *testing.T) {
	for _, megabytes := range []int{0, 1, 3, 16} {
		table := newTranspositionTable(megabytes)
		size := len(table.slots)
		if size&(size-1) != 0 || size*ttEntryBytes > max(1, megabytes)<<20 {
			t.Errorf("newTranspositionTable(%d) has %d slots, want a power of two within the budget", megabytes, size)
		}
	}
}

func TestTranspositionTable_ClearForgetsEverything(t *testing.T) {
	table := newTranspositionTable(1)
	table.store(42, ttEntry{score: 1, depth: 1, bound: boundExact})
	table.clear()
	if _, found := table.probe(42); found {
		t.Error("probe found an entry after clear")
	}
}

func TestScoreToTT_KeepsMateDistanceFromTheNode(t *testing.T) {
	// Mate in 5 plies from the root, found at ply 2: 3 plies from the node.
	stored := scoreToTT(mateScore-5, 2)
	if got := scoreFromTT(stored, 4); got != mateScore-7 {
		t.Errorf("mate reached again at ply 4 scores %d, want %d", got, mateScore-7)
	}
	if got := scoreFromTT(scoreToTT(-mateScore+5, 2), 2); got != -mateScore+5 {
		t.Errorf("mated score round trip = %d, want %d", got, -mateScore+5)
	}
	if got := scoreToTT(150, 9); got != 150 {
		t.Errorf("scoreToTT(150) = %d, want it unchanged", got)
	}
}

func TestTTEntry_CutoffScore(t *testing.T) {
	cases := []struct {
		name    string
		entry   ttEntry
		settled bool
	}{
		{"exact", ttEntry{score: 50, depth: 4, bound: boundExact}, true},
		{"too shallow", ttEntry{score: 50, depth: 3, bound: boundExact}, false},
		{"lower bound above beta", ttEntry{score: 200, depth: 4, bound: boundLower}, true},
		{"lower bound inside the window", ttEntry{score: 50, depth: 4, bound: boundLower}, false},
		{"upper bound below alpha", ttEntry{score: -200, depth: 4, bound: boundUpper}, true},
		{"upper bound inside the window", ttEntry{score: 50, depth: 4, bound: boundUpper}, false},
	}
	for _, tc := range cases {
		if _, settled := tc.entry.cutoffScore(4, 1, -100, 100); settled != tc.settled {
			t.Errorf("%s: settled = %v, want %v", tc.name, settled, tc.settled)
		}
	}
}

func TestSearch_TableKeepsItsKnowledgeForTheNextSearch(t *testing.T) {
	searcher := New(eval.Evaluate, Options{HashMB: 4})
	board := boardFromFEN(t, kiwipeteFEN)
	first := searcher.Search(context.Background(), board, Limits{Depth: 5}, nil)
	second := searcher.Search(context.Background(), board, Limits{Depth: 5}, nil)
	if second.Nodes >= first.Nodes/2 {
		t.Errorf("repeating a search took %d nodes after %d, want the table to save most of them", second.Nodes, first.Nodes)
	}
	searcher.Clear()
	if _, found := searcher.tt.probe(board.Hash()); found {
		t.Error("the root is still in the table after Clear")
	}
	if first.Best.IsNull() || second.Best == chess.NullSearchMove {
		t.Error("a search returned no move")
	}
}
