package search

import (
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Time management turns the UCI clock fields into two budgets (#41). The
// soft budget is checked between iterations: past half of it, the next
// iteration would most likely not finish, so the search stops. The hard
// budget is checked inside the tree and is never exceeded by more than a
// poll interval.
const (
	// moveOverhead is kept back from every budget for the latency of the
	// process and the GUI, so the engine does not lose on time.
	moveOverhead = 30 * time.Millisecond
	// defaultMovesToGo is how many more moves a sudden-death clock is
	// assumed to have to last.
	defaultMovesToGo = 30
	// minimumBudget leaves time for depth 1 even with almost no clock.
	minimumBudget = time.Millisecond
	// hardBudgetFactor lets one move take a few times its share when an
	// iteration runs long, but never more than a third of the clock.
	hardBudgetFactor = 4
	hardClockShare   = 3
)

// timeBudget is how long a search may take; limited false means no limit.
type timeBudget struct {
	soft, hard time.Duration
	limited    bool
}

// budgetFor reads the limits of the side to move. MoveTime wins over the
// clocks, and Infinite over both.
//
//	budgetFor(Limits{WTime: time.Minute}, chess.White) // soft 2s, hard 8s
func budgetFor(limits Limits, side chess.Color) timeBudget {
	if limits.Infinite {
		return timeBudget{}
	}
	if limits.MoveTime > 0 {
		fixed := max(limits.MoveTime-moveOverhead, minimumBudget)
		return timeBudget{soft: 2 * fixed, hard: fixed, limited: true}
	}
	remaining, increment := clockOf(limits, side)
	if remaining <= 0 {
		return timeBudget{}
	}
	return clockBudget(remaining, increment, limits.MovesToGo)
}

// clockBudget shares the remaining time among the moves still to play.
func clockBudget(remaining, increment time.Duration, movesToGo int) timeBudget {
	if movesToGo <= 0 {
		movesToGo = defaultMovesToGo
	}
	usable := max(remaining-moveOverhead, minimumBudget)
	share := min(usable/time.Duration(movesToGo)+increment*3/4, usable)
	hard := min(share*hardBudgetFactor, usable/hardClockShare)
	return timeBudget{soft: share, hard: max(hard, min(share, usable)), limited: true}
}

func clockOf(limits Limits, side chess.Color) (remaining, increment time.Duration) {
	if side == chess.White {
		return limits.WTime, limits.WInc
	}
	return limits.BTime, limits.BInc
}

// startNextIteration reports whether another iteration is worth starting
// after elapsed: it costs about as much as all the previous ones together.
func (b timeBudget) startNextIteration(elapsed time.Duration) bool {
	return !b.limited || elapsed < b.soft/2
}
