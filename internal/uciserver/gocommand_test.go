package uciserver

import (
	"reflect"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/internal/search"
)

func TestParseGo(t *testing.T) {
	cases := []struct {
		args    []string
		want    goRequest
		ignored []string
	}{
		{nil, goRequest{}, nil},
		{[]string{"depth", "7"}, goRequest{limits: search.Limits{Depth: 7}}, nil},
		{[]string{"movetime", "1500", "nodes", "9000"}, goRequest{limits: search.Limits{MoveTime: 1500 * time.Millisecond, Nodes: 9000}}, nil},
		{[]string{"wtime", "60000", "btime", "-20", "winc", "1000", "binc", "1000", "movestogo", "12"},
			goRequest{limits: search.Limits{WTime: time.Minute, WInc: time.Second, BInc: time.Second, MovesToGo: 12}}, nil},
		{[]string{"infinite"}, goRequest{limits: search.Limits{Infinite: true}, waitForStop: true}, nil},
		{[]string{"ponder", "wtime", "1000"}, goRequest{limits: search.Limits{Infinite: true, WTime: time.Second}, waitForStop: true}, nil},
		{[]string{"searchmoves", "depth", "x", "depth", "3"}, goRequest{limits: search.Limits{Depth: 3}}, []string{"searchmoves", "depth", "x"}},
	}
	for _, tc := range cases {
		got, ignored := parseGo(tc.args)
		if got != tc.want || !reflect.DeepEqual(ignored, tc.ignored) {
			t.Errorf("parseGo(%q) = %+v ignoring %q, want %+v ignoring %q", tc.args, got, ignored, tc.want, tc.ignored)
		}
	}
}

func TestInfoLine(t *testing.T) {
	info := search.Info{Depth: 4, Score: 31, Nodes: 2000, Elapsed: 500 * time.Millisecond}
	if got, want := infoLine(info), "info depth 4 score cp 31 nodes 2000 nps 4000 time 500 pv "; got != want {
		t.Errorf("infoLine = %q, want %q", got, want)
	}
	mate := search.Info{Depth: 3, Score: 32000 - 3}
	if got, want := infoLine(mate), "info depth 3 score mate 2 nodes 0 nps 0 time 0 pv "; got != want {
		t.Errorf("infoLine(mate) = %q, want %q", got, want)
	}
}

func TestParseSetOption(t *testing.T) {
	cases := []struct {
		args        []string
		name, value string
		fails       bool
	}{
		{[]string{"name", "Hash", "value", "64"}, "Hash", "64", false},
		{[]string{"name", "Clear", "Hash"}, "Clear Hash", "", false},
		{[]string{"Hash", "64"}, "", "", true},
	}
	for _, tc := range cases {
		name, value, err := parseSetOption(tc.args)
		if name != tc.name || value != tc.value || (err != nil) != tc.fails {
			t.Errorf("parseSetOption(%q) = %q, %q, %v", tc.args, name, value, err)
		}
	}
}

func TestParsePosition(t *testing.T) {
	cases := []struct {
		args  []string
		fen   string
		fails bool
	}{
		{[]string{"startpos"}, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", false},
		{[]string{"startpos", "moves", "e2e4", "c7c5"}, "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2", false},
		{[]string{"fen", "4k3/P7/8/8/8/8/8/4K3", "w", "-", "-", "0", "1", "moves", "a7a8N"}, "N3k3/8/8/8/8/8/8/4K3 b - - 0 1", false},
		{[]string{"startpos", "e2e4"}, "", true},
		{[]string{"fen", "not", "a", "fen"}, "", true},
		{[]string{"middlegame"}, "", true},
		{nil, "", true},
	}
	for _, tc := range cases {
		board, err := parsePosition(tc.args)
		if (err != nil) != tc.fails || (err == nil && board.FEN() != tc.fen) {
			t.Errorf("parsePosition(%q) = %v, want FEN %q (fails %v)", tc.args, err, tc.fen, tc.fails)
		}
	}
}

func TestParsePosition_RepetitionSurvivesAClaimedDraw(t *testing.T) {
	// The knights repeat the position three times; a GUI may still send it.
	moves := []string{"startpos", "moves", "g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1", "f6g8", "e2e4"}
	if _, err := parsePosition(moves); err != nil {
		t.Errorf("parsePosition after a threefold repetition: %v", err)
	}
}
