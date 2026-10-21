package chess

import (
	"errors"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/timer"
	"github.com/idylicaro/go-chess/internal/uci"
)

// Types of the supporting packages under internal/, re-exported so callers
// outside this module can name everything the Game API takes and returns.
type (
	Color      = common.Color
	PieceType  = common.PieceType
	Move       = uci.Move
	TimeConfig = timer.TimeConfig
	Clock      = timer.Clock
	// SystemClock reads the wall clock.
	SystemClock = timer.SystemClock
)

const (
	White = common.White
	Black = common.Black
)

// Pieces a pawn can promote to, for Move.Promotion. The zero value means no
// promotion (and promotes to a queen if the move reaches the last rank).
const (
	PromoteToQueen  = common.Queen
	PromoteToRook   = common.Rook
	PromoteToBishop = common.Bishop
	PromoteToKnight = common.Knight
)

// Common time controls for NewTimedGame.
var (
	Blitz3Plus2       = timer.Blitz3Plus2
	Rapid10Plus0      = timer.Rapid10Plus0
	Classical90Plus30 = timer.Classical90Plus30
)

// ParseMove reads a move in UCI notation.
//
//	move, err := ParseMove("e7e8q")
func ParseMove(uciMove string) (Move, error) {
	return uci.ParseMove(uciMove)
}

// ErrGameOver is returned (wrapped) by every action that needs a game in progress.
var ErrGameOver = errors.New("game is over")

// ErrNoMoveToUndo is returned by Undo when there is nothing to take back.
var ErrNoMoveToUndo = errors.New("no move to undo")

// Result is the outcome of a game, in PGN terms.
type Result int

// Results of a game.
const (
	ResultOngoing Result = iota
	ResultWhiteWins
	ResultBlackWins
	ResultDraw
)

// String is the PGN result token: "*", "1-0", "0-1" or "1/2-1/2".
func (r Result) String() string {
	switch r {
	case ResultWhiteWins:
		return "1-0"
	case ResultBlackWins:
		return "0-1"
	case ResultDraw:
		return "1/2-1/2"
	default:
		return "*"
	}
}

// DrawReason says why a game ended in a draw.
type DrawReason int

// Reasons a game can end in a draw.
const (
	DrawNone DrawReason = iota
	DrawStalemate
	DrawFiftyMoveRule
	DrawThreefoldRepetition
	DrawInsufficientMaterial
	DrawAgreement
	// DrawTimeoutVsInsufficientMaterial: a clock ran out, but the opponent
	// could not have checkmated with the pieces left, so it is not a loss.
	DrawTimeoutVsInsufficientMaterial
)

// String names the draw reason; DrawNone is "None".
func (r DrawReason) String() string {
	switch r {
	case DrawStalemate:
		return "Stalemate"
	case DrawFiftyMoveRule:
		return "FiftyMoveRule"
	case DrawThreefoldRepetition:
		return "ThreefoldRepetition"
	case DrawInsufficientMaterial:
		return "InsufficientMaterial"
	case DrawAgreement:
		return "Agreement"
	case DrawTimeoutVsInsufficientMaterial:
		return "TimeoutVsInsufficientMaterial"
	default:
		return "None"
	}
}
