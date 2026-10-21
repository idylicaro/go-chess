package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

type screen int

const (
	screenMenu screen = iota
	screenGame
)

// played is a move made in the game and how it reads in SAN.
type played struct {
	move chess.Move
	san  string
}

// promotion is a pawn move waiting for the piece the player picks. The
// candidates are the legal moves, one per piece.
type promotion struct {
	candidates []chess.Move
	choice     int
}

// Model is the state of the TUI. It is a value: Update returns the next one.
type Model struct {
	cfg    Config
	screen screen
	game   *chess.Game

	history []played
	flipped bool

	// The keyboard cursor, the piece picked up and where it can go.
	cursor   chess.Position
	selected *chess.Position
	targets  []chess.Position

	promoting *promotion
	message   string // one line under the board: why a move was refused, how the game ended

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
	return Model{cfg: cfg, game: game, cursor: chess.Position{X: 4, Y: 6}}, nil // e2
}

// Init has nothing to start.
func (m Model) Init() tea.Cmd { return nil }

// Update handles a message: a key, the mouse, a resize or, later, a tick or
// an engine reply.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.promoting != nil { // q is the queen here, not "quit"
		return m.updatePromotion(msg), nil
	}
	key := msg.String()
	if key == "ctrl+c" || key == "q" {
		return m, tea.Quit
	}
	if m.screen == screenMenu {
		switch key {
		case "enter", "n", " ":
			m.screen = screenGame
		case "esc":
			return m, tea.Quit
		}
		return m, nil
	}
	return m.updateGameKey(key), nil
}

func (m Model) updateGameKey(key string) Model {
	switch key {
	case "left", "h":
		return m.moveCursor(-1, 0)
	case "right", "l":
		return m.moveCursor(1, 0)
	case "up", "k":
		return m.moveCursor(0, -1)
	case "down", "j":
		return m.moveCursor(0, 1)
	case "enter", " ":
		return m.activate(m.cursor)
	case "esc":
		if m.selected != nil {
			return m.deselect()
		}
		m.screen = screenMenu
	case "f":
		m.flipped = !m.flipped
	case "u":
		return m.undo()
	}
	return m
}

// moveCursor moves the cursor one square on the screen, whichever way the
// board faces, and stops at the edge.
func (m Model) moveCursor(dx, dy int) Model {
	if m.flipped {
		dx, dy = -dx, -dy
	}
	m.cursor.X = min(max(m.cursor.X+dx, 0), 7)
	m.cursor.Y = min(max(m.cursor.Y+dy, 0), 7)
	return m
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.screen != screenGame || m.promoting != nil ||
		msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	square, ok := m.squareAt(msg.X, msg.Y)
	if !ok {
		return m, nil
	}
	m.cursor = square
	return m.activate(square), nil
}

// activate acts on a square: with a piece picked up it moves there or
// drops the piece, otherwise it picks up the piece on the square.
func (m Model) activate(square chess.Position) Model {
	m.message = ""
	if m.game.IsOver() {
		return m
	}
	if m.selected != nil {
		from := *m.selected
		for _, target := range m.targets {
			if target == square {
				return m.moveTo(from, square)
			}
		}
		m = m.deselect()
		if square == from {
			return m
		}
	}
	return m.pickUp(square)
}

func (m Model) pickUp(square chess.Position) Model {
	piece, err := m.game.GetBoard().GetPiece(square.X, square.Y)
	if err != nil || piece == nil {
		return m
	}
	if piece.GetColor() != m.game.GetCurrentTurn() {
		m.message = fmt.Sprintf("That is %s's piece, and it is %s's turn.", piece.GetColor(), m.game.GetCurrentTurn())
		return m
	}
	targets := render.Targets(m.game, square)
	if len(targets) == 0 {
		m.message = fmt.Sprintf("The %s on %s has no legal move.", piece.GetType(), render.SquareName(square))
		return m
	}
	m.selected, m.targets = &square, targets
	return m
}

func (m Model) deselect() Model {
	m.selected, m.targets = nil, nil
	return m
}

// moveTo plays the legal move from one square to another, or asks which
// piece a pawn promotes to.
func (m Model) moveTo(from, to chess.Position) Model {
	var candidates []chess.Move
	for _, move := range m.game.LegalMoves() {
		if move.FromX == from.X && move.FromY == from.Y && move.ToX == to.X && move.ToY == to.Y {
			candidates = append(candidates, move)
		}
	}
	switch {
	case len(candidates) == 0:
		return m.deselect()
	case len(candidates) > 1: // a promotion: one move per piece
		m.promoting = &promotion{candidates: candidates}
		return m
	}
	return m.play(candidates[0])
}

// play makes a legal move and clears the selection.
func (m Model) play(move chess.Move) Model {
	m = m.deselect()
	m.promoting = nil
	san, err := m.game.SAN(move)
	if err == nil {
		err = m.game.Move(move)
	}
	if err != nil {
		m.message = err.Error()
		return m
	}
	m.history = append(m.history, played{move, san})
	if m.game.IsOver() {
		m.message = render.Status(m.game)
	}
	return m
}

func (m Model) undo() Model {
	m = m.deselect()
	if err := m.game.Undo(); err != nil {
		m.message = "Nothing to take back."
		return m
	}
	m.history = m.history[:len(m.history)-1]
	m.message = ""
	return m
}

// promotionPieces is the order the dialog offers the pieces in.
var promotionPieces = []struct {
	key   string
	name  string
	piece chess.PieceType
}{
	{"q", "Queen", chess.PromoteToQueen},
	{"r", "Rook", chess.PromoteToRook},
	{"b", "Bishop", chess.PromoteToBishop},
	{"n", "Knight", chess.PromoteToKnight},
}

func (m Model) updatePromotion(msg tea.KeyMsg) Model {
	switch key := msg.String(); key {
	case "esc", "ctrl+c":
		m.promoting = nil
		return m.deselect()
	case "left", "h":
		m.promoting.choice = (m.promoting.choice + len(promotionPieces) - 1) % len(promotionPieces)
	case "right", "l":
		m.promoting.choice = (m.promoting.choice + 1) % len(promotionPieces)
	case "enter", " ":
		return m.promoteTo(promotionPieces[m.promoting.choice].piece)
	default:
		for _, option := range promotionPieces {
			if key == option.key {
				return m.promoteTo(option.piece)
			}
		}
	}
	return m
}

func (m Model) promoteTo(piece chess.PieceType) Model {
	for _, move := range m.promoting.candidates {
		if move.Promotion == piece {
			return m.play(move)
		}
	}
	return m
}
