package search

import (
	"context"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/internal/eval"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// steppingClock is a fake time source that moves forward by step on every
// reading, so a timed search ends after a known number of clock checks.
type steppingClock struct {
	current time.Time
	step    time.Duration
}

func (c *steppingClock) now() time.Time {
	c.current = c.current.Add(c.step)
	return c.current
}

func TestBudgetFor(t *testing.T) {
	cases := []struct {
		name       string
		limits     Limits
		side       chess.Color
		soft, hard time.Duration
		limited    bool
	}{
		{"no limits", Limits{}, chess.White, 0, 0, false},
		{"infinite ignores the clock", Limits{Infinite: true, WTime: time.Minute}, chess.White, 0, 0, false},
		{"movetime", Limits{MoveTime: time.Second}, chess.White, 1940 * time.Millisecond, 970 * time.Millisecond, true},
		{"movetime wins over the clock", Limits{MoveTime: time.Second, WTime: time.Minute}, chess.White, 1940 * time.Millisecond, 970 * time.Millisecond, true},
		{"sudden death", Limits{WTime: 61*time.Second + moveOverhead}, chess.White, 61 * time.Second / 30, 4 * (61 * time.Second / 30), true},
		{"reads the side to move's clock", Limits{WTime: time.Hour, BTime: 31*time.Second + moveOverhead, BInc: 4 * time.Second}, chess.Black, 31*time.Second/30 + 3*time.Second, 31 * time.Second / 3, true},
		{"moves to go", Limits{WTime: 10*time.Second + moveOverhead, MovesToGo: 5}, chess.White, 2 * time.Second, 10 * time.Second / 3, true},
		{"last move before the control", Limits{WTime: 3*time.Second + moveOverhead, MovesToGo: 1}, chess.White, 3 * time.Second, 3 * time.Second, true},
		{"flag about to fall", Limits{WTime: 10 * time.Millisecond}, chess.White, time.Millisecond / 30, 4 * (time.Millisecond / 30), true},
	}
	for _, tc := range cases {
		got := budgetFor(tc.limits, tc.side)
		if got.limited != tc.limited || got.soft != tc.soft || got.hard != tc.hard {
			t.Errorf("%s: budget = %+v, want soft %v hard %v limited %v", tc.name, got, tc.soft, tc.hard, tc.limited)
		}
	}
}

func TestTimeBudget_NoNewIterationPastHalfTheSoftBudget(t *testing.T) {
	budget := timeBudget{soft: time.Second, hard: 3 * time.Second, limited: true}
	if !budget.startNextIteration(400*time.Millisecond) || budget.startNextIteration(600*time.Millisecond) {
		t.Error("want a new iteration at 400ms and none at 600ms of a 1s soft budget")
	}
	if !(timeBudget{}).startNextIteration(time.Hour) {
		t.Error("an unlimited budget stopped deepening")
	}
}

func TestSearch_MoveTimeStopsTheSearch(t *testing.T) {
	searcher := New(eval.Evaluate, Options{HashMB: 1})
	clock := &steppingClock{current: time.Unix(0, 0), step: time.Millisecond}
	searcher.now = clock.now
	result := searcher.Search(context.Background(), boardFromFEN(t, kiwipeteFEN), Limits{MoveTime: 100 * time.Millisecond}, nil)
	if result.Best.IsNull() || result.Depth >= maxPly {
		t.Fatalf("timed search: best %s at depth %d, want a move before the depth limit", result.Best, result.Depth)
	}
	if elapsed := clock.current.Sub(time.Unix(0, 0)); elapsed > 200*time.Millisecond {
		t.Errorf("the fake clock reached %v, want the search stopped near 70ms", elapsed)
	}
}

func TestSearch_RealClockMoveTimeIsRespected(t *testing.T) {
	start := time.Now()
	result := New(eval.Evaluate, Options{HashMB: 1}).Search(context.Background(), boardFromFEN(t, kiwipeteFEN), Limits{MoveTime: 150 * time.Millisecond}, nil)
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond || result.Best.IsNull() {
		t.Errorf("movetime 150ms took %v and returned %s", elapsed, result.Best)
	}
}
