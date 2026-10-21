package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// styleFlags are the flags that choose how boards are drawn.
type styleFlags struct {
	noColor, ascii bool
}

func addStyleFlags(fs *flag.FlagSet) *styleFlags {
	f := &styleFlags{}
	fs.BoolVar(&f.noColor, "no-color", false, "print the board without colors")
	fs.BoolVar(&f.ascii, "ascii", false, "draw pieces with letters instead of Unicode symbols")
	return f
}

// style is what the terminal supports, narrowed by the flags.
func (e env) style(f *styleFlags) render.Style {
	style := render.DetectStyle(e.out, e.getenv)
	if f.noColor {
		style.Color = false
	}
	if f.ascii {
		style.Unicode = false
	}
	return style
}

// printPosition writes the state line and the board, with the last move and
// a king in check highlighted. opts carries the style, the orientation and
// any selection; last may be nil.
func printPosition(w io.Writer, game *chess.Game, opts render.Options, last *chess.Move) {
	opts.LastMove = last
	opts.Check = render.CheckSquare(game)
	fmt.Fprint(w, stateLine(game))
	fmt.Fprint(w, render.Board(game.GetBoard(), opts))
}

// stateLine describes the state of the game and, on a timed game, the clocks.
func stateLine(game *chess.Game) string {
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

	line := fmt.Sprintf("\n--- Current State: %s | Turn: %s ---\n", state, turn)
	if game.HasTimer() {
		whiteTime, err1 := game.GetRemainingTime(common.White)
		blackTime, err2 := game.GetRemainingTime(common.Black)
		if err1 == nil && err2 == nil {
			line += fmt.Sprintf("⏰ White: %v | Black: %v\n", formatDuration(whiteTime), formatDuration(blackTime))
		}
	}
	return line
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
