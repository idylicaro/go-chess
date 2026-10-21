package cli

import (
	"fmt"
	"time"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// engineThinkTime is how long the engine thinks per move on an untimed game.
const engineThinkTime = time.Second

// drawCheckTime is how long the engine looks at a position before it answers
// a draw offer.
const drawCheckTime = 300 * time.Millisecond

// opponent is the engine the player faces and the side the player has.
type opponent struct {
	engine *engine.Engine
	human  chess.Color
}

func (o *opponent) side() chess.Color {
	if o.human == chess.White {
		return chess.Black
	}
	return chess.White
}

func (s *session) startEngine(path string, level int, human chess.Color) error {
	eng, err := engine.Start(path)
	if err != nil {
		return err
	}
	if level >= 0 {
		if err := eng.SetOption("Skill Level", fmt.Sprint(level)); err != nil {
			eng.Close()
			return err
		}
	}
	if err := eng.NewGame(); err != nil {
		eng.Close()
		return err
	}
	s.opponent = &opponent{engine: eng, human: human}
	fmt.Fprintf(s.out, "Playing against %s as %s.\n", eng.Name, s.opponent.human)
	return nil
}

func (s *session) engineToMove() bool {
	return s.opponent != nil && !s.game.IsOver() && s.game.GetCurrentTurn() == s.opponent.side()
}

// limits are the search limits for the engine's turn: its clock when the
// game is timed, a fixed think time otherwise.
func (s *session) limits() engine.Limits {
	if s.timeControl == nil {
		return engine.Limits{MoveTime: engineThinkTime}
	}
	white, _ := s.game.GetRemainingTime(chess.White)
	black, _ := s.game.GetRemainingTime(chess.Black)
	return engine.Limits{WTime: white, BTime: black, WInc: s.timeControl.Increment, BInc: s.timeControl.Increment}
}

// engineMove asks the engine for its move and plays it.
func (s *session) engineMove() error {
	fmt.Fprintln(s.out, "Engine is thinking...")
	res, err := s.opponent.engine.Search(s.ctx, s.game.FEN(), s.limits(), nil)
	if err != nil {
		return fmt.Errorf("engine: %w", err)
	}
	move, err := chess.ParseMove(res.Best)
	if err != nil {
		return fmt.Errorf("engine answered %q: %w", res.Best, err)
	}
	san, sanErr := s.game.SAN(move)
	if sanErr != nil {
		san = res.Best
	}
	if err := s.play(move); err != nil {
		return fmt.Errorf("engine played %q: %w", res.Best, err)
	}
	fmt.Fprintf(s.out, "Engine plays %s.\n", san)
	return nil
}

// offerDraw offers a draw for the player. Against an engine the answer is
// immediate: it accepts when it does not think it stands better.
func (s *session) offerDraw() error {
	if err := s.game.OfferDraw(s.humanSide()); err != nil {
		return err
	}
	if s.opponent == nil {
		fmt.Fprintf(s.out, "%v offers a draw. The opponent can type \"accept\" or \"decline\".\n", s.humanSide())
		return nil
	}
	accept, err := s.engineAcceptsDraw()
	if err != nil {
		return err
	}
	if accept {
		fmt.Fprintln(s.out, "Engine accepts the draw.")
		return s.game.AcceptDraw(s.opponent.side())
	}
	fmt.Fprintln(s.out, "Engine declines the draw.")
	return s.game.DeclineDraw(s.opponent.side())
}

func (s *session) engineAcceptsDraw() (bool, error) {
	var last engine.Info
	_, err := s.opponent.engine.Search(s.ctx, s.game.FEN(), engine.Limits{MoveTime: drawCheckTime}, func(info engine.Info) {
		if info.HasScore && info.MultiPV <= 1 {
			last = info
		}
	})
	if err != nil {
		return false, fmt.Errorf("engine: %w", err)
	}
	// The score is for the side to move, which is the engine's only when
	// the player has just offered on the engine's turn; normalize to the engine.
	score := last.Score
	if s.game.GetCurrentTurn() != s.opponent.side() {
		score.Value = -score.Value
	}
	if score.Mate {
		return score.Value < 0, nil
	}
	return last.HasScore && score.Value <= 0, nil
}
