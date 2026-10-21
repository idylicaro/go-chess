package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/moveinput"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func runPlay(e env, args []string) error {
	fs := newFlagSet(e, "play", "[flags]", "Play a game in the terminal. Moves are typed in SAN (Nf3, exd5, O-O) or UCI (e2e4).\nType \"help\" during the game for the other commands.")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	fmt.Fprintln(e.out, "Starting Go Chess Engine...")

	scanner := bufio.NewScanner(e.in)

	// Ask user about timer
	fmt.Fprint(e.out, "Do you want to play with timer? (y/n): ")
	var game *chess.Game

	if scanner.Scan() {
		response := strings.TrimSpace(scanner.Text())
		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			// Ask for time format
			fmt.Fprintln(e.out, "Choose time format:")
			fmt.Fprintln(e.out, "1. Blitz (3+2)")
			fmt.Fprintln(e.out, "2. Rapid (10+0)")
			fmt.Fprintln(e.out, "3. Classical (90+30)")
			fmt.Fprint(e.out, "Enter choice (1-3): ")

			if scanner.Scan() {
				choice := strings.TrimSpace(scanner.Text())
				var timeConfig chess.TimeConfig

				switch choice {
				case "1":
					timeConfig = chess.Blitz3Plus2
					fmt.Fprintln(e.out, "Selected: Blitz 3+2")
				case "2":
					timeConfig = chess.Rapid10Plus0
					fmt.Fprintln(e.out, "Selected: Rapid 10+0")
				case "3":
					timeConfig = chess.Classical90Plus30
					fmt.Fprintln(e.out, "Selected: Classical 90+30")
				default:
					timeConfig = chess.Blitz3Plus2
					fmt.Fprintln(e.out, "Default: Blitz 3+2")
				}

				game = chess.NewTimedGame(timeConfig, chess.SystemClock{})
				if err := game.StartTimer(); err != nil {
					return fmt.Errorf("starting timer: %w", err)
				}
			} else {
				game = chess.NewGame()
				fmt.Fprintln(e.out, "Game created without timer!")
			}
		} else {
			game = chess.NewGame()
		}
	} else {
		game = chess.NewGame()
	}

	s := &session{env: e, game: game}
	for {
		s.show()
		fmt.Fprintf(e.out, "> Enter a move (SAN or UCI) or a command, \"help\" lists them (%s): ", game.GetCurrentTurn())
		if !scanner.Scan() {
			return nil
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" || input == "quit" {
			fmt.Fprintln(e.out, "Exiting game.")
			return nil
		}
		if !s.command(input) {
			move, err := moveinput.Parse(game, input)
			if err != nil {
				fmt.Fprintf(e.out, "Error: %v\n", err)
				continue
			}
			if err := game.Move(move); err != nil {
				fmt.Fprintf(e.out, "Error: %v\n", err)
				continue
			}
		}
		if game.IsOver() {
			s.show()
			fmt.Fprintf(e.out, "Game over! Result: %s\n", game.Result())
			return nil
		}
	}
}

// session is a game in progress on the terminal.
type session struct {
	env
	game    *chess.Game
	flipped bool // show the board from Black's side
}

func (s *session) show() {
	displayGameState(s.out, s.game)
	displayBoard(s.out, s.game.GetBoard(), nil, s.flipped)
}

// gameCommands lists the in-game commands for "help", in display order.
var gameCommands = []struct{ name, help string }{
	{"moves", "list the legal moves"},
	{"undo", "take back the last move"},
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
	turn := game.GetCurrentTurn()
	var err error
	switch input {
	case "resign":
		err = game.Resign(turn)
	case "draw":
		err = game.OfferDraw(turn)
		if err == nil {
			fmt.Fprintf(s.out, "%v offers a draw. The opponent can type \"accept\" or \"decline\".\n", turn)
		}
	case "accept":
		err = game.AcceptDraw(s.answerer())
	case "decline":
		err = game.DeclineDraw(s.answerer())
		if err == nil {
			fmt.Fprintln(s.out, "Draw offer declined.")
		}
	case "undo":
		err = game.Undo()
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

// answerer is the side that answers a pending draw offer: the opponent of
// whoever made it, which is not necessarily the side to move.
func (s *session) answerer() chess.Color {
	if by, pending := s.game.PendingDrawOffer(); pending && by == chess.White {
		return chess.Black
	}
	return chess.White
}

func (s *session) pgnHeaders() chess.PGNHeaders {
	return chess.PGNHeaders{
		Event: "Casual game",
		Site:  "go-chess CLI",
		Date:  s.now().Format("2006.01.02"),
		White: "White",
		Black: "Black",
	}
}
