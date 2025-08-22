package chess

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
	"log"
	"os"
	"strings"
	"time"
)

// DebugLevel represents different levels of debugging
type DebugLevel int

const (
	DebugNone DebugLevel = iota
	DebugBasic
	DebugDetailed
	DebugVerbose
)

// DebugConfig holds the debug configuration
type DebugConfig struct {
	Enabled     bool
	Level       DebugLevel
	LogToFile   bool
	LogFilePath string
	ShowTiming  bool
	logger      *log.Logger
}

var Debug = &DebugConfig{
	Enabled:     false,
	Level:       DebugNone,
	LogToFile:   false,
	LogFilePath: "chess_debug.log",
	ShowTiming:  false,
}

// InitDebug initializes the debug system
func InitDebug(level DebugLevel, logToFile bool, logPath string) error {
	Debug.Enabled = level > DebugNone
	Debug.Level = level
	Debug.LogToFile = logToFile
	Debug.ShowTiming = level >= DebugDetailed

	if logToFile {
		Debug.LogFilePath = logPath
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open debug log file: %v", err)
		}
		Debug.logger = log.New(file, "", log.LstdFlags)
	} else {
		Debug.logger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	}

	Debug.Log(DebugBasic, "Debug mode initialized - Level: %v, LogToFile: %v", level, logToFile)
	return nil
}

// Log logs a debug message if the level is appropriate
func (d *DebugConfig) Log(level DebugLevel, format string, args ...interface{}) {
	if !d.Enabled || level > d.Level {
		return
	}

	prefix := ""
	switch level {
	case DebugBasic:
		prefix = "[INFO] "
	case DebugDetailed:
		prefix = "[DETAIL] "
	case DebugVerbose:
		prefix = "[VERBOSE] "
	}

	message := fmt.Sprintf(prefix+format, args...)
	if d.logger != nil {
		d.logger.Println(message)
	}
}

// LogTiming logs timing information for a function
func (d *DebugConfig) LogTiming(functionName string, start time.Time) {
	if !d.Enabled || !d.ShowTiming {
		return
	}

	duration := time.Since(start)
	d.Log(DebugDetailed, "%s took %v", functionName, duration)
}

// LogMove logs move information
func (d *DebugConfig) LogMove(move string, player string, gameState GameState) {
	if !d.Enabled {
		return
	}

	d.Log(DebugBasic, "Move: %s by %s, Game State: %v", move, player, gameState)
}

// LogBoard logs the current board state
func (d *DebugConfig) LogBoard(board IBoard) {
	if !d.Enabled || d.Level < DebugDetailed {
		return
	}

	d.Log(DebugDetailed, "Current board state:")
	boardString := d.formatBoard(board)
	d.Log(DebugDetailed, "\n%s", boardString)
}

// LogLegalMoves logs all legal moves for current player
func (d *DebugConfig) LogLegalMoves(game *Game) {
	if !d.Enabled || d.Level < DebugVerbose {
		return
	}

	allMoves := game.getAllLegalMoves(game.currentTurn)
	d.Log(DebugVerbose, "Legal moves for %v: %d moves", game.currentTurn, len(allMoves))

	if d.Level >= DebugVerbose {
		for i, move := range allMoves {
			d.Log(DebugVerbose, "  Move %d: (%d,%d)", i+1, move.X, move.Y)
		}
	}
}

// LogTimer logs timer information
func (d *DebugConfig) LogTimer(whiteTime, blackTime time.Duration, currentPlayer string) {
	if !d.Enabled || d.Level < DebugDetailed {
		return
	}

	d.Log(DebugDetailed, "Timer - White: %v, Black: %v, Current: %s",
		whiteTime, blackTime, currentPlayer)
}

// LogGameState logs detailed game state information
func (d *DebugConfig) LogGameState(game *Game) {
	if !d.Enabled || d.Level < DebugDetailed {
		return
	}

	d.Log(DebugDetailed, "=== Game State Debug ===")
	d.Log(DebugDetailed, "Current Turn: %v", game.currentTurn)
	d.Log(DebugDetailed, "Game State: %v", game.state)
	d.Log(DebugDetailed, "Half Move Clock: %d", game.halfMoveClock)
	d.Log(DebugDetailed, "Move History Length: %d", len(game.moveHistory))

	if game.enPassantTarget != nil {
		d.Log(DebugDetailed, "En Passant Target: (%d,%d)",
			game.enPassantTarget.X, game.enPassantTarget.Y)
	} else {
		d.Log(DebugDetailed, "En Passant Target: None")
	}

	if game.HasTimer() {
		whiteTime, _ := game.GetRemainingTime(common.White)
		blackTime, _ := game.GetRemainingTime(common.Black)
		d.LogTimer(whiteTime, blackTime, game.currentTurn.String())
	}

	d.Log(DebugDetailed, "========================")
}

// formatBoard creates a string representation of the board for debugging
func (d *DebugConfig) formatBoard(board IBoard) string {
	var sb strings.Builder
	sb.WriteString("  a b c d e f g h\n")

	for y := 0; y < 8; y++ {
		sb.WriteString(fmt.Sprintf("%d ", 8-y))
		for x := 0; x < 8; x++ {
			piece, err := board.GetPiece(x, y)
			if err != nil || piece == nil {
				sb.WriteString(". ")
			} else {
				sb.WriteString(fmt.Sprintf("%s ", d.getPieceSymbol(piece)))
			}
		}
		sb.WriteString(fmt.Sprintf(" %d\n", 8-y))
	}
	sb.WriteString("  a b c d e f g h")

	return sb.String()
}

// getPieceSymbol returns a debug symbol for a piece
func (d *DebugConfig) getPieceSymbol(piece IPiece) string {
	symbols := map[string]string{
		"WhitePawn": "♙", "BlackPawn": "♟",
		"WhiteRook": "♖", "BlackRook": "♜",
		"WhiteKnight": "♘", "BlackKnight": "♞",
		"WhiteBishop": "♗", "BlackBishop": "♝",
		"WhiteQueen": "♕", "BlackQueen": "♛",
		"WhiteKing": "♔", "BlackKing": "♚",
	}

	color := "White"
	if piece.GetColor().String() == "Black" {
		color = "Black"
	}

	key := color + piece.GetType().String()
	if symbol, exists := symbols[key]; exists {
		return symbol
	}
	return "?"
}

// LogError logs error information
func (d *DebugConfig) LogError(context string, err error) {
	if !d.Enabled {
		return
	}

	d.Log(DebugBasic, "ERROR in %s: %v", context, err)
}

// LogPerformanceMetrics logs performance-related metrics
func (d *DebugConfig) LogPerformanceMetrics(game *Game) {
	if !d.Enabled || d.Level < DebugVerbose {
		return
	}

	// Count pieces
	pieceCount := 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := game.board.GetPiece(x, y)
			if piece != nil {
				pieceCount++
			}
		}
	}

	d.Log(DebugVerbose, "Performance Metrics:")
	d.Log(DebugVerbose, "  Pieces on board: %d", pieceCount)
	d.Log(DebugVerbose, "  Move history length: %d", len(game.moveHistory))
	d.Log(DebugVerbose, "  Position history size: %d", len(game.positionHistory))
}

// EnableFromEnvironment enables debug based on environment variables
func EnableFromEnvironment() error {
	debugLevel := os.Getenv("CHESS_DEBUG_LEVEL")
	debugFile := os.Getenv("CHESS_DEBUG_FILE")

	if debugLevel == "" {
		return nil // No debug requested
	}

	var level DebugLevel
	switch strings.ToLower(debugLevel) {
	case "basic", "1":
		level = DebugBasic
	case "detailed", "2":
		level = DebugDetailed
	case "verbose", "3":
		level = DebugVerbose
	default:
		return fmt.Errorf("invalid debug level: %s", debugLevel)
	}

	logToFile := debugFile != ""
	if !logToFile {
		debugFile = "chess_debug.log"
	}

	return InitDebug(level, logToFile, debugFile)
}
