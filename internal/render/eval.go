package render

import (
	"fmt"
	"math"
	"strings"
)

// Eval is an engine evaluation from White's point of view.
type Eval struct {
	// Mate is true when Value counts moves to mate instead of centipawns.
	// A positive value favors White: White gives mate, or is ahead.
	Mate  bool
	Value int
}

// String formats the evaluation as pawns ("+0.34") or as mate in N ("#3"
// when White mates, "#-3" when Black does).
func (e Eval) String() string {
	if e.Mate {
		return fmt.Sprintf("#%d", e.Value)
	}
	return fmt.Sprintf("%+.2f", float64(e.Value)/100)
}

// WhiteShare is the fraction of the bar that belongs to White, from 0 to 1:
// a logistic curve over the centipawn score, so +4 pawns is already about
// three quarters and a mate fills the bar.
func (e Eval) WhiteShare() float64 {
	switch {
	case e.Mate && e.Value > 0:
		return 1
	case e.Mate && e.Value < 0:
		return 0
	case e.Mate:
		return 0.5
	}
	return 1 / (1 + math.Exp(-float64(e.Value)/400))
}

// EvalBar draws an evaluation as a bar of width cells, White's share filled.
// A decided game fills it, but a close one never drops a side below one cell.
func EvalBar(e Eval, width int, style Style) string {
	white := int(math.Round(e.WhiteShare() * float64(width)))
	if !e.Mate {
		white = min(max(white, 1), width-1)
	}
	if style.Color {
		return fmt.Sprintf("\x1b[48;5;231m%s\x1b[48;5;238m%s\x1b[0m", strings.Repeat(" ", white), strings.Repeat(" ", width-white))
	}
	return strings.Repeat("█", white) + strings.Repeat("░", width-white)
}
