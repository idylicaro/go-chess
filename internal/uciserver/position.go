package uciserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// fenFieldCount is the number of fields of a full FEN.
const fenFieldCount = 6

// handlePosition sets up "position startpos|fen <fen> [moves m1 m2 ...]".
// On an error the previous position is kept.
func (s *Server) handlePosition(_ context.Context, args []string) error {
	s.stopSearch()
	board, err := parsePosition(args)
	if err != nil {
		return err
	}
	s.board = board
	return nil
}

// parsePosition plays the moves on a SearchBoard rather than a Game: a Game
// refuses moves once a draw rule ends it, but a GUI may play on (a
// threefold repetition it does not claim), and the board keeps the
// repetition history the search needs.
func parsePosition(args []string) (*chess.SearchBoard, error) {
	game, rest, err := parseStartingPosition(args)
	if err != nil {
		return nil, err
	}
	board := chess.NewSearchBoard(game)
	if len(rest) == 0 {
		return board, nil
	}
	if rest[0] != "moves" {
		return nil, fmt.Errorf("position: unexpected %q, expected \"moves\" after the position", rest[0])
	}
	return board, playMoves(board, rest[1:])
}

// parseStartingPosition reads "startpos" or "fen" and its six fields, and
// returns what follows.
func parseStartingPosition(args []string) (*chess.Game, []string, error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("position: missing position, expected \"startpos\" or \"fen <fen>\"")
	}
	switch args[0] {
	case "startpos":
		return chess.NewGame(), args[1:], nil
	case "fen":
		fields := args[1:min(len(args), 1+fenFieldCount)]
		game, err := chess.ParseFEN(strings.Join(fields, " "))
		if err != nil {
			return nil, nil, fmt.Errorf("position: %w", err)
		}
		return game, args[1+len(fields):], nil
	}
	return nil, nil, fmt.Errorf("position: unknown position %q, expected \"startpos\" or \"fen\"", args[0])
}

func playMoves(board *chess.SearchBoard, moves []string) error {
	for _, notation := range moves {
		if !playMove(board, strings.ToLower(notation)) {
			return fmt.Errorf("position: move %q is not legal in %q, expected a legal move in UCI notation like e2e4 or e7e8q", notation, board.FEN())
		}
	}
	return nil
}

// playMove makes the legal move written as notation, if there is one.
func playMove(board *chess.SearchBoard, notation string) bool {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if move.String() == notation {
			_, legal := board.MakeMove(move)
			return legal
		}
	}
	return false
}
