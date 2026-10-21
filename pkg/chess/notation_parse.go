package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/uci"
	"regexp"
	"strings"
)

// sanPattern captures: piece letter, origin file, origin rank, capture
// marker, destination, promotion piece. Castling is handled separately.
var sanPattern = regexp.MustCompile(`^([NBRQK])?([a-h])?([1-8])?(x)?([a-h][1-8])(?:=?([NBRQ]))?$`)

// sanPieceTypes maps SAN piece letters to piece types.
var sanPieceTypes = map[string]common.PieceType{
	"": common.Pawn, "N": common.Knight, "B": common.Bishop, "R": common.Rook, "Q": common.Queen, "K": common.King,
}

// sanRequest is what a SAN string pins down about the move it names.
type sanRequest struct {
	pieceType  common.PieceType
	fromFile   string
	fromRank   string
	to         string
	promotion  common.PieceType // Pawn means "no promotion given"
	isCastling bool
	kingSide   bool
}

// ParseSAN resolves a SAN string to the legal move it names in the current
// position (#9). It tolerates what readers accept in practice: a missing or
// extra "x", redundant disambiguation, "e8Q" without "=", "0-0" for "O-O",
// and trailing "+", "#", "!" or "?" annotations.
//
//	move, err := NewGame().ParseSAN("Nf3") // g1f3
func (g *Game) ParseSAN(san string) (uci.Move, error) {
	request, err := parseSANRequest(san)
	if err != nil {
		return uci.Move{}, err
	}
	var matches []uci.Move
	for _, move := range g.legalUCIMoves() {
		if g.matchesSANRequest(move, request) {
			matches = append(matches, move)
		}
	}
	return pickSANMatch(san, matches, g.FEN())
}

func parseSANRequest(san string) (sanRequest, error) {
	clean := strings.TrimRight(san, "+#!?")
	switch clean {
	case "O-O", "0-0":
		return sanRequest{isCastling: true, kingSide: true}, nil
	case "O-O-O", "0-0-0":
		return sanRequest{isCastling: true}, nil
	}
	parts := sanPattern.FindStringSubmatch(clean)
	if parts == nil {
		return sanRequest{}, fmt.Errorf("invalid SAN %q: expected e.g. \"e4\", \"Nbd2\", \"exd5\", \"e8=Q\" or \"O-O\"", san)
	}
	request := sanRequest{pieceType: sanPieceTypes[parts[1]], fromFile: parts[2], fromRank: parts[3], to: parts[5]}
	if parts[6] != "" {
		request.promotion = sanPieceTypes[parts[6]]
	}
	return request, nil
}

func (g *Game) matchesSANRequest(move uci.Move, request sanRequest) bool {
	piece, _ := g.board.GetPiece(move.FromX, move.FromY)
	if request.isCastling {
		return isCastlingMove(piece, move) && (move.ToX > move.FromX) == request.kingSide
	}
	origin := squareName(move.FromX, move.FromY)
	return piece.GetType() == request.pieceType &&
		!isCastlingMove(piece, move) &&
		squareName(move.ToX, move.ToY) == request.to &&
		strings.HasPrefix(origin, request.fromFile) &&
		strings.HasSuffix(origin, request.fromRank) &&
		move.Promotion == request.promotion
}

// pickSANMatch requires exactly one legal move to match.
func pickSANMatch(san string, matches []uci.Move, fen string) (uci.Move, error) {
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return uci.Move{}, fmt.Errorf("SAN %q matches no legal move in position %q (a promotion needs its piece, e.g. \"e8=Q\")", san, fen)
	default:
		return uci.Move{}, fmt.Errorf("SAN %q is ambiguous in position %q: matches %v, add the origin file or rank", san, fen, matches)
	}
}
