package match

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// fakeFactory hands out fake players by engine path and remembers them.
type fakeFactory struct {
	mu      sync.Mutex
	started []Player
}

func (f *fakeFactory) start(spec EngineSpec) (Player, error) {
	var player Player
	switch spec.Path {
	case "legal":
		player = &firstLegalMovePlayer{}
	case "illegal":
		player = &illegalMovePlayer{}
	default:
		return nil, errors.New("no such engine")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, player)
	return player, nil
}

func testConfig(first, second string) Config {
	return Config{
		Engines:     [2]EngineSpec{{Name: "first", Path: first}, {Name: "second", Path: second}},
		Openings:    BuiltinOpenings()[:3],
		TimeControl: TimeControl{Base: time.Minute},
		Concurrency: 3,
		MaxPairs:    30,
		MaxPlies:    60,
	}
}

func TestTournament_SPRTStopsAOneSidedMatch(t *testing.T) {
	config := testConfig("legal", "illegal")
	config.SPRT = &SPRT{Elo0: 0, Elo1: 10, Alpha: 0.05, Beta: 0.05}
	factory := &fakeFactory{}
	var reports int
	summary, err := NewTournament(config, factory.start, func(Summary) { reports++ }).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Verdict != AcceptH1 || summary.Losses != 0 || summary.Pairs[4] != summary.Pairs.Pairs() {
		t.Errorf("summary %s, want only won pairs and H1 accepted", summary)
	}
	if summary.Pairs.Pairs() >= config.MaxPairs || reports != summary.Pairs.Pairs() {
		t.Errorf("%d pairs and %d reports, want the SPRT to stop before %d with one report per pair", summary.Pairs.Pairs(), reports, config.MaxPairs)
	}
	for _, player := range factory.started {
		if !player.(interface{ isClosed() bool }).isClosed() {
			t.Error("a player was left running")
		}
	}
}

func TestTournament_PlaysEveryPairWithoutSPRT(t *testing.T) {
	config := testConfig("legal", "legal")
	config.MaxPairs = 4
	var pgn bytes.Buffer
	config.PGN = &pgn
	summary, err := NewTournament(config, (&fakeFactory{}).start, nil).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Pairs.Pairs() != 4 || summary.Wins+summary.Losses+summary.Draws != 8 {
		t.Errorf("summary %s, want 4 pairs of 8 games", summary)
	}
	if games := strings.Count(pgn.String(), "[Event "); games != 8 || !strings.Contains(pgn.String(), `[White "second"]`) {
		t.Errorf("PGN has %d games, want 8 with both engines as White", games)
	}
}

func TestTournament_ReportsAnEngineThatDoesNotStart(t *testing.T) {
	_, err := NewTournament(testConfig("legal", "missing"), (&fakeFactory{}).start, nil).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "engine 2 (missing)") {
		t.Errorf("err = %v, want one naming engine 2", err)
	}
}

func TestSummary_AddCountsFromTheFirstEnginesView(t *testing.T) {
	var summary Summary
	// Engines[0] wins as White and draws as Black: 1.5 of 2.
	summary.add(pairResult{games: [2]GameRecord{{Result: chess.ResultWhiteWins}, {Result: chess.ResultDraw}}})
	// Engines[0] loses as White and wins as Black: 1 of 2.
	summary.add(pairResult{games: [2]GameRecord{{Result: chess.ResultBlackWins}, {Result: chess.ResultBlackWins}}})
	if summary.Wins != 2 || summary.Losses != 1 || summary.Draws != 1 || summary.Pairs != (Pentanomial{0, 0, 1, 1, 0}) {
		t.Errorf("summary = %+v", summary)
	}
}
