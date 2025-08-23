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

type PlayerTimer struct {
	remaining time.Duration
	increment time.Duration
	active    bool
	lastStart time.Time
}

type GameTimer struct {
	white       *PlayerTimer
	black       *PlayerTimer
	activeColor common.Color
	paused      bool
	mu          sync.Mutex
	onTimeout   func(common.Color) // callback
}

func NewGameTimer(config TimeConfig) *GameTimer {
	return &GameTimer{
		white: &PlayerTimer{
			remaining: config.InitialTime,
			increment: config.Increment,
		},
		black: &PlayerTimer{
			remaining: config.InitialTime,
			increment: config.Increment,
		},
		paused: true,
	}
}

func (gt *GameTimer) Start(color common.Color) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	gt.activeColor = color
	gt.paused = false

	var player *PlayerTimer
	if color == common.White {
		player = gt.white
	} else {
		player = gt.black
	}

	player.active = true
	player.lastStart = time.Now()
}

func (gt *GameTimer) SwitchPlayer(newColor common.Color) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	var prevPlayer, nextPlayer *PlayerTimer
	if gt.activeColor == common.White {
		prevPlayer = gt.white
		nextPlayer = gt.black
	} else {
		prevPlayer = gt.black
		nextPlayer = gt.white
	}

	// Stop the previous player's clock
	timeElapsed := time.Since(prevPlayer.lastStart)
	prevPlayer.active = false
	prevPlayer.remaining -= timeElapsed
	prevPlayer.remaining += prevPlayer.increment

	if prevPlayer.remaining < 0 && gt.onTimeout != nil {
		gt.onTimeout(gt.activeColor)
	}

	nextPlayer.active = true
	nextPlayer.lastStart = time.Now()

	gt.activeColor = newColor
}
func (gt *GameTimer) GetRemainingTime(color common.Color) time.Duration {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	var player *PlayerTimer
	if color == common.White {
		player = gt.white
	} else {
		player = gt.black
	}

	if player.active {
		return player.remaining - time.Since(player.lastStart)
	}

	return player.remaining
}

func (gt *GameTimer) IsTimeout(color common.Color) bool {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	if color == common.White {
		return gt.white.remaining <= 0
	}
	return gt.black.remaining <= 0
}

func (gt *GameTimer) SetTimeoutCallback(callback func(common.Color)) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.onTimeout = callback
}

func (gt *GameTimer) Pause() {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.paused = true
}

func (gt *GameTimer) Resume() {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	gt.paused = false
}
