package main

import (
	"bufio"
	"fmt"
	"idylicaro/go-chess/internal/chess"
	"idylicaro/go-chess/internal/debug"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/uci"
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
				var timeConfig timer.TimeConfig

				switch choice {
				case "1":
					timeConfig = timer.Blitz3Plus2
					fmt.Println("Selected: Blitz 3+2")
				case "2":
					timeConfig = timer.Rapid10Plus0
					fmt.Println("Selected: Rapid 10+0")
				case "3":
					timeConfig = timer.Classical90Plus30
					fmt.Println("Selected: Classical 90+30")
				default:
					timeConfig = timer.Blitz3Plus2
					fmt.Println("Default: Blitz 3+2")
				}

				game = chess.NewTimedGame(timeConfig, timer.SystemClock{})
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

		fmt.Printf("> Enter your move (%s): ", game.GetCurrentTurn().String())
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" || input == "quit" {
			fmt.Println("Exiting game.")
			return
		}

		// 3. Convert the input to UCI format
		move, err := uci.ParseMove(input)
		fmt.Println(move)
		if err != nil {
			fmt.Printf("Error: %v. Please enter a move in UCI format (e.g., e2e4)\n", err)
			continue
		}

		// Try to make the move
		err = game.Move(move)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Check the GameState to see if the game has ended
		if game.GetState() == chess.Checkmate ||
			game.GetState() == chess.Stalemate ||
			game.GetState() == chess.Draw ||
			game.GetState() == chess.TimeoutWhite ||
			game.GetState() == chess.TimeoutBlack {
			DisplayGameState(game)
			DisplayBoard(game.GetBoard(), nil)
			fmt.Println("Game over!")
			return
		}
	}
}
