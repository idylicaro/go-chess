// Package moveinput turns what a player typed into a legal chess.Move, in SAN
// or UCI notation, and says why when it cannot.
package moveinput

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Error is a rejected input. Kind is chess.ErrInvalidNotation,
// chess.ErrIllegalMove or chess.ErrAmbiguousMove, so callers can tell the
// three apart with errors.Is; Error() is the message to show the player.
type Error struct {
	Kind error
	Text string
}

func (e *Error) Error() string { return e.Text }
func (e *Error) Unwrap() error { return e.Kind }

const notationHelp = "write it in SAN (Nf3, exd5, O-O, e8=Q) or UCI (e2e4, e7e8q)"

var (
	// uciLike is any input shaped like two squares, so that a bad square
	// ("e2e9") is reported as a bad UCI move and not as bad SAN.
	uciLike = regexp.MustCompile(`^[a-z][0-9][a-z][0-9][a-z]?$`)
	// sanTarget captures the piece letter and destination of a SAN string.
	sanTarget = regexp.MustCompile(`^([NBRQK])?[a-h]?[1-8]?x?([a-h][1-8])`)
	// promotionSquare is a bare pawn move to the last rank, which needs a piece.
	promotionSquare = regexp.MustCompile(`^(?:[a-h]x)?[a-h][18]$`)
)

// Parse reads text as a move of the side to move in game. Errors are *Error.
//
//	move, err := moveinput.Parse(game, "Nf3")
func Parse(game *chess.Game, text string) (chess.Move, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return chess.Move{}, &Error{chess.ErrInvalidNotation, "type a move: " + notationHelp}
	}
	if lower := strings.ToLower(text); uciLike.MatchString(lower) {
		return parseUCI(game, text, lower)
	}
	return parseSAN(game, text)
}

func parseUCI(game *chess.Game, text, lower string) (chess.Move, error) {
	move, err := chess.ParseMove(lower)
	if err != nil {
		return chess.Move{}, &Error{chess.ErrInvalidNotation, fmt.Sprintf("%q is not a valid move: %s", text, trimKind(err))}
	}
	for _, legal := range game.LegalMoves() {
		if legal.FromX == move.FromX && legal.FromY == move.FromY && legal.ToX == move.ToX && legal.ToY == move.ToY &&
			(legal.Promotion == move.Promotion || move.Promotion == 0) {
			return move, nil
		}
	}
	return chess.Move{}, &Error{chess.ErrIllegalMove, fmt.Sprintf("%s is not legal: %s", text, explainIllegal(game, move))}
}

func parseSAN(game *chess.Game, text string) (chess.Move, error) {
	move, err := game.ParseSAN(text)
	switch {
	case err == nil:
		return move, nil
	case errors.Is(err, chess.ErrAmbiguousMove):
		return chess.Move{}, &Error{chess.ErrAmbiguousMove, fmt.Sprintf("%q is ambiguous: %s", text, ambiguityHint(game, text))}
	case errors.Is(err, chess.ErrIllegalMove):
		reason := "no piece can make that move"
		switch {
		case promotionSquare.MatchString(strings.TrimRight(text, "+#!?")):
			reason = `a pawn that reaches the last rank must name its piece, like "` + text + `=Q"`
		case game.GetState() == chess.Check:
			reason = "you are in check, and this move does not get out of it"
		}
		return chess.Move{}, &Error{chess.ErrIllegalMove, fmt.Sprintf("%s is not legal: %s", text, reason)}
	default:
		return chess.Move{}, &Error{chess.ErrInvalidNotation, fmt.Sprintf("%q is not a move: %s", text, notationHelp)}
	}
}

// trimKind drops the leading "invalid move notation: " that every parse
// error carries, since the message already says the input is not a move.
func trimKind(err error) string {
	return strings.TrimPrefix(err.Error(), chess.ErrInvalidNotation.Error()+": ")
}

// ambiguityHint lists the SAN of every legal move that text could mean.
func ambiguityHint(game *chess.Game, text string) string {
	parts := sanTarget.FindStringSubmatch(strings.TrimRight(text, "+#!?"))
	if parts == nil {
		return "add the origin file or rank"
	}
	var options []string
	for _, legal := range game.LegalMoves() {
		san, err := game.SAN(legal)
		if err != nil {
			continue
		}
		if target := sanTarget.FindStringSubmatch(strings.TrimRight(san, "+#")); target != nil &&
			target[1] == parts[1] && target[2] == parts[2] {
			options = append(options, san)
		}
	}
	return "it could be " + strings.Join(options, " or ") + ", so say which"
}

// explainIllegal says why a well-formed UCI move cannot be played.
func explainIllegal(game *chess.Game, move chess.Move) string {
	from := square(move.FromX, move.FromY)
	piece, err := game.GetBoard().GetPiece(move.FromX, move.FromY)
	if err != nil || piece == nil {
		return "there is no piece on " + from
	}
	turn := game.GetCurrentTurn()
	if piece.GetColor() != turn {
		return fmt.Sprintf("the piece on %s is %s's, but it is %s's turn", from, piece.GetColor(), turn)
	}
	to := square(move.ToX, move.ToY)
	for _, target := range piece.GetValidMoves(move.FromX, move.FromY, game.GetBoard()) {
		if target.X == move.ToX && target.Y == move.ToY {
			if game.GetState() == chess.Check {
				return "you are in check, and this move does not get out of it"
			}
			return "it would leave your king in check"
		}
	}
	return fmt.Sprintf("the %s on %s cannot move to %s", strings.ToLower(piece.GetType().String()), from, to)
}

func square(x, y int) string {
	return string([]byte{byte('a' + x), byte('8' - y)})
}
