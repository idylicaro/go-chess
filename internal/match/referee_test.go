package match

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func testReferee() referee {
	return referee{timeControl: TimeControl{Base: time.Minute}, timeMargin: defaultTimeMargin, maxPlies: defaultMaxPlies, now: time.Now}
}

func TestReferee_PlaysToCheckmate(t *testing.T) {
	white := &scriptedPlayer{script: []string{"e2e4", "f1c4", "d1h5", "h5f7"}}
	black := &scriptedPlayer{script: []string{"e7e5", "b8c6", "g8f6"}}
	record, err := testReferee().play(context.Background(), [2]Player{white, black}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if record.Result != chess.ResultWhiteWins || record.Termination != "Checkmate" {
		t.Errorf("result %s (%s), want 1-0 by checkmate", record.Result, record.Termination)
	}
	if white.newGames.Load() != 1 || black.newGames.Load() != 1 {
		t.Error("players were not told a new game began")
	}
}

func TestReferee_StartsFromTheOpening(t *testing.T) {
	white := &scriptedPlayer{script: []string{"d1h5", "h5f7"}}
	black := &scriptedPlayer{script: []string{"g8f6"}}
	opening := Opening{"e2e4", "e7e5", "f1c4", "b8c6"}
	record, err := testReferee().play(context.Background(), [2]Player{white, black}, opening)
	if err != nil || record.Result != chess.ResultWhiteWins {
		t.Errorf("result %s, %v, want the scholar's mate from the opening", record.Result, err)
	}
}

func TestReferee_IllegalMoveLoses(t *testing.T) {
	record, err := testReferee().play(context.Background(), [2]Player{&firstLegalMovePlayer{}, &illegalMovePlayer{}}, nil)
	if err != nil || record.Result != chess.ResultWhiteWins || !strings.Contains(record.Termination, "illegal move a1a1") {
		t.Errorf("result %s (%s), %v, want Black to lose for an illegal move", record.Result, record.Termination, err)
	}
}

func TestReferee_LossOnTimeAndClockReporting(t *testing.T) {
	clock := &manualClock{current: time.Unix(0, 0)}
	r := referee{timeControl: TimeControl{Base: 10 * time.Second, Increment: time.Second}, timeMargin: defaultTimeMargin, maxPlies: defaultMaxPlies, now: clock.now}
	white := &slowPlayer{clock: clock, thinking: time.Second}
	black := &slowPlayer{clock: clock, thinking: 3 * time.Second}
	record, err := r.play(context.Background(), [2]Player{white, black}, nil)
	if err != nil || record.Result != chess.ResultWhiteWins || !strings.Contains(record.Termination, "Black loses on time") {
		t.Fatalf("result %s (%s), %v, want Black to lose on time", record.Result, record.Termination, err)
	}
	// White spends exactly its increment, so its clock never moves.
	if white.lastClock.WTime != 10*time.Second || white.lastClock.WInc != time.Second {
		t.Errorf("White was last told %+v, want 10s and a 1s increment", white.lastClock)
	}
}

func TestReferee_LongGameIsAdjudicatedADraw(t *testing.T) {
	r := testReferee()
	r.maxPlies = 10
	record, err := r.play(context.Background(), [2]Player{&firstLegalMovePlayer{}, &firstLegalMovePlayer{}}, nil)
	if err != nil || record.Result != chess.ResultDraw || !strings.Contains(record.Termination, "maximum length") {
		t.Errorf("result %s (%s), %v, want a draw at the maximum length", record.Result, record.Termination, err)
	}
}

func TestReferee_CancelledMatchAbortsTheGame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := testReferee().play(ctx, [2]Player{&firstLegalMovePlayer{}, &firstLegalMovePlayer{}}, nil); err != errAborted {
		t.Errorf("err = %v, want errAborted", err)
	}
}

func TestTimeControl_String(t *testing.T) {
	if got := (TimeControl{Base: 10 * time.Second, Increment: 100 * time.Millisecond}).String(); got != "10+0.1" {
		t.Errorf("String = %q, want 10+0.1", got)
	}
}

func TestParseTimeControl(t *testing.T) {
	cases := []struct {
		text  string
		want  TimeControl
		fails bool
	}{
		{"10+0.1", TimeControl{Base: 10 * time.Second, Increment: 100 * time.Millisecond}, false},
		{"60", TimeControl{Base: time.Minute}, false},
		{"0+1", TimeControl{}, true},
		{"5+x", TimeControl{}, true},
		{"fast", TimeControl{}, true},
	}
	for _, tc := range cases {
		got, err := ParseTimeControl(tc.text)
		if got != tc.want || (err != nil) != tc.fails {
			t.Errorf("ParseTimeControl(%q) = %+v, %v", tc.text, got, err)
		}
	}
}
