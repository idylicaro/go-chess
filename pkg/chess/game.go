package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/debug"
	"github.com/idylicaro/go-chess/internal/timer"
	"github.com/idylicaro/go-chess/internal/uci"
	"time"
)

// Game is a chess game: the board, whose turn it is, the rules that end it
// and, optionally, a clock. Create one with NewGame, NewTimedGame or ParseFEN.
// A Game is not safe for concurrent use.
type Game struct {
	board           *Board
	currentTurn     common.Color
	state           GameState
	enPassantTarget *Position
	castlingRights  CastlingRights   // Castles still available (FEN castling field)
	halfMoveClock   int              // Plies since the last capture or pawn move, for the fifty-move rule
	fullMoveNumber  int              // Starts at 1, incremented after each Black move (FEN)
	initialFEN      string           // Position the game started from, replayed for PGN export
	positionHistory []uint64         // Zobrist keys of the positions reached, for threefold repetition
	hash            uint64           // Zobrist key of the current position, kept by makeMove
	moveHistory     []uci.Move       // For move history tracking
	gameTimer       *timer.GameTimer // Timer control
	drawReason      DrawReason       // Why the game is drawn; DrawNone otherwise
	drawOfferedBy   *common.Color    // Side with a pending draw offer, if any
	undoStack       []gameSnapshot   // One entry per move played with Move, for Undo
}

// fiftyMoveRulePlies is 50 moves per side; halfMoveClock counts plies (#5).
const fiftyMoveRulePlies = 100

// GameState is the coarse state of a game. Result and DrawReason give the
// outcome in PGN terms.
type GameState int

// Values of GameState. Check is a game still in progress with the side to
// move in check; Draw, the timeouts and the resignations are final.
const (
	InProgress GameState = iota
	Check
	Checkmate
	Stalemate
	Draw
	TimeoutWhite
	TimeoutBlack
	ResignedWhite
	ResignedBlack
)

// NewGame creates an untimed game from the standard starting position.
func NewGame() *Game {
	game := &Game{
		board:           newInitialBoard(),
		currentTurn:     common.White,
		state:           InProgress,
		enPassantTarget: nil,
		castlingRights:  AllCastlingRights,
		fullMoveNumber:  1,
		initialFEN:      StartingFEN,
		gameTimer:       nil, // No timer by default
	}
	game.hash = game.computeHash()
	// The starting position counts towards threefold repetition, as it
	// already did for games from ParseFEN.
	game.positionHistory = []uint64{game.hash}
	return game
}

// NewTimedGame creates a game with a chess clock reading time from clock.
// Timeouts are detected by polling the clock on each move, so no timeout
// callback is installed (the old one printed from inside the library, #6).
//
//	game := NewTimedGame(timer.Blitz3Plus2, timer.SystemClock{})
//	_ = game.StartTimer()
func NewTimedGame(timeConfig timer.TimeConfig, clock timer.Clock) *Game {
	game := NewGame()
	game.gameTimer = timer.NewGameTimerWithClock(timeConfig, clock)
	return game
}

// GetCurrentTurn returns the side to move.
func (g *Game) GetCurrentTurn() common.Color {
	return g.currentTurn
}

// GetState returns the current state of the game.
func (g *Game) GetState() GameState {
	return g.state
}

// GetBoard returns the live board, for display. Do not modify it.
func (g *Game) GetBoard() IBoard {
	return g.board
}

// Timer related methods
func (g *Game) StartTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Start(g.currentTurn)
	return nil
}

// GetRemainingTime returns the time left for color, or an error on an untimed game.
func (g *Game) GetRemainingTime(color common.Color) (time.Duration, error) {
	if g.gameTimer == nil {
		return 0, fmt.Errorf("no timer configured for this game")
	}
	return g.gameTimer.GetRemainingTime(color), nil
}

// PauseTimer stops the running clock; it errors on an untimed game.
func (g *Game) PauseTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Pause()
	return nil
}

// ResumeTimer restarts a paused clock; it errors on an untimed game.
func (g *Game) ResumeTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Resume()
	return nil
}

// HasTimer reports whether the game was created with a clock.
func (g *Game) HasTimer() bool {
	return g.gameTimer != nil
}

// Move plays a legal move for the side to move and updates the state. It
// returns an error, leaving the game unchanged, if the move is illegal or the
// game is over. A pawn move to the last rank without Promotion promotes to a queen.
func (g *Game) Move(move uci.Move) error {
	defer func(start time.Time) {
		debug.LogTiming("Game.Move", start)
	}(time.Now())

	moveStr := fmt.Sprintf("(%d,%d)->(%d,%d)", move.FromX, move.FromY, move.ToX, move.ToY)
	debug.NewChessDebugger().LogMove(moveStr, g.currentTurn.String(), g.state)

	if g.IsOver() {
		return fmt.Errorf("move %s rejected, the game is already over (%s): %w", move, g.Result(), ErrGameOver)
	}
	if err := g.rejectIfOutOfTime(); err != nil {
		return err
	}

	snapshot := g.snapshot()
	if err := g.play(move); err != nil {
		return err
	}
	g.undoStack = append(g.undoStack, snapshot)

	debugger := debug.NewChessDebugger()
	debugger.LogGameState(g)
	debugger.LogBoard(g.board)

	return nil
}

// play applies a move without the game-over, clock and undo bookkeeping of
// Move. Search code (PGN replay, SAN suffixes) uses it on throwaway copies.
func (g *Game) play(move uci.Move) error {
	if err := g.requireOwnPieceOn(move); err != nil {
		return err
	}
	legal, ok := g.findGenMove(move)
	if !ok {
		return fmt.Errorf("%w: %s for %v in position %q: the piece cannot go there or it would leave the king in check (see LegalMoves)", ErrIllegalMove, move, g.currentTurn, g.FEN())
	}
	g.makeMove(legal)
	g.moveHistory = append(g.moveHistory, move)
	g.drawOfferedBy = nil
	if g.gameTimer != nil {
		g.gameTimer.SwitchPlayer(g.currentTurn)
	}
	g.updateGameState()
	return nil
}

// requireOwnPieceOn explains why a move from an empty or enemy square fails.
func (g *Game) requireOwnPieceOn(move uci.Move) error {
	piece, err := g.board.GetPiece(move.FromX, move.FromY)
	if err != nil {
		return fmt.Errorf("move %s: %w", move, err)
	}
	if piece == nil {
		return fmt.Errorf("%w: move %s: there is no piece on %s", ErrIllegalMove, move, squareName(move.FromX, move.FromY))
	}
	if piece.GetColor() != g.currentTurn {
		return fmt.Errorf("%w: move %s: the piece on %s is %v's but it is %v's turn", ErrIllegalMove, move, squareName(move.FromX, move.FromY), piece.GetColor(), g.currentTurn)
	}
	return nil
}

// findGenMove returns the legal move matching move, with a missing or
// invalid promotion piece read as a queen.
func (g *Game) findGenMove(move uci.Move) (SearchMove, bool) {
	if g.board.IsValidCoordinates(move.ToX, move.ToY) != nil {
		return SearchMove{}, false
	}
	wanted := SearchMove{
		from:      int8(squareIndex(move.FromX, move.FromY)),
		to:        int8(squareIndex(move.ToX, move.ToY)),
		promotion: g.normalizedPromotion(move),
	}
	var buffer [maxMovesPerPosition]SearchMove
	for _, legal := range g.appendLegalMoves(buffer[:0]) {
		if legal == wanted {
			return legal, true
		}
	}
	return SearchMove{}, false
}

func (g *Game) getOpponentColor() common.Color {
	return opposite(g.currentTurn)
}

func opposite(color common.Color) common.Color {
	if color == common.White {
		return common.Black
	}
	return common.White
}

// rejectIfOutOfTime blocks a move by a side whose clock has run out, before
// anything on the board changes. Previously the clock was only checked after
// the move had been applied, using stale banked time, so it never fired (#6).
func (g *Game) rejectIfOutOfTime() error {
	if state, reason, timedOut := g.timeoutOutcome(); timedOut {
		g.endGame(state, reason)
		return fmt.Errorf("move rejected, the game is over (%s, %s): %w", g.Result(), state, ErrGameOver)
	}
	return nil
}

// timeoutOutcome reports how the game ends if either clock has run out: a
// loss on time for the flagged side, or a draw when the other side could not
// checkmate even with the flagged side's help (FIDE 6.9).
func (g *Game) timeoutOutcome() (GameState, DrawReason, bool) {
	if g.gameTimer == nil {
		return InProgress, DrawNone, false
	}
	for _, flagged := range []common.Color{common.White, common.Black} {
		if !g.gameTimer.IsTimeout(flagged) {
			continue
		}
		if !g.canPossiblyMate(opposite(flagged)) {
			return Draw, DrawTimeoutVsInsufficientMaterial, true
		}
		if flagged == common.White {
			return TimeoutWhite, DrawNone, true
		}
		return TimeoutBlack, DrawNone, true
	}
	return InProgress, DrawNone, false
}

func (g *Game) updateGameState() {
	g.drawReason = DrawNone
	if state, reason, timedOut := g.timeoutOutcome(); timedOut {
		g.state, g.drawReason = state, reason
		return
	}

	hasLegalMove := g.hasLegalMove()

	isInCheck := g.isSideToMoveInCheck()

	if isInCheck {
		if !hasLegalMove {
			g.state = Checkmate
		} else {
			g.state = Check
		}
	} else {
		if !hasLegalMove {
			g.state = Stalemate
			g.drawReason = DrawStalemate
		} else {
			g.state = InProgress
		}
	}
	// Checkmate ends the game before any draw rule can apply to it.
	if g.state == Checkmate {
		return
	}
	if reason := g.automaticDrawReason(); reason != DrawNone {
		g.state, g.drawReason = Draw, reason
	}
}

// automaticDrawReason returns the draw rule that currently applies, or DrawNone.
// It also records the position for the repetition count.
func (g *Game) automaticDrawReason() DrawReason {
	repeated := g.isThreefoldRepetition()
	switch {
	case g.halfMoveClock >= fiftyMoveRulePlies:
		return DrawFiftyMoveRule
	case repeated:
		return DrawThreefoldRepetition
	case g.isInsufficientMaterial():
		return DrawInsufficientMaterial
	}
	return DrawNone
}

// GetDebugInfo implements debug.GameStateProvider
func (g *Game) GetDebugInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["Current Turn"] = g.currentTurn.String()
	info["Game State"] = g.state.String()
	info["Half Move Clock"] = g.halfMoveClock
	info["Move History Length"] = len(g.moveHistory)

	if g.enPassantTarget != nil {
		info["En Passant Target"] = fmt.Sprintf("(%d,%d)", g.enPassantTarget.X, g.enPassantTarget.Y)
	} else {
		info["En Passant Target"] = "None"
	}

	if g.HasTimer() {
		whiteTime, _ := g.GetRemainingTime(common.White)
		blackTime, _ := g.GetRemainingTime(common.Black)
		info["White Time"] = whiteTime.String()
		info["Black Time"] = blackTime.String()
	}

	return info
}

// String names the state, for logs and the CLI.
func (gs GameState) String() string {
	switch gs {
	case InProgress:
		return "InProgress"
	case Check:
		return "Check"
	case Checkmate:
		return "Checkmate"
	case Stalemate:
		return "Stalemate"
	case Draw:
		return "Draw"
	case TimeoutWhite:
		return "TimeoutWhite"
	case TimeoutBlack:
		return "TimeoutBlack"
	case ResignedWhite:
		return "ResignedWhite"
	case ResignedBlack:
		return "ResignedBlack"
	default:
		return "Unknown"
	}
}
