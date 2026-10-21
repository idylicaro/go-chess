package search

import (
	"context"
	"testing"

	"github.com/idylicaro/go-chess/internal/eval"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Positions shared by the search tests.
const (
	kiwipeteFEN = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	// scholarsMateFEN: Qxf7 is mate.
	scholarsMateFEN = "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4"
	// rookMateInTwoFEN: Kb6 Kb8 Rh8 (or Kc7 Ka7 Ra1).
	rookMateInTwoFEN = "k7/8/2K5/8/8/8/8/7R w - - 0 1"
	// stalemateFEN: Black to move has no legal move and is not in check.
	stalemateFEN = "k7/2Q5/1K6/8/8/8/8/8 b - - 0 1"
	// hangingQueenFEN: the black queen on d5 is free for the e4 pawn.
	hangingQueenFEN = "rnb1kbnr/ppp1pppp/8/3q4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 3"
)

func TestSearch_FindsMateInOne(t *testing.T) {
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, scholarsMateFEN), Limits{Depth: 3}, nil)
	if result.Best.String() != "h5f7" {
		t.Errorf("best move = %s, want h5f7 (Qxf7#)", result.Best)
	}
	if moves, ok := MateInMoves(result.Score); !ok || moves != 1 {
		t.Errorf("score %d reads as mate %d (%v), want mate 1", result.Score, moves, ok)
	}
}

func TestSearch_FindsMateInTwo(t *testing.T) {
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, rookMateInTwoFEN), Limits{Depth: 5}, nil)
	if moves, ok := MateInMoves(result.Score); !ok || moves != 2 {
		t.Errorf("score %d reads as mate %d (%v), want mate 2", result.Score, moves, ok)
	}
}

func TestSearch_TakesAHangingQueen(t *testing.T) {
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, hangingQueenFEN), Limits{Depth: 3}, nil)
	if result.Best.String() != "e4d5" {
		t.Errorf("best move = %s, want e4d5", result.Best)
	}
}

func TestSearch_NoLegalMoveGivesANullBest(t *testing.T) {
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, stalemateFEN), Limits{Depth: 4}, nil)
	if !result.Best.IsNull() || result.Score != drawScore {
		t.Errorf("stalemate: best %s score %d, want the null move and %d", result.Best, result.Score, drawScore)
	}
}

func TestSearch_LeavesTheBoardAsItFoundIt(t *testing.T) {
	board := boardFromFEN(t, kiwipeteFEN)
	before := board.FEN()
	New(eval.Evaluate).Search(context.Background(), board, Limits{Depth: 3}, nil)
	if after := board.FEN(); after != before {
		t.Errorf("board changed from %q to %q", before, after)
	}
}

func TestSearch_ReportsEveryDepthUpToTheLimit(t *testing.T) {
	var depths []int
	report := func(info Info) { depths = append(depths, info.Depth) }
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, chess.StartingFEN), Limits{Depth: 4}, report)
	if len(depths) != 4 || depths[3] != 4 || result.Depth != 4 {
		t.Errorf("reported depths %v, result depth %d, want 1..4", depths, result.Depth)
	}
}

func TestSearch_PrincipalVariationIsPlayable(t *testing.T) {
	board := boardFromFEN(t, kiwipeteFEN)
	result := New(eval.Evaluate).Search(context.Background(), board, Limits{Depth: 4}, nil)
	if len(result.PV) == 0 || result.PV[0] != result.Best {
		t.Fatalf("PV %v does not start with the best move %s", result.PV, result.Best)
	}
	for _, move := range result.PV {
		if _, legal := board.MakeMove(move); !legal {
			t.Fatalf("PV move %s is illegal in %q", move, board.FEN())
		}
	}
}

func TestSearch_StopsAtTheNodeLimit(t *testing.T) {
	result := New(eval.Evaluate).Search(context.Background(), boardFromFEN(t, kiwipeteFEN), Limits{Nodes: 20000}, nil)
	if result.Best.IsNull() || result.Nodes > 20000+2*pollInterval {
		t.Errorf("node-limited search: best %s after %d nodes, want a move within about 20000", result.Best, result.Nodes)
	}
}

func TestSearch_CancelledContextStillReturnsAMove(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := New(eval.Evaluate).Search(ctx, boardFromFEN(t, kiwipeteFEN), Limits{}, nil)
	if result.Best.IsNull() || result.Depth != 1 {
		t.Errorf("cancelled search: best %s at depth %d, want a depth-1 move", result.Best, result.Depth)
	}
}

func TestSearch_ScoresARepetitionAsADraw(t *testing.T) {
	// Black is a rook down but can repeat: after Ng1 Ng8 the position is back.
	game, err := chess.ParseFEN("rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1")
	if err != nil {
		t.Fatal(err)
	}
	board := chess.NewSearchBoard(game)
	playUCI(t, board, "g8f6", "f3g1")
	if board.IsRepetition() {
		t.Fatal("no repetition expected yet")
	}
	playUCI(t, board, "f6g8")
	w := newWorker(board, eval.Evaluate, newStopper(context.Background(), 0))
	if score := w.negamax(2, 1, -infinity, infinity); score != drawScore {
		t.Errorf("negamax in a repeated position = %d, want %d", score, drawScore)
	}
}

// TestNegamax_MatchesPlainMinimax checks that alpha-beta pruning changes
// only the work, never the score of a full-window search.
func TestNegamax_MatchesPlainMinimax(t *testing.T) {
	for _, fen := range []string{chess.StartingFEN, scholarsMateFEN, hangingQueenFEN, benchmarkFENs["endgame"]} {
		board := boardFromFEN(t, fen)
		want := minimax(board, 2, 0)
		w := newWorker(board, eval.Evaluate, newStopper(context.Background(), 0))
		if got := w.negamax(2, 0, -infinity, infinity); got != want {
			t.Errorf("%q: negamax(2) = %d, minimax(2) = %d", fen, got, want)
		}
	}
}

// minimax is the reference: every move searched, with the same draw and
// mate rules as negamax and its full-window quiescence search at the leaves.
func minimax(board *chess.SearchBoard, depth, ply int) int {
	if ply > 0 && (board.IsRepetition() || board.HalfMoveClock() >= fiftyMovePlies) {
		return drawScore
	}
	if depth == 0 {
		return newWorker(board, eval.Evaluate, newStopper(context.Background(), 0)).quiesce(ply, -infinity, infinity)
	}
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	best, legalMoves := -infinity, 0
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if undo, legal := board.MakeMove(move); legal {
			legalMoves++
			best = max(best, -minimax(board, depth-1, ply+1))
			board.UnmakeMove(move, undo)
		}
	}
	if legalMoves > 0 {
		return best
	}
	if board.InCheck() {
		return matedIn(ply)
	}
	return drawScore
}

func TestMateInMoves(t *testing.T) {
	cases := []struct {
		score, moves int
		ok           bool
	}{
		{mateScore - 1, 1, true}, {mateScore - 3, 2, true}, {-mateScore + 2, -1, true}, {-mateScore + 4, -2, true}, {250, 0, false}, {-250, 0, false},
	}
	for _, tc := range cases {
		if moves, ok := MateInMoves(tc.score); moves != tc.moves || ok != tc.ok {
			t.Errorf("MateInMoves(%d) = %d, %v, want %d, %v", tc.score, moves, ok, tc.moves, tc.ok)
		}
	}
}

func boardFromFEN(t testing.TB, fen string) *chess.SearchBoard {
	t.Helper()
	game, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	return chess.NewSearchBoard(game)
}

// playUCI makes each move, given in UCI notation, on board.
func playUCI(t testing.TB, board *chess.SearchBoard, moves ...string) {
	t.Helper()
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	for _, notation := range moves {
		found := false
		for _, move := range board.GenerateMoves(buffer[:0]) {
			if move.String() != notation {
				continue
			}
			if _, legal := board.MakeMove(move); legal {
				found = true
			}
			break
		}
		if !found {
			t.Fatalf("move %s is not legal in %q", notation, board.FEN())
		}
	}
}
