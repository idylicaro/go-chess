// Package chess is a chess rules engine: legal move generation, check,
// checkmate and the draw rules, FEN, SAN and PGN, undo, and an optional
// injectable chess clock.
//
// A Game is created with NewGame, NewTimedGame or ParseFEN. Moves are given
// in UCI notation (ParseMove) or SAN (Game.ParseSAN) and played with
// Game.Move. After each move, Game.Result and Game.IsOver say whether the
// game ended and how.
//
// Perft and Divide check move generation against known node counts.
//
// docs/API.md in the repository lists what is stable.
package chess
