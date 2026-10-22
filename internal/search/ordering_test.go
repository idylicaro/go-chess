package search

import (
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// orderingFEN: White can take the d5 queen with the e4 pawn or the c3
// knight; every other move is quiet.
const orderingFEN = "4k3/8/8/3q4/4P3/2N5/8/4K3 w - - 0 1"

func TestMovePicker_HashMoveThenMostValuableVictimLeastValuableAttacker(t *testing.T) {
	board := boardFromFEN(t, orderingFEN)
	hash := findMove(t, board, "e1d1")
	order := pickedOrder(board, &moveOrderer{}, hash, 0)
	want := []string{"e1d1", "e4d5", "c3d5"}
	for i, notation := range want {
		if order[i] != notation {
			t.Fatalf("picked order starts %v, want %v", order[:len(want)], want)
		}
	}
}

func TestMovePicker_KillersBeforeHistoryBeforeOtherQuietMoves(t *testing.T) {
	board := boardFromFEN(t, "4k3/8/8/8/8/8/8/R3K3 w - - 0 1")
	orderer := &moveOrderer{}
	orderer.recordCutoff(chess.White, findMove(t, board, "a1a5"), 1, 3)
	orderer.recordCutoff(chess.White, findMove(t, board, "a1a2"), 1, 3) // a1a5 becomes the second killer
	orderer.history[chess.White][findMove(t, board, "e1f2").From()][findMove(t, board, "e1f2").To()] = 50
	order := pickedOrder(board, orderer, chess.NullSearchMove, 3)
	want := []string{"a1a2", "a1a5", "e1f2"}
	for i, notation := range want {
		if order[i] != notation {
			t.Fatalf("picked order starts %v, want %v", order[:len(want)], want)
		}
	}
}

func TestMovePicker_UnderpromotionsComeLast(t *testing.T) {
	board := boardFromFEN(t, "4k3/P7/8/8/8/8/8/4K3 w - - 0 1")
	order := pickedOrder(board, &moveOrderer{}, chess.NullSearchMove, 0)
	if order[0] != "a7a8q" || order[len(order)-1] != "a7a8n" {
		t.Errorf("picked order %v, want a7a8q first and a7a8n last", order)
	}
}

func TestMoveOrderer_HistoryIsHalvedAtTheLimit(t *testing.T) {
	orderer := &moveOrderer{}
	move := findMove(t, boardFromFEN(t, chess.StartingFEN), "g1f3")
	orderer.history[chess.White][move.From()][move.To()] = historyLimit - 1
	orderer.recordCutoff(chess.White, move, 1, 0)
	if got := orderer.history[chess.White][move.From()][move.To()]; got != historyLimit/2 {
		t.Errorf("history after reaching the limit = %d, want %d", got, historyLimit/2)
	}
}

func pickedOrder(board *chess.SearchBoard, orderer *moveOrderer, hash chess.SearchMove, ply int) []string {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	picker := orderer.newMovePicker(board, board.GenerateMoves(buffer[:0]), hash, ply)
	var order []string
	for move, ok := picker.pick(); ok; move, ok = picker.pick() {
		order = append(order, move.String())
	}
	return order
}

func findMove(t testing.TB, board *chess.SearchBoard, notation string) chess.SearchMove {
	t.Helper()
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if move.String() == notation {
			return move
		}
	}
	t.Fatalf("no move %s in %q", notation, board.FEN())
	return chess.NullSearchMove
}
