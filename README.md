# Go Chess

[![CI](https://github.com/idylicaro/go-chess/actions/workflows/ci.yml/badge.svg)](https://github.com/idylicaro/go-chess/actions/workflows/ci.yml)

A chess rules library for Go, with a terminal client (CLI and TUI). The library, `pkg/chess`, has no dependencies outside the standard library; only the client uses [bubbletea](https://github.com/charmbracelet/bubbletea) and [lipgloss](https://github.com/charmbracelet/lipgloss).

- Full legal-move generation: castling, en passant, promotion, check, checkmate and stalemate.
- Draw rules: fifty-move, threefold repetition, insufficient material, agreement, and timeout against insufficient material.
- Resignation, draw offers and `Undo`.
- FEN import/export, SAN parsing/formatting and PGN export.
- A chess clock with an injectable time source.
- Perft tests against the six reference positions.
- Its own UCI engine, `bin/chess-engine`: alpha-beta search with a transposition table, quiescence, null-move pruning, late-move reductions, futility pruning, Lazy SMP and Polyglot opening books, measured by SPRT with the built-in match runner.

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
make build                      # builds bin/chess
bin/chess play                  # two players at one keyboard
bin/chess play --engine stockfish --level 5 --color black --time 5+3
bin/chess tui --engine stockfish            # full-screen interface
bin/chess tui --analyze --engine stockfish  # engine evaluation and main line
bin/chess pgn game.pgn                      # show a game; add --replay to step through it
bin/chess perft 5 --divide                  # check move generation
bin/chess help                              # every command; `chess <command> --help` for its flags
```

| Command | What it does |
|---|---|
| `play` | Plays in the terminal. Moves in SAN (`Nf3`, `exd5`, `O-O`) or UCI (`e2e4`); `help` inside lists `undo`, `resign`, `draw`, `fen`, `pgn`, `flip` and `moves <square>`. `--pgn-out file` (or `-`) writes the PGN when it ends. |
| `tui` | Full-screen interface (bubbletea): arrows and enter or the mouse to move, a dialog for promotions, clocks, an engine opponent and an analysis mode. `q` quits. |
| `analyze` | Asks a UCI engine for the best move and the main line of `--fen`. |
| `perft` | Counts the nodes of the legal-move tree. |
| `fen` | Validates a FEN and shows the position. |
| `pgn` | Reads a PGN and shows the game; `--replay` steps through it with ←/→. |
| `match` | Plays two UCI engines against each other in game pairs, with Elo error bars and an optional SPRT (`--sprt 0,10`); `--pgn` saves the games. |

The board uses colors and Unicode pieces when the terminal supports them. `--no-color` and `--ascii` (or `NO_COLOR`) turn them off, and without a UTF-8 locale or a terminal it falls back to ASCII. Engines are any UCI engine; `--level` sets Stockfish's `Skill Level`.

## Engine

`make build` also builds `bin/chess-engine`, a UCI engine for any chess GUI (Arena, Cute Chess, BanksiaGUI) or for the client itself:

```bash
bin/chess play --engine bin/chess-engine --time 5+3
bin/chess analyze --engine bin/chess-engine --fen "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4"
```

| UCI option | Default | What it does |
|---|---|---|
| `Hash` | 16 | transposition table size in MB |
| `Threads` | 1 | goroutines searching together (Lazy SMP) |
| `OwnBook`, `BookFile` | off, empty | play from a Polyglot `.bin` opening book |
| `NullMove`, `LMR`, `Futility` | on | the selective-search techniques, switchable to measure them |

`chess match` plays two UCI engines against each other in game pairs (each opening with both colors) and reports the Elo difference with 95% error bars; `--sprt 0,10` stops as soon as a sequential probability ratio test decides. `scripts/sprt.sh [base-ref]` builds the engine at an earlier commit and at the working tree and runs that test, with `chess match` or, with `RUNNER=cutechess`, cutechess-cli.

```bash
bin/chess match --engine1 bin/chess-engine --engine2 old-engine --tc 10+0.1 --sprt 0,10
scripts/sprt.sh main -- --tc 5+0.05
```

The measurements behind each technique are in `docs/labs/3.0-*.md`.

## Project layout

```
cmd/cli/        terminal client (entry point)
cmd/engine/     the UCI engine
pkg/chess/      public API: board, pieces, game, FEN, SAN, PGN, perft, SearchBoard
internal/       cli, tui, render (board, panel), engine (UCI client), pgnread,
                moveinput, plus common types, clock, UCI move parsing, debug logging;
                for the engine: eval, search, uciserver, book (Polyglot), match (SPRT)
docs/           API surface, testing guide, labs (one report per experiment)
scripts/        benchmark script
```

`pkg/chess` re-exports the few types it needs from `internal/` (`Color`, `Move`, `TimeConfig`, `Clock`), so callers never import `internal/`.

## Development

```bash
make check        # gofmt, go vet and go test -race (what CI runs)
make perft        # full perft suite (~600M nodes, ~15s)
make fuzz         # fuzz FEN, SAN and PGN (FUZZTIME=2m to run longer)
make bench        # benchmarks
go test ./internal/search -run '^$' -bench Search   # search nodes and nodes/s by pruning
make help         # every target
```

[docs/API.md](docs/API.md) lists the stable public API. See [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for testing, benchmarking and debug logging.

## License

[MIT](LICENSE)
