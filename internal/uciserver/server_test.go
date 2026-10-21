package uciserver

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/internal/eval"
	"github.com/idylicaro/go-chess/internal/search"
)

// replyTimeout bounds every wait for a reply, so a broken server fails the
// test instead of hanging it.
const replyTimeout = 5 * time.Second

// uciSession runs a Server over pipes, the way a GUI talks to the engine.
type uciSession struct {
	t       *testing.T
	in      *io.PipeWriter
	replies chan string
	done    chan error
}

func startSession(t *testing.T) *uciSession {
	t.Helper()
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	session := &uciSession{t: t, in: inWriter, replies: make(chan string, 1024), done: make(chan error, 1)}
	server := New(Identity{Name: "test engine", Author: "tests"}, outWriter, testSearcher)
	go func() {
		session.done <- server.Run(context.Background(), inReader)
		outWriter.Close()
	}()
	go session.collect(outReader)
	t.Cleanup(func() { inWriter.Close() })
	return session
}

func testSearcher(options search.Options) *search.Searcher {
	options.HashMB = 1 // keep the tests' memory small whatever was asked
	return search.New(eval.Evaluate, options)
}

func (s *uciSession) collect(out io.Reader) {
	lines := bufio.NewScanner(out)
	for lines.Scan() {
		s.replies <- lines.Text()
	}
	close(s.replies)
}

func (s *uciSession) send(command string) {
	s.t.Helper()
	if _, err := io.WriteString(s.in, command+"\n"); err != nil {
		s.t.Fatalf("send %q: %v", command, err)
	}
}

// expect returns the first reply starting with prefix, and the ones before.
func (s *uciSession) expect(prefix string) (string, []string) {
	s.t.Helper()
	var skipped []string
	timeout := time.After(replyTimeout)
	for {
		select {
		case line, open := <-s.replies:
			if !open {
				s.t.Fatalf("output ended waiting for %q after %q", prefix, skipped)
			}
			if strings.HasPrefix(line, prefix) {
				return line, skipped
			}
			skipped = append(skipped, line)
		case <-timeout:
			s.t.Fatalf("no reply starting with %q after %q", prefix, skipped)
		}
	}
}

// expectNoneFor fails if a reply starting with prefix arrives within wait.
func (s *uciSession) expectNoneFor(prefix string, wait time.Duration) {
	s.t.Helper()
	timeout := time.After(wait)
	for {
		select {
		case line := <-s.replies:
			if strings.HasPrefix(line, prefix) {
				s.t.Fatalf("unexpected reply %q", line)
			}
		case <-timeout:
			return
		}
	}
}

func TestServer_Handshake(t *testing.T) {
	session := startSession(t)
	session.send("uci")
	_, before := session.expect("uciok")
	want := []string{"id name test engine", "id author tests", "option name Hash type spin default 16 min 1 max 4096"}
	for i, line := range want {
		if i >= len(before) || before[i] != line {
			t.Fatalf("handshake = %q, want it to start with %q", before, want)
		}
	}
	session.send("isready")
	session.expect("readyok")
}

func TestServer_SearchesTheGivenPosition(t *testing.T) {
	session := startSession(t)
	session.send("position fen r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")
	session.send("go depth 3")
	best, infos := session.expect("bestmove")
	if best != "bestmove h5f7" {
		t.Errorf("got %q, want bestmove h5f7 (mate in one)", best)
	}
	if len(infos) == 0 || !strings.Contains(infos[len(infos)-1], "score mate 1") {
		t.Errorf("info lines %q, want the last one to announce mate 1", infos)
	}
}

func TestServer_PositionWithMoves(t *testing.T) {
	session := startSession(t)
	// After 1.f3 e5 2.g4 Black mates with Qh4.
	session.send("position startpos moves f2f3 e7e5 g2g4")
	session.send("go depth 2")
	if best, _ := session.expect("bestmove"); best != "bestmove d8h4" {
		t.Errorf("got %q, want bestmove d8h4", best)
	}
}

func TestServer_InfiniteSearchWaitsForStop(t *testing.T) {
	session := startSession(t)
	// Mate in one: the search itself ends at once, but must not say so.
	session.send("position fen r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")
	session.send("go infinite")
	session.expectNoneFor("bestmove", 200*time.Millisecond)
	session.send("isready")
	session.expect("readyok")
	session.send("stop")
	if best, _ := session.expect("bestmove"); best != "bestmove h5f7" {
		t.Errorf("got %q, want bestmove h5f7", best)
	}
}

func TestServer_StopEndsALongSearch(t *testing.T) {
	session := startSession(t)
	session.send("position startpos")
	session.send("go wtime 3600000 btime 3600000")
	session.expect("info depth 1")
	start := time.Now()
	session.send("stop")
	session.expect("bestmove")
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("bestmove came %v after stop, want promptly", elapsed)
	}
}

func TestServer_NoLegalMoveIsBestmoveNull(t *testing.T) {
	session := startSession(t)
	session.send("position fen k7/2Q5/1K6/8/8/8/8/8 b - - 0 1")
	session.send("go depth 3")
	if best, _ := session.expect("bestmove"); best != "bestmove 0000" {
		t.Errorf("stalemate: got %q, want bestmove 0000", best)
	}
}

func TestServer_ReportsBadInputAndKeepsThePosition(t *testing.T) {
	session := startSession(t)
	session.send("position startpos moves f2f3 e7e5 g2g4")
	session.send("position startpos moves e2e5")
	if line, _ := session.expect("info string"); !strings.Contains(line, "e2e5") {
		t.Errorf("got %q, want the illegal move named", line)
	}
	session.send("setoption name Hash value lots")
	session.expect("info string setoption: Hash")
	session.send("go depth 2 searchmoves e2e4")
	session.expect("info string go: ignoring")
	if best, _ := session.expect("bestmove"); best != "bestmove d8h4" {
		t.Errorf("got %q, want the earlier position searched (bestmove d8h4)", best)
	}
}

func TestServer_SetOptionNewGameAndQuit(t *testing.T) {
	session := startSession(t)
	session.send("setoption name Hash value 2")
	session.send("setoption name Clear Hash")
	session.send("ucinewgame")
	session.send("go depth 1")
	session.expect("bestmove")
	session.send("quit")
	select {
	case err := <-session.done:
		if err != nil {
			t.Errorf("Run after quit = %v, want nil", err)
		}
	case <-time.After(replyTimeout):
		t.Fatal("Run did not return after quit")
	}
}

// TestServer_DrivenByTheProjectsUCIClient plays the server against the
// client the CLI uses for Stockfish, so both sides of the protocol agree.
func TestServer_DrivenByTheProjectsUCIClient(t *testing.T) {
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	server := New(Identity{Name: "test engine", Author: "tests"}, outWriter, testSearcher)
	go func() {
		_ = server.Run(context.Background(), inReader)
		outWriter.Close()
	}()
	client, err := engine.New(inWriter, outReader, nil, nil)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	defer client.Close()
	result, err := client.Search(context.Background(), "k7/8/2K5/8/8/8/8/7R w - - 0 1", engine.Limits{Depth: 5}, nil)
	if err != nil || result.Best == "" || client.Name != "test engine" {
		t.Errorf("search = %+v, %v (engine %q), want a move from the test engine", result, err, client.Name)
	}
}
