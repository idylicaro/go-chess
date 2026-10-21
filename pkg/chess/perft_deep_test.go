//go:build perft

package chess

import (
	"math"
	"testing"
)

// TestPerftDeep runs every reference depth. It takes tens of seconds, so it
// is behind the perft build tag: `make perft` or `go test -tags perft`.
func TestPerftDeep(t *testing.T) {
	runPerftCases(t, math.MaxInt)
}
