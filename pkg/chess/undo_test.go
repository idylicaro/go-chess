package chess

import (
	"errors"
	"testing"
	"time"

	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/timer"
	"github.com/idylicaro/go-chess/internal/timer/timertest"
)

func TestUndo(t *testing.T) {
	t.Run("restores every position of a game with castling, en passant and capture", func(t *testing.T) {
		game := NewGame()
		moves := []string{"e2e4", "d7d5", "e4e5", "f7f5", "e5f6", "g8f6", "g1f3", "e7e6", "f1e2", "f8e7", "e1g1"}
		fens := []string{game.FEN()}
		for _, move := range moves {
			playMoves(t, game, move)
			fens = append(fens, game.FEN())
		}
		for i := len(moves) - 1; i >= 0; i-- {
			if err := game.Undo(); err != nil {
				t.Fatalf("Undo before %s: %v", moves[i], err)
			}
			if got := game.FEN(); got != fens[i] {
				t.Fatalf("after undoing %s: FEN = %q, want %q", moves[i], got, fens[i])
			}
		}
		if len(game.moveHistory) != 0 {
			t.Errorf("moveHistory has %d moves after undoing all", len(game.moveHistory))
		}
	})

	t.Run("takes back a promotion", func(t *testing.T) {
		const fen = "8/P6k/8/8/8/8/8/K7 w - - 0 1"
		game := mustFEN(t, fen)
		playMoves(t, game, "a7a8q")
		if err := game.Undo(); err != nil {
			t.Fatal(err)
		}
		if got := game.FEN(); got != fen {
			t.Errorf("FEN = %q, want %q", got, fen)
		}
	})

	t.Run("nothing to undo", func(t *testing.T) {
		if err := NewGame().Undo(); !errors.Is(err, ErrNoMoveToUndo) {
			t.Errorf("err = %v, want ErrNoMoveToUndo", err)
		}
	})

	t.Run("reopens a game ended by checkmate", func(t *testing.T) {
		game := NewGame()
		playMoves(t, game, "f2f3", "e7e5", "g2g4", "d8h4")
		if game.GetState() != Checkmate {
			t.Fatalf("state = %v, want Checkmate", game.GetState())
		}
		if err := game.Undo(); err != nil {
			t.Fatal(err)
		}
		if game.IsOver() || game.Result() != ResultOngoing || len(game.LegalMoves()) == 0 {
			t.Errorf("game still over after undo: state %v", game.GetState())
		}
	})

	t.Run("reopens a game drawn by the fifty-move rule", func(t *testing.T) {
		game := NewGame()
		game.halfMoveClock = 99
		playMoves(t, game, "g1f3")
		if err := game.Undo(); err != nil {
			t.Fatal(err)
		}
		if game.IsOver() || game.DrawReason() != DrawNone || game.halfMoveClock != 99 {
			t.Errorf("state = %v, reason = %v, clock = %d", game.GetState(), game.DrawReason(), game.halfMoveClock)
		}
	})

	t.Run("refuses once a player resigned", func(t *testing.T) {
		game := NewGame()
		playMoves(t, game, "e2e4")
		_ = game.Resign(common.Black)
		if err := game.Undo(); !errors.Is(err, ErrGameOver) {
			t.Errorf("err = %v, want ErrGameOver", err)
		}
	})

	t.Run("exported PGN no longer has the undone move", func(t *testing.T) {
		game := NewGame()
		playMoves(t, game, "e2e4", "e7e5")
		_ = game.Undo()
		pgn, err := game.PGN(PGNHeaders{})
		if err != nil {
			t.Fatal(err)
		}
		if want := "1. e4 *\n"; len(pgn) < len(want) || pgn[len(pgn)-len(want):] != want {
			t.Errorf("PGN movetext = %q, want it to end with %q", pgn, want)
		}
	})

	t.Run("returns the turn on the clock and withdraws the increment", func(t *testing.T) {
		clock := timertest.NewFakeClock()
		game := newStartedTimedGame(t, timer.TimeConfig{InitialTime: 10 * time.Second, Increment: 2 * time.Second}, clock)
		playTimedMoves(t, game, clock, time.Second, "e2e4")
		assertClock(t, game, common.White, 11*time.Second)

		if err := game.Undo(); err != nil {
			t.Fatal(err)
		}
		assertClock(t, game, common.White, 9*time.Second)
		clock.Advance(time.Second)
		assertClock(t, game, common.White, 8*time.Second) // White is on the move again
		assertClock(t, game, common.Black, 10*time.Second)
	})
}
