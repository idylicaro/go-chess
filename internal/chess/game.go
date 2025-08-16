package chess

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/uci"
)

type Game struct {
	board           IBoard
	currentTurn     Color
	state           GameState
	enPassantTarget *Position
}

type Color int

const (
	White Color = iota
	Black
)

type GameState int

const (
	InProgress GameState = iota
	Check
	Checkmate
	Stalemate
	Draw
)

func NewGame() *Game {
	return &Game{
		board:           NewInitialBoard(),
		currentTurn:     White,
		state:           InProgress,
		enPassantTarget: nil,
	}
}

func (g *Game) GetCurrentTurn() Color {
	return g.currentTurn
}

func (g *Game) GetState() GameState {
	return g.state
}

func (g *Game) GetBoard() IBoard {
	return g.board
}

func (g *Game) Move(move uci.Move) error {
	pieceToMove, _ := g.board.GetPiece(move.FromX, move.FromY)

	fmt.Printf("[DEBUG] Moving piece: %v | PieceColor: %v | Check Bool: %v\n", pieceToMove, pieceToMove.GetColor(), pieceToMove.GetColor() != g.currentTurn)
	if pieceToMove == nil || pieceToMove.GetColor() != g.currentTurn {
		return fmt.Errorf("not your piece or not your turn")
	}

	legalMoves, err := g.getLegalMovesForPiece(move.FromX, move.FromY)
	if err != nil {
		return err
	}

	isLegal := false
	for _, legalMove := range legalMoves {
		if legalMove.X == move.ToX && legalMove.Y == move.ToY {
			isLegal = true
			break
		}
	}
	if !isLegal {
		return fmt.Errorf("illegal move: king would be in check or move is not valid")
	}

	if pieceToMove.GetType() == common.King {
		isCastling := (move.ToX == move.FromX+2 || move.ToX == move.FromX-2)
		if isCastling {
			// Determine the rook's position and its destination
			rookFromX := 0
			rookToX := 0
			if move.ToX > move.FromX { // Short castling
				rookFromX = 7
				rookToX = move.FromX + 1
			} else { // Long castling
				rookFromX = 0
				rookToX = move.FromX - 1
			}
			rookFromY := move.FromY
			rookToY := move.FromY

			g.board.MovePiece(rookFromX, rookFromY, rookToX, rookToY)
		}
	}

	// En passant capture
	if pieceToMove.GetType() == common.Pawn && g.enPassantTarget != nil &&
		move.ToX == g.enPassantTarget.X && move.ToY == g.enPassantTarget.Y {

		// Calculate the position of the captured pawn
		capturedPawnY := move.FromY
		if pieceToMove.GetColor() == White {
			capturedPawnY = move.ToY + 1
		} else {
			capturedPawnY = move.ToY - 1
		}

		g.board.RemovePiece(move.ToX, capturedPawnY)
	}

	isPromotion := false
	if pieceToMove.GetType() == common.Pawn {
		if (pieceToMove.GetColor() == White && move.ToY == 0) ||
			(pieceToMove.GetColor() == Black && move.ToY == 7) {
			isPromotion = true
		}
	}

	if isPromotion {
		// Remove the pawn from the board
		g.board.RemovePiece(move.FromX, move.FromY)

		if move.Promotion == common.Rook {
			g.board.AddPiece(NewRook(pieceToMove.GetColor()), move.ToX, move.ToY)
		} else if move.Promotion == common.Bishop {
			g.board.AddPiece(NewBishop(pieceToMove.GetColor()), move.ToX, move.ToY)
		} else if move.Promotion == common.Knight {
			g.board.AddPiece(NewKnight(pieceToMove.GetColor()), move.ToX, move.ToY)
		} else { // Queen is the default promotion
			g.board.AddPiece(NewQueen(pieceToMove.GetColor()), move.ToX, move.ToY)
		}
	} else {
		// if not a promotion, move the piece
		g.board.MovePiece(move.FromX, move.FromY, move.ToX, move.ToY)
	}

	// Clear en passant target of last move
	g.enPassantTarget = nil
	// En passant Check
	if pieceToMove.GetType() == common.Pawn {
		deltaY := move.ToY - move.FromY
		if (pieceToMove.GetColor() == White && deltaY == -2) || (pieceToMove.GetColor() == Black && deltaY == 2) {
			targetY := move.FromY + deltaY/2
			g.enPassantTarget = &Position{X: move.ToX, Y: targetY}
		}
	}

	g.currentTurn = g.getOpponentColor()

	g.updateGameState()

	return nil
}

func (g *Game) getLegalMovesForPiece(fromX, fromY int) ([]Position, error) {
	piece, err := g.board.GetPiece(fromX, fromY)
	if err != nil {
		return nil, err
	}
	if piece.GetColor() != g.currentTurn {
		return nil, fmt.Errorf("this piece is not your color")
	}

	// 1. Obtenha todos os movimentos possíveis da peça (válidos, mas não necessariamente legais)
	var possibleMoves []Position
	if pawn, ok := piece.(*Pawn); ok {
		possibleMoves = pawn.GetValidMovesWithEnPassant(fromX, fromY, g.board, g.enPassantTarget)
	} else {
		possibleMoves = piece.GetValidMoves(fromX, fromY, g.board)
	}

	var legalMoves []Position
	for _, move := range possibleMoves {
		// 2. Crie uma CÓPIA do tabuleiro para simular o movimento
		boardCopy := g.board.Copy()

		// 3. Simule o movimento na cópia
		_ = boardCopy.MovePiece(fromX, fromY, move.X, move.Y) // Ignore o erro pois o move é válido

		// 4. Encontre a nova posição do rei
		kingX, kingY := boardCopy.GetKingPosition(g.currentTurn)

		// 5. Verifique se o rei está em xeque na cópia do tabuleiro
		if !boardCopy.IsSquareAttacked(kingX, kingY, g.getOpponentColor()) {
			// Se o rei não está em xeque, o movimento é legal
			legalMoves = append(legalMoves, move)
		}
	}

	return legalMoves, nil
}

func (g *Game) getOpponentColor() Color {
	if g.currentTurn == White {
		return Black
	}
	return White
}

// getAllLegalMoves returns a slice of all legal move positions for the given color.
func (g *Game) getAllLegalMoves(color Color) []Position {
	var allMoves []Position
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece != nil && piece.GetColor() == color {
				moves, _ := g.getLegalMovesForPiece(x, y)
				allMoves = append(allMoves, moves...)
			}
		}
	}
	return allMoves
}

func (g *Game) updateGameState() {
	allLegalMoves := g.getAllLegalMoves(g.currentTurn)

	kingX, kingY := g.board.GetKingPosition(g.currentTurn)
	isInCheck := g.board.IsSquareAttacked(kingX, kingY, g.getOpponentColor())

	if isInCheck {
		if len(allLegalMoves) == 0 {
			g.state = Checkmate
		} else {
			g.state = Check
		}
	} else {
		if len(allLegalMoves) == 0 {
			g.state = Stalemate
		} else {
			g.state = InProgress
		}
	}
}

func (c Color) String() string {
	switch c {
	case White:
		return "White"
	case Black:
		return "Black"
	default:
		return "Unknown"
	}
}
