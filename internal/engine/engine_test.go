package engine

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseInfo(t *testing.T) {
	tests := []struct {
		line string
		want Info
		ok   bool
	}{
		{"info depth 12 seldepth 18 multipv 1 score cp 34 nodes 1000 time 250 pv e2e4 e7e5 g1f3",
			Info{Depth: 12, MultiPV: 1, Score: Score{Value: 34}, HasScore: true, Nodes: 1000, Time: 250 * time.Millisecond, PV: []string{"e2e4", "e7e5", "g1f3"}}, true},
		{"info depth 5 score mate -2 pv h5h7", Info{Depth: 5, MultiPV: 1, Score: Score{Mate: true, Value: -2}, HasScore: true, PV: []string{"h5h7"}}, true},
		{"info string NNUE evaluation using nn.nnue", Info{}, false},
		{"info depth 3 currmove e2e4 currmovenumber 1", Info{}, false},
		{"bestmove e2e4", Info{}, false},
		{"", Info{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseInfo(tt.line)
		if ok != tt.ok {
			t.Errorf("ParseInfo(%q) ok = %v, want %v", tt.line, ok, tt.ok)
			continue
		}
		if !ok {
			continue
		}
		if got.Depth != tt.want.Depth || got.Score != tt.want.Score || got.HasScore != tt.want.HasScore ||
			got.Nodes != tt.want.Nodes || got.Time != tt.want.Time || strings.Join(got.PV, " ") != strings.Join(tt.want.PV, " ") {
			t.Errorf("ParseInfo(%q) = %+v, want %+v", tt.line, got, tt.want)
		}
	}
}

func TestScoreString(t *testing.T) {
	if got := (Score{Value: 34}).String(); got != "+0.34" {
		t.Errorf("got %q", got)
	}
	if got := (Score{Value: -120}).String(); got != "-1.20" {
		t.Errorf("got %q", got)
	}
	if got := (Score{Mate: true, Value: 3}).String(); got != "#3" {
		t.Errorf("got %q", got)
	}
}

func TestLimitsCommand(t *testing.T) {
	tests := []struct {
		limits Limits
		want   string
	}{
		{Limits{Depth: 10}, "go depth 10"},
		{Limits{MoveTime: 1500 * time.Millisecond}, "go movetime 1500"},
		{Limits{}, "go infinite"},
		{Limits{WTime: time.Minute, BTime: 30 * time.Second, WInc: time.Second}, "go wtime 60000 btime 30000 winc 1000"},
		{Limits{WTime: -time.Second, BTime: time.Minute}, "go wtime 0 btime 60000"},
	}
	for _, tt := range tests {
		if got := tt.limits.command(); got != tt.want {
			t.Errorf("command() = %q, want %q", got, tt.want)
		}
	}
}

// fake runs a scripted UCI engine on the other end of two pipes. respond
// maps a command to the lines the engine prints for it.
type fake struct {
	commands chan string
}

func newFake(t *testing.T, respond func(command string) []string) (*Engine, *fake) {
	t.Helper()
	toEngine, engineIn := io.Pipe()
	engineOut, fromEngine := io.Pipe()
	f := &fake{commands: make(chan string, 64)}
	go func() {
		defer fromEngine.Close()
		scanner := bufio.NewScanner(toEngine)
		for scanner.Scan() {
			command := scanner.Text()
			f.commands <- command
			if command == "quit" {
				return
			}
			for _, line := range respond(command) {
				_, _ = io.WriteString(fromEngine, line+"\n")
			}
		}
	}()
	e, err := New(engineIn, engineOut, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e, f
}

func basicEngine(command string) []string {
	switch {
	case command == "uci":
		return []string{"id name FakeFish 1", "id author nobody", "uciok"}
	case command == "isready":
		return []string{"readyok"}
	case strings.HasPrefix(command, "go"):
		return []string{
			"info string warming up",
			"info depth 1 score cp 20 pv e2e4",
			"info depth 2 score cp 25 pv e2e4 e7e5",
			"bestmove e2e4 ponder e7e5",
		}
	}
	return nil
}

func TestHandshakeReadsName(t *testing.T) {
	e, _ := newFake(t, basicEngine)
	if e.Name != "FakeFish 1" {
		t.Errorf("Name = %q", e.Name)
	}
}

func TestSearchReportsInfoAndBestMove(t *testing.T) {
	e, f := newFake(t, basicEngine)
	var infos []Info
	res, err := e.Search(context.Background(), "startfen", Limits{Depth: 2}, func(i Info) { infos = append(infos, i) })
	if err != nil {
		t.Fatal(err)
	}
	if res.Best != "e2e4" || res.Ponder != "e7e5" {
		t.Errorf("result = %+v", res)
	}
	if len(infos) != 2 || infos[1].Depth != 2 || infos[1].Score.Value != 25 {
		t.Errorf("infos = %+v", infos)
	}
	var sent []string
	for len(f.commands) > 0 {
		sent = append(sent, <-f.commands)
	}
	got := strings.Join(sent, "|")
	if !strings.Contains(got, "position fen startfen|go depth 2") {
		t.Errorf("commands = %s", got)
	}
}

func TestSearchCancelStopsEngine(t *testing.T) {
	e, f := newFake(t, func(command string) []string {
		switch command {
		case "go infinite":
			return []string{"info depth 1 score cp 10 pv e2e4"} // keeps thinking
		case "stop":
			return []string{"bestmove e2e4"}
		}
		return basicEngine(command)
	})
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan error, 1)
	go func() {
		_, err := e.Search(ctx, "fen", Limits{}, func(Info) { cancel() })
		got <- err
	}()
	select {
	case err := <-got:
		if err != context.Canceled {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Search did not return after cancel")
	}
	var sawStop bool
	for len(f.commands) > 0 {
		sawStop = sawStop || <-f.commands == "stop"
	}
	if !sawStop {
		t.Error("engine never got \"stop\"")
	}
	// The engine is still usable for the next search.
	if _, err := e.Search(context.Background(), "fen", Limits{Depth: 1}, nil); err != nil {
		t.Errorf("search after cancel: %v", err)
	}
}

func TestEngineExitIsAnError(t *testing.T) {
	toEngine, engineIn := io.Pipe()
	engineOut, fromEngine := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(toEngine)
		for scanner.Scan() {
			if scanner.Text() == "uci" {
				_, _ = io.WriteString(fromEngine, "uciok\n")
			}
			if scanner.Text() == "isready" {
				_, _ = io.WriteString(fromEngine, "readyok\n")
			}
			if strings.HasPrefix(scanner.Text(), "go") {
				fromEngine.Close() // engine crashes mid-search
			}
		}
	}()
	e, err := New(engineIn, engineOut, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Search(context.Background(), "fen", Limits{Depth: 1}, nil); err == nil {
		t.Error("expected an error when the engine exits mid-search")
	}
}

func TestStartMissingBinary(t *testing.T) {
	if _, err := Start("/nonexistent/engine"); err == nil {
		t.Error("expected an error for a missing binary")
	}
}

func TestSANAndLine(t *testing.T) {
	start := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	pv := []string{"e2e4", "e7e5", "g1f3", "b8c6"}
	if got := strings.Join(SAN(start, pv), " "); got != "e4 e5 Nf3 Nc6" {
		t.Errorf("SAN = %q", got)
	}
	if got := Line(start, pv); got != "1. e4 e5 2. Nf3 Nc6" {
		t.Errorf("Line = %q", got)
	}
	black := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 7"
	if got := Line(black, []string{"e7e5", "g1f3", "b8c6"}); got != "7... e5 8. Nf3 Nc6" {
		t.Errorf("Line from Black = %q", got)
	}
	// It stops at a move that is not legal, and survives junk.
	if got := Line(start, []string{"e2e4", "e2e4", "e7e5"}); got != "1. e4" {
		t.Errorf("Line with an illegal move = %q", got)
	}
	if got := Line(start, []string{"zz"}); got != "" {
		t.Errorf("Line of junk = %q", got)
	}
	if got := SAN("junk", pv); got != nil {
		t.Errorf("SAN from a bad FEN = %v", got)
	}
}

func TestSearchGameSendsTheMoves(t *testing.T) {
	e, f := newFake(t, basicEngine)
	if _, err := e.SearchGame(context.Background(), "startfen", []string{"e2e4", "e7e5"}, Limits{Depth: 1}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SearchGame(context.Background(), "otherfen", nil, Limits{Depth: 1}, nil); err != nil {
		t.Fatal(err)
	}
	var positions []string
	for len(f.commands) > 0 {
		if command := <-f.commands; strings.HasPrefix(command, "position") {
			positions = append(positions, command)
		}
	}
	want := []string{"position fen startfen moves e2e4 e7e5", "position fen otherfen"}
	if strings.Join(positions, "|") != strings.Join(want, "|") {
		t.Errorf("position commands = %q, want %q", positions, want)
	}
}
