package chess

import (
	"fmt"
	"testing"
)

// perftShallowNodeLimit keeps the default `go test` run fast; depths above it
// run only with `-tags perft` (see perft_deep_test.go and `make perft`).
const perftShallowNodeLimit = 1_000_000

// perftCases holds the reference counts from
// https://www.chessprogramming.org/Perft_Results, indexed by depth-1.
var perftCases = []struct {
	position string
	nodes    []int
}{
	{"initial", []int{20, 400, 8902, 197281, 4865609, 119060324}},
	{"kiwipete", []int{48, 2039, 97862, 4085603, 193690690}},
	{"position3", []int{14, 191, 2812, 43238, 674624, 11030083}},
	{"position4", []int{6, 264, 9467, 422333, 15833292}},
	{"position5", []int{44, 1486, 62379, 2103487, 89941194}},
	{"position6", []int{46, 2079, 89890, 3894594, 164075551}},
}

// runPerftCases checks every reference depth whose node count is <= maxNodes.
func runPerftCases(t *testing.T, maxNodes int) {
	for _, tc := range perftCases {
		for depth := 1; depth <= len(tc.nodes); depth++ {
			want := tc.nodes[depth-1]
			if want > maxNodes {
				continue
			}
			t.Run(fmt.Sprintf("%s/depth%d", tc.position, depth), func(t *testing.T) {
				t.Parallel() // the deep cases are independent and CPU-bound
				if got := mustParseFEN(t, perftReferenceFENs[tc.position]).Perft(depth); got != want {
					t.Errorf("Perft(%d) = %d, want %d", depth, got, want)
				}
			})
		}
	}
}

func TestPerft(t *testing.T) {
	runPerftCases(t, perftShallowNodeLimit)
}

func TestPerft_DepthZeroIsOneNode(t *testing.T) {
	if got := NewGame().Perft(0); got != 1 {
		t.Errorf("Perft(0) = %d, want 1", got)
	}
}

func TestDivide_SplitsPerftByFirstMove(t *testing.T) {
	counts := NewGame().Divide(2)
	if len(counts) != 20 {
		t.Fatalf("Divide(2) has %d first moves, want 20", len(counts))
	}
	total := 0
	for _, nodes := range counts {
		total += nodes
	}
	if total != 400 || counts["e2e4"] != 20 || counts["g1f3"] != 20 {
		t.Errorf("Divide(2): total %d (want 400), e2e4 %d, g1f3 %d (want 20 each)", total, counts["e2e4"], counts["g1f3"])
	}
}

func TestDivide_ListsCastlingAndPromotions(t *testing.T) {
	counts := mustParseFEN(t, "r3k3/1P6/8/8/8/8/8/R3K2R w KQq - 0 1").Divide(1)
	for _, move := range []string{"e1g1", "e1c1", "b7b8q", "b7b8n", "b7a8r", "b7a8b"} {
		if counts[move] != 1 {
			t.Errorf("Divide(1)[%q] = %d, want 1 (moves: %v)", move, counts[move], counts)
		}
	}
}

func TestDivide_NonPositiveDepthIsEmpty(t *testing.T) {
	if counts := NewGame().Divide(0); len(counts) != 0 {
		t.Errorf("Divide(0) = %v, want empty", counts)
	}
}
