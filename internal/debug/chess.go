package debug

import (
	"fmt"
	"strings"
	"time"
)

// ChessDebugger provides chess-specific debugging functionality
type ChessDebugger struct {
	*DebugConfig
}

// NewChessDebugger creates a new chess-specific debugger
func NewChessDebugger() *ChessDebugger {
	return &ChessDebugger{Debug}
}

// LogMove logs move information - generic interface
func (cd *ChessDebugger) LogMove(move string, player string, gameState interface{}) {
	if !cd.Enabled {
		return
	}

	cd.Log(DebugBasic, "Move: %s by %s, Game State: %v", move, player, gameState)
}

// LogTimer logs timer information
func (cd *ChessDebugger) LogTimer(whiteTime, blackTime time.Duration, currentPlayer string) {
	if !cd.Enabled || cd.Level < DebugDetailed {
		return
	}

	cd.Log(DebugDetailed, "Timer - White: %v, Black: %v, Current: %s",
		whiteTime, blackTime, currentPlayer)
}

// LogBoard logs the current board state using a generic board interface
func (cd *ChessDebugger) LogBoard(board BoardFormatter) {
	if !cd.Enabled || cd.Level < DebugDetailed {
		return
	}

	cd.Log(DebugDetailed, "Current board state:")
	boardString := board.FormatForDebug()
	cd.Log(DebugDetailed, "\n%s", boardString)
}

// LogGameState logs detailed game state information
func (cd *ChessDebugger) LogGameState(gameState GameStateProvider) {
	if !cd.Enabled || cd.Level < DebugDetailed {
		return
	}

	cd.Log(DebugDetailed, "=== Game State Debug ===")
	state := gameState.GetDebugInfo()

	for key, value := range state {
		cd.Log(DebugDetailed, "%s: %v", key, value)
	}

	cd.Log(DebugDetailed, "========================")
}

// LogPerformanceMetrics logs performance-related metrics
func (cd *ChessDebugger) LogPerformanceMetrics(metrics map[string]interface{}) {
	if !cd.Enabled || cd.Level < DebugVerbose {
		return
	}

	cd.Log(DebugVerbose, "Performance Metrics:")
	for key, value := range metrics {
		cd.Log(DebugVerbose, "  %s: %v", key, value)
	}
}

// Interfaces for decoupling
type BoardFormatter interface {
	FormatForDebug() string
}

type GameStateProvider interface {
	GetDebugInfo() map[string]interface{}
}

// FormatBoardGeneric creates a generic board string representation
func FormatBoardGeneric(pieces [][]string) string {
	var sb strings.Builder
	sb.WriteString("  a b c d e f g h\n")

	for y := 0; y < 8; y++ {
		sb.WriteString(fmt.Sprintf("%d ", 8-y))
		for x := 0; x < 8; x++ {
			if len(pieces) > y && len(pieces[y]) > x && pieces[y][x] != "" {
				sb.WriteString(fmt.Sprintf("%s ", pieces[y][x]))
			} else {
				sb.WriteString(". ")
			}
		}
		sb.WriteString(fmt.Sprintf(" %d\n", 8-y))
	}
	sb.WriteString("  a b c d e f g h")

	return sb.String()
}
