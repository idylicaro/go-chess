package match

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// TimeControl is a game's clock: Base for the game plus Increment a move.
type TimeControl struct {
	Base, Increment time.Duration
}

// String is the PGN TimeControl tag, seconds+increment, like "10+0.1".
func (tc TimeControl) String() string {
	return fmt.Sprintf("%g+%g", tc.Base.Seconds(), tc.Increment.Seconds())
}

// GameRecord is a finished game.
type GameRecord struct {
	Game        *chess.Game
	Result      chess.Result
	Termination string // why it ended, for the PGN and the report
}

// referee plays one game between two players and enforces the clock.
type referee struct {
	timeControl TimeControl
	// timeMargin is how far past zero a clock may go before the game is
	// lost on time: process latency, not thinking, at fast controls.
	timeMargin time.Duration
	// maxPlies adjudicates a draw in a game that runs too long.
	maxPlies int
	now      func() time.Time
}

// errAborted means the match was cancelled during the game.
var errAborted = errors.New("game aborted")

// play runs the game from opening; players[chess.White] moves first.
func (r referee) play(ctx context.Context, players [2]Player, opening Opening) (GameRecord, error) {
	game, moves, err := startGame(players, opening)
	if err != nil {
		return GameRecord{}, err
	}
	clock := [2]time.Duration{r.timeControl.Base, r.timeControl.Base}
	for !game.IsOver() && len(moves) < r.maxPlies {
		side := game.GetCurrentTurn()
		move, lost, err := r.askMove(ctx, players[side], moves, &clock, side)
		if err != nil || lost != "" {
			return forfeit(game, side, lost), err
		}
		if err := playUCI(game, move); err != nil {
			return forfeit(game, side, fmt.Sprintf("illegal move %s", move)), nil
		}
		moves = append(moves, move)
	}
	return finishedRecord(game), nil
}

// startGame plays the opening and tells both players a new game begins.
func startGame(players [2]Player, opening Opening) (*chess.Game, []string, error) {
	game := chess.NewGame()
	for _, move := range opening {
		if err := playUCI(game, move); err != nil {
			return nil, nil, fmt.Errorf("opening %v: %w", opening, err)
		}
	}
	for _, player := range players {
		if err := player.NewGame(); err != nil {
			return nil, nil, err
		}
	}
	return game, append([]string(nil), opening...), nil
}

// askMove asks side's player for a move and charges its clock. lost names
// the reason when the player forfeits (time or a crash).
func (r referee) askMove(ctx context.Context, player Player, moves []string, clock *[2]time.Duration, side chess.Color) (move, lost string, err error) {
	moveCtx, cancel := context.WithTimeout(ctx, clock[side]+r.timeMargin)
	defer cancel()
	start := r.now()
	move, searchErr := player.BestMove(moveCtx, moves, r.clockFor(*clock))
	clock[side] -= r.now().Sub(start)
	switch {
	case ctx.Err() != nil:
		return "", "", errAborted
	case clock[side] < -r.timeMargin:
		return "", "loses on time", nil
	case searchErr != nil:
		return "", fmt.Sprintf("engine error: %v", searchErr), nil
	}
	clock[side] += r.timeControl.Increment
	return move, "", nil
}

func (r referee) clockFor(clock [2]time.Duration) Clock {
	return Clock{
		WTime: max(clock[chess.White], 0), BTime: max(clock[chess.Black], 0),
		WInc: r.timeControl.Increment, BInc: r.timeControl.Increment,
	}
}

func playUCI(game *chess.Game, notation string) error {
	move, err := chess.ParseMove(notation)
	if err != nil {
		return err
	}
	return game.Move(move)
}

// forfeit ends the game with a loss for side.
func forfeit(game *chess.Game, side chess.Color, reason string) GameRecord {
	result := chess.ResultWhiteWins
	if side == chess.White {
		result = chess.ResultBlackWins
	}
	return GameRecord{Game: game, Result: result, Termination: fmt.Sprintf("%s %s", side, reason)}
}

// finishedRecord reads how a game that was played out ended.
func finishedRecord(game *chess.Game) GameRecord {
	if !game.IsOver() {
		return GameRecord{Game: game, Result: chess.ResultDraw, Termination: "adjudicated draw: maximum length"}
	}
	termination := game.GetState().String()
	if reason := game.DrawReason(); reason != chess.DrawNone {
		termination = reason.String()
	}
	return GameRecord{Game: game, Result: game.Result(), Termination: termination}
}

// ParseTimeControl reads seconds plus an optional increment in seconds, as
// cutechess-cli writes it: "10+0.1", "60".
func ParseTimeControl(text string) (TimeControl, error) {
	baseText, incrementText, hasIncrement := strings.Cut(text, "+")
	base, err := strconv.ParseFloat(baseText, 64)
	if err != nil || base <= 0 {
		return TimeControl{}, fmt.Errorf("invalid time control %q: expected seconds plus an optional increment in seconds, like 10+0.1", text)
	}
	increment := 0.0
	if hasIncrement {
		increment, err = strconv.ParseFloat(incrementText, 64)
		if err != nil || increment < 0 {
			return TimeControl{}, fmt.Errorf("invalid time control %q: the increment must be a number of seconds, like 10+0.1", text)
		}
	}
	return TimeControl{Base: seconds(base), Increment: seconds(increment)}, nil
}

func seconds(value float64) time.Duration {
	return time.Duration(value * float64(time.Second))
}
