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
}

// 256-color palette: board squares, piece colors and highlights.
const (
	lightSquare = 187
	darkSquare  = 107
	whitePiece  = 231
	blackPiece  = 16
	marker      = 238
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
			out.WriteString(square(b, x, y, opts.Style))
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

// square draws one 3-column square.
func square(b chess.IBoard, x, y int, style Style) string {
	glyph := " "
	var white bool
	piece, err := b.GetPiece(x, y)
	hasPiece := err == nil && piece != nil
	if hasPiece {
		white = piece.GetColor() == chess.White
		glyph = pieceGlyph(piece.GetType(), white, style)
	} else if !style.Color {
		glyph = "."
	}
	cell := " " + glyph + " "
	if !style.Color {
		return cell
	}
	background := darkSquare
	if (x+y)%2 == 0 {
		background = lightSquare
	}
	foreground := blackPiece
	if white {
		foreground = whitePiece
	}
	return fmt.Sprintf("\x1b[48;5;%d;38;5;%dm%s\x1b[0m", background, foreground, cell)
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
