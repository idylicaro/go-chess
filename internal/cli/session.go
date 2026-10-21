package cli

import (
	"fmt"
	"os"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// style says how the board is drawn.
type style struct {
	color   bool // ANSI colors
	unicode bool // chess symbols instead of letters
}

// session is a game in progress on the terminal.
type session struct {
	env
	game        *chess.Game
	flipped     bool // show the board from Black's side
	style       style
	timeControl *chess.TimeConfig // nil on an untimed game

	pgnOut string // where to write the PGN when the session ends; empty for nowhere

	opponent *opponent // nil when two players share the keyboard
	plies    int       // moves played in this session, to undo a pair against the engine
}

func (s *session) show() {
	displayGameState(s.out, s.game)
	displayBoard(s.out, s.game.GetBoard(), nil, s.flipped)
}

// play makes a move for the side to move.
func (s *session) play(move chess.Move) error {
	if err := s.game.Move(move); err != nil {
		return err
	}
	s.plies++
	return nil
}

func (s *session) close() {
	if s.opponent != nil {
		s.opponent.engine.Close()
	}
}

// gameCommands lists the in-game commands for "help", in display order.
var gameCommands = []struct{ name, help string }{
	{"moves", "list the legal moves"},
	{"undo", "take back the last move (yours and the engine's reply, against an engine)"},
	{"resign", "resign the game"},
	{"draw", "offer a draw"},
	{"accept", "accept the opponent's draw offer"},
	{"decline", "decline the opponent's draw offer"},
	{"fen", "print the position as FEN"},
	{"pgn", "print the game so far as PGN"},
	{"flip", "turn the board around"},
	{"help", "show this list"},
	{"exit", "leave the game (also: quit)"},
}

// command runs input if it is an in-game command and reports whether it was.
func (s *session) command(input string) bool {
	game := s.game
	var err error
	switch input {
	case "resign":
		err = game.Resign(s.humanSide())
	case "draw":
		err = s.offerDraw()
	case "accept":
		err = game.AcceptDraw(s.answerer())
	case "decline":
		err = game.DeclineDraw(s.answerer())
		if err == nil {
			fmt.Fprintln(s.out, "Draw offer declined.")
		}
	case "undo":
		err = s.undo()
	case "moves":
		for _, move := range game.LegalMoves() {
			san, sanErr := game.SAN(move)
			if sanErr != nil {
				san = move.String()
			}
			fmt.Fprintf(s.out, "%s ", san)
		}
		fmt.Fprintln(s.out)
	case "fen":
		fmt.Fprintln(s.out, game.FEN())
	case "pgn":
		var text string
		if text, err = game.PGN(s.pgnHeaders()); err == nil {
			fmt.Fprint(s.out, text)
		}
	case "flip":
		s.flipped = !s.flipped
	case "help":
		for _, c := range gameCommands {
			fmt.Fprintf(s.out, "  %-8s %s\n", c.name, c.help)
		}
		fmt.Fprintln(s.out, "Anything else is read as a move, in SAN (Nf3, exd5, O-O, e8=Q) or UCI (e2e4, e7e8q).")
	default:
		return false
	}
	if err != nil {
		fmt.Fprintf(s.out, "Error: %v\n", err)
	}
	return true
}

// humanSide is the side the person at the keyboard plays: against an engine
// always the same one, otherwise whoever is to move.
func (s *session) humanSide() chess.Color {
	if s.opponent != nil {
		return s.opponent.human
	}
	return s.game.GetCurrentTurn()
}

// answerer is the side that answers a pending draw offer: the opponent of
// whoever made it, which is not necessarily the side to move.
func (s *session) answerer() chess.Color {
	if by, pending := s.game.PendingDrawOffer(); pending && by == chess.White {
		return chess.Black
	}
	return chess.White
}

func (s *session) undo() error {
	if s.opponent == nil {
		if err := s.game.Undo(); err != nil {
			return err
		}
		s.plies--
		return nil
	}
	// Take back the engine's reply along with the move it answered, so it is
	// the player's turn again; the engine would just replay otherwise.
	if s.plies < 2 {
		return fmt.Errorf("%w yet: the engine has made the only move so far", chess.ErrNoMoveToUndo)
	}
	for range 2 {
		if err := s.game.Undo(); err != nil {
			return err
		}
		s.plies--
	}
	return nil
}

func (s *session) pgnHeaders() chess.PGNHeaders {
	headers := chess.PGNHeaders{
		Event: "Casual game",
		Site:  "go-chess CLI",
		Date:  s.now().Format("2006.01.02"),
		White: "White",
		Black: "Black",
	}
	if s.timeControl != nil {
		headers.TimeControl = formatTimeControl(*s.timeControl)
	}
	if s.opponent != nil {
		human, engine := "Player", s.opponent.engine.Name
		if engine == "" {
			engine = "Engine"
		}
		if s.opponent.human == chess.White {
			headers.White, headers.Black = human, engine
		} else {
			headers.White, headers.Black = engine, human
		}
	}
	return headers
}

// exportPGN writes the game to the --pgn-out destination, if there is one.
func (s *session) exportPGN() error {
	if s.pgnOut == "" {
		return nil
	}
	text, err := s.game.PGN(s.pgnHeaders())
	if err != nil {
		return err
	}
	if s.pgnOut == "-" {
		_, err = fmt.Fprint(s.out, "\n"+text)
		return err
	}
	if err := os.WriteFile(s.pgnOut, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing PGN: %w", err)
	}
	fmt.Fprintf(s.out, "PGN saved to %s\n", s.pgnOut)
	return nil
}
