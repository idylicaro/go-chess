package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/pkg/chess"
)

type screen int

const (
	screenMenu screen = iota
	screenGame
)

// Model is the state of the TUI. It is a value: Update returns the next one.
type Model struct {
	cfg    Config
	screen screen
	game   *chess.Game

	flipped bool

	// Terminal size, known after the first WindowSizeMsg.
	width, height int
}

// New builds the model for a session. The game starts on the first screen
// the player leaves.
func New(cfg Config) (Model, error) {
	if cfg.FEN == "" {
		cfg.FEN = chess.StartingFEN
	}
	game, err := chess.ParseFEN(cfg.FEN)
	if err != nil {
		return Model{}, fmt.Errorf("invalid starting position: %w", err)
	}
	return Model{cfg: cfg, game: game}, nil
}

// Init has nothing to start.
func (m Model) Init() tea.Cmd { return nil }

// Update handles a message: a key, a resize or, later, a tick or an engine reply.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key := msg.String(); key {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	switch m.screen {
	case screenMenu:
		switch msg.String() {
		case "enter", "n", " ":
			m.screen = screenGame
		case "esc":
			return m, tea.Quit
		}
	case screenGame:
		switch msg.String() {
		case "esc":
			m.screen = screenMenu
		case "f":
			m.flipped = !m.flipped
		}
	}
	return m, nil
}
