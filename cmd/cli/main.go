package main

import (
	"idylicaro/go-chess/internal/chess"
)

func main() {
	b := chess.NewInitialBoard()

	DisplayBoard(b, chess.White, nil)

}
