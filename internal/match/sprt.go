package match

import (
	"math"
)

// The statistics of a match are kept per game pair, as fishtest does: the
// two games of a pair share an opening with colors swapped, so their
// results are correlated, and counting pairs (the pentanomial model) gives
// honest error bars where counting games would not (#43).

// Pentanomial counts game pairs by the first engine's score in the pair:
// index 0 is 0/2 (both lost) up to index 4, 2/2 (both won).
type Pentanomial [5]int

// pairScores are the per-game scores of each Pentanomial bucket.
var pairScores = [5]float64{0, 0.25, 0.5, 0.75, 1}

// regularization is a tiny pseudo-count added to every bucket so the
// variance is never zero after a few one-sided pairs (as in fishtest).
const regularization = 1e-3

// Pairs is the number of pairs played.
func (p Pentanomial) Pairs() int {
	return p[0] + p[1] + p[2] + p[3] + p[4]
}

// meanAndVariance are the mean per-game score and its variance per pair.
func (p Pentanomial) meanAndVariance() (mean, variance float64) {
	total := 0.0
	for i, count := range p {
		weight := float64(count) + regularization
		total += weight
		mean += weight * pairScores[i]
	}
	mean /= total
	for i, count := range p {
		deviation := pairScores[i] - mean
		variance += (float64(count) + regularization) * deviation * deviation
	}
	return mean, variance / total
}

// SPRT is a sequential probability ratio test between H0, "the first
// engine is elo0 stronger", and H1, "it is elo1 stronger", with the error
// rates alpha (accepting H1 when H0 holds) and beta (the reverse).
type SPRT struct {
	Elo0, Elo1, Alpha, Beta float64
}

// Verdict is the state of an SPRT.
type Verdict int

// Values of Verdict.
const (
	Undecided Verdict = iota
	AcceptH0          // the data favor elo0: the first engine is not elo1 stronger
	AcceptH1          // the data favor elo1: the first engine is more than elo0 stronger
)

// String names the verdict for the match report.
func (v Verdict) String() string {
	switch v {
	case AcceptH0:
		return "H0 accepted"
	case AcceptH1:
		return "H1 accepted"
	default:
		return "undecided"
	}
}

// Bounds are the log-likelihood ratios at which the test stops.
//
//	SPRT{Alpha: 0.05, Beta: 0.05}.Bounds() // -2.94, 2.94
func (s SPRT) Bounds() (lower, upper float64) {
	return math.Log(s.Beta / (1 - s.Alpha)), math.Log((1 - s.Beta) / s.Alpha)
}

// LLR is the log-likelihood ratio of the pairs so far, in the generalized
// SPRT approximation for a normal pair score: N (s1-s0)(2x - s0 - s1) / 2v,
// with s0 and s1 the expected scores at elo0 and elo1 (logistic Elo).
func (s SPRT) LLR(pairs Pentanomial) float64 {
	if pairs.Pairs() == 0 {
		return 0
	}
	mean, variance := pairs.meanAndVariance()
	s0, s1 := expectedScore(s.Elo0), expectedScore(s.Elo1)
	return float64(pairs.Pairs()) * (s1 - s0) * (2*mean - s0 - s1) / (2 * variance)
}

// minimumPairsForVerdict keeps the test from deciding on a handful of
// pairs: the normal approximation behind LLR needs a variance estimate,
// and after one or two pairs it is close to zero, which once accepted H1
// after a single won pair.
const minimumPairsForVerdict = 10

// Decide compares the LLR with the bounds.
func (s SPRT) Decide(pairs Pentanomial) Verdict {
	if pairs.Pairs() < minimumPairsForVerdict {
		return Undecided
	}
	lower, upper := s.Bounds()
	switch llr := s.LLR(pairs); {
	case llr >= upper:
		return AcceptH1
	case llr <= lower:
		return AcceptH0
	}
	return Undecided
}

// expectedScore is the score of an engine elo points stronger.
func expectedScore(elo float64) float64 {
	return 1 / (1 + math.Pow(10, -elo/400))
}

// eloOfScore is the inverse of expectedScore.
func eloOfScore(score float64) float64 {
	return -400 * math.Log10(1/score-1)
}

// EloEstimate is the Elo difference the pairs show for the first engine,
// with the half-width of its 95% confidence interval.
//
//	elo, margin := pairs.EloEstimate() // "+12.3 ± 8.1"
func (p Pentanomial) EloEstimate() (elo, margin float64) {
	if p.Pairs() == 0 {
		return 0, math.Inf(1)
	}
	mean, variance := p.meanAndVariance()
	spread := 1.96 * math.Sqrt(variance/float64(p.Pairs()))
	low, high := eloOfScore(max(mean-spread, 1e-6)), eloOfScore(min(mean+spread, 1-1e-6))
	return eloOfScore(mean), (high - low) / 2
}
