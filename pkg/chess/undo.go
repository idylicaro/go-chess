package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
)

// gameSnapshot is everything Move changes, captured just before it does.
type gameSnapshot struct {
	board           IBoard
	currentTurn     common.Color
	state           GameState
	drawReason      DrawReason
	enPassantTarget *Position
	castlingRights  CastlingRights
	halfMoveClock   int
	fullMoveNumber  int
	positionHistory map[string]int
}

func (g *Game) snapshot() gameSnapshot {
	history := make(map[string]int, len(g.positionHistory))
	for position, count := range g.positionHistory {
		history[position] = count
	}
	return gameSnapshot{
		board:           g.board.Copy(),
		currentTurn:     g.currentTurn,
		state:           g.state,
		drawReason:      g.drawReason,
		enPassantTarget: g.enPassantTarget, // replaced, never mutated, by Move
		castlingRights:  g.castlingRights,
		halfMoveClock:   g.halfMoveClock,
		fullMoveNumber:  g.fullMoveNumber,
		positionHistory: history,
	}
}

// Undo takes back the last move played with Move, restoring the position,
// castling rights, clocks counters and draw bookkeeping. Games ended by the
// clock, resignation or agreement cannot be undone, since no move ended them.
// On a timed game the increment is withdrawn but spent time is not returned.
//
//	_ = game.Move(move)
//	_ = game.Undo() // back to the position before move
func (g *Game) Undo() error {
	if g.IsOver() && !g.endedOnBoard() {
		return fmt.Errorf("cannot undo, the game ended off the board (%s, %s): %w", g.Result(), g.state, ErrGameOver)
	}
	if len(g.undoStack) == 0 {
		return ErrNoMoveToUndo
	}
	last := len(g.undoStack) - 1
	snapshot := g.undoStack[last]
	g.undoStack = g.undoStack[:last]

	g.board = snapshot.board
	g.currentTurn = snapshot.currentTurn
	g.state = snapshot.state
	g.drawReason = snapshot.drawReason
	g.enPassantTarget = snapshot.enPassantTarget
	g.castlingRights = snapshot.castlingRights
	g.halfMoveClock = snapshot.halfMoveClock
	g.fullMoveNumber = snapshot.fullMoveNumber
	g.positionHistory = snapshot.positionHistory
	g.moveHistory = g.moveHistory[:len(g.moveHistory)-1]
	g.drawOfferedBy = nil
	if g.gameTimer != nil {
		g.gameTimer.Rewind(g.currentTurn)
	}
	return nil
}

// endedOnBoard is true when the position itself ended the game.
func (g *Game) endedOnBoard() bool {
	switch g.state {
	case Checkmate, Stalemate:
		return true
	case Draw:
		return g.drawReason != DrawAgreement && g.drawReason != DrawTimeoutVsInsufficientMaterial
	default:
		return false
	}
}
