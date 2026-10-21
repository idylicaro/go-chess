package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/idylicaro/go-chess/internal/render"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("107"))
	helpStyle  = lipgloss.NewStyle().Faint(true)
	frameStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)
)

// View draws the current screen, centered when the terminal size is known.
func (m Model) View() string {
	var body string
	switch m.screen {
	case screenGame:
		body = m.gameView()
	default:
		body = m.menuView()
	}
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
	board := render.Board(m.game.GetBoard(), render.Options{
		Style:    m.cfg.Style,
		Flipped:  m.flipped,
		Check:    render.CheckSquare(m.game),
		Material: true,
	})
	panel := render.Panel{FirstMoveNumber: 1, Status: render.Status(m.game)}
	position := strings.TrimRight(render.SideBySide(board, panel.String(), 3), "\n")
	help := helpStyle.Render("f flip · esc menu · q quit")
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("♞ go-chess"), "", position, "", help)
}
