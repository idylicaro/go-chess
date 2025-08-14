package main

import (
	"fmt"
	"idylicaro/go-chess/internal/chess"
)

func main() {
	b := chess.NewBoard(nil)
	bi := chess.NewBishop(chess.White)
	err := b.AddPiece(bi, 3, 5)
	if err != nil {
		fmt.Println("Error adding piece:", err)
		return
	}

	DisplayBoard(b, bi.GetValidMoves(3, 5, b))

}
