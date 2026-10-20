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
	castlingRights  CastlingRights   // Castles still available (FEN castling field)
	halfMoveClock   int              // Plies since the last capture or pawn move, for the fifty-move rule
	fullMoveNumber  int              // Starts at 1, incremented after each Black move (FEN)
	positionHistory map[string]int   // For threefold repetition
	moveHistory     []uci.Move       // For move history tracking
	gameTimer       *timer.GameTimer // Timer control
	//TODO: Draw Reason
}

// fiftyMoveRulePlies is 50 moves per side; halfMoveClock counts plies (#5).
const fiftyMoveRulePlies = 100

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
		castlingRights:  AllCastlingRights,
		fullMoveNumber:  1,
		gameTimer:       nil, // No timer by default
	}
}

// NewTimedGame creates a game with a chess clock reading time from clock.
// Timeouts are detected by polling the clock on each move, so no timeout
// callback is installed (the old one printed from inside the library, #6).
//
//	game := NewTimedGame(timer.Blitz3Plus2, timer.SystemClock{})
//	_ = game.StartTimer()
func NewTimedGame(timeConfig timer.TimeConfig, clock timer.Clock) *Game {
	game := NewGame()
	game.gameTimer = timer.NewGameTimerWithClock(timeConfig, clock)
	return game
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

	if err := g.rejectIfOutOfTime(); err != nil {
		return err
	}

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

	if err := applyMoveToBoard(g.board, move, g.enPassantTarget); err != nil {
		return err
	}
	g.castlingRights = g.castlingRights.afterMove(pieceToMove, move)
	if g.currentTurn == common.Black {
		g.fullMoveNumber++
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

	// Switch turn first
	nextTurn := g.getOpponentColor()

	if g.gameTimer != nil {
		g.gameTimer.SwitchPlayer(nextTurn)
	}

	// Update current turn
	g.currentTurn = nextTurn

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
	if piece.GetType() == common.King {
		possibleMoves = append(possibleMoves, g.castlingMoves(Position{X: fromX, Y: fromY})...)
	}

	var legalMoves []Position
	for _, move := range possibleMoves {
		// 2. Crie uma CÓPIA do tabuleiro para simular o movimento
		boardCopy := g.board.Copy()

		// 3. Simule o movimento na cópia — com roque, en passant e promoção completos (#4)
		simulated := uci.Move{FromX: fromX, FromY: fromY, ToX: move.X, ToY: move.Y}
		moveErr := applyMoveToBoard(boardCopy, simulated, g.enPassantTarget)
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

// rejectIfOutOfTime blocks a move by a side whose clock has run out, before
// anything on the board changes. Previously the clock was only checked after
// the move had been applied, using stale banked time, so it never fired (#6).
func (g *Game) rejectIfOutOfTime() error {
	if state, timedOut := g.timeoutState(); timedOut {
		g.state = state
		return fmt.Errorf("game over: %v ran out of time (state %v)", g.currentTurn, state)
	}
	return nil
}

// timeoutState returns TimeoutWhite/TimeoutBlack when either clock has run out.
func (g *Game) timeoutState() (GameState, bool) {
	if g.gameTimer == nil {
		return InProgress, false
	}
	if g.gameTimer.IsTimeout(common.White) {
		return TimeoutWhite, true
	}
	if g.gameTimer.IsTimeout(common.Black) {
		return TimeoutBlack, true
	}
	return InProgress, false
}

func (g *Game) updateGameState() {
	if state, timedOut := g.timeoutState(); timedOut {
		g.state = state
		return
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
	if g.halfMoveClock >= fiftyMoveRulePlies {
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

	hash += fmt.Sprintf("castle:%d", g.castlingRights)
	return hash
}

// placedPiece is a non-king piece together with the square it stands on.
type placedPiece struct {
	pieceType common.PieceType
	position  Position
}

// isInsufficientMaterial reports whether neither side can ever deliver mate.
// Covers: bare kings, a single knight, and any number of bishops (either side)
// all standing on squares of the same color — which includes K+B vs K and
// K+B vs K+B with same-colored bishops (#2).
//
//	game.isInsufficientMaterial() // true for K+B(c1) vs K+B(f8)
func (g *Game) isInsufficientMaterial() bool {
	material := g.collectNonKingPieces()
	if len(material) == 0 {
		return true
	}
	if len(material) == 1 && material[0].pieceType == common.Knight {
		return true
	}
	return allBishopsOnSameSquareColor(material)
}

// collectNonKingPieces lists every piece on the board except the kings.
func (g *Game) collectNonKingPieces() []placedPiece {
	var material []placedPiece
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece == nil || piece.GetType() == common.King {
				continue
			}
			material = append(material, placedPiece{piece.GetType(), Position{x, y}})
		}
	}
	return material
}

// allBishopsOnSameSquareColor is true when material is non-empty, made only of
// bishops, and every bishop stands on the same square color.
func allBishopsOnSameSquareColor(material []placedPiece) bool {
	if len(material) == 0 {
		return false
	}
	firstSquareColor := squareColor(material[0].position)
	for _, placed := range material {
		if placed.pieceType != common.Bishop || squareColor(placed.position) != firstSquareColor {
			return false
		}
	}
	return true
}

// squareColor returns 0 for light squares and 1 for dark squares.
func squareColor(position Position) int {
	return (position.X + position.Y) % 2
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
	case TimeoutWhite:
		return "TimeoutWhite"
	case TimeoutBlack:
		return "TimeoutBlack"
	default:
		return "Unknown"
	}
}
