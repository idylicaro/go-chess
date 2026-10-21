package render

import (
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Options say how to draw a board.
type Options struct {
	Style Style
	// Flipped draws the board from Black's side.
	Flipped bool

	// Highlights, all optional.
	LastMove *chess.Move      // origin and destination of the last move played
	Check    *chess.Position  // king in check
	Selected *chess.Position  // piece picked up
	Targets  []chess.Position // where the selected piece can go
}

// 256-color palette: board squares, piece colors and highlights.
const (
	lightSquare = 187
	darkSquare  = 107
	whitePiece  = 231
	blackPiece  = 16

	lastLight, lastDark = 229, 185 // squares of the last move
	checkSquare         = 160      // king in check
	selectedSquare      = 117      // piece picked up
	captureSquare       = 174      // piece the selected one can take
	targetMarker        = 22       // dot on an empty square the selected piece can reach
)

// mark is what is drawn on a square besides its piece.
type mark int

const (
	plain mark = iota
	lastMove
	check
	selected
	target // empty square the selected piece can reach
	capture
)

// Glyphs by piece type, indexed White, Black. With color both sides use the
// solid symbols and differ by color; without it White gets the outlines.
var (
	outlineGlyphs = map[common.PieceType]string{common.Pawn: "♙", common.Knight: "♘", common.Bishop: "♗", common.Rook: "♖", common.Queen: "♕", common.King: "♔"}
	solidGlyphs   = map[common.PieceType]string{common.Pawn: "♟", common.Knight: "♞", common.Bishop: "♝", common.Rook: "♜", common.Queen: "♛", common.King: "♚"}
	letters       = map[common.PieceType]string{common.Pawn: "p", common.Knight: "n", common.Bishop: "b", common.Rook: "r", common.Queen: "q", common.King: "k"}
)

// Board draws the board with rank and file labels. White is at the bottom
// unless Options.Flipped.
//
//	fmt.Print(render.Board(game.GetBoard(), render.Options{Style: style}))
func Board(b chess.IBoard, opts Options) string {
	var out strings.Builder
	for row := 0; row < 8; row++ {
		y := row
		if opts.Flipped {
			y = 7 - row
		}
		fmt.Fprintf(&out, "%d ", 8-y)
		for col := 0; col < 8; col++ {
			x := col
			if opts.Flipped {
				x = 7 - col
			}
			out.WriteString(square(b, x, y, opts.Style, opts.markAt(b, x, y)))
		}
		fmt.Fprintf(&out, " %d\n", 8-y)
	}
	out.WriteString("  ")
	for col := 0; col < 8; col++ {
		x := col
		if opts.Flipped {
			x = 7 - col
		}
		fmt.Fprintf(&out, " %c ", 'a'+x)
	}
	return strings.TrimRight(out.String(), " ") + "\n"
}

// markAt decides which highlight a square gets; a king in check beats the
// selection, which beats a target, which beats the last move.
func (o Options) markAt(b chess.IBoard, x, y int) mark {
	at := func(p *chess.Position) bool { return p != nil && p.X == x && p.Y == y }
	switch {
	case at(o.Check):
		return check
	case at(o.Selected):
		return selected
	}
	for _, t := range o.Targets {
		if t.X == x && t.Y == y {
			if piece, err := b.GetPiece(x, y); err == nil && piece != nil {
				return capture
			}
			return target
		}
	}
	if m := o.LastMove; m != nil && (m.FromX == x && m.FromY == y || m.ToX == x && m.ToY == y) {
		return lastMove
	}
	return plain
}

// markers wrap the glyph on a square without color, where the background
// cannot carry the highlight.
var markers = map[mark][2]string{
	lastMove: {"[", "]"}, check: {"<", ">"}, selected: {"{", "}"}, capture: {"(", ")"},
}

// square draws one 3-column square.
func square(b chess.IBoard, x, y int, style Style, m mark) string {
	glyph := " "
	var white bool
	piece, err := b.GetPiece(x, y)
	if err == nil && piece != nil {
		white = piece.GetColor() == chess.White
		glyph = pieceGlyph(piece.GetType(), white, style)
	} else if m == target {
		glyph = "*"
		if style.Unicode {
			glyph = "•"
		}
	} else if !style.Color {
		glyph = "."
	}
	if !style.Color {
		left, right := " ", " "
		if pair, ok := markers[m]; ok {
			left, right = pair[0], pair[1]
		}
		return left + glyph + right
	}
	light := (x+y)%2 == 0
	background := darkSquare
	if light {
		background = lightSquare
	}
	switch m {
	case lastMove:
		background = lastDark
		if light {
			background = lastLight
		}
	case check:
		background = checkSquare
	case selected:
		background = selectedSquare
	case capture:
		background = captureSquare
	}
	foreground := blackPiece
	switch {
	case white || m == check:
		foreground = whitePiece
	case m == target:
		foreground = targetMarker
	}
	return fmt.Sprintf("\x1b[48;5;%d;38;5;%dm %s \x1b[0m", background, foreground, glyph)
}

func pieceGlyph(t common.PieceType, white bool, style Style) string {
	switch {
	case !style.Unicode && white:
		return strings.ToUpper(letters[t])
	case !style.Unicode:
		return letters[t]
	case white && !style.Color:
		return outlineGlyphs[t]
	default:
		return solidGlyphs[t]
	}
}
