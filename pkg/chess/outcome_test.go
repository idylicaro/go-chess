package chess

import (
	"errors"
	"testing"
	"time"

	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/timer/timertest"
)

func playMoves(t *testing.T, game *Game, moves ...string) {
	t.Helper()
	for _, notation := range moves {
		if err := game.Move(mustParseUCI(t, notation)); err != nil {
			t.Fatalf("Move(%s): %v", notation, err)
		}
	}
}

func mustFEN(t *testing.T, fen string) *Game {
	t.Helper()
	game, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	return game
}

func TestLegalMoves(t *testing.T) {
	t.Run("initial position has 20 moves", func(t *testing.T) {
		if got := len(NewGame().LegalMoves()); got != 20 {
			t.Errorf("len(LegalMoves) = %d, want 20", got)
		}
	})

	t.Run("a promotion expands into four moves", func(t *testing.T) {
		game := mustFEN(t, "8/P6k/8/8/8/8/8/K7 w - - 0 1")
		promotions := 0
		for _, move := range game.LegalMoves() {
			if move.FromX == 0 && move.FromY == 1 && move.ToY == 0 {
				promotions++
			}
		}
		if promotions != 4 {
			t.Errorf("a7-a8 promotions = %d, want 4", promotions)
		}
	})

	t.Run("empty once the game is over", func(t *testing.T) {
		game := NewGame()
		if err := game.Resign(common.White); err != nil {
			t.Fatal(err)
		}
		if moves := game.LegalMoves(); len(moves) != 0 {
			t.Errorf("LegalMoves after resignation = %v, want none", moves)
		}
	})
}

func TestResultAndDrawReason(t *testing.T) {
	foolsMate := func(t *testing.T) *Game {
		game := NewGame()
		playMoves(t, game, "f2f3", "e7e5", "g2g4", "d8h4")
		return game
	}
	fiftyMoves := func(t *testing.T) *Game {
		game := NewGame()
		game.halfMoveClock = 99
		playMoves(t, game, "g1f3")
		return game
	}
	threefold := func(t *testing.T) *Game {
		game := NewGame()
		for cycle := 0; cycle < 4; cycle++ {
			for _, move := range []string{"g1f3", "g8f6", "f3g1", "f6g8"} {
				if !game.IsOver() {
					playMoves(t, game, move)
				}
			}
		}
		return game
	}

	tests := []struct {
		name       string
		game       func(t *testing.T) *Game
		wantResult Result
		wantPGN    string
		wantReason DrawReason
	}{
		{"new game", func(t *testing.T) *Game { return NewGame() }, ResultOngoing, "*", DrawNone},
		{"checkmate", foolsMate, ResultBlackWins, "0-1", DrawNone},
		{"stalemate", func(t *testing.T) *Game { return mustFEN(t, "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1") }, ResultDraw, "1/2-1/2", DrawStalemate},
		{"fifty-move rule", fiftyMoves, ResultDraw, "1/2-1/2", DrawFiftyMoveRule},
		{"threefold repetition", threefold, ResultDraw, "1/2-1/2", DrawThreefoldRepetition},
		{"insufficient material", func(t *testing.T) *Game { return mustFEN(t, "8/8/8/8/8/8/8/K6k w - - 0 1") }, ResultDraw, "1/2-1/2", DrawInsufficientMaterial},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := tt.game(t)
			if got := game.Result(); got != tt.wantResult {
				t.Errorf("Result() = %v, want %v", got, tt.wantResult)
			}
			if got := game.Result().String(); got != tt.wantPGN {
				t.Errorf("Result().String() = %q, want %q", got, tt.wantPGN)
			}
			if got := game.DrawReason(); got != tt.wantReason {
				t.Errorf("DrawReason() = %v, want %v", got, tt.wantReason)
			}
		})
	}
}

func TestResign(t *testing.T) {
	game := NewGame()
	playMoves(t, game, "e2e4")

	// Resigning is allowed out of turn: here White resigns on Black's move.
	if err := game.Resign(common.White); err != nil {
		t.Fatalf("Resign: %v", err)
	}
	if game.GetState() != ResignedWhite || game.Result() != ResultBlackWins || !game.IsOver() {
		t.Errorf("state = %v, result = %v, over = %v", game.GetState(), game.Result(), game.IsOver())
	}
	if err := game.Move(mustParseUCI(t, "e7e5")); !errors.Is(err, ErrGameOver) {
		t.Errorf("Move after resignation: err = %v, want ErrGameOver", err)
	}
	if err := game.Resign(common.Black); !errors.Is(err, ErrGameOver) {
		t.Errorf("second Resign: err = %v, want ErrGameOver", err)
	}
}

func TestDrawOffer(t *testing.T) {
	t.Run("accepted by the opponent", func(t *testing.T) {
		game := NewGame()
		if err := game.OfferDraw(common.White); err != nil {
			t.Fatal(err)
		}
		if by, pending := game.PendingDrawOffer(); !pending || by != common.White {
			t.Errorf("PendingDrawOffer = %v, %v", by, pending)
		}
		if err := game.AcceptDraw(common.White); err == nil {
			t.Error("offering side accepted its own offer")
		}
		if err := game.AcceptDraw(common.Black); err != nil {
			t.Fatal(err)
		}
		if game.GetState() != Draw || game.DrawReason() != DrawAgreement || game.Result() != ResultDraw {
			t.Errorf("state = %v, reason = %v, result = %v", game.GetState(), game.DrawReason(), game.Result())
		}
	})

	t.Run("accepting without an offer fails", func(t *testing.T) {
		if err := NewGame().AcceptDraw(common.Black); err == nil {
			t.Error("AcceptDraw with no offer succeeded")
		}
	})

	t.Run("offering twice fails", func(t *testing.T) {
		game := NewGame()
		_ = game.OfferDraw(common.White)
		if err := game.OfferDraw(common.White); err == nil {
			t.Error("second offer by the same side succeeded")
		}
	})

	t.Run("declined explicitly", func(t *testing.T) {
		game := NewGame()
		_ = game.OfferDraw(common.White)
		if err := game.DeclineDraw(common.Black); err != nil {
			t.Fatal(err)
		}
		if _, pending := game.PendingDrawOffer(); pending || game.IsOver() {
			t.Errorf("pending = %v, over = %v after decline", pending, game.IsOver())
		}
	})

	t.Run("a move declines it", func(t *testing.T) {
		game := NewGame()
		_ = game.OfferDraw(common.White)
		playMoves(t, game, "e2e4")
		if _, pending := game.PendingDrawOffer(); pending {
			t.Error("offer survived a move")
		}
	})
}

func TestTimeoutVersusInsufficientMaterial(t *testing.T) {
	flaggedGame := func(t *testing.T, fen string) *Game {
		game := mustFEN(t, fen)
		clock := timertest.NewFakeClock()
		game.gameTimer = timer.NewGameTimerWithClock(timer.TimeConfig{InitialTime: time.Second}, clock)
		game.gameTimer.Start(game.GetCurrentTurn())
		clock.Advance(2 * time.Second)
		return game
	}

	tests := []struct {
		name       string
		fen        string
		wantState  GameState
		wantResult Result
		wantReason DrawReason
	}{
		{"opponent has a bare king: draw", "4k3/8/8/8/8/8/P7/4K3 w - - 0 1", Draw, ResultDraw, DrawTimeoutVsInsufficientMaterial},
		{"opponent has a pawn: loss on time", "4k3/p7/8/8/8/8/8/4K3 w - - 0 1", TimeoutWhite, ResultBlackWins, DrawNone},
		{"lone knight against a pawn: loss on time", "4k3/p7/8/8/8/8/8/3NK3 b - - 0 1", TimeoutBlack, ResultWhiteWins, DrawNone},
		{"two knights can mate: loss on time", "4k3/8/8/8/8/8/P7/2NNK3 b - - 0 1", TimeoutBlack, ResultWhiteWins, DrawNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := flaggedGame(t, tt.fen)
			err := game.Move(game.LegalMoves()[0])
			if !errors.Is(err, ErrGameOver) {
				t.Fatalf("Move after flagging: err = %v, want ErrGameOver", err)
			}
			if game.GetState() != tt.wantState || game.Result() != tt.wantResult || game.DrawReason() != tt.wantReason {
				t.Errorf("state = %v, result = %v, reason = %v; want %v, %v, %v",
					game.GetState(), game.Result(), game.DrawReason(), tt.wantState, tt.wantResult, tt.wantReason)
			}
		})
	}
}
