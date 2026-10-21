package match

import (
	"strings"
	"testing"
)

func TestBuiltinOpenings_AllParseAndDiffer(t *testing.T) {
	openings := BuiltinOpenings()
	if len(openings) < 40 {
		t.Fatalf("%d built-in openings, want at least 40", len(openings))
	}
	seen := map[string]bool{}
	for _, opening := range openings {
		key := strings.Join(opening, " ")
		if seen[key] {
			t.Errorf("opening %q is listed twice", key)
		}
		seen[key] = true
	}
}

func TestParseOpenings(t *testing.T) {
	openings, err := ParseOpenings("# comment\n\ne4 e5 Nf3\nd4 d5 c4 dxc4\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(openings) != 2 || strings.Join(openings[1], " ") != "d2d4 d7d5 c2c4 d5c4" {
		t.Errorf("ParseOpenings = %q", openings)
	}
}

func TestParseOpenings_RejectsAnIllegalMoveAndEmptyInput(t *testing.T) {
	if _, err := ParseOpenings("e4 e5\ne4 Ke2"); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("ParseOpenings with Ke2 for Black: err = %v, want one naming line 2", err)
	}
	if _, err := ParseOpenings("# nothing\n"); err == nil {
		t.Error("ParseOpenings of no openings: want an error")
	}
}
