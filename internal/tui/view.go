package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("107"))
	helpStyle    = lipgloss.NewStyle().Faint(true)
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("172"))
	frameStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)
	gameStyle    = lipgloss.NewStyle().Padding(1, 2)
)

// Layout of the game screen, which the mouse needs to find the squares: the
// board starts after the left padding and the rank label ("8 "), and below
// the padding, the title, a blank line and the line of captured pieces.
const (
	gamePadX      = 2
	gamePadY      = 1
	boardLabelW   = 2
	squareW       = 3
	headerLines   = 2 // title and a blank line
	materialLines = 1 // above the board
)

// View draws the current screen.
func (m Model) View() string {
	if m.screen == screenGame {
		return gameStyle.Render(m.gameView())
	}
	body := m.menuView()
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}
	return body
}

func (m Model) menuView() string {
	lines := []string{
		titleStyle.Render("♞ go-chess"),
		"",
		"Play chess in the terminal.",
		"",
		"  enter   start a game",
		"  q       quit",
	}
	return frameStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) gameView() string {
	opts := render.Options{
		Style:    m.cfg.Style,
		Flipped:  m.flipped,
		Check:    render.CheckSquare(m.game),
		Selected: m.selected,
		Targets:  m.targets,
		Cursor:   &m.cursor,
		Material: true,
	}
	if n := len(m.history); n > 0 {
		opts.LastMove = &m.history[n-1].move
	}
	moves := make([]string, len(m.history))
	for i, h := range m.history {
		moves[i] = h.san
	}
	number, blackFirst := firstMove(m.cfg.FEN)
	panel := render.Panel{Moves: moves, FirstMoveNumber: number, BlackFirst: blackFirst, Status: render.Status(m.game)}
	if m.game.HasTimer() {
		white, _ := m.game.GetRemainingTime(chess.White)
		black, _ := m.game.GetRemainingTime(chess.Black)
		panel.Clocks = &render.Clocks{White: white, Black: black, Active: m.game.GetCurrentTurn(), Stopped: m.game.IsOver()}
	}
	position := strings.TrimRight(render.SideBySide(render.Board(m.game.GetBoard(), opts), panel.String(), 3), "\n")
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("♞ go-chess"), "", position, "", m.footer())
}

// footer is the line under the board: the promotion dialog, a message or the keys.
func (m Model) footer() string {
	switch {
	case m.promoting != nil:
		return m.promotionDialog()
	case m.message != "":
		return messageStyle.Render(m.message)
	}
	return helpStyle.Render("arrows move · enter or click select · u undo · f flip · esc menu · q quit")
}

func (m Model) promotionDialog() string {
	var options []string
	for i, option := range promotionPieces {
		label := fmt.Sprintf("[%s] %s", strings.ToUpper(option.key), option.name)
		if i == m.promoting.choice {
			label = lipgloss.NewStyle().Reverse(true).Render(label)
		}
		options = append(options, label)
	}
	return "Promote to: " + strings.Join(options, "  ") + "\n" + helpStyle.Render("press a letter, or ←/→ and enter · esc cancels")
}

// squareAt maps a screen position to the board square drawn there.
func (m Model) squareAt(x, y int) (chess.Position, bool) {
	col := x - gamePadX - boardLabelW
	row := y - gamePadY - headerLines - materialLines
	if col < 0 || row < 0 || col >= 8*squareW || row >= 8 {
		return chess.Position{}, false
	}
	col /= squareW
	if m.flipped {
		col, row = 7-col, 7-row
	}
	return chess.Position{X: col, Y: row}, true
}

// firstMove is the number of the first move of a game started from fen, and
// whether Black made it.
func firstMove(fen string) (number int, blackFirst bool) {
	number = 1
	fields := strings.Fields(fen)
	if len(fields) >= 2 {
		blackFirst = fields[1] == "b"
	}
	if len(fields) == 6 {
		fmt.Sscanf(fields[5], "%d", &number)
	}
	return max(number, 1), blackFirst
}
