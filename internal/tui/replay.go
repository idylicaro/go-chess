package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/idylicaro/go-chess/internal/pgnread"
	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// pageStep is how many plies PgUp and PgDn move.
const pageStep = 10

// ReplayConfig is what a replay session is set up with.
type ReplayConfig struct {
	// Game is the game read from a PGN file.
	Game pgnread.Game
	// Style says what the terminal can draw.
	Style render.Style
}

// Replay steps through a game that was already played. Going forward plays
// the next move on a live game and going back undoes it, so a step costs one
// move however long the game is, and nothing is replayed from the start.
type Replay struct {
	cfg    ReplayConfig
	game   *chess.Game
	moves  []chess.Move // the whole game
	sans   []string     // how each move reads in SAN
	ply    int          // moves played on game so far
	number int          // number of the first move, 1 unless the game began from a FEN
	black  bool         // Black made the first move

	flipped bool
	width   int
	height  int
}

// NewReplay prepares the replay of a game: it plays the moves once to
// check them and write them in SAN, then starts from the first position.
func NewReplay(cfg ReplayConfig) (Replay, error) {
	walker, err := startingGame(cfg.Game)
	if err != nil {
		return Replay{}, fmt.Errorf("bad FEN tag: %w", err)
	}
	moves := make([]chess.Move, 0, len(cfg.Game.Moves))
	sans := make([]string, 0, len(cfg.Game.Moves))
	for i, token := range cfg.Game.Moves {
		move, err := walker.ParseSAN(token)
		if err == nil {
			var san string
			if san, err = walker.SAN(move); err == nil {
				sans, moves = append(sans, san), append(moves, move)
				err = walker.Move(move)
			}
		}
		if err != nil {
			return Replay{}, fmt.Errorf("move %d (%s): %w", i/2+1, token, err)
		}
	}
	start, _ := startingGame(cfg.Game)
	number, black := firstMove(startFEN(cfg.Game))
	return Replay{cfg: cfg, game: start, moves: moves, sans: sans, number: number, black: black}, nil
}

func startFEN(g pgnread.Game) string {
	if fen := g.Tag("FEN"); fen != "" {
		return fen
	}
	return chess.StartingFEN
}

func startingGame(g pgnread.Game) (*chess.Game, error) {
	return chess.ParseFEN(startFEN(g))
}

// Init has nothing to start.
func (r Replay) Init() tea.Cmd { return nil }

// Update handles the keys that move through the game.
func (r Replay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return r, tea.Quit
		case "right", "l", " ":
			return r.stepTo(r.ply + 1), nil
		case "left", "h":
			return r.stepTo(r.ply - 1), nil
		case "pgdown", "down", "j":
			return r.stepTo(r.ply + pageStep), nil
		case "pgup", "up", "k":
			return r.stepTo(r.ply - pageStep), nil
		case "home", "g":
			return r.stepTo(0), nil
		case "end", "G":
			return r.stepTo(len(r.moves)), nil
		case "f":
			r.flipped = !r.flipped
		}
	}
	return r, nil
}

// stepTo moves to a ply by undoing or playing moves one at a time from
// where it is, and stops at the ends of the game.
func (r Replay) stepTo(ply int) Replay {
	ply = min(max(ply, 0), len(r.moves))
	for r.ply > ply {
		if err := r.game.Undo(); err != nil {
			break
		}
		r.ply--
	}
	for r.ply < ply {
		if err := r.game.Move(r.moves[r.ply]); err != nil {
			break
		}
		r.ply++
	}
	return r
}

// Ply is the number of moves played so far.
func (r Replay) Ply() int { return r.ply }

// Moves is the number of moves in the game.
func (r Replay) Moves() int { return len(r.moves) }

// FEN is the position being shown.
func (r Replay) FEN() string { return r.game.FEN() }

// View draws the position, the moves up to it and the game's tags.
func (r Replay) View() string {
	opts := render.Options{
		Style:    r.cfg.Style,
		Flipped:  r.flipped,
		Check:    render.CheckSquare(r.game),
		Material: true,
	}
	if r.ply > 0 {
		opts.LastMove = &r.moves[r.ply-1]
	}
	panel := render.Panel{Moves: r.sans[:r.ply], FirstMoveNumber: r.number, BlackFirst: r.black, Status: r.status()}
	position := strings.TrimRight(render.Beside(render.Board(r.game.GetBoard(), opts), opts, panel.String(), 3), "\n")
	help := helpStyle.Render("←/→ step · pgup/pgdn ±10 · home/end · f flip · q quit")
	return gameStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(r.header()), "", position, "", help))
}

// header names the players and the event.
func (r Replay) header() string {
	g := r.cfg.Game
	white, black := orDash(g.Tag("White")), orDash(g.Tag("Black"))
	line := fmt.Sprintf("♞ %s vs %s", white, black)
	var extra []string
	for _, name := range []string{"Event", "Date", "Result"} {
		if v := g.Tag(name); v != "" && v != "?" && v != "????.??.??" {
			extra = append(extra, v)
		}
	}
	if len(extra) > 0 {
		line += "  ·  " + strings.Join(extra, " · ")
	}
	return line
}

func orDash(s string) string {
	if s == "" || s == "?" {
		return "—"
	}
	return s
}

// status says where in the game the replay is: "After 12... Nf6 · 24/105".
func (r Replay) status() string {
	switch {
	case r.ply == 0:
		return fmt.Sprintf("Start · %d moves", len(r.moves))
	case r.ply == len(r.moves):
		return fmt.Sprintf("End · %d moves · %s", len(r.moves), strings.TrimSpace(r.resultText()))
	}
	return fmt.Sprintf("After %s · %d/%d", r.plyLabel(r.ply-1), r.ply, len(r.moves))
}

// plyLabel names the move at index i the way a score sheet does: "12. Nf3"
// for White's move and "12... Nc6" for Black's.
func (r Replay) plyLabel(i int) string {
	offset := i
	if r.black {
		offset++ // the game starts with Black, so Black's move 1 is index 0
	}
	number := r.number + offset/2
	if offset%2 == 0 {
		return fmt.Sprintf("%d. %s", number, r.sans[i])
	}
	return fmt.Sprintf("%d... %s", number, r.sans[i])
}

func (r Replay) resultText() string {
	if res := r.cfg.Game.Result; res != "" {
		return res
	}
	return render.Status(r.game)
}

// RunReplay shows the replay on out, reading keys from in, until the player
// quits. It returns ctx.Err() when ctx ends first.
func RunReplay(ctx context.Context, cfg ReplayConfig, in io.Reader, out io.Writer) error {
	model, err := NewReplay(cfg)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen()).Run()
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
