package chess

import (
	"fmt"
	"testing"
)

func TestPerftParallel_MatchesPerft(t *testing.T) {
	for name, fen := range perftReferenceFENs {
		game := mustParseFEN(t, fen)
		want := game.Perft(3)
		for _, workers := range []int{0, 1, 2, 7, 64} {
			if got := game.PerftParallel(3, workers); got != want {
				t.Errorf("%s: PerftParallel(3, %d) = %d, want %d", name, workers, got, want)
			}
		}
	}
}

func TestPerftParallel_ShallowDepths(t *testing.T) {
	for depth, want := range []int{1, 20} {
		if got := NewGame().PerftParallel(depth, 4); got != want {
			t.Errorf("PerftParallel(%d, 4) = %d, want %d", depth, got, want)
		}
	}
}

func TestPerftParallel_LeavesTheGameUntouched(t *testing.T) {
	game := mustParseFEN(t, perftReferenceFENs["kiwipete"])
	before := game.FEN()
	game.PerftParallel(3, 4)
	if after := game.FEN(); after != before {
		t.Errorf("PerftParallel changed the game from %q to %q", before, after)
	}
}

func TestRootMoveQueue_HandsOutEachMoveOnce(t *testing.T) {
	queue := &rootMoveQueue{moves: []SearchMove{{from: 1}, {from: 2}}}
	first, okFirst := queue.take()
	second, okSecond := queue.take()
	_, okThird := queue.take()
	if !okFirst || !okSecond || okThird || first.from != 1 || second.from != 2 {
		t.Errorf("take() gave %v %v, %v %v, then ok=%v; want moves 1 and 2, then empty", first, okFirst, second, okSecond, okThird)
	}
}

// BenchmarkPerftParallel measures the speedup by worker count for lab #35:
// compare nodes/s across the workers=N sub-benchmarks.
func BenchmarkPerftParallel(b *testing.B) {
	for _, position := range []string{"initial", "kiwipete"} {
		depth := map[string]int{"initial": 5, "kiwipete": 4}[position]
		game, err := ParseFEN(perftReferenceFENs[position])
		if err != nil {
			b.Fatalf("ParseFEN(%q): %v", position, err)
		}
		for _, workers := range []int{1, 2, 4, 8, 16} {
			b.Run(fmt.Sprintf("%s/workers=%d", position, workers), func(b *testing.B) {
				nodes := 0
				for i := 0; i < b.N; i++ {
					nodes += game.PerftParallel(depth, workers)
				}
				b.ReportMetric(float64(nodes)/b.Elapsed().Seconds(), "nodes/s")
			})
		}
	}
}
