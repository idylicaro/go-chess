package render

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Status says what is happening in a game, for the status bar.
//
//	render.Status(game) // "Black to move, in check"
func Status(game *chess.Game) string {
	turn := game.GetCurrentTurn()
	other := chess.White
	if turn == chess.White {
		other = chess.Black
	}
	switch game.GetState() {
	case chess.Check:
		return fmt.Sprintf("%s to move, in check", turn)
	case chess.Checkmate:
		return fmt.Sprintf("Checkmate, %s wins", other)
	case chess.Stalemate:
		return "Stalemate, draw"
	case chess.Draw:
		return fmt.Sprintf("Draw (%s)", game.DrawReason())
	case chess.ResignedWhite:
		return "White resigned, Black wins"
	case chess.ResignedBlack:
		return "Black resigned, White wins"
	case chess.TimeoutWhite:
		return "White ran out of time, Black wins"
	case chess.TimeoutBlack:
		return "Black ran out of time, White wins"
	default:
		return fmt.Sprintf("%s to move", turn)
	}
}

// Clocks are the time left of each side.
type Clocks struct {
	White, Black time.Duration
	// Active is the side whose clock is running; ignored once Stopped.
	Active  chess.Color
	Stopped bool
}

// Panel is the text beside the board: clocks, the move list and a status bar.
type Panel struct {
	// Clocks is nil on an untimed game.
	Clocks *Clocks
	// Moves are the SAN of every move played, oldest first.
	Moves []string
	// FirstMoveNumber is the number of the first move in Moves, 1 for a game
	// from the standard position, and BlackFirst says Black made it.
	FirstMoveNumber int
	BlackFirst      bool
	Status          string
	// Analysis, if set, adds the engine's evaluation and main line.
	Analysis *Analysis
	// Rows is how many lines of moves fit; older moves scroll off. Zero means 8.
	Rows int
}

// Analysis is what an engine thinks of the position.
type Analysis struct {
	Eval  Eval
	Depth int
	// Line is the main line, as numbered SAN.
	Line  string
	Style Style
	// Pending is true until the engine has said anything about the position.
	Pending bool
}

const (
	panelWidth  = 26
	defaultRows = 8
)

// String draws the panel, one line per row, ending in a newline.
func (p Panel) String() string {
	rule := strings.Repeat("─", panelWidth)
	var lines []string
	if c := p.Clocks; c != nil {
		lines = append(lines, clockLine("Black", c.Black, c, chess.Black), clockLine("White", c.White, c, chess.White), rule)
	}
	lines = append(lines, p.moveLines()...)
	lines = append(lines, rule)
	if a := p.Analysis; a != nil {
		lines = append(lines, a.lines()...)
		lines = append(lines, rule)
	}
	lines = append(lines, p.Status)
	return strings.Join(lines, "\n") + "\n"
}

func clockLine(name string, left time.Duration, c *Clocks, side chess.Color) string {
	marker := "  "
	if !c.Stopped && c.Active == side {
		marker = "▶ "
	}
	return fmt.Sprintf("%s%-6s %*s", marker, name, panelWidth-9, FormatClock(left))
}

// FormatClock formats a time left as mm:ss, or h:mm:ss from an hour on;
// time already run out is 00:00.
func FormatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int(d.Seconds())
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

// moveLines lays the moves out two to a row, "12. Nf3     Nc6", showing the
// last Rows rows.
func (p Panel) moveLines() []string {
	number := max(p.FirstMoveNumber, 1)
	var rows [][3]string // number, White's move, Black's move
	for i, san := range p.Moves {
		whiteMove := (i%2 == 0) != p.BlackFirst
		switch {
		case whiteMove:
			rows = append(rows, [3]string{fmt.Sprintf("%d.", number), san, ""})
		case len(rows) == 0 || rows[len(rows)-1][2] != "":
			rows = append(rows, [3]string{fmt.Sprintf("%d.", number), "...", san}) // Black's move opens the list
		default:
			rows[len(rows)-1][2] = san
		}
		if !whiteMove {
			number++
		}
	}
	limit := p.Rows
	if limit <= 0 {
		limit = defaultRows
	}
	if len(rows) > limit {
		rows = rows[len(rows)-limit:]
	}
	lines := make([]string, 0, limit)
	for _, row := range rows {
		lines = append(lines, strings.TrimRight(fmt.Sprintf("%4s %-9s %s", row[0], row[1], row[2]), " "))
	}
	for len(lines) < limit { // keep the panel height steady as the game grows
		lines = append(lines, "")
	}
	return lines
}

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// width is the number of columns a string takes on screen: its runes,
// without color sequences.
func width(s string) int {
	return utf8.RuneCountInString(ansiSequence.ReplaceAllString(s, ""))
}

// SideBySide puts right next to left, gap columns apart. Both may carry
// color sequences. Lines of left are padded to its widest, so the columns
// line up.
func SideBySide(left, right string, gap int) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(right, "\n"), "\n")
	column := 0
	for _, line := range leftLines {
		column = max(column, width(line))
	}
	var out strings.Builder
	for i := 0; i < max(len(leftLines), len(rightLines)); i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		if r == "" {
			out.WriteString(strings.TrimRight(l, " ") + "\n")
			continue
		}
		out.WriteString(l + strings.Repeat(" ", column-width(l)+gap) + r + "\n")
	}
	return out.String()
}

// maxLineRows is how many rows of the main line the panel shows.
const maxLineRows = 3

func (a Analysis) lines() []string {
	if a.Pending {
		return []string{"Analyzing…"}
	}
	lines := []string{
		fmt.Sprintf("Eval %-7s depth %d", a.Eval, a.Depth),
		EvalBar(a.Eval, panelWidth, a.Style),
	}
	return append(lines, wrapWords(a.Line, panelWidth, maxLineRows)...)
}

// wrapWords breaks text into lines of at most width runes, keeping at most
// rows lines and ending the last with an ellipsis if text was cut.
func wrapWords(text string, width, rows int) []string {
	var lines []string
	current := ""
	words := strings.Fields(text)
	for _, word := range words {
		switch {
		case current == "":
			current = word
		case utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
		if len(lines) == rows {
			lines[rows-1] = truncate(lines[rows-1], width-1) + "…"
			return lines
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}
