package timer_test

import (
	"testing"
	"time"

	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/timer/timertest"
)

func newTestTimer(initial, increment time.Duration) (*timer.GameTimer, *timertest.FakeClock) {
	clock := timertest.NewFakeClock()
	config := timer.TimeConfig{InitialTime: initial, Increment: increment}
	return timer.NewGameTimerWithClock(config, clock), clock
}

func assertRemaining(t *testing.T, gt *timer.GameTimer, color common.Color, want time.Duration) {
	t.Helper()
	if got := gt.GetRemainingTime(color); got != want {
		t.Errorf("remaining time for %v = %v, want %v", color, got, want)
	}
}

// Full flow with increment: elapsed time is charged to the side to move and
// the increment is credited when it hands the clock over.
func TestGameTimer_FullFlowWithIncrement(t *testing.T) {
	gt, clock := newTestTimer(10*time.Second, time.Second)
	gt.Start(common.White)

	clock.Advance(2 * time.Second)
	assertRemaining(t, gt, common.White, 8*time.Second)

	gt.SwitchPlayer(common.Black)
	assertRemaining(t, gt, common.White, 9*time.Second)

	clock.Advance(3 * time.Second)
	assertRemaining(t, gt, common.Black, 7*time.Second)
	assertRemaining(t, gt, common.White, 9*time.Second) // idle clock does not run

	gt.SwitchPlayer(common.White)
	assertRemaining(t, gt, common.Black, 8*time.Second)
}

func TestGameTimer_RemainingBeforeStart(t *testing.T) {
	gt, clock := newTestTimer(10*time.Second, 0)
	clock.Advance(5 * time.Second)
	assertRemaining(t, gt, common.White, 10*time.Second)
	assertRemaining(t, gt, common.Black, 10*time.Second)
}

// Regression for #6: IsTimeout ignored the running clock, so a player who
// had run out of time was still allowed to move.
func TestGameTimer_IsTimeoutWhileClockRuns(t *testing.T) {
	gt, clock := newTestTimer(time.Second, 0)
	gt.Start(common.White)

	clock.Advance(999 * time.Millisecond)
	if gt.IsTimeout(common.White) {
		t.Fatal("IsTimeout(White) = true with 1ms left")
	}
	clock.Advance(time.Millisecond)
	if !gt.IsTimeout(common.White) {
		t.Error("IsTimeout(White) = false after the whole second elapsed")
	}
	if gt.IsTimeout(common.Black) {
		t.Error("IsTimeout(Black) = true although Black's clock never ran")
	}
}

// Regression for #6: Pause only set a flag and the clock kept running.
func TestGameTimer_PauseFreezesAndResumeContinues(t *testing.T) {
	gt, clock := newTestTimer(10*time.Second, 0)
	gt.Start(common.White)
	clock.Advance(2 * time.Second)

	gt.Pause()
	clock.Advance(time.Minute)
	assertRemaining(t, gt, common.White, 8*time.Second)

	gt.Resume()
	clock.Advance(3 * time.Second)
	assertRemaining(t, gt, common.White, 5*time.Second)
}

func TestGameTimer_PauseTwiceDoesNotDoubleCharge(t *testing.T) {
	gt, clock := newTestTimer(10*time.Second, 0)
	gt.Start(common.White)
	clock.Advance(2 * time.Second)
	gt.Pause()
	gt.Pause()
	assertRemaining(t, gt, common.White, 8*time.Second)
}

func TestGameTimer_TimeoutCallbackOnSwitch(t *testing.T) {
	gt, clock := newTestTimer(time.Second, 5*time.Second)
	var flagged []common.Color
	gt.SetTimeoutCallback(func(color common.Color) { flagged = append(flagged, color) })

	gt.Start(common.White)
	clock.Advance(2 * time.Second)
	gt.SwitchPlayer(common.Black)

	if len(flagged) != 1 || flagged[0] != common.White {
		t.Fatalf("timeout callback calls = %v, want [White]", flagged)
	}
	if !gt.IsTimeout(common.White) {
		t.Error("a flagged player must not be rescued by the increment")
	}
}

// Regression for #6: the callback ran while the timer mutex was held, so
// calling back into the timer from it deadlocked.
func TestGameTimer_TimeoutCallbackCanUseTimer(t *testing.T) {
	gt, clock := newTestTimer(time.Second, 0)
	done := make(chan time.Duration, 1)
	gt.SetTimeoutCallback(func(color common.Color) { done <- gt.GetRemainingTime(color) })

	gt.Start(common.White)
	clock.Advance(2 * time.Second)
	go gt.SwitchPlayer(common.Black)

	select {
	case remaining := <-done:
		if remaining != -time.Second {
			t.Errorf("remaining seen by callback = %v, want -1s", remaining)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout callback deadlocked calling GetRemainingTime")
	}
}

func TestGameTimer_RewindReturnsTurnWithoutIncrement(t *testing.T) {
	gt, clock := newTestTimer(10*time.Second, 2*time.Second)
	gt.Start(common.White)
	clock.Advance(time.Second)
	gt.SwitchPlayer(common.Black)
	clock.Advance(3 * time.Second)

	gt.Rewind(common.White)

	if got := gt.GetRemainingTime(common.White); got != 9*time.Second {
		t.Errorf("White = %v, want 9s (10s - 1s spent, increment withdrawn)", got)
	}
	if got := gt.GetRemainingTime(common.Black); got != 7*time.Second {
		t.Errorf("Black = %v, want 7s (time spent before the take-back is not returned)", got)
	}
	clock.Advance(time.Second)
	if got := gt.GetRemainingTime(common.White); got != 8*time.Second {
		t.Errorf("White = %v after 1s, want 8s: the clock should run for White again", got)
	}
	if got := gt.GetRemainingTime(common.Black); got != 7*time.Second {
		t.Errorf("Black = %v, want 7s: its clock should be stopped", got)
	}
}
