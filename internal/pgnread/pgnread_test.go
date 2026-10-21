package pgnread

import (
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

const sample = `[Event "Test"]
[White "Alice \"Al\" Smith"]
[Black "Bob"]
[Result "1-0"]

1. e4 {best by test} e5 2. Qh5 $1 Nc6 (2... Nf6 3. Qxe5+) 3. Bc4 ; line comment
Nf6?? 4. Qxf7# 1-0

[Event "Second"]

1. d4 d5 *
`

func TestParse(t *testing.T) {
	games, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("got %d games, want 2", len(games))
	}
	first := games[0]
	if first.Tag("White") != `Alice "Al" Smith` || first.Tag("Black") != "Bob" || first.Tag("Missing") != "" {
		t.Errorf("tags = %+v", first.Tags)
	}
	if got := strings.Join(first.Moves, " "); got != "e4 e5 Qh5 Nc6 Bc4 Nf6?? Qxf7#" {
		t.Errorf("moves = %q", got)
	}
	if first.Result != "1-0" {
		t.Errorf("result = %q", first.Result)
	}
	if games[1].Tag("Event") != "Second" || strings.Join(games[1].Moves, " ") != "d4 d5" || games[1].Result != "*" {
		t.Errorf("second game = %+v", games[1])
	}
}

func TestParseMoveNumbersGluedToMoves(t *testing.T) {
	games, err := Parse("1.e4 e5 2.Nf3 2...Nc6 1/2-1/2")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(games[0].Moves, " "); got != "e4 e5 Nf3 Nc6" {
		t.Errorf("moves = %q", got)
	}
}

func TestParseErrors(t *testing.T) {
	for _, text := range []string{
		"", "   \n", `[White "A`, "[White]", "1. e4 {open", "1. e4 (2. d4", "[White \"A\" x",
	} {
		if _, err := Parse(text); err == nil {
			t.Errorf("Parse(%q) succeeded, want an error", text)
		}
	}
}

func TestReplayRoundTripsExportedPGN(t *testing.T) {
	game := chess.NewGame()
	for _, san := range []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6"} {
		move, err := game.ParseSAN(san)
		if err != nil {
			t.Fatal(err)
		}
		if err := game.Move(move); err != nil {
			t.Fatal(err)
		}
	}
	text, err := game.PGN(chess.PGNHeaders{White: "A", Black: "B"})
	if err != nil {
		t.Fatal(err)
	}
	games, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	replay, moves, err := games[0].Replay()
	if err != nil {
		t.Fatal(err)
	}
	if replay.FEN() != game.FEN() || len(moves) != 6 {
		t.Errorf("replay FEN %q (%d moves), want %q", replay.FEN(), len(moves), game.FEN())
	}
}

func TestReplayUsesFENTag(t *testing.T) {
	fen := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	games, err := Parse("[FEN \"" + fen + "\"]\n\n1. e4 Kd7 *")
	if err != nil {
		t.Fatal(err)
	}
	game, _, err := games[0].Replay()
	if err != nil {
		t.Fatal(err)
	}
	if want := "8/3k4/8/8/4P3/8/8/4K3 w - - 1 2"; game.FEN() != want {
		t.Errorf("FEN = %q, want %q", game.FEN(), want)
	}
}

func TestReplayReportsTheBadMove(t *testing.T) {
	games, _ := Parse("1. e4 e5 2. Ke3 *")
	_, _, err := games[0].Replay()
	if err == nil || !strings.Contains(err.Error(), "move 2 (Ke3)") {
		t.Errorf("err = %v, want it to name move 2 (Ke3)", err)
	}
	games, _ = Parse("[FEN \"nonsense\"]\n\n1. e4 *")
	if _, _, err := games[0].Replay(); err == nil {
		t.Error("expected an error for a bad FEN tag")
	}
}

func FuzzParse(f *testing.F) {
	f.Add(sample)
	f.Add("[A \"b\"]\n1. e4 {x (y} ( 1. d4 ) e5 *")
	f.Fuzz(func(t *testing.T, text string) {
		games, err := Parse(text)
		if err == nil && len(games) == 0 {
			t.Fatal("no error and no games")
		}
	})
}
