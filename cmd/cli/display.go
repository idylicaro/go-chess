package main

import (
	"fmt"
	"idylicaro/go-chess/internal/chess"
	common "idylicaro/go-chess/internal/common"
)

func DisplayBoard(b chess.IBoard, validMoves []chess.Position) {
	fmt.Println("  -----------------")
	for y := 0; y < 8; y++ {
		fmt.Printf("%d |", 8-y)
		for x := 0; x < 8; x++ {
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
	fmt.Println("    a b c d e f g h")
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
