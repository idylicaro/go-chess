//go:build perft

package chess

import (
	"math"
	"testing"
)

// TestPerftDeep runs every reference depth, about 600M nodes. It takes ~15s
// on 8 cores (the cases run in parallel), so it is behind the perft build tag:
// `make perft` or `go test -tags perft`.
func TestPerftDeep(t *testing.T) {
	runPerftCases(t, math.MaxInt)
}
