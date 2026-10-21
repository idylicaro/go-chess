package cli

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func runPerft(e env, args []string) error {
	fs := newFlagSet(e, "perft", "[flags] <depth>", "Count the leaf nodes of the legal-move tree to <depth> plies,\nto check move generation against known values.")
	fen := fs.String("fen", chess.StartingFEN, "position to start from")
	divide := fs.Bool("divide", false, "also print the node count under each first move")
	workers := fs.Int("workers", 0, "goroutines sharing the root moves; 0 uses every CPU (ignored with --divide)")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usagef("expected exactly one <depth>")
	}
	depth, err := strconv.Atoi(rest[0])
	if err != nil || depth < 0 {
		return usagef("invalid depth %q: expected a number of plies, like 4", rest[0])
	}
	game, err := chess.ParseFEN(*fen)
	if err != nil {
		return err
	}

	start := time.Now()
	var nodes int
	if *divide {
		counts := game.Divide(depth)
		moves := make([]string, 0, len(counts))
		for move, n := range counts {
			moves = append(moves, move)
			nodes += n
		}
		sort.Strings(moves)
		for _, move := range moves {
			fmt.Fprintf(e.out, "%s: %d\n", move, counts[move])
		}
		fmt.Fprintln(e.out)
	} else {
		nodes = game.PerftParallel(depth, *workers)
	}
	elapsed := time.Since(start)
	fmt.Fprintf(e.out, "nodes: %d\n", nodes)
	if elapsed > 0 {
		fmt.Fprintf(e.out, "time:  %s (%.0f nodes/s)\n", elapsed.Round(time.Millisecond), float64(nodes)/elapsed.Seconds())
	}
	return nil
}
