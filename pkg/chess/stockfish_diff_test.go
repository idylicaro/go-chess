//go:build stockfish

package chess_test

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Differential test against Stockfish (lab #17). For positions reached by
// random games it compares the legal move list (perft 1 divide) and the perft
// 2 node count with Stockfish's.
//
//	STOCKFISH=/path/to/stockfish STOCKFISH_POSITIONS=100000 \
//	  go test -tags stockfish -run TestAgainstStockfish -timeout 0 ./pkg/chess

type stockfish struct {
	in  io.WriteCloser
	out *bufio.Scanner
	cmd *exec.Cmd
}

func startStockfish(t *testing.T) *stockfish {
	path := os.Getenv("STOCKFISH")
	if path == "" {
		path = "stockfish"
	}
	cmd := exec.Command(path)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("stockfish not available: %v", err)
	}
	sf := &stockfish{in: in, out: bufio.NewScanner(out), cmd: cmd}
	sf.send("uci")
	sf.readUntil("uciok", nil)
	return sf
}

func (s *stockfish) send(command string) { fmt.Fprintln(s.in, command) }

// readUntil feeds every line to visit until a line starts with prefix.
func (s *stockfish) readUntil(prefix string, visit func(string)) {
	for s.out.Scan() {
		line := s.out.Text()
		if strings.HasPrefix(line, prefix) {
			return
		}
		if visit != nil {
			visit(line)
		}
	}
}

func (s *stockfish) close() {
	s.send("quit")
	_ = s.cmd.Wait()
}

// divide returns Stockfish's perft(depth) split by first move, plus the total.
func (s *stockfish) divide(fen string, depth int) (map[string]int, int) {
	s.send("position fen " + fen)
	s.send("go perft " + strconv.Itoa(depth))
	moves := map[string]int{}
	total := 0
	s.readUntil("Nodes searched", func(line string) {
		move, count, ok := strings.Cut(line, ": ")
		if !ok {
			return
		}
		n, err := strconv.Atoi(strings.TrimSpace(count))
		if err == nil {
			moves[move] = n
			total += n
		}
	})
	return moves, total
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestAgainstStockfish(t *testing.T) {
	positions := 20000
	if v := os.Getenv("STOCKFISH_POSITIONS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatal(err)
		}
		positions = n
	}
	seed := int64(17)
	if v := os.Getenv("STOCKFISH_SEED"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = n
	}
	sf := startStockfish(t)
	defer sf.close()
	rng := rand.New(rand.NewSource(seed))

	checked, divergences := 0, 0
	report := func(format string, args ...any) {
		divergences++
		t.Errorf(format, args...)
		if divergences >= 20 {
			t.Fatal("too many divergences, stopping")
		}
	}

	for checked < positions {
		game := chess.NewGame()
		for ply := 0; ply < 300 && !game.IsOver() && checked < positions; ply++ {
			fen := game.FEN()
			mine1 := game.Divide(1)
			theirs1, _ := sf.divide(fen, 1)
			if len(theirs1) == 0 {
				t.Fatalf("stockfish returned no moves at %q: perft output not parsed", fen)
			}
			if strings.Join(sortedKeys(mine1), " ") != strings.Join(sortedKeys(theirs1), " ") {
				report("legal moves differ at %q\n  ours:   %v\n  theirs: %v", fen, sortedKeys(mine1), sortedKeys(theirs1))
			} else if _, theirs2 := sf.divide(fen, 2); game.Perft(2) != theirs2 {
				report("perft(2) differs at %q: ours %d, stockfish %d", fen, game.Perft(2), theirs2)
			}
			checked++

			moves := game.LegalMoves()
			if len(moves) == 0 {
				break
			}
			if err := game.Move(moves[rng.Intn(len(moves))]); err != nil {
				t.Fatalf("legal move rejected at %q: %v", fen, err)
			}
		}
	}
	t.Logf("compared %d positions, %d divergences (seed %d)", checked, divergences, seed)
}
