package main

import (
	"fmt"
	"idylicaro/go-chess/internal/chess"
	"idylicaro/go-chess/internal/uci"
)

func main() {
	// Create a new game
	game := chess.NewGame()

	// Create the notation handler
	notation := chess.NewAlgebraicNotation(game.GetBoard(), game)

	fmt.Println("=== Algebraic Notation Example ===")

	// Example 1: Try to generate notation for e2-e4
	move := uci.Move{
		FromX: 4, FromY: 6, // e2 in internal coordinates
		ToX: 4, ToY: 4, // e4 in internal coordinates
	}

	san, err := notation.GenerateMove(move)
	if err != nil {
		fmt.Printf("Error generating SAN: %v\n", err)
	} else {
		fmt.Printf("UCI %v -> SAN: %s\n", move, san)
	}

	// Example 2: Try to parse a SAN move
	sanMove := "e4"
	parsedMove, err := notation.ParseMove(sanMove)
	if err != nil {
		fmt.Printf("Error parsing SAN '%s': %v\n", sanMove, err)
	} else {
		fmt.Printf("SAN '%s' -> UCI: %v\n", sanMove, parsedMove)
	}

	// Example 3: Show coordinate conversion
	converter := &chess.CoordinateConverter{}

	testCoords := []struct {
		x, y int
		name string
	}{
		{4, 6, "e2"},
		{4, 4, "e4"},
		{6, 7, "g1"},
		{5, 5, "f3"},
	}

	fmt.Println("\n=== Coordinate Conversion Examples ===")
	for _, coord := range testCoords {
		algebraic, _ := converter.ToAlgebraic(coord.x, coord.y)
		fmt.Printf("Internal (%d,%d) -> Algebraic: %s (expected: %s)\n",
			coord.x, coord.y, algebraic, coord.name)
	}
}
