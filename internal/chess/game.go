package chess

import (
	"fmt"
	common "idylicaro/go-chess/internal/common"
	"idylicaro/go-chess/internal/debug"
	"idylicaro/go-chess/internal/timer"
	"idylicaro/go-chess/internal/uci"
	"time"
)

type Game struct {
	board           IBoard
	currentTurn     common.Color
	state           GameState
	enPassantTarget *Position
	halfMoveClock   int              // For fifty-move rule
	positionHistory map[string]int   // For threefold repetition
	moveHistory     []uci.Move       // For move history tracking
	gameTimer       *timer.GameTimer // Timer control
	//TODO: Draw Reason
}

type GameState int

const (
	InProgress GameState = iota
	Check
	Checkmate
	Stalemate
	Draw
	TimeoutWhite
	TimeoutBlack
)

func NewGame() *Game {
	return &Game{
		board:           NewInitialBoard(),
		currentTurn:     common.White,
		state:           InProgress,
		enPassantTarget: nil,
		gameTimer:       nil, // No timer by default
	}
}

func NewGameWithTimer(timeConfig timer.TimeConfig) *Game {
	gameTimer := timer.NewGameTimer(timeConfig)

	// Set timeout callback
	gameTimer.SetTimeoutCallback(func(color common.Color) {
		fmt.Printf("Timeout for %v player!", color)
		// This will be called when timeout occurs
		// We'll handle it in the Move method
	})

	return &Game{
		board:           NewInitialBoard(),
		currentTurn:     common.White,
		state:           InProgress,
		enPassantTarget: nil,
		gameTimer:       gameTimer,
	}
}

func (g *Game) GetCurrentTurn() common.Color {
	return g.currentTurn
}

func (g *Game) GetState() GameState {
	return g.state
}

func (g *Game) GetBoard() IBoard {
	return g.board
}

// Timer related methods
func (g *Game) StartTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Start(g.currentTurn)
	return nil
}

func (g *Game) GetRemainingTime(color common.Color) (time.Duration, error) {
	if g.gameTimer == nil {
		return 0, fmt.Errorf("no timer configured for this game")
	}
	return g.gameTimer.GetRemainingTime(color), nil
}

func (g *Game) PauseTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Pause()
	return nil
}

func (g *Game) ResumeTimer() error {
	if g.gameTimer == nil {
		return fmt.Errorf("no timer configured for this game")
	}
	g.gameTimer.Resume()
	return nil
}

func (g *Game) HasTimer() bool {
	return g.gameTimer != nil
}

func (g *Game) Move(move uci.Move) error {
	defer func(start time.Time) {
		debug.LogTiming("Game.Move", start)
	}(time.Now())

	moveStr := fmt.Sprintf("(%d,%d)->(%d,%d)", move.FromX, move.FromY, move.ToX, move.ToY)
	debug.NewChessDebugger().LogMove(moveStr, g.currentTurn.String(), g.state)

	pieceToMove, _ := g.board.GetPiece(move.FromX, move.FromY)

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

	targetPiece, _ := g.board.GetPiece(move.ToX, move.ToY)
	if targetPiece != nil || pieceToMove.GetType() == common.Pawn {
		g.halfMoveClock = 0
	} else {
		g.halfMoveClock++
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
		var capturedPawnY int
		if pieceToMove.GetColor() == common.White {
			capturedPawnY = move.ToY + 1
		} else {
			capturedPawnY = move.ToY - 1
		}

		g.board.RemovePiece(move.ToX, capturedPawnY)
	}

	isPromotion := false
	if pieceToMove.GetType() == common.Pawn {
		if (pieceToMove.GetColor() == common.White && move.ToY == 0) ||
			(pieceToMove.GetColor() == common.Black && move.ToY == 7) {
			isPromotion = true
		}
	}

	if isPromotion {
		err := g.board.RemovePiece(move.FromX, move.FromY)
		if err != nil {
			return fmt.Errorf("failed to remove pawn during promotion: %v", err)
		}
		_ = g.board.RemovePiece(move.ToX, move.ToY) // Ignore error if square is empty

		var promotedPiece IPiece
		if move.Promotion == common.Rook {
			promotedPiece = NewRook(pieceToMove.GetColor())
		} else if move.Promotion == common.Bishop {
			promotedPiece = NewBishop(pieceToMove.GetColor())
		} else if move.Promotion == common.Knight {
			promotedPiece = NewKnight(pieceToMove.GetColor())
		} else { // Queen is the default promotion
			promotedPiece = NewQueen(pieceToMove.GetColor())
		}

		err = g.board.SetPiece(promotedPiece, move.ToX, move.ToY)
		if err != nil {
			return fmt.Errorf("failed to set promoted piece: %v", err)
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
		if (pieceToMove.GetColor() == common.White && deltaY == -2) || (pieceToMove.GetColor() == common.Black && deltaY == 2) {
			targetY := move.FromY + deltaY/2
			g.enPassantTarget = &Position{X: move.ToX, Y: targetY}
		}
	}

	g.moveHistory = append(g.moveHistory, move)

	// Handle timer if present
	if g.gameTimer != nil {
		// Check for timeout before switching
		if g.gameTimer.IsTimeout(g.getOpponentColor()) {
			if g.getOpponentColor() == common.White {
				g.state = TimeoutWhite
			} else {
				g.state = TimeoutBlack
			}
			return fmt.Errorf("game over - timeout")
		}

		// Switch timer to current player
		g.gameTimer.SwitchPlayer(g.currentTurn)
	}

	// Switch turn
	g.currentTurn = g.getOpponentColor()

	g.updateGameState()

	debugger := debug.NewChessDebugger()
	debugger.LogGameState(g)
	if boardFormatter, ok := g.board.(debug.BoardFormatter); ok {
		debugger.LogBoard(boardFormatter)
	}

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
		// Special castling validation for King pieces
		if piece.GetType() == common.King {
			isCastling := (move.X == fromX+2 || move.X == fromX-2)
			if isCastling {
				// Castling rules:
				// 1. King cannot be in check
				// 2. King cannot pass through check
				// 3. King cannot end in check (already validated below)

				kingX, kingY := fromX, fromY
				opponentColor := g.getOpponentColor()

				if g.board.IsSquareAttacked(kingX, kingY, opponentColor) {
					continue
				}

				var passedSquares []Position
				if move.X > fromX { // Kingside castling
					passedSquares = []Position{{X: fromX + 1, Y: fromY}, {X: fromX + 2, Y: fromY}}
				} else { // Queenside castling
					passedSquares = []Position{{X: fromX - 1, Y: fromY}, {X: fromX - 2, Y: fromY}}
				}

				canCastle := true
				for _, square := range passedSquares {
					if g.board.IsSquareAttacked(square.X, square.Y, opponentColor) {
						canCastle = false
						break
					}
				}

				if !canCastle {
					continue
				}
			}
		}

		// 2. Crie uma CÓPIA do tabuleiro para simular o movimento
		boardCopy := g.board.Copy()

		// 3. Simule o movimento na cópia
		moveErr := boardCopy.MovePiece(fromX, fromY, move.X, move.Y)
		if moveErr != nil {
			continue
		}

		// 4. Encontre a nova posição do rei
		kingX, kingY := boardCopy.GetKingPosition(g.currentTurn)

		// 5. Verifique se o rei está em xeque na cópia do tabuleiro
		isAttacked := boardCopy.IsSquareAttacked(kingX, kingY, g.getOpponentColor())
		if !isAttacked {
			// Se o rei não está em xeque, o movimento é legal
			legalMoves = append(legalMoves, move)
		}
	}

	return legalMoves, nil
}

func (g *Game) getOpponentColor() common.Color {
	if g.currentTurn == common.White {
		return common.Black
	}
	return common.White
}

// getAllLegalMoves returns a slice of all legal move positions for the given color.
func (g *Game) getAllLegalMoves(color common.Color) []Position {
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
	// Check for timeout first
	if g.gameTimer != nil {
		if g.gameTimer.IsTimeout(common.White) {
			g.state = TimeoutWhite
			return
		}
		if g.gameTimer.IsTimeout(common.Black) {
			g.state = TimeoutBlack
			return
		}
	}

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
	if g.isDraw() {
		g.state = Draw
	}
}

func (g *Game) isDraw() bool {
	if g.halfMoveClock >= 50 {
		return true
	}
	if g.isThreefoldRepetition() {
		return true
	}
	if g.isInsufficientMaterial() {
		return true
	}
	return false
}

func (g *Game) isThreefoldRepetition() bool {
	// Gerar hash da posição atual
	currentHash := g.generatePositionHash()

	// Inicializar o mapa se não existir
	if g.positionHistory == nil {
		g.positionHistory = make(map[string]int)
	}

	// Incrementar o contador para a posição atual
	g.positionHistory[currentHash]++

	// Verificar se a posição se repetiu 3 vezes
	return g.positionHistory[currentHash] >= 3
}

func (g *Game) generatePositionHash() string {
	hash := ""

	// Pieces position
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece != nil {
				hash += fmt.Sprintf("%d%d%s%s", x, y, piece.GetColor(), piece.GetType())
			}
		}
	}

	hash += fmt.Sprintf("turn:%s", g.currentTurn)

	if g.enPassantTarget != nil {
		hash += fmt.Sprintf("ep:%d%d", g.enPassantTarget.X, g.enPassantTarget.Y)
	}

	hash += fmt.Sprintf("castle:%t%t", g.board.HasCastlingRights(common.White), g.board.HasCastlingRights(common.Black))
	return hash
}

func (g *Game) isInsufficientMaterial() bool {
	pieces := make(map[common.PieceType]int)
	bishops := make(map[common.Color][]Position) // Para rastrear cor dos bispos

	// Contar peças
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece != nil && piece.GetType() != common.King {
				pieces[piece.GetType()]++
				if piece.GetType() == common.Bishop {
					bishops[piece.GetColor()] = append(bishops[piece.GetColor()], Position{x, y})
				}
			}
		}
	}

	totalPieces := 0
	for _, count := range pieces {
		totalPieces += count
	}

	// Specific checks
	// only kings
	if totalPieces == 0 {
		return true
	}
	// King + Bishop vs King
	if totalPieces == 1 && pieces[common.Bishop] == 1 {
		return true
	}
	// King + Knight vs King
	if totalPieces == 1 && pieces[common.Knight] == 1 {
		return true
	}
	// King and Bishop vs King and Bishop (Bishop into a diff color squares)
	if totalPieces == 2 && pieces[common.Bishop] == 2 {
		// Check if the bishops are on opposite colored squares
		if (bishops[common.White][0].X+bishops[common.White][0].Y)%2 != (bishops[common.Black][0].X+bishops[common.Black][0].Y)%2 {
			return true
		}
	}

	return false
}

// GetDebugInfo implements debug.GameStateProvider
func (g *Game) GetDebugInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["Current Turn"] = g.currentTurn.String()
	info["Game State"] = g.state.String()
	info["Half Move Clock"] = g.halfMoveClock
	info["Move History Length"] = len(g.moveHistory)

	if g.enPassantTarget != nil {
		info["En Passant Target"] = fmt.Sprintf("(%d,%d)", g.enPassantTarget.X, g.enPassantTarget.Y)
	} else {
		info["En Passant Target"] = "None"
	}

	if g.HasTimer() {
		whiteTime, _ := g.GetRemainingTime(common.White)
		blackTime, _ := g.GetRemainingTime(common.Black)
		info["White Time"] = whiteTime.String()
		info["Black Time"] = blackTime.String()
	}

	return info
}

// String methods for debug output
func (gs GameState) String() string {
	switch gs {
	case InProgress:
		return "InProgress"
	case Check:
		return "Check"
	case Checkmate:
		return "Checkmate"
	case Stalemate:
		return "Stalemate"
	case Draw:
		return "Draw"
	default:
		return "Unknown"
	}
}
