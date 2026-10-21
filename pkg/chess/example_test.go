package chess_test

import (
	"fmt"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func ExampleNewGame() {
	game := chess.NewGame()
	move, _ := chess.ParseMove("e2e4")
	if err := game.Move(move); err != nil {
		fmt.Println(err)
	}
	fmt.Println(game.FEN())
	// Output: rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1
}

func ExampleParseFEN() {
	game, err := chess.ParseFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(game.IsOver(), game.Result(), game.DrawReason())
	// Output: true 1/2-1/2 Stalemate
}

func ExampleParseFEN_invalid() {
	_, err := chess.ParseFEN("not a fen")
	fmt.Println(err != nil)
	// Output: true
}

func ExampleGame_ParseSAN() {
	game := chess.NewGame()
	move, err := game.ParseSAN("Nf3")
	if err != nil {
		fmt.Println(err)
		return
	}
	_ = game.Move(move)
	fmt.Println(game.FEN())
	// Output: rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1
}

func ExampleGame_SAN() {
	game := chess.NewGame()
	move, _ := chess.ParseMove("g1f3")
	san, _ := game.SAN(move)
	fmt.Println(san)
	// Output: Nf3
}

func ExampleGame_PGN() {
	game := chess.NewGame()
	for _, san := range []string{"e4", "e5", "Nf3"} {
		move, _ := game.ParseSAN(san)
		_ = game.Move(move)
	}
	pgn, _ := game.PGN(chess.PGNHeaders{White: "Alice", Black: "Bob"})
	fmt.Print(pgn)
	// Output:
	// [Event "?"]
	// [Site "?"]
	// [Date "????.??.??"]
	// [Round "?"]
	// [White "Alice"]
	// [Black "Bob"]
	// [Result "*"]
	//
	// 1. e4 e5 2. Nf3 *
}

func ExampleGame_Undo() {
	game := chess.NewGame()
	move, _ := chess.ParseMove("e2e4")
	_ = game.Move(move)
	_ = game.Undo()
	fmt.Println(game.FEN() == chess.StartingFEN)
	// Output: true
}

func ExampleGame_Perft() {
	fmt.Println(chess.NewGame().Perft(3))
	// Output: 8902
}
