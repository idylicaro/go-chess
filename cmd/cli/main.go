package main

import (
	"bufio"
	"fmt"
	"idylicaro/go-chess/internal/chess"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/uci"
	"os"
	"strings"
)

func main() {
	fmt.Println("Starting Go Chess Engine...")

	// Ask if user wants to play with timer
	fmt.Print("Do you want to play with timer? (y/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	var game *chess.Game

	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer == "y" || answer == "yes" {
			// Create game with timer (Rapid 10+0)
			game = chess.NewGameWithTimer(timer.Rapid10Plus0)
			fmt.Println("Game created with Rapid 10 timer!")

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
