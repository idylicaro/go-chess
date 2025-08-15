package main

import (
	"fmt"
	"idylicaro/go-chess/internal/chess"
	common "idylicaro/go-chess/internal/common"
)

func DisplayBoard(b chess.IBoard, perspective chess.Color, validMoves []chess.Position) {
	fmt.Println("  -----------------")
	var yStart, yEnd, yStep int
	var xStart, xEnd, xStep int
	var colLabels string
	if perspective == chess.Black {
		yStart, yEnd, yStep = 0, 8, 1
		xStart, xEnd, xStep = 7, -1, -1
		colLabels = "    h g f e d c b a"
	} else {
		yStart, yEnd, yStep = 7, -1, -1
		xStart, xEnd, xStep = 0, 8, 1
		colLabels = "    a b c d e f g h"
	}
	for y := yStart; y != yEnd; y += yStep {
		if perspective == chess.Black {
			fmt.Printf("%d |", y+1)
		} else {
			fmt.Printf("%d |", 8-y)
		}
		for x := xStart; x != xEnd; x += xStep {
			piece, err := b.GetPiece(x, y)
			if err == nil && piece != nil {
				fmt.Printf(" %s", getPieceChar(piece))
			} else {
				isMove := false
				if len(validMoves) > 0 {
					for _, move := range validMoves {
						if move.X == x && move.Y == y {
							fmt.Printf(" x")
							isMove = true
							break
						}
					}
				}
				if !isMove {
					fmt.Printf(" .")
				}
			}
		}
		fmt.Println(" |")
	}
	fmt.Println("  -----------------")
	fmt.Println(colLabels)
}

// getPieceChar converte uma peça em sua representação de caractere
func getPieceChar(piece chess.IPiece) string {
	switch piece.GetType() {
	case common.Pawn:
		if piece.GetColor() == chess.White {
			return "P"
		}
		return "p"
	case common.Rook:
		if piece.GetColor() == chess.White {
			return "R"
		}
		return "r"
	case common.Knight:
		if piece.GetColor() == chess.White {
			return "N"
		}
		return "n"
	case common.Bishop:
		if piece.GetColor() == chess.White {
			return "B"
		}
		return "b"
	case common.Queen:
		if piece.GetColor() == chess.White {
			return "Q"
		}
		return "q"
	case common.King:
		if piece.GetColor() == chess.White {
			return "K"
		}
		return "k"
	}
	return " "
}
