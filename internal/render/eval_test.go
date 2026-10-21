package render

import (
	"strings"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func TestEvalString(t *testing.T) {
	tests := map[Eval]string{
		{Value: 34}: "+0.34", {Value: -120}: "-1.20", {Value: 0}: "+0.00", {Mate: true, Value: 3}: "#3", {Mate: true, Value: -2}: "#-2",
	}
	for e, want := range tests {
		if got := e.String(); got != want {
			t.Errorf("%+v: %q, want %q", e, got, want)
		}
	}
}

func TestWhiteShare(t *testing.T) {
	if got := (Eval{}).WhiteShare(); got != 0.5 {
		t.Errorf("equal position = %v", got)
	}
	if (Eval{Value: 400}).WhiteShare() < 0.7 || (Eval{Value: 400}).WhiteShare() > 0.78 {
		t.Errorf("+4 pawns = %v, want about three quarters", (Eval{Value: 400}).WhiteShare())
	}
	if (Eval{Value: 100}).WhiteShare()+(Eval{Value: -100}).WhiteShare() != 1 {
		t.Error("the bar must be symmetric")
	}
	if (Eval{Mate: true, Value: 2}).WhiteShare() != 1 || (Eval{Mate: true, Value: -2}).WhiteShare() != 0 || (Eval{Mate: true}).WhiteShare() != 0.5 {
		t.Error("mate should fill the bar for the side that mates")
	}
}

func TestEvalBar(t *testing.T) {
	cases := []struct {
		eval  Eval
		white int
	}{
		{Eval{}, 13}, {Eval{Value: 10000}, 25}, {Eval{Value: -10000}, 1},
		{Eval{Mate: true, Value: 1}, 26}, {Eval{Mate: true, Value: -1}, 0},
	}
	for _, c := range cases {
		bar := EvalBar(c.eval, 26, Style{})
		if strings.Count(bar, "█") != c.white || strings.Count(bar, "░") != 26-c.white {
			t.Errorf("%+v: %q, want %d white cells of 26", c.eval, bar, c.white)
		}
	}
	colored := EvalBar(Eval{}, 26, Style{Color: true})
	if !strings.Contains(colored, "48;5;231m"+strings.Repeat(" ", 13)) || !strings.HasSuffix(colored, "\x1b[0m") {
		t.Errorf("colored bar = %q", colored)
	}
}

func TestPanelWithAnalysis(t *testing.T) {
	golden(t, "panel_analysis", Panel{
		Clocks:          &Clocks{White: 5 * time.Minute, Black: 5 * time.Minute, Active: chess.White},
		Moves:           []string{"e4", "e5"},
		FirstMoveNumber: 1,
		Status:          "White to move",
		Analysis: &Analysis{
			Eval: Eval{Value: 34}, Depth: 18,
			Line: "2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Nb8 10. d4 Nbd7",
		},
	}.String())
}

func TestWrapWords(t *testing.T) {
	if got := wrapWords("aaa bbb ccc", 7, 3); strings.Join(got, "|") != "aaa bbb|ccc" {
		t.Errorf("got %q", got)
	}
	if got := wrapWords("", 7, 3); len(got) != 0 {
		t.Errorf("empty text: %q", got)
	}
	got := wrapWords("aaa bbb ccc ddd eee fff", 7, 2)
	if len(got) != 2 || got[0] != "aaa bbb" || !strings.HasSuffix(got[1], "…") || len([]rune(got[1])) > 7 {
		t.Errorf("a long line should be cut with an ellipsis: %q", got)
	}
	if got := wrapWords("aaa bbb ccc", 7, 2); strings.Join(got, "|") != "aaa bbb|ccc" {
		t.Errorf("text that just fits must not be cut: %q", got)
	}
}
