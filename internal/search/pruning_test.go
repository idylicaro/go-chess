package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
)

// pruningConfigs are the steps of lab #44, each adding one technique.
var pruningConfigs = []struct {
	name    string
	pruning Pruning
}{
	{"none", Pruning{}},
	{"null", Pruning{NullMove: true}},
	{"null+lmr", Pruning{NullMove: true, LateMoveReductions: true}},
	{"all", AllPruning},
}

func TestSearch_PruningKeepsTheTactics(t *testing.T) {
	cases := []struct {
		fen, best string
		mate      int
		depth     int
	}{
		{scholarsMateFEN, "h5f7", 1, 4},
		{rookMateInTwoFEN, "", 2, 6},
		{hangingQueenFEN, "e4d5", 0, 5},
		{"4k3/8/4p3/3p4/8/8/8/3QK3 w - - 0 1", "", 0, 5}, // must not play Qxd5
	}
	for _, config := range pruningConfigs {
		for _, tc := range cases {
			searcher := New(eval.Evaluate, Options{HashMB: 1, Pruning: config.pruning})
			result := searcher.Search(context.Background(), boardFromFEN(t, tc.fen), Limits{Depth: tc.depth}, nil)
			if tc.best != "" && result.Best.String() != tc.best {
				t.Errorf("%s, %q: best %s, want %s", config.name, tc.fen, result.Best, tc.best)
			}
			if moves, _ := MateInMoves(result.Score); tc.mate != 0 && moves != tc.mate {
				t.Errorf("%s, %q: score %d, want mate %d", config.name, tc.fen, result.Score, tc.mate)
			}
			if result.Best.String() == "d1d5" {
				t.Errorf("%s: plays Qxd5 into exd5", config.name)
			}
		}
	}
}

func TestPruning_ReducesTheNodesToAFixedDepth(t *testing.T) {
	var previous int64
	for i, config := range pruningConfigs[:3] {
		searcher := New(eval.Evaluate, Options{HashMB: 4, Pruning: config.pruning})
		nodes := searcher.Search(context.Background(), boardFromFEN(t, kiwipeteFEN), Limits{Depth: 6}, nil).Nodes
		if i > 0 && nodes >= previous {
			t.Errorf("%s searched %d nodes to depth 6, want fewer than the %d before it", config.name, nodes, previous)
		}
		previous = nodes
	}
}

func TestHasPieces_KingAndPawnsOnlyIsZugzwangTerritory(t *testing.T) {
	if newTestWorker(boardFromFEN(t, "4k3/4p3/8/8/8/8/4P3/4K3 w - - 0 1")).hasPieces() {
		t.Error("hasPieces with king and pawns = true, want false")
	}
	if !newTestWorker(boardFromFEN(t, "4k3/8/8/8/8/8/4P3/4KN2 w - - 0 1")).hasPieces() {
		t.Error("hasPieces with a knight = false, want true")
	}
}

func TestTryNullMove_SkippedInCheckAndWithoutPieces(t *testing.T) {
	w := newTestWorker(boardFromFEN(t, "4k3/4p3/8/8/8/8/4P3/4K3 w - - 0 1"))
	w.pruning = AllPruning
	n := w.newNode(6, 1, -infinity)
	if _, cut := w.tryNullMove(n, -1000, true); cut {
		t.Error("null move tried with only king and pawns")
	}
	checked := newTestWorker(boardFromFEN(t, "4k3/8/8/8/8/8/8/R3K2r w - - 0 1"))
	checked.pruning = AllPruning
	if _, cut := checked.tryNullMove(checked.newNode(6, 1, -infinity), -1000, true); cut {
		t.Error("null move tried in check")
	}
}

func TestReduction(t *testing.T) {
	w := newTestWorker(boardFromFEN(t, kiwipeteFEN))
	w.pruning = AllPruning
	n := node{depth: 8, ply: 2}
	if r := w.reduction(n, 1, true); r != 0 {
		t.Errorf("reduction of the second move = %d, want 0", r)
	}
	if r := w.reduction(n, 20, false); r != 0 {
		t.Errorf("reduction of a capture or check = %d, want 0", r)
	}
	late := w.reduction(n, 20, true)
	if late < 1 || late > n.depth-2 || w.reduction(n, 40, true) < late {
		t.Errorf("reduction of late moves = %d, want at least 1, at most depth-2, growing with the index", late)
	}
}

func TestIsFutile(t *testing.T) {
	w := newTestWorker(boardFromFEN(t, kiwipeteFEN))
	w.pruning = AllPruning
	quiet := node{depth: 1, ply: 3, staticEval: 0}
	if !w.isFutile(quiet, 200) {
		t.Error("eval 0 with alpha 200 at depth 1 should be futile")
	}
	if w.isFutile(quiet, 100) || w.isFutile(node{depth: 3, ply: 3}, 1000) || w.isFutile(node{depth: 1, ply: 3, inCheck: true}, 1000) {
		t.Error("futile within the margin, too deep or in check")
	}
	if w.isFutile(quiet, mateBound) {
		t.Error("futile against a mate score")
	}
}
