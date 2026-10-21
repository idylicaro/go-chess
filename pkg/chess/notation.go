package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
)

// SAN formats a legal move of the side to move in Standard Algebraic
// Notation, including castling, en passant, promotion, minimal
// disambiguation and the check (+) or mate (#) suffix (#9).
// A promotion without a piece (zero Promotion) is a queen promotion, as in Move.
//
//	NewGame().SAN(uci.Move{FromX: 6, FromY: 7, ToX: 5, ToY: 5}) // "Nf3"
func (g *Game) SAN(move uci.Move) (string, error) {
	legal, err := g.findLegalMove(move)
	if err != nil {
		return "", err
	}
	return g.sanWithoutSuffix(legal) + g.checkSuffix(legal), nil
}

// findLegalMove returns the legal move matching move, with Promotion
// normalized: queen when a promotion has no valid piece, zero otherwise.
func (g *Game) findLegalMove(move uci.Move) (uci.Move, error) {
	move.Promotion = g.normalizedPromotion(move)
	for _, legal := range g.legalUCIMoves() {
		if legal == move {
			return legal, nil
		}
	}
	return uci.Move{}, fmt.Errorf("move %s is not legal in position %q", move, g.FEN())
}

func (g *Game) normalizedPromotion(move uci.Move) common.PieceType {
	piece, _ := g.board.GetPiece(move.FromX, move.FromY)
	if piece == nil || !isPromotionMove(piece, move) {
		return common.Pawn
	}
	for _, choice := range promotionChoices {
		if move.Promotion == choice {
			return choice
		}
	}
	return common.Queen
}

// sanWithoutSuffix assumes move is legal in the current position.
func (g *Game) sanWithoutSuffix(move uci.Move) string {
	piece, _ := g.board.GetPiece(move.FromX, move.FromY)
	if isCastlingMove(piece, move) {
		if move.ToX > move.FromX {
			return "O-O"
		}
		return "O-O-O"
	}
	if piece.GetType() == common.Pawn {
		return g.pawnSAN(piece, move)
	}
	capture := ""
	if g.isCapture(piece, move) {
		capture = "x"
	}
	return sanPieceLetter(piece.GetType()) + g.disambiguation(piece, move) + capture + squareName(move.ToX, move.ToY)
}

// pawnSAN writes "e4", "exd5" (also en passant) and "e8=Q".
func (g *Game) pawnSAN(pawn IPiece, move uci.Move) string {
	san := squareName(move.ToX, move.ToY)
	if g.isCapture(pawn, move) {
		san = squareName(move.FromX, move.FromY)[:1] + "x" + san
	}
	if isPromotionMove(pawn, move) {
		san += "=" + sanPieceLetter(move.Promotion)
	}
	return san
}

func (g *Game) isCapture(piece IPiece, move uci.Move) bool {
	target, _ := g.board.GetPiece(move.ToX, move.ToY)
	return target != nil || isEnPassantCapture(piece, move, g.enPassantTarget)
}

// disambiguation returns the shortest origin hint (file, else rank, else
// both) that separates move from other legal moves of the same piece type
// to the same square. Only legal moves count, so a pinned twin is ignored.
func (g *Game) disambiguation(piece IPiece, move uci.Move) string {
	rivals := g.rivalOrigins(piece, move)
	if len(rivals) == 0 {
		return ""
	}
	sameFile, sameRank := false, false
	for _, rival := range rivals {
		sameFile = sameFile || rival.X == move.FromX
		sameRank = sameRank || rival.Y == move.FromY
	}
	origin := squareName(move.FromX, move.FromY)
	switch {
	case !sameFile:
		return origin[:1]
	case !sameRank:
		return origin[1:]
	default:
		return origin
	}
}

// rivalOrigins lists the other squares from which a piece of the same type
// can legally reach move's destination.
func (g *Game) rivalOrigins(piece IPiece, move uci.Move) []Position {
	var rivals []Position
	for _, other := range g.legalUCIMoves() {
		sameSquare := other.FromX == move.FromX && other.FromY == move.FromY
		if sameSquare || other.ToX != move.ToX || other.ToY != move.ToY {
			continue
		}
		if rival, _ := g.board.GetPiece(other.FromX, other.FromY); rival.GetType() == piece.GetType() {
			rivals = append(rivals, Position{X: other.FromX, Y: other.FromY})
		}
	}
	return rivals
}

// checkSuffix plays move on a copy: "#" if it mates, "+" if it checks.
func (g *Game) checkSuffix(move uci.Move) string {
	child := g.childAfter(move)
	if !child.isSideToMoveInCheck() {
		return ""
	}
	if len(child.legalUCIMoves()) == 0 {
		return "#"
	}
	return "+"
}

func (g *Game) isSideToMoveInCheck() bool {
	kingX, kingY := g.board.GetKingPosition(g.currentTurn)
	return g.board.IsSquareAttacked(kingX, kingY, g.getOpponentColor())
}

// sanPieceLetter is "" for pawns and N/B/R/Q/K otherwise.
func sanPieceLetter(pieceType common.PieceType) string {
	return (&CoordinateConverter{}).GetPieceSymbol(pieceType)
}

// squareName formats internal coordinates known to be on the board, e.g. "e4".
func squareName(x, y int) string {
	square, _ := (&CoordinateConverter{}).ToAlgebraic(x, y)
	return square
}
