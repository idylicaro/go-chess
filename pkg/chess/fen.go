package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"strconv"
	"strings"
	"unicode"
)

// StartingFEN is the standard initial position.
const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// fenLetterByType maps piece types to their lowercase (black) FEN letter.
var fenLetterByType = map[common.PieceType]rune{
	common.Pawn: 'p', common.Knight: 'n', common.Bishop: 'b',
	common.Rook: 'r', common.Queen: 'q', common.King: 'k',
}

// ParseFEN builds a game from a Forsyth–Edwards Notation string. The last
// two fields (halfmove clock, fullmove number) are optional and default to
// "0 1", as many tools omit them.
//
//	game, err := ParseFEN("8/8/8/8/8/8/8/K6k w - - 0 1")
func ParseFEN(fen string) (*Game, error) {
	fields := strings.Fields(fen)
	if len(fields) != 4 && len(fields) != 6 {
		return nil, fmt.Errorf("invalid FEN %q: got %d fields, expected 6 (or 4 without the move counters)", fen, len(fields))
	}
	if len(fields) == 4 {
		fields = append(fields, "0", "1")
	}
	game, err := gameFromFENFields(fields)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN %q: %w", fen, err)
	}
	game.initialFEN = game.FEN()
	game.updateGameState()
	return game, nil
}

// gameFromFENFields parses the six FEN fields in order.
func gameFromFENFields(fields []string) (*Game, error) {
	game := &Game{state: InProgress}
	var err error
	if game.board, err = parseFENPlacement(fields[0]); err != nil {
		return nil, err
	}
	if game.currentTurn, err = parseFENSideToMove(fields[1]); err != nil {
		return nil, err
	}
	if game.castlingRights, err = parseFENCastling(fields[2]); err != nil {
		return nil, err
	}
	if game.enPassantTarget, err = parseFENEnPassant(fields[3], game.currentTurn); err != nil {
		return nil, err
	}
	if game.halfMoveClock, err = parseFENCounter("halfmove clock", fields[4], 0); err != nil {
		return nil, err
	}
	game.fullMoveNumber, err = parseFENCounter("fullmove number", fields[5], 1)
	return game, err
}

// parseFENPlacement reads the eight ranks, from rank 8 (internal row 0) down
// to rank 1 (row 7), and requires exactly one king per side.
func parseFENPlacement(placement string) (IBoard, error) {
	ranks := strings.Split(placement, "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("piece placement %q has %d ranks, expected 8 separated by '/'", placement, len(ranks))
	}
	board := newBoard(nil)
	for row, rank := range ranks {
		if err := placeFENRank(board, row, rank); err != nil {
			return nil, err
		}
	}
	return board, requireOneKingEach(board)
}

// placeFENRank places one rank's pieces; digits skip that many empty squares.
func placeFENRank(board IBoard, row int, rank string) error {
	column := 0
	for _, symbol := range rank {
		if symbol >= '1' && symbol <= '8' {
			column += int(symbol - '0')
			continue
		}
		piece, ok := pieceFromFENLetter(symbol)
		if !ok {
			return fmt.Errorf("rank %q has piece letter %q, expected one of pnbrqkPNBRQK or 1-8", rank, symbol)
		}
		if column < 8 {
			_ = board.SetPiece(piece, column, row)
		}
		column++
	}
	if column != 8 {
		return fmt.Errorf("rank %q describes %d squares, expected 8", rank, column)
	}
	return nil
}

// requireOneKingEach also records the king positions the board relies on.
func requireOneKingEach(board IBoard) error {
	kings := map[common.Color][]Position{}
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if piece, _ := board.GetPiece(x, y); piece != nil && piece.GetType() == common.King {
				kings[piece.GetColor()] = append(kings[piece.GetColor()], Position{X: x, Y: y})
			}
		}
	}
	for _, color := range []common.Color{common.White, common.Black} {
		if len(kings[color]) != 1 {
			return fmt.Errorf("piece placement has %d %v kings, expected exactly 1", len(kings[color]), color)
		}
		board.SetKingPosition(color, kings[color][0].X, kings[color][0].Y)
	}
	return nil
}

func pieceFromFENLetter(letter rune) (IPiece, bool) {
	color := common.Black
	if unicode.IsUpper(letter) {
		color = common.White
	}
	for pieceType, fenLetter := range fenLetterByType {
		if unicode.ToLower(letter) == fenLetter {
			return newPiece(pieceType, color), true
		}
	}
	return nil, false
}

func parseFENSideToMove(field string) (common.Color, error) {
	switch field {
	case "w":
		return common.White, nil
	case "b":
		return common.Black, nil
	}
	return common.White, fmt.Errorf("side to move %q, expected \"w\" or \"b\"", field)
}

// parseFENCastling accepts "-" or a non-repeating subset of "KQkq".
func parseFENCastling(field string) (CastlingRights, error) {
	if field == "-" {
		return NoCastlingRights, nil
	}
	rights := NoCastlingRights
	for i := 0; i < len(field); i++ {
		right, ok := castlingRightByFENLetter(field[i])
		if !ok || rights.Has(right) {
			return NoCastlingRights, fmt.Errorf("castling rights %q, expected \"-\" or a subset of \"KQkq\" without repeats", field)
		}
		rights |= right
	}
	return rights, nil
}

func castlingRightByFENLetter(letter byte) (CastlingRights, bool) {
	for _, side := range castlingSides {
		if side.fenLetter == letter {
			return side.right, true
		}
	}
	return NoCastlingRights, false
}

// parseFENEnPassant accepts "-" or the square behind a pawn that just moved
// two squares: rank 6 when White is to move, rank 3 when Black is.
func parseFENEnPassant(field string, sideToMove common.Color) (*Position, error) {
	if field == "-" {
		return nil, nil
	}
	expectedRank := byte('6')
	if sideToMove == common.Black {
		expectedRank = '3'
	}
	x, y, err := (&coordinateConverter{}).FromAlgebraic(field)
	if err != nil || field[1] != expectedRank {
		return nil, fmt.Errorf("en passant square %q, expected \"-\" or a square on rank %c", field, expectedRank)
	}
	return &Position{X: x, Y: y}, nil
}

func parseFENCounter(name, field string, minimum int) (int, error) {
	value, err := strconv.Atoi(field)
	if err != nil || value < minimum {
		return 0, fmt.Errorf("%s %q, expected an integer >= %d", name, field, minimum)
	}
	return value, nil
}
