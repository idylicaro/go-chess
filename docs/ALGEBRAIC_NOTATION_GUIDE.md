# Algebraic Notation Implementation Guide

## Overview

This guide will walk you through implementing Standard Algebraic Notation (SAN) parsing and generation in your Go chess engine.

## Phase 1: Understanding the Components

### 1.1 What you've already built:

- ✅ `CoordinateConverter` - converts between internal coordinates and algebraic squares
- ✅ `AlgebraicNotation` struct - skeleton for the main functionality
- ✅ Test framework - ready to guide your development

### 1.2 SAN Components to understand:

```
Move: "Nf3"
- N = piece type (Knight)
- f3 = destination square

Move: "Nbd2"
- N = piece type (Knight)
- b = disambiguation (source file)
- d2 = destination square

Move: "exd5"
- e = source file (pawn capture)
- x = capture symbol
- d5 = destination square
```

## Phase 2: Implementation Steps

### Step 1: Update GetPieceSymbol method

First, fix the `GetPieceSymbol` method in `coordinate_converter.go`:

```go
// Update this method to use your actual PieceType enum
func (cc *CoordinateConverter) GetPieceSymbol(pieceType common.PieceType) string {
	switch pieceType {
	case common.Pawn:
		return ""
	case common.Knight:
		return "N"
	case common.Bishop:
		return "B"
	case common.Rook:
		return "R"
	case common.Queen:
		return "Q"
	case common.King:
		return "K"
	default:
		return "?"
	}
}
```

### Step 2: Implement basic GenerateMove

Start with simple cases (no captures, no checks, no disambiguation):

```go
func (an *AlgebraicNotation) GenerateMove(move uci.Move) (string, error) {
	// 1. Get the piece being moved
	piece, err := an.board.GetPiece(move.FromX, move.FromY)
	if err != nil {
		return "", err
	}

	// 2. Get piece symbol
	converter := &CoordinateConverter{}
	symbol := converter.GetPieceSymbol(piece.GetType())

	// 3. Get destination square
	destSquare, err := converter.ToAlgebraic(move.ToX, move.ToY)
	if err != nil {
		return "", err
	}

	// 4. Combine (for now, no captures/checks/disambiguation)
	return symbol + destSquare, nil
}
```

### Step 3: Add capture detection

Enhance GenerateMove to detect captures:

```go
// Check if destination square is occupied by opponent
destPiece, destErr := an.board.GetPiece(move.ToX, move.ToY)
isCapture := destErr == nil && destPiece != nil && destPiece.GetColor() != piece.GetColor()

// For pawns, show source file on captures
if piece.GetType() == common.Pawn && isCapture {
    sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)
    return sourceSquare[0:1] + "x" + destSquare, nil
}

// For other pieces
captureSymbol := ""
if isCapture {
    captureSymbol = "x"
}

return symbol + captureSymbol + destSquare, nil
```

### Step 4: Implement basic ParseMove

Start parsing simple moves:

```go
func (an *AlgebraicNotation) ParseMove(sanMove string) (uci.Move, error) {
	// Remove check/checkmate symbols
	cleanMove := strings.TrimSuffix(strings.TrimSuffix(sanMove, "+"), "#")

	// Handle castling first
	if cleanMove == "O-O" {
		// TODO: Return kingside castling move
	}
	if cleanMove == "O-O-O" {
		// TODO: Return queenside castling move
	}

	// Parse destination square (always last 2 characters)
	if len(cleanMove) < 2 {
		return uci.Move{}, fmt.Errorf("invalid move: %s", sanMove)
	}

	destSquare := cleanMove[len(cleanMove)-2:]
	converter := &CoordinateConverter{}
	destX, destY, err := converter.FromAlgebraic(destSquare)
	if err != nil {
		return uci.Move{}, err
	}

	// Determine piece type
	var pieceType common.PieceType
	if len(cleanMove) == 2 {
		// Pawn move (e.g., "e4")
		pieceType = common.Pawn
	} else {
		// Named piece (e.g., "Nf3")
		switch cleanMove[0] {
		case 'N':
			pieceType = common.Knight
		case 'B':
			pieceType = common.Bishop
		case 'R':
			pieceType = common.Rook
		case 'Q':
			pieceType = common.Queen
		case 'K':
			pieceType = common.King
		default:
			return uci.Move{}, fmt.Errorf("unknown piece: %c", cleanMove[0])
		}
	}

	// Find the source square
	// TODO: Search board for pieces of this type that can reach destX, destY
	// TODO: Handle disambiguation if multiple pieces can reach the destination

	return uci.Move{}, fmt.Errorf("not fully implemented yet")
}
```

## Phase 3: Disambiguation Logic

### Step 5: Implement disambiguation detection

```go
func (an *AlgebraicNotation) isAmbiguous(move uci.Move, piece IPiece) bool {
	// Find all pieces of same type and color that can reach the destination
	candidateMoves := []uci.Move{}

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			candidate, err := an.board.GetPiece(x, y)
			if err != nil || candidate == nil {
				continue
			}

			// Same type and color?
			if candidate.GetType() == piece.GetType() && candidate.GetColor() == piece.GetColor() {
				// Can this piece reach the destination?
				validMoves := candidate.GetValidMoves(x, y, an.board)
				for _, validMove := range validMoves {
					if validMove.X == move.ToX && validMove.Y == move.ToY {
						candidateMoves = append(candidateMoves, uci.Move{
							FromX: x, FromY: y,
							ToX: move.ToX, ToY: move.ToY,
						})
					}
				}
			}
		}
	}

	return len(candidateMoves) > 1
}
```

### Step 6: Implement disambiguation string generation

```go
func (an *AlgebraicNotation) getDisambiguation(move uci.Move, piece IPiece) string {
	// Similar logic to isAmbiguous, but determine what disambiguation is needed
	// Priority: file > rank > both

	// Find all conflicting pieces
	conflicts := an.findConflictingMoves(move, piece)

	// Check if file disambiguation is enough
	fileConflict := false
	for _, conflict := range conflicts {
		if conflict.FromX == move.FromX {
			fileConflict = true
			break
		}
	}

	if !fileConflict {
		// File disambiguation is sufficient
		converter := &CoordinateConverter{}
		sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)
		return sourceSquare[0:1] // return file only
	}

	// Check if rank disambiguation is enough
	rankConflict := false
	for _, conflict := range conflicts {
		if conflict.FromY == move.FromY {
			rankConflict = true
			break
		}
	}

	if !rankConflict {
		// Rank disambiguation is sufficient
		converter := &CoordinateConverter{}
		sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)
		return sourceSquare[1:2] // return rank only
	}

	// Need full square disambiguation
	converter := &CoordinateConverter{}
	sourceSquare, _ := converter.ToAlgebraic(move.FromX, move.FromY)
	return sourceSquare
}
```

## Phase 4: Testing Strategy

1. **Start simple**: Test pawn moves and basic piece moves
2. **Add complexity gradually**: captures, then disambiguation, then special moves
3. **Use TDD**: Write a failing test, make it pass, refactor
4. **Test edge cases**: multiple disambiguations, en passant, promotion

## Phase 5: Integration with Game

Once your notation system works, integrate it into your `Game` struct:

```go
// Add to Game struct
func (g *Game) MakeMove(sanMove string) error {
	notation := NewAlgebraicNotation(g.board, g)
	uciMove, err := notation.ParseMove(sanMove)
	if err != nil {
		return err
	}

	// Use existing MakeMove logic with UCI move
	return g.makeMoveUCI(uciMove)
}

func (g *Game) GetLastMoveNotation() (string, error) {
	if len(g.moveHistory) == 0 {
		return "", fmt.Errorf("no moves in history")
	}

	lastMove := g.moveHistory[len(g.moveHistory)-1]
	notation := NewAlgebraicNotation(g.board, g)
	return notation.GenerateMove(lastMove)
}
```

## Key Learning Points

1. **Separation of Concerns**: Notice how we separated coordinate conversion, notation parsing/generation, and disambiguation into different responsibilities.

2. **Test-Driven Development**: The tests guide your implementation and catch regressions.

3. **Incremental Implementation**: Start with simple cases and add complexity gradually.

4. **Disambiguation Priority**: File > Rank > Full square - this is the standard priority in chess notation.

## Next Steps

1. Start with Step 1 (fix GetPieceSymbol)
2. Run the existing tests to see them fail
3. Implement GenerateMove for simple cases
4. Make the tests pass one by one
5. Add more complex test cases as you implement features

Remember: The goal is to learn by implementing, not to get perfect code immediately. Start simple and build up!
