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
	styleOpts := addStyleFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	cfg := tui.Config{FEN: *fen, Style: e.style(styleOpts)}
	if _, err := tui.New(cfg); err != nil {
		return err
	}
	if !render.IsTerminal(e.out) {
		return errors.New("the TUI needs a terminal; use \"chess play\" for plain output")
	}
	return tui.Run(e.ctx, cfg, e.in, e.out)
}
