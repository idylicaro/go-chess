package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/engine"
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

	// The clock starts the first time the game screen shows and stops while
	// the menu does. tickGen tells the current chain of ticks from stale ones.
	clockStarted bool
	tickGen      int

	// Playing against an engine: a search runs in a command while thinking;
	// searchGen tells its answer from that of an abandoned search.
	thinking     bool
	engineDown   bool // the engine failed; no more searches until the player takes a move back
	frame        int
	searchGen    int
	cancelSearch context.CancelFunc

	// Analysis mode: the position being analyzed and what the engine said.
	analysisFEN string
	analysis    engine.Info
	hasAnalysis bool

	// Terminal size, known after the first WindowSizeMsg.
	width, height int
}

// New builds the model for a session. The game starts on the first screen
// the player leaves.
func New(cfg Config) (Model, error) {
	if cfg.FEN == "" {
		cfg.FEN = chess.StartingFEN
	}
	var game *chess.Game
	var err error
	if cfg.TimeControl != nil {
		clock := cfg.Clock
		if clock == nil {
			clock = chess.SystemClock{}
		}
		game, err = chess.ParseFENTimed(cfg.FEN, *cfg.TimeControl, clock)
	} else {
		game, err = chess.ParseFEN(cfg.FEN)
	}
	if err != nil {
		return Model{}, fmt.Errorf("invalid starting position: %w", err)
	}
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = defaultTickInterval
	}
	m := Model{cfg: cfg, game: game, cursor: chess.Position{X: 4, Y: 6}} // e2
	if cfg.Engine != nil && !cfg.Analyze && cfg.HumanColor == chess.Black {
		m.flipped, m.cursor = true, chess.Position{X: 4, Y: 1} // e7
	}
	return m, nil
}

// Init has nothing to start.
func (m Model) Init() tea.Cmd { return nil }

// Update handles a message (a key, the mouse, a resize, a tick or an engine
// reply) and, if that leaves the engine to move, starts its search.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	model := next.(Model)
	if model.screen != screenGame || model.promoting != nil {
		return model, cmd
	}
	model, search := model.startSearch()
	model, analysis := model.syncAnalysis()
	return model, tea.Batch(cmd, search, analysis)
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tickMsg:
		return m.updateTick(msg)
	case engineMoveMsg:
		return m.updateEngineMove(msg)
	case spinMsg:
		return m.updateSpin(msg)
	case analysisMsg:
		return m.updateAnalysis(msg)
	}
	return m, nil
}

// tickMsg wakes the model to redraw the clocks and look for a flag.
type tickMsg struct{ gen int }

func (m Model) tick() tea.Cmd {
	gen := m.tickGen
	return tea.Tick(m.cfg.TickInterval, func(time.Time) tea.Msg { return tickMsg{gen} })
}

func (m Model) updateTick(msg tickMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.tickGen || m.screen != screenGame || !m.game.HasTimer() || m.game.IsOver() {
		return m, nil
	}
	if m.game.CheckTimeout() {
		m = m.stopSearch().deselect()
		m.promoting = nil
		m.message = render.Status(m.game)
		return m, nil
	}
	return m, m.tick()
}

// enterGame shows the game screen and starts or resumes its clock.
func (m Model) enterGame() (Model, tea.Cmd) {
	m.screen = screenGame
	if !m.game.HasTimer() {
		return m, nil
	}
	if m.clockStarted {
		_ = m.game.ResumeTimer()
	} else {
		_ = m.game.StartTimer()
		m.clockStarted = true
	}
	m.tickGen++
	return m, m.tick()
}

// leaveGame goes back to the menu and stops the clock.
func (m Model) leaveGame() Model {
	m = m.stopSearch() // it starts again when the player comes back
	m.analysisFEN = ""
	m.screen = screenMenu
	if m.game.HasTimer() && !m.game.IsOver() {
		_ = m.game.PauseTimer()
	}
	return m
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.promoting != nil { // q is the queen here, not "quit"
		return m.updatePromotion(msg), nil
	}
	key := msg.String()
	if key == "ctrl+c" || key == "q" {
		m.stopSearch()
		return m, tea.Quit
	}
	if m.screen == screenMenu {
		switch key {
		case "enter", "n", " ":
			return m.enterGame()
		case "esc":
			return m, tea.Quit
		}
		return m, nil
	}
	return m.updateGameKey(key)
}

func (m Model) updateGameKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "left", "h":
		return m.moveCursor(-1, 0), nil
	case "right", "l":
		return m.moveCursor(1, 0), nil
	case "up", "k":
		return m.moveCursor(0, -1), nil
	case "down", "j":
		return m.moveCursor(0, 1), nil
	case "enter", " ":
		return m.activate(m.cursor), nil
	case "esc":
		if m.selected != nil {
			return m.deselect(), nil
		}
		return m.leaveGame(), nil
	case "f":
		m.flipped = !m.flipped
	case "u":
		return m.undo(), nil
	}
	return m, nil
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
	if m.thinking {
		m.message = "Wait for the engine's move."
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
	if errors.Is(err, chess.ErrGameOver) { // the mover's clock ran out first
		m.message = render.Status(m.game)
		return m
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

// undo takes the last move back. Against an engine it takes back the
// engine's reply along with the move it answered, so it is the player's turn
// again; if it is still the engine's turn, because it is thinking or has
// failed, only the player's move goes.
func (m Model) undo() Model {
	m = m.deselect()
	plies := 1
	if m.cfg.Engine != nil && !m.cfg.Analyze && m.game.GetCurrentTurn() == m.cfg.HumanColor {
		plies = 2
	}
	if len(m.history) < plies {
		m.message = "Nothing to take back."
		return m
	}
	m = m.stopSearch()
	m.engineDown = false
	for range plies {
		if err := m.game.Undo(); err != nil {
			m.message = "Nothing to take back."
			return m
		}
		m.history = m.history[:len(m.history)-1]
	}
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
