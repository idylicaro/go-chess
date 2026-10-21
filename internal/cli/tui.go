package cli

import (
	"errors"

	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/internal/tui"
	"github.com/idylicaro/go-chess/pkg/chess"
)

func runTUI(e env, args []string) error {
	fs := newFlagSet(e, "tui", "[flags]", "Play in a full-screen terminal interface. Press q to quit.")
	fen := fs.String("fen", chess.StartingFEN, "position to start from")
	enginePath := fs.String("engine", "", "UCI engine to play against, like stockfish (default: two players at one keyboard)")
	level := fs.Int("level", -1, "engine strength from 0 to 20 (needs --engine)")
	color := fs.String("color", "white", "your side against the engine: white or black (needs --engine)")
	analyze := fs.Bool("analyze", false, "let the engine analyze the position instead of playing; move both sides by hand (needs --engine)")
	timeText := fs.String("time", "", "time control as minutes+increment seconds, like 5+3 (default: untimed)")
	styleOpts := addStyleFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	if *enginePath == "" && (*level >= 0 || *color != "white") {
		return usagef("--level and --color only apply when playing against an engine; add --engine")
	}
	if *analyze && *enginePath == "" {
		return usagef("--analyze needs an engine to analyze with; add --engine")
	}
	if *analyze && (*level >= 0 || *color != "white") {
		return usagef("--level and --color do not apply to --analyze: the engine analyzes at full strength")
	}
	if *level > maxSkillLevel {
		return usagef("invalid --level %d: expected 0 to %d", *level, maxSkillLevel)
	}
	if *color != "white" && *color != "black" {
		return usagef("invalid --color %q: expected white or black", *color)
	}
	cfg := tui.Config{FEN: *fen, Style: e.style(styleOpts)}
	if *timeText != "" {
		config, err := parseTimeControl(*timeText)
		if err != nil {
			return usageError{err.Error()}
		}
		cfg.TimeControl = &config
	}
	if _, err := tui.New(cfg); err != nil {
		return err
	}
	if !render.IsTerminal(e.out) {
		return errors.New("the TUI needs a terminal; use \"chess play\" for plain output")
	}
	if *enginePath != "" {
		cfg.Analyze = *analyze
		eng, err := openEngine(*enginePath, *level)
		if err != nil {
			return err
		}
		defer eng.Close()
		cfg.Engine, cfg.EngineName = eng, eng.Name
		if *color == "black" {
			cfg.HumanColor = chess.Black
		}
	}
	return tui.Run(e.ctx, cfg, e.in, e.out)
}
