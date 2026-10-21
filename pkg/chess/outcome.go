package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
)

// IsOver reports whether the game has ended, by the board, the clock,
// resignation or agreement.
func (g *Game) IsOver() bool {
	switch g.state {
	case InProgress, Check:
		return false
	default:
		return true
	}
}

// Result is the game's outcome: ResultOngoing while it is being played.
//
//	game.Result().String() // "1-0"
func (g *Game) Result() Result {
	switch g.state {
	case Checkmate:
		return winnerResult(opposite(g.currentTurn))
	case TimeoutWhite, ResignedWhite:
		return ResultBlackWins
	case TimeoutBlack, ResignedBlack:
		return ResultWhiteWins
	case Stalemate, Draw:
		return ResultDraw
	default:
		return ResultOngoing
	}
}

func winnerResult(winner common.Color) Result {
	if winner == common.White {
		return ResultWhiteWins
	}
	return ResultBlackWins
}

// DrawReason is why the game was drawn, or DrawNone if it was not.
func (g *Game) DrawReason() DrawReason {
	return g.drawReason
}

// LegalMoves lists every legal move of the side to move, expanding a
// promotion into one move per piece. It is empty once the game is over.
//
//	NewGame().LegalMoves() // 20 moves
func (g *Game) LegalMoves() []Move {
	if g.IsOver() {
		return nil
	}
	return g.legalUCIMoves()
}

// Resign ends the game with a win for the opponent of color. Either side may
// resign at any time, not only on its turn.
func (g *Game) Resign(color Color) error {
	if err := g.requireInProgress("resign"); err != nil {
		return err
	}
	if color == common.White {
		g.endGame(ResignedWhite, DrawNone)
	} else {
		g.endGame(ResignedBlack, DrawNone)
	}
	return nil
}

// OfferDraw records a draw offer from color. It stands until the opponent
// answers with AcceptDraw or DeclineDraw, or plays a move, which declines it.
func (g *Game) OfferDraw(color Color) error {
	if err := g.requireInProgress("offer a draw"); err != nil {
		return err
	}
	if g.drawOfferedBy != nil && *g.drawOfferedBy == color {
		return fmt.Errorf("%v has already offered a draw", color)
	}
	g.drawOfferedBy = &color
	return nil
}

// PendingDrawOffer reports which side has an unanswered draw offer.
func (g *Game) PendingDrawOffer() (by Color, pending bool) {
	if g.drawOfferedBy == nil {
		return 0, false
	}
	return *g.drawOfferedBy, true
}

// AcceptDraw ends the game in a draw if the opponent of color offered one.
func (g *Game) AcceptDraw(color Color) error {
	if err := g.requireOfferToAnswer(color, "accept"); err != nil {
		return err
	}
	g.endGame(Draw, DrawAgreement)
	return nil
}

// DeclineDraw rejects the opponent's pending offer; the game goes on.
func (g *Game) DeclineDraw(color Color) error {
	if err := g.requireOfferToAnswer(color, "decline"); err != nil {
		return err
	}
	g.drawOfferedBy = nil
	return nil
}

func (g *Game) requireOfferToAnswer(color Color, verb string) error {
	if err := g.requireInProgress(verb + " a draw"); err != nil {
		return err
	}
	if g.drawOfferedBy == nil || *g.drawOfferedBy == color {
		return fmt.Errorf("cannot %s a draw: %v has no draw offer from %v to answer", verb, color, opposite(color))
	}
	return nil
}

func (g *Game) requireInProgress(action string) error {
	if g.IsOver() {
		return fmt.Errorf("cannot %s, the game is already over (%s): %w", action, g.Result(), ErrGameOver)
	}
	return nil
}

// endGame finishes the game outside the normal move flow and stops the clock.
func (g *Game) endGame(state GameState, reason DrawReason) {
	g.state, g.drawReason = state, reason
	g.drawOfferedBy = nil
	if g.gameTimer != nil {
		g.gameTimer.Pause()
	}
}
