package cli

import (
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/pgnread"
	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func runPGN(e env, args []string) error {
	fs := newFlagSet(e, "pgn", "[flags] <file|->", "Read a PGN file (or standard input with -), replay it and show its tags,\nmoves in SAN and the final position.")
	number := fs.Int("game", 1, "which game of the file to show, counting from 1")
	styleOpts := addStyleFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usagef("expected exactly one PGN file, or - for standard input")
	}
	text, err := readInput(e, rest[0])
	if err != nil {
		return err
	}
	games, err := pgnread.Parse(text)
	if err != nil {
		return err
	}
	if *number < 1 || *number > len(games) {
		return usagef("--game %d: the file has %d game(s)", *number, len(games))
	}
	parsed := games[*number-1]
	game, moves, err := parsed.Replay()
	if err != nil {
		return err
	}

	if len(games) > 1 {
		fmt.Fprintf(e.out, "Game %d of %d\n", *number, len(games))
	}
	for _, tag := range parsed.Tags {
		fmt.Fprintf(e.out, "%-8s %s\n", tag.Name+":", tag.Value)
	}
	fmt.Fprintln(e.out)
	fmt.Fprintln(e.out, numberedMoves(parsed.Moves, startFEN(parsed)))
	fmt.Fprintln(e.out)
	var last *chess.Move
	if len(moves) > 0 {
		last = &moves[len(moves)-1]
	}
	firstNumber, blackFirst := moveNumbering(startFEN(parsed))
	e.printPosition(game, render.Options{Style: e.style(styleOpts), Material: true, Captured: parsed.Tag("FEN") == ""},
		render.Panel{Moves: parsed.Moves, FirstMoveNumber: firstNumber, BlackFirst: blackFirst}, last)
	fmt.Fprintf(e.out, "FEN: %s\n", game.FEN())
	if parsed.Result != "" {
		fmt.Fprintf(e.out, "Result: %s\n", parsed.Result)
	}
	return nil
}

// numberedMoves formats SAN moves as "1. e4 e5 2. Nf3". startFEN says whose
// turn the first move is and which move number it has.
func numberedMoves(moves []string, startFEN string) string {
	number, blackToMove := moveNumbering(startFEN)
	var b strings.Builder
	for i, san := range moves {
		whiteMove := (i%2 == 0) != blackToMove
		if i > 0 {
			b.WriteByte(' ')
		}
		switch {
		case whiteMove:
			fmt.Fprintf(&b, "%d. ", number)
		case i == 0:
			fmt.Fprintf(&b, "%d... ", number)
		}
		b.WriteString(san)
		if !whiteMove {
			number++
		}
	}
	return b.String()
}

func startFEN(g pgnread.Game) string {
	if fen := g.Tag("FEN"); fen != "" {
		return fen
	}
	return chess.StartingFEN
}
