package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
)

func TestSearch_QuiescenceSeesTheRecapture(t *testing.T) {
	// Qxd5 wins a pawn at depth 1 but loses the queen to exd5.
	board := boardFromFEN(t, "4k3/8/4p3/3p4/8/8/8/3QK3 w - - 0 1")
	result := New(eval.Evaluate).Search(context.Background(), board, Limits{Depth: 1}, nil)
	if result.Best.String() == "d1d5" {
		t.Errorf("depth-1 search plays Qxd5 (score %d), want the recapture seen", result.Score)
	}
}

func TestQuiesce_QuietPositionIsTheStaticEvaluation(t *testing.T) {
	board := boardFromFEN(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
	w := newWorker(board, eval.Evaluate, newStopper(context.Background(), 0))
	if got, want := w.quiesce(0, -infinity, infinity), eval.Evaluate(board); got != want {
		t.Errorf("quiesce = %d, want the static evaluation %d", got, want)
	}
}

func TestQuiesce_FindsAMateAmongTheEvasions(t *testing.T) {
	// Black is checkmated: quiescence must not stand pat in check.
	board := boardFromFEN(t, "R5k1/5ppp/8/8/8/8/8/6K1 b - - 0 1")
	w := newWorker(board, eval.Evaluate, newStopper(context.Background(), 0))
	if got := w.quiesce(0, -infinity, infinity); got != matedIn(0) {
		t.Errorf("quiesce of a mated position = %d, want %d", got, matedIn(0))
	}
}

func TestQuiesce_WinsAnUndefendedPiece(t *testing.T) {
	board := boardFromFEN(t, "4k3/8/8/3n4/8/8/8/3QK3 w - - 0 1")
	w := newWorker(board, eval.Evaluate, newStopper(context.Background(), 0))
	if got, static := w.quiesce(0, -infinity, infinity), eval.Evaluate(board); got < static+250 {
		t.Errorf("quiesce = %d with static %d, want Qxd5 counted", got, static)
	}
}
