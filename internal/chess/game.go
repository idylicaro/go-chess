package chess

type Game struct {
	board         Board
	currentPlayer Color
	state         GameState
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

func (b *Game) GetCurrentPlayer() Color {
	return b.currentPlayer
}

func (b *Game) GetState() GameState {
	return b.state
}

func (b *Game) Move() 
