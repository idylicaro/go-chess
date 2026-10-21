package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/moveinput"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func runPlay(e env, args []string) error {
	fs := newFlagSet(e, "play", "[flags]", "Play a game in the terminal. Moves are typed in SAN (Nf3, exd5, O-O) or UCI (e2e4).")
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

	for {
		displayGameState(e.out, game)
		displayBoard(e.out, game.GetBoard(), nil)

		fmt.Fprintf(e.out, "> Enter a move (SAN or UCI), or resign|draw|accept|decline|undo|moves (%s): ", game.GetCurrentTurn().String())
		if !scanner.Scan() {
			return nil
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" || input == "quit" {
			fmt.Fprintln(e.out, "Exiting game.")
			return nil
		}
		if runGameCommand(e, game, input) {
			if game.IsOver() {
				finish(e, game)
				return nil
			}
			continue
		}

		move, err := moveinput.Parse(game, input)
		if err != nil {
			fmt.Fprintf(e.out, "Error: %v\n", err)
			continue
		}
		if err := game.Move(move); err != nil {
			fmt.Fprintf(e.out, "Error: %v\n", err)
			continue
		}
		if game.IsOver() {
			finish(e, game)
			return nil
		}
	}
}

// runGameCommand handles the non-move commands and reports whether input was one.
func runGameCommand(e env, game *chess.Game, input string) bool {
	turn := game.GetCurrentTurn()
	var err error
	switch input {
	case "resign":
		err = game.Resign(turn)
	case "draw":
		err = game.OfferDraw(turn)
		if err == nil {
			fmt.Fprintf(e.out, "%v offers a draw. The opponent can type \"accept\" or \"decline\".\n", turn)
		}
	case "accept":
		err = game.AcceptDraw(turn)
	case "decline":
		err = game.DeclineDraw(turn)
	case "undo":
		err = game.Undo()
	case "moves":
		for _, move := range game.LegalMoves() {
			fmt.Fprintf(e.out, "%s ", move)
		}
		fmt.Fprintln(e.out)
	default:
		return false
	}
	if err != nil {
		fmt.Fprintf(e.out, "Error: %v\n", err)
	}
	return true
}

func finish(e env, game *chess.Game) {
	displayGameState(e.out, game)
	displayBoard(e.out, game.GetBoard(), nil)
	fmt.Fprintf(e.out, "Game over! Result: %s\n", game.Result())
}
