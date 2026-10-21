# Go Chess

[![CI](https://github.com/idylicaro/go-chess/actions/workflows/ci.yml/badge.svg)](https://github.com/idylicaro/go-chess/actions/workflows/ci.yml)

A chess rules library for Go, with a small terminal client. It has no dependencies outside the standard library.

- Full legal-move generation: castling, en passant, promotion, check, checkmate and stalemate.
- Draw rules: fifty-move, threefold repetition, insufficient material, agreement, and timeout against insufficient material.
- Resignation, draw offers and `Undo`.
- FEN import/export, SAN parsing/formatting and PGN export.
- A chess clock with an injectable time source.
- Perft tests against the six reference positions.

## Install

```bash
go get github.com/idylicaro/go-chess
```

The module path is `github.com/idylicaro/go-chess`; the importable package is `github.com/idylicaro/go-chess/pkg/chess`.

## Usage

```go
package main

import (
	"fmt"

	"github.com/idylicaro/go-chess/pkg/chess"
)

func main() {
	game := chess.NewGame()

	for _, notation := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
		move, err := chess.ParseMove(notation) // UCI notation
		if err != nil {
			panic(err)
		}
		if err := game.Move(move); err != nil {
			panic(err)
		}
	}

	fmt.Println(game.Result())            // 0-1
	fmt.Println(game.GetState())          // Checkmate
	fmt.Println(len(game.LegalMoves()))   // 0

	_ = game.Undo()                       // take the last move back
	fmt.Println(game.Result())            // *
}
```

Other entry points:

| Need | API |
|---|---|
| Start from a position | `chess.ParseFEN(fen)`, `game.FEN()` |
| Moves in SAN | `game.SAN(move)`, `game.ParseSAN(san)` |
| Export a game | `game.PGN(chess.PGNHeaders{...})` |
| How the game ended | `game.Result()`, `game.DrawReason()`, `game.IsOver()` |
| Resign / draw by agreement | `game.Resign(color)`, `game.OfferDraw(color)`, `game.AcceptDraw(color)`, `game.DeclineDraw(color)` |
| Timed games | `chess.NewTimedGame(chess.Blitz3Plus2, chess.SystemClock{})` |
| Move-generator check | `game.Perft(depth)`, `game.Divide(depth)` |

Errors from a game that has already ended wrap `chess.ErrGameOver`; use `errors.Is`.

## Command-line client

```bash
make run          # builds bin/chess and starts it
```

Moves are typed in UCI notation (`e2e4`, `e7e8q`). Other commands: `moves`, `undo`, `resign`, `draw`, `accept`, `decline`, `exit`.

## Project layout

```
cmd/cli/        terminal client
pkg/chess/      public API: board, pieces, game, FEN, SAN, PGN, perft
internal/       common types, clock, UCI move parsing, debug logging
docs/           testing guide
scripts/        benchmark script
```

`pkg/chess` re-exports the few types it needs from `internal/` (`Color`, `Move`, `TimeConfig`, `Clock`), so callers never import `internal/`.

## Development

```bash
make check        # gofmt, go vet and go test -race (what CI runs)
make perft        # full perft suite (slow)
make bench        # benchmarks
make help         # every target
```

See [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for testing, benchmarking and debug logging.

## License

[MIT](LICENSE)
