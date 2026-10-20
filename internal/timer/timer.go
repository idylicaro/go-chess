package timer

import (
	common "idylicaro/go-chess/internal/common"
	"sync"
	"time"
)

type TimeConfig struct {
	InitialTime time.Duration
	Increment   time.Duration
}

// Constants for formats
var (
	Blitz3Plus2       = TimeConfig{3 * time.Minute, 2 * time.Second}
	Rapid10Plus0      = TimeConfig{10 * time.Minute, 0}
	Classical90Plus30 = TimeConfig{90 * time.Minute, 30 * time.Second}
)

// PlayerTimer holds one side's banked time. Time spent on the current turn
// is not subtracted until the turn ends or the clock is paused.
type PlayerTimer struct {
	remaining time.Duration
	increment time.Duration
}

// GameTimer is a two-player chess clock. Only the side to move runs, and
// only while the timer is running (started and not paused).
type GameTimer struct {
	white         *PlayerTimer
	black         *PlayerTimer
	activeColor   common.Color
	running       bool
	turnStartedAt time.Time
	clock         Clock
	mu            sync.Mutex
	onTimeout     func(common.Color) // callback, always invoked without holding mu
}

// NewGameTimerWithClock creates a stopped clock that reads time from clock.
//
//	gt := NewGameTimerWithClock(Blitz3Plus2, SystemClock{})
//	gt.Start(common.White)
func NewGameTimerWithClock(config TimeConfig, clock Clock) *GameTimer {
	return &GameTimer{
		white: &PlayerTimer{remaining: config.InitialTime, increment: config.Increment},
		black: &PlayerTimer{remaining: config.InitialTime, increment: config.Increment},
		clock: clock,
	}
}

// NewGameTimer creates a stopped clock that reads the wall clock.
//
// Deprecated: use NewGameTimerWithClock(config, SystemClock{}).
func NewGameTimer(config TimeConfig) *GameTimer {
	return NewGameTimerWithClock(config, SystemClock{})
}

// Start runs color's clock from now.
func (gt *GameTimer) Start(color common.Color) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.activeColor = color
	gt.running = true
	gt.turnStartedAt = gt.clock.Now()
}

// SwitchPlayer ends the active side's turn, credits its increment unless it
// has already run out of time, and starts newColor's clock. The timeout
// callback fires for a side that ran out (#6).
func (gt *GameTimer) SwitchPlayer(newColor common.Color) {
	flaggedColor, flagged, callback := gt.handOver(newColor)
	if flagged && callback != nil {
		callback(flaggedColor)
	}
}

// handOver does the locked part of SwitchPlayer and reports who, if anyone,
// ran out of time, so the callback can run after the lock is released.
func (gt *GameTimer) handOver(newColor common.Color) (common.Color, bool, func(common.Color)) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.chargeElapsedLocked()
	previousColor := gt.activeColor
	previous := gt.playerLocked(previousColor)
	flagged := previous.remaining <= 0
	if !flagged {
		previous.remaining += previous.increment
	}
	gt.activeColor = newColor
	gt.turnStartedAt = gt.clock.Now()
	return previousColor, flagged, gt.onTimeout
}

// GetRemainingTime returns color's time left, including the running turn.
// It is negative once the side has run out.
func (gt *GameTimer) GetRemainingTime(color common.Color) time.Duration {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	return gt.remainingLocked(color)
}

// IsTimeout reports whether color has run out of time, counting the turn in progress.
func (gt *GameTimer) IsTimeout(color common.Color) bool {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	return gt.remainingLocked(color) <= 0
}

func (gt *GameTimer) SetTimeoutCallback(callback func(common.Color)) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.onTimeout = callback
}

// Pause freezes both clocks; time spent so far in the turn is kept.
func (gt *GameTimer) Pause() {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.chargeElapsedLocked()
	gt.running = false
}

// Resume restarts the active side's clock after Pause.
func (gt *GameTimer) Resume() {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	if gt.running {
		return
	}
	gt.running = true
	gt.turnStartedAt = gt.clock.Now()
}

// chargeElapsedLocked moves the running turn's elapsed time into the active
// side's banked time. Callers must hold mu.
func (gt *GameTimer) chargeElapsedLocked() {
	if !gt.running {
		return
	}
	now := gt.clock.Now()
	gt.playerLocked(gt.activeColor).remaining -= now.Sub(gt.turnStartedAt)
	gt.turnStartedAt = now
}

func (gt *GameTimer) remainingLocked(color common.Color) time.Duration {
	remaining := gt.playerLocked(color).remaining
	if gt.running && color == gt.activeColor {
		remaining -= gt.clock.Now().Sub(gt.turnStartedAt)
	}
	return remaining
}

func (gt *GameTimer) playerLocked(color common.Color) *PlayerTimer {
	if color == common.White {
		return gt.white
	}
	return gt.black
}
