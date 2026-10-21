package cli

import (
	"fmt"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func runFEN(e env, args []string) error {
	fs := newFlagSet(e, "fen", "[<fen>]", "Check a FEN and show the position: the normalized FEN, the board, whose turn\nit is and how many legal moves there are. Without <fen> it shows the starting position.\nQuote the FEN so the shell passes it as one argument.")
	styleOpts := addStyleFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return usagef("expected one quoted FEN, got %d arguments", len(rest))
	}
	fen := chess.StartingFEN
	if len(rest) == 1 {
		fen = rest[0]
	}
	game, err := chess.ParseFEN(fen)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "FEN:         %s\n", game.FEN())
	printPosition(e.out, game, e.style(styleOpts), false)
	fmt.Fprintf(e.out, "Legal moves: %d\n", len(game.LegalMoves()))
	return nil
}
