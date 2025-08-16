package main

import (
	"bufio"
	"fmt"
	"idylicaro/go-chess/internal/chess"
	"idylicaro/go-chess/internal/uci"
	"os"
	"strings"
)

func main() {
	fmt.Println("Starting Go Chess Engine...")

	game := chess.NewGame()

	scanner := bufio.NewScanner(os.Stdin)

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
		if game.GetState() == chess.Checkmate || game.GetState() == chess.Stalemate || game.GetState() == chess.Draw {
			DisplayGameState(game)
			DisplayBoard(game.GetBoard(), nil)
			fmt.Println("Game over!")
			return
		}
	}
}
