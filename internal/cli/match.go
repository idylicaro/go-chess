package cli

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/idylicaro/go-chess/internal/match"
)

// defaultMatchEngine is the engine "make build" produces.
const defaultMatchEngine = "bin/chess-engine"

// engineOptions collects repeated --option Name=Value flags.
type engineOptions map[string]string

func (o engineOptions) String() string {
	parts := make([]string, 0, len(o))
	for name, value := range o {
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, ",")
}

func (o engineOptions) Set(text string) error {
	name, value, ok := strings.Cut(text, "=")
	if !ok || name == "" {
		return fmt.Errorf("invalid option %q: expected Name=Value, like Hash=64", text)
	}
	o[name] = value
	return nil
}

// matchFlags are the flags of "chess match".
type matchFlags struct {
	engines                          [2]match.EngineSpec
	timeControl, sprt, openings, pgn string
	concurrency, pairs, maxPlies     int
	alpha, beta                      float64
}

func defineMatchFlags(fs *flag.FlagSet) *matchFlags {
	f := &matchFlags{}
	for i := range f.engines {
		f.engines[i].Options = engineOptions{}
		fs.StringVar(&f.engines[i].Path, fmt.Sprintf("engine%d", i+1), defaultMatchEngine, fmt.Sprintf("path of engine %d", i+1))
		fs.StringVar(&f.engines[i].Name, fmt.Sprintf("name%d", i+1), fmt.Sprintf("engine%d", i+1), fmt.Sprintf("name of engine %d in the report and the PGN", i+1))
		fs.Var(engineOptions(f.engines[i].Options), fmt.Sprintf("option%d", i+1), fmt.Sprintf("UCI option Name=Value for engine %d; repeat for more", i+1))
	}
	fs.StringVar(&f.timeControl, "tc", "10+0.1", "time control as seconds+increment, like 10+0.1")
	fs.IntVar(&f.concurrency, "concurrency", max(1, runtime.NumCPU()/2), "games played at once")
	fs.IntVar(&f.pairs, "pairs", 500, "game pairs to play at most (each opening once with each color)")
	fs.StringVar(&f.sprt, "sprt", "", "stop early with an SPRT between elo0,elo1, like 0,10 (default: play every pair)")
	fs.Float64Var(&f.alpha, "alpha", 0.05, "SPRT false-positive rate")
	fs.Float64Var(&f.beta, "beta", 0.05, "SPRT false-negative rate")
	fs.StringVar(&f.openings, "openings", "", "file with one opening per line in SAN (default: the built-in set)")
	fs.StringVar(&f.pgn, "pgn", "", "write every game to this PGN file")
	fs.IntVar(&f.maxPlies, "max-plies", 400, "adjudicate a draw after this many plies")
	return f
}

func runMatch(e env, args []string) error {
	fs := newFlagSet(e, "match", "[flags]", "Play two UCI engines against each other and measure the difference:\ngame pairs with colors swapped, Elo with 95% error bars and an optional SPRT.\nEngine 1 is the one measured, engine 2 the baseline.")
	flags := defineMatchFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	config, closePGN, err := flags.config(e)
	if err != nil {
		return err
	}
	defer closePGN()
	return playMatch(e, config)
}

func playMatch(e env, config match.Config) error {
	report := func(summary match.Summary) { fmt.Fprintln(e.out, summary) }
	summary, err := match.NewTournament(config, match.StartUCIPlayer, report).Run(e.ctx)
	if err != nil {
		return err
	}
	elo, margin := summary.Pairs.EloEstimate()
	fmt.Fprintf(e.out, "\n%s vs %s: Elo %+.1f ± %.1f over %d games, %s\n", config.Engines[0].Name, config.Engines[1].Name,
		elo, margin, 2*summary.Pairs.Pairs(), summary.Verdict)
	return nil
}

// config turns the flags into a match configuration; closePGN closes the
// PGN file, if one was opened.
func (f *matchFlags) config(e env) (config match.Config, closePGN func(), err error) {
	closePGN = func() {}
	config = match.Config{Engines: f.engines, Concurrency: f.concurrency, MaxPairs: f.pairs, MaxPlies: f.maxPlies}
	if config.TimeControl, err = match.ParseTimeControl(f.timeControl); err != nil {
		return config, closePGN, usagef("%v", err)
	}
	if config.SPRT, err = f.parseSPRT(); err != nil {
		return config, closePGN, err
	}
	if config.Openings, err = f.loadOpenings(e); err != nil {
		return config, closePGN, err
	}
	if f.pgn != "" {
		file, err := os.Create(f.pgn)
		if err != nil {
			return config, closePGN, err
		}
		config.PGN, closePGN = file, func() { _ = file.Close() }
	}
	return config, closePGN, nil
}

func (f *matchFlags) parseSPRT() (*match.SPRT, error) {
	if f.sprt == "" {
		return nil, nil
	}
	elo0Text, elo1Text, ok := strings.Cut(f.sprt, ",")
	elo0, err0 := strconv.ParseFloat(elo0Text, 64)
	elo1, err1 := strconv.ParseFloat(elo1Text, 64)
	if !ok || err0 != nil || err1 != nil || elo1 <= elo0 {
		return nil, usagef("invalid --sprt %q: expected elo0,elo1 with elo0 < elo1, like 0,10", f.sprt)
	}
	if f.alpha <= 0 || f.alpha >= 1 || f.beta <= 0 || f.beta >= 1 {
		return nil, usagef("invalid --alpha %g or --beta %g: expected rates between 0 and 1, like 0.05", f.alpha, f.beta)
	}
	return &match.SPRT{Elo0: elo0, Elo1: elo1, Alpha: f.alpha, Beta: f.beta}, nil
}

func (f *matchFlags) loadOpenings(e env) ([]match.Opening, error) {
	if f.openings == "" {
		return match.BuiltinOpenings(), nil
	}
	text, err := readInput(e, f.openings)
	if err != nil {
		return nil, err
	}
	return match.ParseOpenings(text)
}
