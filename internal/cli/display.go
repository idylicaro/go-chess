package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

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

// minSideBySide is the terminal width from which the panel goes beside the
// board; below it the panel is printed under the board.
const minSideBySide = 62

// printPosition writes the board, with the last move and a king in check
// highlighted, next to the panel. opts carries the style, the orientation
// and any selection; last may be nil.
func (e env) printPosition(game *chess.Game, opts render.Options, panel render.Panel, last *chess.Move) {
	opts.LastMove = last
	opts.Check = render.CheckSquare(game)
	panel.Status = render.Status(game)
	if game.HasTimer() {
		white, _ := game.GetRemainingTime(chess.White)
		black, _ := game.GetRemainingTime(chess.Black)
		panel.Clocks = &render.Clocks{White: white, Black: black, Active: game.GetCurrentTurn(), Stopped: game.IsOver()}
	}
	board := render.Board(game.GetBoard(), opts)
	text := board + panel.String()
	if columns, err := strconv.Atoi(e.getenv("COLUMNS")); err != nil || columns == 0 || columns >= minSideBySide {
		text = render.Beside(board, opts, panel.String(), 3)
	}
	fmt.Fprintln(e.out)
	fmt.Fprint(e.out, text)
}

// moveNumbering says, from the FEN a game started at, which number the first
// move has and whether Black made it.
func moveNumbering(startFEN string) (number int, blackFirst bool) {
	number = 1
	if fields := strings.Fields(startFEN); len(fields) >= 2 {
		blackFirst = fields[1] == "b"
		if len(fields) == 6 {
			if n, err := strconv.Atoi(fields[5]); err == nil && n > 0 {
				number = n
			}
		}
	}
	return number, blackFirst
}
