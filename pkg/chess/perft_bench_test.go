package chess

import "testing"

// BenchmarkPerft is the reference speed measure for move generation (#12):
// each sub-benchmark reports nodes/s, so `benchstat` can compare runs before
// and after a change to the generator.
func BenchmarkPerft(b *testing.B) {
	cases := []struct {
		position string
		depth    int
	}{
		{"initial", 4},
		{"kiwipete", 3},
		{"position3", 4},
	}
	for _, tc := range cases {
		b.Run(tc.position, func(b *testing.B) {
			game, err := ParseFEN(perftReferenceFENs[tc.position])
			if err != nil {
				b.Fatalf("ParseFEN(%q): %v", tc.position, err)
			}
			nodes := 0
			for i := 0; i < b.N; i++ {
				nodes += game.Perft(tc.depth)
			}
			b.ReportMetric(float64(nodes)/b.Elapsed().Seconds(), "nodes/s")
		})
	}
}
