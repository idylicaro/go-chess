package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/internal/pgnread"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// TestMain lets the test binary double as a scripted UCI engine: analyze
// tests point --engine at os.Args[0] with GO_CHESS_FAKE_UCI set.
func TestMain(m *testing.M) {
	if os.Getenv("GO_CHESS_FAKE_UCI") != "" {
		fakeEngine()
		return
	}
	os.Exit(m.Run())
}

// fakeEngine speaks just enough UCI. Mode "1" is a fixed script for analyze;
// mode "2" plays the first legal move of whatever position it is given and
// reports the score in GO_CHESS_FAKE_SCORE centipawns for the side to move.
func fakeEngine() {
	scanner := bufio.NewScanner(os.Stdin)
	var fen string
	for scanner.Scan() {
		command := scanner.Text()
		switch {
		case command == "uci":
			fmt.Println("id name FakeFish\nuciok")
		case command == "isready":
			fmt.Println("readyok")
		case strings.HasPrefix(command, "position fen "):
			fen = strings.TrimPrefix(command, "position fen ")
		case strings.HasPrefix(command, "go") && os.Getenv("GO_CHESS_FAKE_UCI") == "1":
			fmt.Println("info depth 1 score cp 30 pv e2e4 e7e5")
			fmt.Println("info depth 2 score cp 40 pv e2e4 e7e5 g1f3")
			fmt.Println("bestmove e2e4")
		case strings.HasPrefix(command, "go"):
			game, err := chess.ParseFEN(fen)
			if err != nil || len(game.LegalMoves()) == 0 {
				fmt.Println("bestmove (none)")
				continue
			}
			move := game.LegalMoves()[0]
			fmt.Printf("info depth 1 score cp %s pv %s\n", os.Getenv("GO_CHESS_FAKE_SCORE"), move)
			fmt.Printf("bestmove %s\n", move)
		case command == "quit":
			return
		}
	}
}

// run executes the command line and returns the exit status and both outputs.
func run(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	return runEnv(t, nil, stdin, args...)
}

// runEnv is run with environment variables; those not given are unset, so a
// test does not depend on the locale of the machine it runs on.
func runEnv(t *testing.T, vars map[string]string, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = execute(env{
		ctx: context.Background(), in: strings.NewReader(stdin), out: &out, err: &errOut,
		now: time.Now, getenv: func(name string) string { return vars[name] },
	}, args)
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
	code, out, _ := run(t, "e2e4\nexit\n", "play")
	if code != 0 || !strings.Contains(out, "Exiting game.") {
		t.Errorf("exit %d, output %q", code, out)
	}
	// With no command at all the game starts too.
	if code, out, _ := run(t, "quit\n"); code != 0 || !strings.Contains(out, "Starting Go Chess Engine") {
		t.Errorf("exit %d, output %q", code, out)
	}
}

func TestPlayAcceptsSANAndExplainsRejectedMoves(t *testing.T) {
	input := strings.Join([]string{"e4", "e5", "Nf6", "hello", "e2e4", "g8g5", "Nf3", "exit"}, "\n") + "\n"
	code, out, _ := run(t, input, "play")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"Nf6 is not legal",           // knight cannot reach f6 for White
		`"hello" is not a move`,      // malformed
		"e2e4 is not legal",          // pawn already moved: no piece on e2
		"the piece on g8 is Black's", // not your piece... after Nf3? White to move
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

// play runs an untimed game from the scripted lines.
func play(t *testing.T, lines ...string) string {
	t.Helper()
	return playWith(t, nil, lines...)
}

// playWith is play with extra flags.
func playWith(t *testing.T, flags []string, lines ...string) string {
	t.Helper()
	code, out, errOut := run(t, strings.Join(lines, "\n")+"\n", append([]string{"play"}, flags...)...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	return out
}

func TestPlayHelpListsEveryCommand(t *testing.T) {
	out := play(t, "help", "exit")
	for _, c := range gameCommands {
		if !strings.Contains(out, c.name) {
			t.Errorf("help does not list %q", c.name)
		}
	}
}

func TestPlayUndoFENAndMoves(t *testing.T) {
	out := play(t, "e4", "fen", "undo", "fen", "moves", "exit")
	if !strings.Contains(out, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1") {
		t.Errorf("no FEN after e4:\n%s", out)
	}
	if strings.Count(out, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") != 1 {
		t.Errorf("FEN after undo is not the starting one:\n%s", out)
	}
	if !strings.Contains(out, "Nf3 ") || !strings.Contains(out, "e4 ") {
		t.Errorf("moves are not listed in SAN:\n%s", out)
	}
	if out := play(t, "undo", "exit"); !strings.Contains(out, "Error: no move to undo") {
		t.Errorf("undo at the start:\n%s", out)
	}
}

func TestPlayFlipTurnsTheBoardAround(t *testing.T) {
	out := play(t, "flip", "exit")
	if !strings.Contains(out, "h  g  f  e  d  c  b  a") {
		t.Errorf("board not flipped:\n%s", out)
	}
	if out := play(t, "flip", "flip", "exit"); strings.Count(out, "h  g  f  e  d  c  b  a") != 1 {
		t.Errorf("flip twice should show the flipped board once, then restore it:\n%s", out)
	}
}

func TestPlayPGNCommand(t *testing.T) {
	out := play(t, "e4", "e5", "pgn", "exit")
	for _, want := range []string{`[Event "Casual game"]`, `[Result "*"]`, "1. e4 e5 *"} {
		if !strings.Contains(out, want) {
			t.Errorf("pgn output lacks %q:\n%s", want, out)
		}
	}
}

func TestPlayResign(t *testing.T) {
	out := play(t, "e4", "resign")
	if !strings.Contains(out, "Black resigned, White wins") || !strings.Contains(out, "Result: 1-0") {
		t.Errorf("output:\n%s", out)
	}
}

func TestPlayDrawIsAnsweredByTheOpponent(t *testing.T) {
	out := play(t, "e4", "e5", "draw", "accept")
	if !strings.Contains(out, "White offers a draw") || !strings.Contains(out, "Draw (Agreement)") || !strings.Contains(out, "Result: 1/2-1/2") {
		t.Errorf("accept:\n%s", out)
	}
	out = play(t, "draw", "decline", "accept", "exit")
	if !strings.Contains(out, "Draw offer declined.") || !strings.Contains(out, "Error: cannot accept a draw") {
		t.Errorf("decline:\n%s", out)
	}
}

func TestParseTimeControl(t *testing.T) {
	good := map[string]string{"5+3": "300+3", "10": "600", "90+30": "5400+30", "0.5+0": "30"}
	for text, want := range good {
		config, err := parseTimeControl(text)
		if err != nil {
			t.Errorf("%q: %v", text, err)
			continue
		}
		if got := formatTimeControl(config); got != want {
			t.Errorf("%q: formatted %q, want %q", text, got, want)
		}
	}
	for _, text := range []string{"", "x", "0", "-5", "5+", "5+x", "5+-1", "+3"} {
		if _, err := parseTimeControl(text); err == nil {
			t.Errorf("%q: expected an error", text)
		}
	}
}

func TestPlayFlagsStartFromFENAndTimeControl(t *testing.T) {
	out := playWith(t, []string{"--fen", "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1", "--time", "5+3"}, "e4", "fen", "exit")
	if !strings.Contains(out, "4k3/8/8/8/4P3/8/8/4K3 b - e3 0 1") {
		t.Errorf("did not play from the FEN:\n%s", out)
	}
	if !regexp.MustCompile(`White\s+0[45]:\d\d`).MatchString(out) || !regexp.MustCompile(`Black\s+0[45]:\d\d`).MatchString(out) {
		t.Errorf("no clocks:\n%s", out)
	}
}

func TestPlayFlagErrors(t *testing.T) {
	for _, args := range [][]string{
		{"play", "--time", "soon"},
		{"play", "--level", "5"},
		{"play", "--color", "black"},
		{"play", "--engine", "x", "--level", "99"},
		{"play", "--engine", "x", "--color", "red"},
		{"play", "extra"},
	} {
		if code, _, _ := run(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := run(t, "", "play", "--fen", "junk"); code != 1 {
		t.Errorf("bad FEN: exit %d, want 1", code)
	}
	if code, _, _ := run(t, "", "play", "--engine", "/nonexistent/engine"); code != 1 {
		t.Errorf("missing engine: exit %d, want 1", code)
	}
	// Colors and symbols are accepted flags.
	if code, _, _ := run(t, "exit\n", "play", "--no-color", "--ascii"); code != 0 {
		t.Errorf("--no-color --ascii: exit %d", code)
	}
}

func engineFlags(extra ...string) []string {
	return append([]string{"--engine", os.Args[0]}, extra...)
}

func TestPlayAgainstEngine(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	out := playWith(t, engineFlags("--level", "3"), "e4", "pgn", "exit")
	if !strings.Contains(out, "Playing against FakeFish as White.") || !strings.Contains(out, "Engine plays") {
		t.Errorf("no engine reply:\n%s", out)
	}
	if !strings.Contains(out, `[White "Player"]`) || !strings.Contains(out, `[Black "FakeFish"]`) {
		t.Errorf("PGN names:\n%s", out)
	}
}

func TestPlayAgainstEngineAsBlack(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	out := playWith(t, engineFlags("--color", "black"), "undo", "exit")
	if strings.Index(out, "Engine plays") > strings.Index(out, "Enter a move") {
		t.Errorf("the engine, as White, must move before the first prompt:\n%s", out)
	}
	if !strings.Contains(out, "h  g  f  e  d  c  b  a") {
		t.Errorf("board should face Black:\n%s", out)
	}
	if !strings.Contains(out, "the engine has made the only move so far") {
		t.Errorf("undo must not take back only the engine's move:\n%s", out)
	}
}

func TestUndoAgainstEngineTakesBackBothMoves(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	out := playWith(t, engineFlags(), "e4", "undo", "fen", "exit")
	if !strings.Contains(out, "RNBQKBNR w KQkq - 0 1\n") {
		t.Errorf("undo did not restore the start:\n%s", out)
	}
}

func TestDrawOfferToEngine(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	t.Setenv("GO_CHESS_FAKE_SCORE", "0")
	out := playWith(t, engineFlags(), "draw")
	if !strings.Contains(out, "Engine accepts the draw.") || !strings.Contains(out, "Result: 1/2-1/2") {
		t.Errorf("equal position:\n%s", out)
	}
	// The score is for the side to move (the player); +300 means the engine is worse.
	t.Setenv("GO_CHESS_FAKE_SCORE", "300")
	if out := playWith(t, engineFlags(), "draw"); !strings.Contains(out, "Engine accepts the draw.") {
		t.Errorf("player ahead, the engine should accept:\n%s", out)
	}
	t.Setenv("GO_CHESS_FAKE_SCORE", "-300")
	out = playWith(t, engineFlags(), "draw", "exit")
	if !strings.Contains(out, "Engine declines the draw.") || strings.Contains(out, "Game over") {
		t.Errorf("engine ahead should decline:\n%s", out)
	}
}

func TestResignAgainstEngine(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	out := playWith(t, engineFlags("--color", "black"), "resign")
	if !strings.Contains(out, "Black resigned") || !strings.Contains(out, "Result: 1-0") {
		t.Errorf("output:\n%s", out)
	}
}

func TestPGNOutToFileHasHeaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.pgn")
	out := playWith(t, []string{"--time", "5+3", "--pgn-out", path}, "f3", "e5", "g4", "Qh4#")
	if !strings.Contains(out, "PGN saved to "+path) {
		t.Errorf("no confirmation:\n%s", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pgn := string(data)
	for _, want := range []string{`[Result "0-1"]`, `[TimeControl "300+3"]`, `[White "White"]`, "1. f3 e5 2. g4 Qh4# 0-1"} {
		if !strings.Contains(pgn, want) {
			t.Errorf("PGN lacks %q:\n%s", want, pgn)
		}
	}
	if !regexp.MustCompile(`\[Date "\d{4}\.\d{2}\.\d{2}"\]`).MatchString(pgn) {
		t.Errorf("PGN has no date:\n%s", pgn)
	}
	// What was written reads back.
	games, err := pgnread.Parse(pgn)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := games[0].Replay(); err != nil {
		t.Error(err)
	}
}

func TestPGNOutToStdoutAlsoWhenLeavingEarly(t *testing.T) {
	out := playWith(t, []string{"--pgn-out", "-"}, "e4", "exit")
	if !strings.Contains(out, `[Result "*"]`) || !strings.Contains(out, "1. e4 *") || strings.Contains(out, "TimeControl") {
		t.Errorf("output:\n%s", out)
	}
}

func TestPGNOutUnwritable(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "no", "such", "dir", "game.pgn")
	code, _, errOut := run(t, "exit\n", "play", "--pgn-out", bad)
	if code != 1 || !strings.Contains(errOut, "writing PGN") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
}

func TestBoardStyleFollowsTheTerminalAndFlags(t *testing.T) {
	utf8 := map[string]string{"LANG": "en_US.UTF-8"}
	_, out, _ := runEnv(t, utf8, "", "fen")
	if !strings.Contains(out, "♜") || strings.Contains(out, "\x1b[") {
		t.Errorf("UTF-8 locale without a terminal: want Unicode, no color:\n%q", out)
	}
	_, out, _ = runEnv(t, utf8, "", "fen", "--ascii")
	if strings.Contains(out, "♜") || !strings.Contains(out, " r ") {
		t.Errorf("--ascii:\n%s", out)
	}
	_, out, _ = runEnv(t, nil, "", "fen")
	if strings.Contains(out, "♜") {
		t.Errorf("no locale should fall back to ASCII:\n%s", out)
	}
	forced := map[string]string{"LANG": "en_US.UTF-8", "CLICOLOR_FORCE": "1"}
	_, out, _ = runEnv(t, forced, "", "fen")
	if !strings.Contains(out, "\x1b[48;5;") {
		t.Errorf("CLICOLOR_FORCE should color the board:\n%q", out)
	}
	_, out, _ = runEnv(t, forced, "", "fen", "--no-color")
	if strings.Contains(out, "\x1b[") || !strings.Contains(out, "♖") {
		t.Errorf("--no-color should drop the colors and keep Unicode (outline pieces for White):\n%q", out)
	}
	_, out, _ = runEnv(t, forced, "e4\nexit\n", "play", "--no-color", "--ascii")
	if strings.Contains(out, "\x1b[") || strings.Contains(out, "♜") {
		t.Errorf("play --no-color --ascii:\n%q", out)
	}
	_, out, _ = runEnv(t, map[string]string{"NO_COLOR": "1", "CLICOLOR_FORCE": "1"}, "", "fen")
	if strings.Contains(out, "\x1b[") {
		t.Errorf("NO_COLOR must win:\n%q", out)
	}
}

func TestMovesSquareShowsTargets(t *testing.T) {
	out := play(t, "moves g1", "moves e4", "moves a8", "moves z9", "e4", "exit")
	if !strings.Contains(out, "{N}") || strings.Count(out, "*") != 2 {
		t.Errorf("g1 selection (knight with two targets):\n%s", out)
	}
	if !strings.Contains(out, "there is no piece on e4") {
		t.Errorf("empty square:\n%s", out)
	}
	if !strings.Contains(out, "the rook on a8 has no legal move") && !strings.Contains(out, "there is no piece") {
		t.Errorf("blocked piece:\n%s", out)
	}
	if !strings.Contains(out, `"z9" is not a square`) {
		t.Errorf("bad square:\n%s", out)
	}
}

func TestLastMoveAndCheckAreHighlighted(t *testing.T) {
	out := play(t, "e4", "e5", "Qh5", "Nc6", "Qxf7+", "exit")
	if !strings.Contains(out, "[Q]") || !strings.Contains(out, "<k>") {
		t.Errorf("no last-move or check highlight:\n%s", out)
	}
	// After undo the last move is the previous one: Nc6.
	out = play(t, "e4", "e5", "Qh5", "Nc6", "Qxf7+", "undo", "exit")
	blocks := strings.Split(out, "> Enter a move") // what was shown after each prompt
	if last := blocks[len(blocks)-2]; strings.Contains(last, "<k>") || !strings.Contains(last, "[n]") {
		t.Errorf("after undo:\n%s", last)
	}
}

func TestBareSquareIsAPawnMove(t *testing.T) {
	out := play(t, "e4", "fen", "exit")
	if !strings.Contains(out, "4P3") {
		t.Errorf("e4 should move the pawn:\n%s", out)
	}
}

func TestPanelListsMovesAndStatus(t *testing.T) {
	out := play(t, "e4", "e5", "Nf3", "exit")
	for _, want := range []string{"1. e4", "e5", "2. Nf3", "Black to move"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel lacks %q:\n%s", want, out)
		}
	}
	// The panel sits next to the board: a rank and the panel share a line.
	if !regexp.MustCompile(`(?m)^3 .* 3 +2\. Nf3`).MatchString(out) && !regexp.MustCompile(`(?m)^[1-8] .* [1-8] +\s*2\. Nf3`).MatchString(out) {
		t.Errorf("panel is not beside the board:\n%s", out)
	}
}

func TestNarrowTerminalPutsThePanelBelow(t *testing.T) {
	_, out, _ := runEnv(t, map[string]string{"COLUMNS": "40"}, "e4\nexit\n", "play")
	if regexp.MustCompile(`(?m)^[1-8] .* [1-8] +\S`).MatchString(out) {
		t.Errorf("panel beside the board on a 40-column terminal:\n%s", out)
	}
	if !strings.Contains(out, "1. e4") {
		t.Errorf("panel missing:\n%s", out)
	}
}

func TestPGNCommandShowsTheMovesInThePanel(t *testing.T) {
	_, out, _ := run(t, scholarsMate, "pgn", "-")
	if !strings.Contains(out, "Checkmate, White wins") || !strings.Contains(out, "4. Qxf7#") {
		t.Errorf("output:\n%s", out)
	}
}

func TestTUIRequiresATerminal(t *testing.T) {
	code, _, errOut := run(t, "", "tui")
	if code != 1 || !strings.Contains(errOut, "needs a terminal") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	if code, _, _ := run(t, "", "tui", "--fen", "junk"); code != 1 {
		t.Errorf("bad FEN: exit %d", code)
	}
	if code, _, _ := run(t, "", "tui", "--time", "soon"); code != 2 {
		t.Errorf("bad time control: exit %d", code)
	}
	for _, args := range [][]string{
		{"tui", "--level", "5"}, {"tui", "--color", "black"}, {"tui", "--engine", "x", "--level", "99"}, {"tui", "--engine", "x", "--color", "red"},
	} {
		if code, _, _ := run(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := run(t, "", "tui", "extra"); code != 2 {
		t.Errorf("extra argument: exit %d", code)
	}
}
