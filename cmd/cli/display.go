package main

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/pkg/chess"
	"time"
)

func DisplayBoard(b chess.IBoard, validMoves []chess.Position) {
	fmt.Println("  -----------------")
	for y, py := 0, 8; y < 8; y, py = y+1, py-1 { // py == presentationY
		fmt.Printf("%d |", py)
		for x := 0; x < 8; x++ { // col a-h
			piece, err := b.GetPiece(x, y)
			if err == nil && piece != nil {
				fmt.Printf(" %s", getPieceChar(piece))
			} else {
				isMove := false
				for _, move := range validMoves {
					if move.X == x && move.Y == y {
						fmt.Printf(" x")
						isMove = true
						break
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

func DisplayGameState(game *chess.Game) {
	var turn, state string
	if game.GetCurrentTurn() == common.White {
		turn = "White"
	} else {
		turn = "Black"
	}

	switch game.GetState() {
	case chess.InProgress:
		state = "In Progress"
	case chess.Check:
		state = "Check"
	case chess.Checkmate:
		state = "Checkmate"
	case chess.Stalemate:
		state = "Stalemate"
	case chess.Draw:
		state = "Draw (" + game.DrawReason().String() + ")"
	case chess.ResignedWhite:
		state = "White resigned - Black Wins"
	case chess.ResignedBlack:
		state = "Black resigned - White Wins"
	case chess.TimeoutWhite:
		state = "Timeout - Black Wins"
	case chess.TimeoutBlack:
		state = "Timeout - White Wins"
	}

	fmt.Printf("\n--- Current State: %s | Turn: %s ---\n", state, turn)

	// Display timer information if available
	if game.HasTimer() {
		whiteTime, err1 := game.GetRemainingTime(common.White)
		blackTime, err2 := game.GetRemainingTime(common.Black)

		if err1 == nil && err2 == nil {
			fmt.Printf("⏰ White: %v | Black: %v\n",
				formatDuration(whiteTime),
				formatDuration(blackTime))
		}
	}
}

// getPieceChar converts a piece to its character representation
func getPieceChar(piece chess.IPiece) string {
	switch piece.GetType() {
	case common.Pawn:
		if piece.GetColor() == common.White {
			return "P"
		}
		return "p"
	case common.Rook:
		if piece.GetColor() == common.White {
			return "R"
		}
		return "r"
	case common.Knight:
		if piece.GetColor() == common.White {
			return "N"
		}
		return "n"
	case common.Bishop:
		if piece.GetColor() == common.White {
			return "B"
		}
		return "b"
	case common.Queen:
		if piece.GetColor() == common.White {
			return "Q"
		}
		return "q"
	case common.King:
		if piece.GetColor() == common.White {
			return "K"
		}
		return "k"
	}
	return " "
}

// formatDuration formats a time.Duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "00:00"
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
