package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
)

func TestSearch_SeveralThreadsFindTheSameTactics(t *testing.T) {
	for _, threads := range []int{2, 4} {
		searcher := New(eval.Evaluate, Options{HashMB: 4, Pruning: AllPruning, Threads: threads})
		result := searcher.Search(context.Background(), boardFromFEN(t, scholarsMateFEN), Limits{Depth: 5}, nil)
		if result.Best.String() != "h5f7" {
			t.Errorf("%d threads: best %s, want h5f7", threads, result.Best)
		}
		result = searcher.Search(context.Background(), boardFromFEN(t, hangingQueenFEN), Limits{Depth: 6}, nil)
		if result.Best.String() != "e4d5" {
			t.Errorf("%d threads: best %s, want e4d5", threads, result.Best)
		}
	}
}

func TestSearch_HelpersStopWithTheMainWorkerAndCountTheirNodes(t *testing.T) {
	single := New(eval.Evaluate, Options{HashMB: 4, Pruning: AllPruning, Threads: 1})
	parallel := New(eval.Evaluate, Options{HashMB: 4, Pruning: AllPruning, Threads: 4})
	board := boardFromFEN(t, kiwipeteFEN)
	one := single.Search(context.Background(), board, Limits{Depth: 6}, nil)
	four := parallel.Search(context.Background(), board, Limits{Depth: 6}, nil)
	if four.Depth != 6 || four.Best.IsNull() {
		t.Fatalf("4 threads: depth %d best %s, want depth 6 and a move", four.Depth, four.Best)
	}
	if four.Nodes <= one.Nodes/2 {
		t.Errorf("4 threads counted %d nodes against %d for one, want the helpers' nodes included", four.Nodes, one.Nodes)
	}
}

func TestSearch_NodeLimitCountsEveryThread(t *testing.T) {
	searcher := New(eval.Evaluate, Options{HashMB: 4, Pruning: AllPruning, Threads: 4})
	result := searcher.Search(context.Background(), boardFromFEN(t, kiwipeteFEN), Limits{Nodes: 200000}, nil)
	if result.Nodes > 200000+8*pollInterval {
		t.Errorf("node-limited search with 4 threads took %d nodes, want about 200000", result.Nodes)
	}
}
