package match

import (
	"math"
	"testing"
)

var standardSPRT = SPRT{Elo0: 0, Elo1: 10, Alpha: 0.05, Beta: 0.05}

func TestSPRT_Bounds(t *testing.T) {
	lower, upper := standardSPRT.Bounds()
	if math.Abs(lower+2.944) > 0.001 || math.Abs(upper-2.944) > 0.001 {
		t.Errorf("Bounds = %.3f, %.3f, want -2.944, 2.944", lower, upper)
	}
}

func TestSPRT_NoPairsIsUndecided(t *testing.T) {
	if llr := standardSPRT.LLR(Pentanomial{}); llr != 0 {
		t.Errorf("LLR of no pairs = %v, want 0", llr)
	}
	if verdict := standardSPRT.Decide(Pentanomial{}); verdict != Undecided {
		t.Errorf("Decide of no pairs = %v, want undecided", verdict)
	}
}

func TestSPRT_ClearWinAcceptsH1AndClearLossAcceptsH0(t *testing.T) {
	winning := Pentanomial{5, 20, 50, 60, 25}
	if verdict := standardSPRT.Decide(winning); verdict != AcceptH1 {
		t.Errorf("Decide(%v) = %v (LLR %.2f), want H1", winning, verdict, standardSPRT.LLR(winning))
	}
	losing := Pentanomial{25, 60, 50, 20, 5}
	if verdict := standardSPRT.Decide(losing); verdict != AcceptH0 {
		t.Errorf("Decide(%v) = %v (LLR %.2f), want H0", losing, verdict, standardSPRT.LLR(losing))
	}
}

func TestSPRT_EvenResultsSitBetweenTheHypotheses(t *testing.T) {
	// Equal strength is H0 exactly; the LLR leans to H0 but slowly.
	even := Pentanomial{10, 40, 100, 40, 10}
	llr := standardSPRT.LLR(even)
	if llr >= 0 || standardSPRT.Decide(even) != Undecided {
		t.Errorf("LLR of even pairs = %.2f (%v), want slightly negative and undecided", llr, standardSPRT.Decide(even))
	}
}

func TestSPRT_LLRGrowsWithTheNumberOfPairs(t *testing.T) {
	small, large := Pentanomial{1, 4, 10, 6, 2}, Pentanomial{4, 16, 40, 24, 8}
	if !(standardSPRT.LLR(large) > standardSPRT.LLR(small) && standardSPRT.LLR(small) > 0) {
		t.Errorf("LLR %.2f for %v and %.2f for four times as many, want both positive and growing", standardSPRT.LLR(small), small, standardSPRT.LLR(large))
	}
}

func TestPentanomial_EloEstimate(t *testing.T) {
	// Every pair 1.5/2 is a 75% score: about +191 Elo, with a tiny margin.
	elo, margin := Pentanomial{0, 0, 0, 100, 0}.EloEstimate()
	if math.Abs(elo-190.85) > 0.5 || margin > 5 {
		t.Errorf("EloEstimate = %.2f ± %.2f, want about 190.85 with a small margin", elo, margin)
	}
	elo, margin = Pentanomial{10, 40, 100, 40, 10}.EloEstimate()
	if math.Abs(elo) > 0.01 || margin < 5 || margin > 50 {
		t.Errorf("EloEstimate of even pairs = %.2f ± %.2f, want 0 with a margin of some Elo", elo, margin)
	}
	if _, margin := (Pentanomial{}).EloEstimate(); !math.IsInf(margin, 1) {
		t.Errorf("margin with no pairs = %v, want +Inf", margin)
	}
}

func TestExpectedScoreAndBack(t *testing.T) {
	for _, elo := range []float64{-300, -10, 0, 35, 400} {
		if got := eloOfScore(expectedScore(elo)); math.Abs(got-elo) > 1e-9 {
			t.Errorf("eloOfScore(expectedScore(%v)) = %v", elo, got)
		}
	}
}

func TestSPRT_FewPairsNeverDecide(t *testing.T) {
	// One won pair once gave an LLR of 3.8, past the H1 bound.
	for pairs := 1; pairs < minimumPairsForVerdict; pairs++ {
		if verdict := standardSPRT.Decide(Pentanomial{4: pairs}); verdict != Undecided {
			t.Errorf("Decide after %d won pairs = %v, want undecided", pairs, verdict)
		}
	}
	if verdict := standardSPRT.Decide(Pentanomial{4: minimumPairsForVerdict}); verdict != AcceptH1 {
		t.Errorf("Decide after %d won pairs = %v, want H1", minimumPairsForVerdict, verdict)
	}
}
