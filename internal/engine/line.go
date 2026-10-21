package engine

import (
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// SAN converts a principal variation from UCI to SAN, played from fen. It
// stops at the first move that is not legal there.
//
//	engine.SAN(game.FEN(), []string{"e2e4", "e7e5"}) // [e4 e5]
func SAN(fen string, pv []string) []string {
	game, err := chess.ParseFEN(fen)
	if err != nil {
		return nil
	}
	var out []string
	for _, text := range pv {
		move, err := chess.ParseMove(text)
		if err != nil {
			break
		}
		san, err := game.SAN(move)
		if err != nil || game.Move(move) != nil {
			break
		}
		out = append(out, san)
	}
	return out
}

// Line is a principal variation as numbered SAN, played from fen:
// "12. Nf3 Nc6 13. d4", or "12... Nc6 13. d4" when Black moves first.
func Line(fen string, pv []string) string {
	sans := SAN(fen, pv)
	number, blackToMove := 1, false
	if fields := strings.Fields(fen); len(fields) >= 2 {
		blackToMove = fields[1] == "b"
		if len(fields) == 6 {
			fmt.Sscanf(fields[5], "%d", &number)
		}
	}
	var out []string
	for i, san := range sans {
		whiteMove := (i%2 == 0) != blackToMove
		switch {
		case whiteMove:
			out = append(out, fmt.Sprintf("%d. %s", number, san))
		case i == 0:
			out = append(out, fmt.Sprintf("%d... %s", number, san))
		default:
			out = append(out, san)
		}
		if !whiteMove {
			number++
		}
	}
	return strings.Join(out, " ")
}
