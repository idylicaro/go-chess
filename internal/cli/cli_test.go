package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain lets the test binary double as a scripted UCI engine: analyze
// tests point --engine at os.Args[0] with GO_CHESS_FAKE_UCI set.
func TestMain(m *testing.M) {
	if os.Getenv("GO_CHESS_FAKE_UCI") == "1" {
		fakeEngine()
		return
	}
	os.Exit(m.Run())
}

func fakeEngine() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		switch {
		case command == "uci":
			fmt.Println("id name FakeFish\nuciok")
		case command == "isready":
			fmt.Println("readyok")
		case strings.HasPrefix(command, "go"):
			fmt.Println("info depth 1 score cp 30 pv e2e4 e7e5")
			fmt.Println("info depth 2 score cp 40 pv e2e4 e7e5 g1f3")
			fmt.Println("bestmove e2e4")
		case command == "quit":
			return
		}
	}
}

// run executes the command line and returns the exit status and both outputs.
func run(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(context.Background(), args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		code, out, _ := run(t, "", args...)
		if code != 0 {
			t.Errorf("%v: exit %d", args, code)
		}
		for _, name := range []string{"play", "analyze", "perft", "fen", "pgn"} {
			if !strings.Contains(out, name) {
				t.Errorf("%v: usage does not list %q:\n%s", args, name, out)
			}
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := run(t, "", "castle")
	if code != 2 || !strings.Contains(errOut, `unknown command "castle"`) {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
}

func TestEverySubcommandHasHelp(t *testing.T) {
	for _, c := range commands {
		code, out, errOut := run(t, "", c.name, "--help")
		if code != 0 {
			t.Errorf("%s --help: exit %d", c.name, code)
		}
		if !strings.Contains(errOut, "Usage: chess "+c.name) || out != "" {
			t.Errorf("%s --help: stderr %q, stdout %q", c.name, errOut, out)
		}
		code, out, _ = run(t, "", "help", c.name)
		if code != 0 || out != "" {
			t.Errorf("help %s: exit %d, stdout %q", c.name, code, out)
		}
	}
}

func TestPerft(t *testing.T) {
	code, out, _ := run(t, "", "perft", "3")
	if code != 0 || !strings.Contains(out, "nodes: 8902") {
		t.Errorf("exit %d, output %q", code, out)
	}
	// Flags may follow the depth.
	code, out, _ = run(t, "", "perft", "2", "--divide")
	if code != 0 || !strings.Contains(out, "e2e4: 20") || !strings.Contains(out, "nodes: 400") {
		t.Errorf("exit %d, output %q", code, out)
	}
	kiwipete := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	_, out, _ = run(t, "", "perft", "--fen", kiwipete, "2")
	if !strings.Contains(out, "nodes: 2039") {
		t.Errorf("kiwipete perft 2: %q", out)
	}
}

func TestPerftUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"perft"}, {"perft", "x"}, {"perft", "-1"}, {"perft", "1", "2"}, {"perft", "--nope", "1"}} {
		if code, _, _ := run(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, errOut := run(t, "", "perft", "--fen", "junk", "1"); code != 1 || errOut == "" {
		t.Errorf("bad FEN: exit %d, stderr %q", code, errOut)
	}
}

func TestFEN(t *testing.T) {
	code, out, _ := run(t, "", "fen")
	if code != 0 || !strings.Contains(out, "Legal moves: 20") || !strings.Contains(out, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") {
		t.Errorf("exit %d, output %q", code, out)
	}
	code, out, _ = run(t, "", "fen", "4k3/8/8/8/8/8/8/4K2R w K - 0 1")
	if code != 0 || !strings.Contains(out, "Legal moves: 15") {
		t.Errorf("exit %d, output %q", code, out)
	}
	if code, _, _ := run(t, "", "fen", "not a fen"); code != 1 {
		t.Errorf("bad FEN: exit %d, want 1", code)
	}
	if code, _, _ := run(t, "", "fen", "a", "b"); code != 2 {
		t.Errorf("two args: exit %d, want 2", code)
	}
}

const scholarsMate = `[Event "Mate"]
[White "Alice"]
[Black "Bob"]

1. e4 e5 2. Bc4 Nc6 3. Qh5 Nf6 4. Qxf7# 1-0
`

func TestPGN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.pgn")
	if err := os.WriteFile(path, []byte(scholarsMate), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, "", "pgn", path)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	for _, want := range []string{"White:", "Alice", "1. e4 e5 2. Bc4 Nc6 3. Qh5 Nf6 4. Qxf7#", "Checkmate", "Result: 1-0"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	// Standard input.
	if code, out, _ := run(t, scholarsMate, "pgn", "-"); code != 0 || !strings.Contains(out, "Alice") {
		t.Errorf("stdin: exit %d, output %q", code, out)
	}
}

func TestPGNErrors(t *testing.T) {
	if code, _, _ := run(t, "", "pgn", filepath.Join(t.TempDir(), "missing.pgn")); code != 1 {
		t.Errorf("missing file: exit %d", code)
	}
	if code, _, errOut := run(t, "1. e4 e5 2. Ke3 *", "pgn", "-"); code != 1 || !strings.Contains(errOut, "Ke3") {
		t.Errorf("illegal move: exit %d, stderr %q", code, errOut)
	}
	if code, _, _ := run(t, scholarsMate, "pgn", "--game", "2", "-"); code != 2 {
		t.Errorf("game out of range: exit %d", code)
	}
	if code, _, _ := run(t, "", "pgn"); code != 2 {
		t.Errorf("no file: exit %d", code)
	}
}

func TestNumberedMovesFromBlack(t *testing.T) {
	got := numberedMoves([]string{"Kd7", "e5", "Ke6"}, "4k3/8/8/8/8/8/8/4K3 b - - 3 17")
	if want := "17... Kd7 18. e5 Ke6"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAnalyzeWithEngine(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "1")
	code, out, errOut := run(t, "", "analyze", "--engine", os.Args[0], "--depth", "2")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	for _, want := range []string{"Engine: FakeFish", "depth  2   +0.40  e4 e5 Nf3", "Best move: e4"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestAnalyzeScoreIsFromWhitesView(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "1")
	// Black to move: the engine's +0.40 for the side to move is -0.40 for White.
	_, out, _ := run(t, "", "analyze", "--engine", os.Args[0], "--depth", "2",
		"--fen", "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1")
	if !strings.Contains(out, "-0.40") {
		t.Errorf("output: %s", out)
	}
}

func TestAnalyzeErrors(t *testing.T) {
	if code, _, _ := run(t, "", "analyze", "--engine", "/nonexistent/engine"); code != 1 {
		t.Errorf("missing engine: exit %d", code)
	}
	if code, _, _ := run(t, "", "analyze", "--fen", "junk"); code != 1 {
		t.Errorf("bad FEN: exit %d", code)
	}
	over := "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1" // stalemate
	if code, _, _ := run(t, "", "analyze", "--fen", over); code != 1 {
		t.Errorf("finished game: exit %d", code)
	}
}

func TestPlayQuitsOnExit(t *testing.T) {
	code, out, _ := run(t, "n\ne2e4\nexit\n", "play")
	if code != 0 || !strings.Contains(out, "Exiting game.") {
		t.Errorf("exit %d, output %q", code, out)
	}
	// With no command at all the game starts too.
	if code, out, _ := run(t, "n\nquit\n"); code != 0 || !strings.Contains(out, "Starting Go Chess Engine") {
		t.Errorf("exit %d, output %q", code, out)
	}
}
