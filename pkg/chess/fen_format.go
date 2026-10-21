package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
	"strconv"
	"strings"
	"unicode"
)

// FEN returns the current position in Forsyth–Edwards Notation.
//
//	NewGame().FEN() // "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
func (g *Game) FEN() string {
	return strings.Join([]string{
		formatFENPlacement(g.board),
		formatFENSideToMove(g.currentTurn),
		formatFENCastling(g.castlingRights),
		formatFENEnPassant(g.enPassantTarget),
		strconv.Itoa(g.halfMoveClock),
		strconv.Itoa(g.fullMoveNumber),
	}, " ")
}

func formatFENPlacement(board IBoard) string {
	ranks := make([]string, 8)
	for row := 0; row < 8; row++ {
		ranks[row] = formatFENRank(board, row)
	}
	return strings.Join(ranks, "/")
}

// formatFENRank writes one rank, collapsing runs of empty squares into a digit.
func formatFENRank(board IBoard, row int) string {
	var rank strings.Builder
	emptyRun := 0
	for column := 0; column < 8; column++ {
		piece, _ := board.GetPiece(column, row)
		if piece == nil {
			emptyRun++
			continue
		}
		if emptyRun > 0 {
			rank.WriteString(strconv.Itoa(emptyRun))
			emptyRun = 0
		}
		rank.WriteRune(fenLetter(piece))
	}
	if emptyRun > 0 {
		rank.WriteString(strconv.Itoa(emptyRun))
	}
	return rank.String()
}

func fenLetter(piece IPiece) rune {
	letter := fenLetterByType[piece.GetType()]
	if piece.GetColor() == common.White {
		return unicode.ToUpper(letter)
	}
	return letter
}

func formatFENSideToMove(color common.Color) string {
	if color == common.White {
		return "w"
	}
	return "b"
}

func formatFENCastling(rights CastlingRights) string {
	var field strings.Builder
	for _, side := range castlingSides {
		if rights.Has(side.right) {
			field.WriteByte(side.fenLetter)
		}
	}
	if field.Len() == 0 {
		return "-"
	}
	return field.String()
}

func formatFENEnPassant(target *Position) string {
	if target == nil {
		return "-"
	}
	square, err := (&CoordinateConverter{}).ToAlgebraic(target.X, target.Y)
	if err != nil {
		panic(fmt.Sprintf("en passant target %v is off the board: %v", *target, err))
	}
	return square
}
