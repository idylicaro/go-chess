package chess

import (
	"fmt"
	"testing"
)

// searchBoardPerft counts the legal tree through the exported SearchBoard
// API only, the way an engine walks it.
func searchBoardPerft(board *SearchBoard, depth int) int {
	if depth == 0 {
		return 1
	}
	var buffer [MaxSearchMoves]SearchMove
	nodes := 0
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if undo, legal := board.MakeMove(move); legal {
			nodes += searchBoardPerft(board, depth-1)
			board.UnmakeMove(move, undo)
		}
	}
	return nodes
}

func TestSearchBoard_PerftMatchesTheReferenceCounts(t *testing.T) {
	for _, tc := range perftCases {
		for depth := 1; depth <= 3; depth++ {
			t.Run(fmt.Sprintf("%s/depth%d", tc.position, depth), func(t *testing.T) {
				board := NewSearchBoard(mustParseFEN(t, perftReferenceFENs[tc.position]))
				if got, want := searchBoardPerft(board, depth), tc.nodes[depth-1]; got != want {
					t.Errorf("perft(%d) = %d, want %d", depth, got, want)
				}
			})
		}
	}
}

func TestSearchBoard_LeavesTheGameUntouched(t *testing.T) {
	game := NewGame()
	before := game.FEN()
	searchBoardPerft(NewSearchBoard(game), 2)
	board := NewSearchBoard(game)
	board.MakeMove(findSearchMove(t, board, "e2e4"))
	if after := game.FEN(); after != before {
		t.Errorf("game FEN changed from %q to %q", before, after)
	}
}

func TestSearchBoard_MakeMoveRejectsAPinnedPieceAndKeepsThePosition(t *testing.T) {
	// The e2 knight is pinned by the e8 rook.
	board := NewSearchBoard(mustParseFEN(t, "4r1k1/8/8/8/8/8/4N3/4K3 w - - 0 1"))
	before, hash := board.FEN(), board.Hash()
	if _, legal := board.MakeMove(findSearchMove(t, board, "e2c3")); legal {
		t.Fatal("MakeMove(e2c3) is legal, want rejected: the knight is pinned")
	}
	if board.FEN() != before || board.Hash() != hash {
		t.Errorf("after a rejected move FEN = %q, want %q", board.FEN(), before)
	}
}

func TestSearchBoard_NullMovePassesTheTurnAndClearsEnPassant(t *testing.T) {
	board := NewSearchBoard(mustParseFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1"))
	before, hash := board.FEN(), board.Hash()
	undo := board.MakeNullMove()
	if got, want := board.FEN(), "4k3/8/8/3pP3/8/8/8/4K3 b - - 1 1"; got != want {
		t.Errorf("after a null move FEN = %q, want %q", got, want)
	}
	if board.Hash() != board.game.computeHash() {
		t.Error("after a null move the incremental hash differs from the recomputed one")
	}
	board.UnmakeNullMove(undo)
	if board.FEN() != before || board.Hash() != hash {
		t.Errorf("after UnmakeNullMove FEN = %q, want %q", board.FEN(), before)
	}
}

func TestSearchBoard_IsRepetitionAfterKnightsGoBackAndForth(t *testing.T) {
	board := NewSearchBoard(NewGame())
	for _, notation := range []string{"g1f3", "g8f6", "f3g1"} {
		playSearchMove(t, board, notation)
		if board.IsRepetition() {
			t.Fatalf("IsRepetition after %s = true, want false", notation)
		}
	}
	playSearchMove(t, board, "f6g8")
	if !board.IsRepetition() {
		t.Error("IsRepetition back at the starting position = false, want true")
	}
}

func TestSearchBoard_IsRepetitionSeesTheGamesPastMoves(t *testing.T) {
	game := NewGame()
	playMoves(t, game, "g1f3", "g8f6", "f3g1")
	board := NewSearchBoard(game)
	playSearchMove(t, board, "f6g8")
	if !board.IsRepetition() {
		t.Error("IsRepetition of a position reached in the game = false, want true")
	}
}

func TestSearchBoard_IsRepetitionAfterAPawnMove(t *testing.T) {
	board := NewSearchBoard(NewGame())
	// The position after e5 has an en passant square, so it never repeats;
	// the one after the first Nf3 does.
	for _, notation := range []string{"e2e4", "e7e5", "g1f3", "g8f6", "f3g1", "f6g8", "g1f3"} {
		playSearchMove(t, board, notation)
	}
	if !board.IsRepetition() {
		t.Error("IsRepetition after the knights repeat past e4 e5 = false, want true")
	}
	if board.HalfMoveClock() != 5 {
		t.Errorf("HalfMoveClock = %d, want 5", board.HalfMoveClock())
	}
}

func TestSearchBoard_GenerateCapturesKeepsOnlyCapturesAndPromotions(t *testing.T) {
	board := NewSearchBoard(mustParseFEN(t, "1n2k3/P7/8/3pP3/8/8/8/4K3 w - d6 0 1"))
	var buffer [MaxSearchMoves]SearchMove
	got := map[string]bool{}
	for _, move := range board.GenerateCaptures(buffer[:0]) {
		got[move.String()] = true
	}
	want := []string{"e5d6", "a7a8q", "a7a8r", "a7a8b", "a7a8n", "a7b8q", "a7b8r", "a7b8b", "a7b8n"}
	if len(got) != len(want) {
		t.Errorf("GenerateCaptures = %v, want %v", got, want)
	}
	for _, notation := range want {
		if !got[notation] {
			t.Errorf("GenerateCaptures misses %s", notation)
		}
	}
}

func TestSearchBoard_CapturedPieceSeesEnPassant(t *testing.T) {
	board := NewSearchBoard(mustParseFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1"))
	captured, ok := board.CapturedPiece(findSearchMove(t, board, "e5d6"))
	if !ok || captured != Pawn {
		t.Errorf("CapturedPiece(e5d6 e.p.) = %v, %v, want Pawn, true", captured, ok)
	}
	if board.IsCapture(findSearchMove(t, board, "e5e6")) {
		t.Error("IsCapture(e5e6) = true, want false")
	}
}

func TestSearchBoard_PieceOnUsesRank8AsSquareZero(t *testing.T) {
	board := NewSearchBoard(NewGame())
	pieceType, color, occupied := board.PieceOn(60)
	if !occupied || pieceType != King || color != White {
		t.Errorf("PieceOn(60) = %v %v %v, want the white king on e1", pieceType, color, occupied)
	}
	if _, _, occupied := board.PieceOn(36); occupied {
		t.Error("PieceOn(36), e4, is occupied in the starting position")
	}
}

func TestSearchBoard_CloneIsIndependent(t *testing.T) {
	board := NewSearchBoard(NewGame())
	clone := board.Clone()
	playSearchMove(t, clone, "e2e4")
	if board.FEN() == clone.FEN() {
		t.Error("a move on the clone changed the original")
	}
}

func TestSearchMove_StringAndNull(t *testing.T) {
	if got := NullSearchMove.String(); got != "0000" || !NullSearchMove.IsNull() {
		t.Errorf("NullSearchMove = %q (null %v), want \"0000\" and null", got, NullSearchMove.IsNull())
	}
	move := SearchMove{from: 12, to: 4, promotion: Queen}
	if got := move.String(); got != "e7e8q" || move.From() != 12 || move.To() != 4 || move.Promotion() != Queen {
		t.Errorf("move = %q from %d to %d promotion %v, want e7e8q 12 4 Queen", got, move.From(), move.To(), move.Promotion())
	}
}

// findSearchMove returns the generated move written as notation in UCI.
func findSearchMove(t *testing.T, board *SearchBoard, notation string) SearchMove {
	t.Helper()
	var buffer [MaxSearchMoves]SearchMove
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if move.String() == notation {
			return move
		}
	}
	t.Fatalf("no generated move %s in %q", notation, board.FEN())
	return NullSearchMove
}

func playSearchMove(t *testing.T, board *SearchBoard, notation string) {
	t.Helper()
	if _, legal := board.MakeMove(findSearchMove(t, board, notation)); !legal {
		t.Fatalf("move %s is illegal in %q", notation, board.FEN())
	}
}

func TestSearchMove_PackRoundTrips(t *testing.T) {
	board := NewSearchBoard(mustParseFEN(t, "1n2k3/P7/8/3pP3/8/8/8/4K3 w - d6 0 1"))
	var buffer [MaxSearchMoves]SearchMove
	for _, move := range append(board.GenerateMoves(buffer[:0]), NullSearchMove) {
		if got := UnpackSearchMove(move.Pack()); got != move {
			t.Errorf("UnpackSearchMove(%s.Pack()) = %s", move, got)
		}
	}
}
