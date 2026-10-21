package main

import (
	"bufio"
	"fmt"
	"idylicaro/go-chess/internal/debug"
	"idylicaro/go-chess/pkg/chess"
	"os"
	"strings"
)

func main() {
	fmt.Println("Starting Go Chess Engine...")

	// Initialize debug mode from environment variables
	debug.EnableFromEnvironment()

	scanner := bufio.NewScanner(os.Stdin)

	// Ask user about timer
	fmt.Print("Do you want to play with timer? (y/n): ")
	var game *chess.Game

	if scanner.Scan() {
		response := strings.TrimSpace(scanner.Text())
		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			// Ask for time format
			fmt.Println("Choose time format:")
			fmt.Println("1. Blitz (3+2)")
			fmt.Println("2. Rapid (10+0)")
			fmt.Println("3. Classical (90+30)")
			fmt.Print("Enter choice (1-3): ")

			if scanner.Scan() {
				choice := strings.TrimSpace(scanner.Text())
				var timeConfig chess.TimeConfig

				switch choice {
				case "1":
					timeConfig = chess.Blitz3Plus2
					fmt.Println("Selected: Blitz 3+2")
				case "2":
					timeConfig = chess.Rapid10Plus0
					fmt.Println("Selected: Rapid 10+0")
				case "3":
					timeConfig = chess.Classical90Plus30
					fmt.Println("Selected: Classical 90+30")
				default:
					timeConfig = chess.Blitz3Plus2
					fmt.Println("Default: Blitz 3+2")
				}

				game = chess.NewTimedGame(timeConfig, chess.SystemClock{})
				err := game.StartTimer()
				if err != nil {
					fmt.Printf("Error starting timer: %v\n", err)
					return
				}
			} else {
				game = chess.NewGame()
				fmt.Println("Game created without timer!")
			}
		} else {
			game = chess.NewGame()
		}
	} else {
		game = chess.NewGame()
	}

	for {
		DisplayGameState(game)
		DisplayBoard(game.GetBoard(), nil)

		fmt.Printf("> Enter a move, or resign|draw|accept|decline|undo|moves (%s): ", game.GetCurrentTurn().String())
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" || input == "quit" {
			fmt.Println("Exiting game.")
			return
		}
		if runCommand(game, input) {
			if game.IsOver() {
				finish(game)
				return
			}
			continue
		}

		// 3. Convert the input to UCI format
		move, err := chess.ParseMove(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Try to make the move
		err = game.Move(move)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if game.IsOver() {
			finish(game)
			return
		}
	}
}

// runCommand handles the non-move commands and reports whether input was one.
func runCommand(game *chess.Game, input string) bool {
	turn := game.GetCurrentTurn()
	var err error
	switch input {
	case "resign":
		err = game.Resign(turn)
	case "draw":
		err = game.OfferDraw(turn)
		if err == nil {
			fmt.Printf("%v offers a draw. The opponent can type \"accept\" or \"decline\".\n", turn)
		}
	case "accept":
		err = game.AcceptDraw(turn)
	case "decline":
		err = game.DeclineDraw(turn)
	case "undo":
		err = game.Undo()
	case "moves":
		for _, move := range game.LegalMoves() {
			fmt.Printf("%s ", move)
		}
		fmt.Println()
	default:
		return false
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	return true
}

func finish(game *chess.Game) {
	DisplayGameState(game)
	DisplayBoard(game.GetBoard(), nil)
	fmt.Printf("Game over! Result: %s\n", game.Result())
}
