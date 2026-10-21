package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
)

// benchmarkFENs are the positions of BenchmarkSearch:
// an opening, a sharp middlegame and an endgame.
var benchmarkFENs = map[string]string{
	"start":    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	"kiwipete": kiwipeteFEN,
	"endgame":  "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
}

// benchmarkDepth is deep enough that move ordering and pruning show.
const benchmarkDepth = 5

// BenchmarkSearch is the reference measure of the search: nodes searched to
// a fixed depth (lower is better pruning) and nodes per second.
//
//	go test ./internal/search -run '^$' -bench Search
func BenchmarkSearch(b *testing.B) {
	for name, fen := range benchmarkFENs {
		b.Run(name, func(b *testing.B) {
			board := boardFromFEN(b, fen)
			var nodes int64
			for range b.N {
				nodes = New(eval.Evaluate).Search(context.Background(), board, Limits{Depth: benchmarkDepth}, nil).Nodes
			}
			b.ReportMetric(float64(nodes), "nodes")
			b.ReportMetric(float64(nodes)*float64(b.N)/b.Elapsed().Seconds(), "nodes/s")
		})
	}
}
