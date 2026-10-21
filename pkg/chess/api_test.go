package chess_test

import (
	"fmt"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// The public API can be driven using only the names exported by pkg/chess.
func Example() {
	game := chess.NewTimedGame(chess.Blitz3Plus2, chess.SystemClock{})
	for _, notation := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
		move, _ := chess.ParseMove(notation)
		if err := game.Move(move); err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println(game.Result(), game.GetState(), len(game.LegalMoves()))

	_ = game.Undo()
	fmt.Println(game.Result(), game.GetCurrentTurn() == chess.Black)
	// Output:
	// 0-1 Checkmate 0
	// * true
}
