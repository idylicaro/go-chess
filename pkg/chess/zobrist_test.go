package chess

import (
	"testing"
)

// TestZobrist_IncrementalMatchesRecomputed walks every position three plies
// deep in the reference positions and compares the hash makeMove kept with
// one built from scratch, before and after unmakeMove.
func TestZobrist_IncrementalMatchesRecomputed(t *testing.T) {
	for name, fen := range perftReferenceFENs {
		assertHashTracksMoves(t, name, mustParseFEN(t, fen), 3)
	}
}

func assertHashTracksMoves(t *testing.T, name string, game *Game, depth int) {
	t.Helper()
	if game.hash != game.computeHash() {
		t.Fatalf("%s: incremental hash drifted in %q", name, game.FEN())
	}
	if depth == 0 {
		return
	}
	for _, move := range game.appendLegalMoves(nil) {
		undo := game.makeMove(move)
		assertHashTracksMoves(t, name, game, depth-1)
		game.unmakeMove(move, undo)
	}
}

func TestZobrist_SeparatesSideCastlingAndEnPassant(t *testing.T) {
	fens := []string{
		"r3k2r/8/8/3pP3/8/8/8/R3K2R w KQkq d6 0 1",
		"r3k2r/8/8/3pP3/8/8/8/R3K2R b KQkq - 0 1",
		"r3k2r/8/8/3pP3/8/8/8/R3K2R w Kkq d6 0 1",
		"r3k2r/8/8/3pP3/8/8/8/R3K2R w KQkq - 0 1",
	}
	seen := map[uint64]string{}
	for _, fen := range fens {
		hash := mustParseFEN(t, fen).hash
		if other, clash := seen[hash]; clash {
			t.Errorf("%q and %q share hash %x", fen, other, hash)
		}
		seen[hash] = fen
	}
}

func TestZobrist_TransposedPositionsShareAHash(t *testing.T) {
	viaKnightFirst, viaPawnFirst := NewGame(), NewGame()
	playUCIMoves(t, viaKnightFirst, "g1f3", "g8f6", "e2e3")
	playUCIMoves(t, viaPawnFirst, "e2e3", "g8f6", "g1f3")
	if viaKnightFirst.hash != viaPawnFirst.hash {
		t.Error("the same position reached by two move orders must have the same hash")
	}
}

func TestThreefoldRepetition_SurvivesUndo(t *testing.T) {
	game := NewGame()
	playUCIMoves(t, game, "g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1")
	if err := game.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	playUCIMoves(t, game, "f3g1", "f6g8")
	if game.DrawReason() != DrawThreefoldRepetition {
		t.Errorf("after undo and replay the start position occurred three times, got state %v reason %v", game.GetState(), game.DrawReason())
	}
}

// Regression: NewGame did not record the starting position, so returning to
// it twice was not a threefold repetition, unlike ParseFEN(StartingFEN).
func TestThreefoldRepetition_CountsTheStartingPosition(t *testing.T) {
	for name, game := range map[string]*Game{"NewGame": NewGame(), "ParseFEN": mustParseFEN(t, StartingFEN)} {
		playUCIMoves(t, game, "g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1", "f6g8")
		if game.DrawReason() != DrawThreefoldRepetition {
			t.Errorf("%s: start position reached three times, got state %v reason %v", name, game.GetState(), game.DrawReason())
		}
	}
}
