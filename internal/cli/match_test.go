package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatch_PlaysPairsBetweenTwoEngines(t *testing.T) {
	t.Setenv("GO_CHESS_FAKE_UCI", "2")
	pgn := filepath.Join(t.TempDir(), "games.pgn")
	code, out, errOut := run(t, "", "match", "--engine1", os.Args[0], "--engine2", os.Args[0], "--name1", "new", "--name2", "base",
		"--pairs", "2", "--concurrency", "2", "--tc", "5+0.1", "--max-plies", "20", "--pgn", pgn)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "pairs 2 ") || !strings.Contains(out, "new vs base: Elo") || !strings.Contains(out, "over 4 games") {
		t.Errorf("output %q, want progress and a final line for 4 games", out)
	}
	written, err := os.ReadFile(pgn)
	if err != nil || strings.Count(string(written), "[Event ") != 4 {
		t.Errorf("PGN file has %d games (%v), want 4", strings.Count(string(written), "[Event "), err)
	}
}

func TestMatch_RejectsBadFlags(t *testing.T) {
	cases := [][]string{
		{"match", "--tc", "fast"},
		{"match", "--sprt", "10,0"},
		{"match", "--sprt", "0,10", "--alpha", "2"},
		{"match", "--option1", "Hash"},
		{"match", "extra"},
	}
	for _, args := range cases {
		if code, _, _ := run(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func TestMatch_ReportsAnEngineThatDoesNotStart(t *testing.T) {
	code, _, errOut := run(t, "", "match", "--engine1", "/no/such/engine", "--pairs", "1", "--concurrency", "1")
	if code != 1 || !strings.Contains(errOut, "/no/such/engine") {
		t.Errorf("exit %d, stderr %q, want 1 naming the engine", code, errOut)
	}
}
