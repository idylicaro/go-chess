package cli

import (
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func runAnalyze(e env, args []string) error {
	fs := newFlagSet(e, "analyze", "[flags]", "Ask a UCI engine for the best move and the main line of a position.\nScores are from White's point of view.")
	fen := fs.String("fen", chess.StartingFEN, "position to analyze")
	enginePath := fs.String("engine", "stockfish", "path of the UCI engine")
	depth := fs.Int("depth", 0, "search depth in plies")
	movetime := fs.Duration("movetime", 0, "search time, like 2s")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	game, err := chess.ParseFEN(*fen)
	if err != nil {
		return err
	}
	if game.IsOver() {
		return fmt.Errorf("the game is already over (%s)", game.Result())
	}
	limits := engine.Limits{Depth: *depth, MoveTime: *movetime}
	if limits.Depth == 0 && limits.MoveTime == 0 {
		limits.Depth = 14
	}

	eng, err := engine.Start(*enginePath)
	if err != nil {
		return err
	}
	defer eng.Close()
	fmt.Fprintf(e.out, "Engine: %s\n", eng.Name)

	var pending *engine.Info
	show := func(info *engine.Info) {
		if info != nil && info.HasScore {
			fmt.Fprintln(e.out, analysisLine(game, *info))
		}
	}
	res, err := eng.Search(e.ctx, game.FEN(), limits, func(info engine.Info) {
		if info.MultiPV > 1 {
			return
		}
		if pending != nil && info.Depth != pending.Depth {
			show(pending)
		}
		pending = &info
	})
	if err != nil {
		return err
	}
	show(pending)
	best := res.Best
	if move, perr := chess.ParseMove(res.Best); perr == nil {
		if san, serr := game.SAN(move); serr == nil {
			best = san
		}
	}
	fmt.Fprintf(e.out, "Best move: %s\n", best)
	return nil
}

// analysisLine formats one depth of the search: "depth 12  +0.34  e4 e5 Nf3".
func analysisLine(game *chess.Game, info engine.Info) string {
	score := info.Score
	if game.GetCurrentTurn() == chess.Black {
		score.Value = -score.Value
	}
	return fmt.Sprintf("depth %2d  %6s  %s", info.Depth, score, strings.Join(pvToSAN(game.FEN(), info.PV), " "))
}

// pvToSAN converts a principal variation from UCI to SAN, stopping at the
// first move that is not legal.
func pvToSAN(fen string, pv []string) []string {
	game, err := chess.ParseFEN(fen)
	if err != nil {
		return pv
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
