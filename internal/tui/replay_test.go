package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/idylicaro/go-chess/internal/pgnread"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// longGame plays pseudo-random legal moves (the seed is fixed, so the game is
// always the same) until it has at least plies moves, and returns it as read
// from its PGN, the way a file would be.
func longGame(tb testing.TB, plies int) pgnread.Game {
	tb.Helper()
	for seed := int64(1); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		game := chess.NewGame()
		for n := 0; n < plies && !game.IsOver(); n++ {
			legal := game.LegalMoves()
			if err := game.Move(legal[rng.Intn(len(legal))]); err != nil {
				tb.Fatal(err)
			}
		}
		if game.IsOver() {
			continue
		}
		text, err := game.PGN(chess.PGNHeaders{Event: "Random", Date: "2026.10.21", White: "Alice", Black: "Bob"})
		if err != nil {
			tb.Fatal(err)
		}
		games, err := pgnread.Parse(text)
		if err != nil {
			tb.Fatal(err)
		}
		return games[0]
	}
	tb.Fatalf("no game of %d plies found", plies)
	return pgnread.Game{}
}

func newReplay(tb testing.TB, g pgnread.Game) Replay {
	tb.Helper()
	r, err := NewReplay(ReplayConfig{Game: g})
	if err != nil {
		tb.Fatal(err)
	}
	return r
}

func send(r Replay, keys ...tea.KeyMsg) Replay {
	for _, key := range keys {
		next, _ := r.Update(key)
		r = next.(Replay)
	}
	return r
}

func key(name string) tea.KeyMsg {
	switch name {
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

// positions are the FENs of the game after 0, 1, ..., n moves, each computed
// by playing the game from the start.
func positions(tb testing.TB, g pgnread.Game) []string {
	tb.Helper()
	game, err := chess.ParseFEN(startFEN(g))
	if err != nil {
		tb.Fatal(err)
	}
	fens := []string{game.FEN()}
	for _, san := range g.Moves {
		move, err := game.ParseSAN(san)
		if err != nil {
			tb.Fatal(err)
		}
		if err := game.Move(move); err != nil {
			tb.Fatal(err)
		}
		fens = append(fens, game.FEN())
	}
	return fens
}

func TestReplayWalksForwardAndBack(t *testing.T) {
	g := longGame(t, 220) // more than 100 moves, as lab 1.2-A asks
	want := positions(t, g)
	r := newReplay(t, g)
	if r.Moves() != len(g.Moves) || r.Moves() < 200 || r.Ply() != 0 || r.FEN() != want[0] {
		t.Fatalf("start: %d plies of %d, FEN %q", r.Ply(), r.Moves(), r.FEN())
	}
	for ply := 1; ply <= r.Moves(); ply++ {
		r = send(r, key("right"))
		if r.Ply() != ply || r.FEN() != want[ply] {
			t.Fatalf("forward to %d: at %d, FEN %q, want %q", ply, r.Ply(), r.FEN(), want[ply])
		}
	}
	for ply := r.Moves() - 1; ply >= 0; ply-- {
		r = send(r, key("left"))
		if r.Ply() != ply || r.FEN() != want[ply] {
			t.Fatalf("back to %d: at %d, FEN %q, want %q", ply, r.Ply(), r.FEN(), want[ply])
		}
	}
}

func TestReplayStopsAtTheEnds(t *testing.T) {
	r := newReplay(t, longGame(t, 30))
	r = send(r, key("left"), key("h"))
	if r.Ply() != 0 {
		t.Errorf("went before the start: %d", r.Ply())
	}
	r = send(r, key("end"), key("right"), key("l"), key("pgdown"))
	if r.Ply() != r.Moves() {
		t.Errorf("went past the end: %d of %d", r.Ply(), r.Moves())
	}
	r = send(r, key("home"), key("pgup"))
	if r.Ply() != 0 {
		t.Errorf("home then pgup: %d", r.Ply())
	}
}

func TestReplayJumps(t *testing.T) {
	g := longGame(t, 120)
	want := positions(t, g)
	r := newReplay(t, g)
	for _, step := range []struct {
		key  string
		want int
	}{{"pgdown", 10}, {"pgdown", 20}, {"pgup", 10}, {"end", len(g.Moves)}, {"pgup", len(g.Moves) - 10}, {"home", 0}, {"j", 10}, {"k", 0}} {
		r = send(r, key(step.key))
		if r.Ply() != step.want || r.FEN() != want[step.want] {
			t.Fatalf("%s: at %d, want %d (FEN ok: %v)", step.key, r.Ply(), step.want, r.FEN() == want[step.want])
		}
	}
}

// Whatever the order of the jumps, the position is the one the game had.
func TestReplayRandomJumpsMatchTheGame(t *testing.T) {
	g := longGame(t, 150)
	want := positions(t, g)
	r := newReplay(t, g)
	rng := rand.New(rand.NewSource(7))
	keys := []string{"right", "left", "pgdown", "pgup", "home", "end"}
	for i := 0; i < 500; i++ {
		r = send(r, key(keys[rng.Intn(len(keys))]))
		if r.FEN() != want[r.Ply()] {
			t.Fatalf("after %d keys the position at ply %d is wrong:\n got %s\nwant %s", i+1, r.Ply(), r.FEN(), want[r.Ply()])
		}
	}
}

func TestReplayViewShowsTheGame(t *testing.T) {
	g := longGame(t, 40)
	r := send(newReplay(t, g), key("right"), key("right"), key("right"))
	v := r.View()
	for _, want := range []string{"Alice vs Bob", "Random", "2026.10.21", "After 2. " + g.Moves[2] + " · 3/40", "1. " + g.Moves[0]} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "2. "+g.Moves[2]+"  ") && strings.Contains(v, g.Moves[3]) {
		t.Errorf("moves after the current one must not show:\n%s", v)
	}
	if start := newReplay(t, g).View(); !strings.Contains(start, "Start · 40 moves") {
		t.Errorf("start status:\n%s", start)
	}
	if end := send(newReplay(t, g), key("end")).View(); !strings.Contains(end, "End · 40 moves") {
		t.Errorf("end status:\n%s", end)
	}
}

func TestReplayFromAFENStartingWithBlack(t *testing.T) {
	games, err := pgnread.Parse("[FEN \"4k3/8/8/8/8/8/4P3/4K3 b - - 0 17\"]\n\n17... Kd7 18. e4 Ke6 *")
	if err != nil {
		t.Fatal(err)
	}
	r := send(newReplay(t, games[0]), key("end"))
	if v := r.View(); !strings.Contains(v, "17. ...") || !strings.Contains(v, "18. e4") {
		t.Errorf("numbering from Black's move:\n%s", v)
	}
	if r.FEN() != "8/8/4k3/8/4P3/8/8/4K3 w - - 1 19" {
		t.Errorf("FEN = %q", r.FEN())
	}
}

func TestReplayRejectsAGameWithAnIllegalMove(t *testing.T) {
	games, _ := pgnread.Parse("1. e4 e5 2. Ke3 *")
	if _, err := NewReplay(ReplayConfig{Game: games[0]}); err == nil || !strings.Contains(err.Error(), "Ke3") {
		t.Errorf("err = %v, want it to name Ke3", err)
	}
	games, _ = pgnread.Parse("[FEN \"junk\"]\n\n1. e4 *")
	if _, err := NewReplay(ReplayConfig{Game: games[0]}); err == nil {
		t.Error("a bad FEN tag should fail")
	}
}

func TestReplayQuitFlipAndSize(t *testing.T) {
	r := newReplay(t, longGame(t, 20))
	for _, name := range []string{"q", "ctrl+c", "esc"} {
		msg := key(name)
		if name == "ctrl+c" {
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		}
		if name == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		}
		if _, cmd := r.Update(msg); !isQuit(cmd) {
			t.Errorf("%s should quit", name)
		}
	}
	flipped := send(r, key("f")).View()
	if !strings.Contains(flipped, "h  g  f  e  d  c  b  a") {
		t.Errorf("f should flip:\n%s", flipped)
	}
	next, _ := r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if next.(Replay).width != 80 {
		t.Error("the window size is kept")
	}
}

func TestReplaySnapshots(t *testing.T) {
	games, err := pgnread.Parse(`[Event "Immortal game"]
[Date "1851.06.21"]
[White "Anderssen"]
[Black "Kieseritzky"]
[Result "1-0"]

1. e4 e5 2. f4 exf4 3. Bc4 Qh4+ 4. Kf1 b5 5. Bxb5 Nf6 6. Nf3 Qh6 7. d3 Nh5 8. Nh4 Qg5
9. Nf5 c6 10. g4 Nf6 11. Rg1 cxb5 12. h4 Qg6 13. h5 Qg5 14. Qf3 Ng8 15. Bxf4 Qf6
16. Nc3 Bc5 17. Nd5 Qxb2 18. Bd6 Bxg1 19. e5 Qxa1+ 20. Ke2 Na6 21. Nxg7+ Kd8
22. Qf6+ Nxf6 23. Be7# 1-0
`)
	if err != nil {
		t.Fatal(err)
	}
	r := newReplay(t, games[0])
	t.Run("start", func(t *testing.T) { golden.RequireEqual(t, []byte(r.View())) })
	middle := send(r, key("pgdown"), key("pgdown"), key("right"), key("right"))
	t.Run("middle", func(t *testing.T) { golden.RequireEqual(t, []byte(middle.View())) })
	t.Run("end", func(t *testing.T) { golden.RequireEqual(t, []byte(send(r, key("end")).View())) })
}

func TestReplayFlowWithTeatest(t *testing.T) {
	g := longGame(t, 110)
	want := positions(t, g)
	model := newReplay(t, g)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(90, 30))
	expect(t, tm, fmt.Sprintf("Start · %d moves", len(g.Moves)))
	tm.Send(key("pgdown"))
	expect(t, tm, fmt.Sprintf("10/%d", len(g.Moves)))
	tm.Send(key("end"))
	expect(t, tm, fmt.Sprintf("End · %d moves", len(g.Moves)))
	tm.Send(key("left"))
	expect(t, tm, fmt.Sprintf("%d/%d", len(g.Moves)-1, len(g.Moves)))
	tm.Send(key("q"))
	final := tm.FinalModel(t).(Replay)
	if final.Ply() != len(g.Moves)-1 || final.FEN() != want[len(g.Moves)-1] {
		t.Errorf("final: ply %d, FEN %q", final.Ply(), final.FEN())
	}
}

// replayLengths are the game lengths, in plies, that the benchmarks run on.
var replayLengths = []int{40, 160, 320}

// BenchmarkReplayStep is one step back and one forward in the middle of a
// game, as the replay does it: undo and play on a live game.
func BenchmarkReplayStep(b *testing.B) {
	for _, plies := range replayLengths {
		b.Run(fmt.Sprintf("plies=%d", plies), func(b *testing.B) {
			r := newReplay(b, longGame(b, plies))
			r = r.stepTo(r.Moves() / 2)
			left, right := key("left"), key("right")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r = send(r, left, right)
			}
		})
	}
}

// BenchmarkReplayStepFromScratch is the same two steps done the naive way:
// build the game again and play every move up to the ply wanted.
func BenchmarkReplayStepFromScratch(b *testing.B) {
	for _, plies := range replayLengths {
		b.Run(fmt.Sprintf("plies=%d", plies), func(b *testing.B) {
			moves := newReplay(b, longGame(b, plies)).moves
			ply := len(moves) / 2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, to := range []int{ply - 1, ply} {
					game := chess.NewGame()
					for _, move := range moves[:to] {
						if err := game.Move(move); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}
