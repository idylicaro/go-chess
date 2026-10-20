package chess

import "testing"

func assertCastling(t *testing.T, game *Game, want string) {
	t.Helper()
	if got := formatFENCastling(game.castlingRights); got != want {
		t.Errorf("castling rights = %q, want %q", got, want)
	}
}

// Castling rights are game state (#7): a king that never moved can still
// have lost the right to castle, which the old hasMoved flags could not express.
func TestCastling_RequiresRight(t *testing.T) {
	game := mustParseFEN(t, "k7/8/8/8/8/8/8/4K2R w - - 0 1")
	assertLegal(t, legalTargets(t, game, "e1"), "g1", false)
}

func TestCastling_LostWhenKingMoves(t *testing.T) {
	game := NewGame()
	playUCIMoves(t, game, "e2e4", "e7e5", "e1e2")
	assertCastling(t, game, "kq")
}

func TestCastling_LostWhenRookMovesOrIsCaptured(t *testing.T) {
	game := mustParseFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	playUCIMoves(t, game, "h1h8") // white rook leaves h1 and captures the rook on h8
	assertCastling(t, game, "Qq")
}

func TestCastling_MovesRookAndKing(t *testing.T) {
	game := mustParseFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	playUCIMoves(t, game, "e1c1", "e8g8")
	assertFEN(t, game, "r4rk1/8/8/8/8/8/8/2KR3R w - - 2 2")
}
