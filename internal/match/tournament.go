// Package match plays engine-versus-engine matches over UCI and measures
// the difference between the two engines: game pairs from a set of
// openings with colors swapped, an Elo estimate with its error bars, and
// an SPRT that stops the match once the result is clear (#43).
package match

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Config describes a match between Engines[0], the one being measured, and
// Engines[1], the baseline.
type Config struct {
	Engines     [2]EngineSpec
	Openings    []Opening
	TimeControl TimeControl
	Concurrency int   // games played at once; at least 1
	MaxPairs    int   // game pairs to play at most
	SPRT        *SPRT // stops early when decided; nil plays MaxPairs
	MaxPlies    int   // a longer game is a draw; 0 means defaultMaxPlies
	PGN         io.Writer
}

const (
	defaultMaxPlies   = 400
	defaultTimeMargin = 100 * time.Millisecond
)

// Summary is the state of a match, reported after every pair and at the end.
type Summary struct {
	Pairs               Pentanomial
	Wins, Losses, Draws int // games, from Engines[0]'s point of view
	Verdict             Verdict
	LLR                 float64
}

// String is one progress line, such as
// "pairs 120  W 45 L 30 D 165  Elo +21.8 ± 19.0  LLR 1.23 (undecided)".
func (s Summary) String() string {
	elo, margin := s.Pairs.EloEstimate()
	return fmt.Sprintf("pairs %d  W %d L %d D %d  Elo %+.1f ± %.1f  LLR %.2f (%s)",
		s.Pairs.Pairs(), s.Wins, s.Losses, s.Draws, elo, margin, s.LLR, s.Verdict)
}

// Tournament runs one match.
type Tournament struct {
	config    Config
	newPlayer PlayerFactory
	referee   referee
	report    func(Summary)
}

// NewTournament prepares a match; report, if not nil, gets every summary.
//
//	tournament := match.NewTournament(config, match.StartUCIPlayer, func(s match.Summary) { fmt.Println(s) })
//	summary, err := tournament.Run(ctx)
func NewTournament(config Config, newPlayer PlayerFactory, report func(Summary)) *Tournament {
	if config.MaxPlies <= 0 {
		config.MaxPlies = defaultMaxPlies
	}
	config.Concurrency = max(config.Concurrency, 1)
	return &Tournament{
		config:    config,
		newPlayer: newPlayer,
		referee:   referee{timeControl: config.TimeControl, timeMargin: defaultTimeMargin, maxPlies: config.MaxPlies, now: time.Now},
		report:    report,
	}
}

// pairResult is the two games of one opening.
type pairResult struct {
	opening int
	games   [2]GameRecord // games[0] has Engines[0] as White
}

// Run plays pairs until MaxPairs, an SPRT verdict or the end of ctx.
func (t *Tournament) Run(ctx context.Context) (Summary, error) {
	if len(t.config.Openings) == 0 {
		return Summary{}, fmt.Errorf("match: no openings, expected at least one")
	}
	results := make(chan pairResult)
	schedule := &pairSchedule{limit: int64(t.config.MaxPairs)}
	errs := t.startWorkers(ctx, schedule, results)
	summary, err := t.collect(results, schedule)
	return summary, errors.Join(err, <-errs)
}

// pairSchedule hands out pair numbers until the limit or a stop.
type pairSchedule struct {
	next    atomic.Int64
	limit   int64
	stopped atomic.Bool
}

func (s *pairSchedule) take() (int, bool) {
	number := s.next.Add(1) - 1
	return int(number), number < s.limit && !s.stopped.Load()
}

// startWorkers runs Concurrency workers; the channel yields their joined
// error once all are done and results is closed.
func (t *Tournament) startWorkers(ctx context.Context, schedule *pairSchedule, results chan<- pairResult) <-chan error {
	var done sync.WaitGroup
	errs := make([]error, t.config.Concurrency)
	for i := range errs {
		done.Add(1)
		go func() {
			defer done.Done()
			errs[i] = t.runWorker(ctx, schedule, results)
		}()
	}
	joined := make(chan error, 1)
	go func() {
		done.Wait()
		close(results)
		joined <- errors.Join(errs...)
	}()
	return joined
}

// runWorker owns one player per engine and plays pairs with them.
func (t *Tournament) runWorker(ctx context.Context, schedule *pairSchedule, results chan<- pairResult) error {
	players, err := t.startPlayers()
	if err != nil {
		schedule.stopped.Store(true)
		return err
	}
	defer closePlayers(players)
	for number, ok := schedule.take(); ok; number, ok = schedule.take() {
		pair, err := t.playPair(ctx, players, number)
		if errors.Is(err, errAborted) {
			return nil
		}
		if err != nil {
			schedule.stopped.Store(true)
			return err
		}
		results <- pair
	}
	return nil
}

func (t *Tournament) startPlayers() ([2]Player, error) {
	var players [2]Player
	for i, spec := range t.config.Engines {
		player, err := t.newPlayer(spec)
		if err != nil {
			closePlayers(players)
			return players, fmt.Errorf("match: starting engine %d (%s): %w", i+1, spec.Path, err)
		}
		players[i] = player
	}
	return players, nil
}

func closePlayers(players [2]Player) {
	for _, player := range players {
		if player != nil {
			_ = player.Close()
		}
	}
}

// playPair plays the pair's opening once with each engine as White.
func (t *Tournament) playPair(ctx context.Context, players [2]Player, number int) (pairResult, error) {
	pair := pairResult{opening: number % len(t.config.Openings)}
	opening := t.config.Openings[pair.opening]
	for i, sides := range [2][2]Player{{players[0], players[1]}, {players[1], players[0]}} {
		record, err := t.referee.play(ctx, sides, opening)
		if err != nil {
			return pairResult{}, err
		}
		pair.games[i] = record
	}
	return pair, nil
}

// collect tallies the pairs as they finish and stops the schedule once the
// SPRT decides.
func (t *Tournament) collect(results <-chan pairResult, schedule *pairSchedule) (Summary, error) {
	var summary Summary
	var pgnErr error
	for pair := range results {
		summary.add(pair)
		if t.config.SPRT != nil {
			summary.LLR, summary.Verdict = t.config.SPRT.LLR(summary.Pairs), t.config.SPRT.Decide(summary.Pairs)
		}
		if summary.Verdict != Undecided {
			schedule.stopped.Store(true)
		}
		pgnErr = errors.Join(pgnErr, t.writePGN(pair))
		if t.report != nil {
			t.report(summary)
		}
	}
	return summary, pgnErr
}

// add counts a pair from Engines[0]'s point of view.
func (s *Summary) add(pair pairResult) {
	points := 0 // half-points of Engines[0] in the pair, 0 to 4
	for i, game := range pair.games {
		ours := ourHalfPoints(game.Result, i == 0)
		points += ours
		switch ours {
		case 2:
			s.Wins++
		case 1:
			s.Draws++
		default:
			s.Losses++
		}
	}
	s.Pairs[points]++
}

// ourHalfPoints is Engines[0]'s score in a game: 2 a win, 1 a draw.
func ourHalfPoints(result chess.Result, oursIsWhite bool) int {
	switch {
	case result == chess.ResultDraw:
		return 1
	case (result == chess.ResultWhiteWins) == oursIsWhite:
		return 2
	}
	return 0
}

func (t *Tournament) writePGN(pair pairResult) error {
	if t.config.PGN == nil {
		return nil
	}
	for i, game := range pair.games {
		white, black := t.config.Engines[i].Name, t.config.Engines[1-i].Name
		headers := chess.PGNHeaders{Event: "go-chess match", Round: fmt.Sprint(pair.opening + 1), White: white, Black: black, TimeControl: t.config.TimeControl.String()}
		text, err := game.Game.PGN(headers)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(t.config.PGN, "%s\n\n", text); err != nil {
			return err
		}
	}
	return nil
}
