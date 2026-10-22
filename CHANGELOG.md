# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

CLI and TUI (milestone v1.2), a much faster move generator (milestone v2 Performance) and the project's own engine (milestone v3). The public API only gains additions.

### Added

- `chess` subcommands: `play`, `tui`, `analyze`, `perft`, `fen` and `pgn`, each with `--help`.
- Moves in SAN or UCI with distinct errors for a malformed, illegal or ambiguous move.
- `play` flags `--fen`, `--time 5+3`, `--engine`, `--level`, `--color`, `--no-color`, `--ascii` and `--pgn-out`, and the in-game commands `undo`, `resign`, `draw`, `accept`, `decline`, `fen`, `pgn`, `flip`, `moves [square]` and `help`.
- A colored Unicode board with an ASCII fallback, highlights for the last move, check and legal destinations, captured pieces and material, and a side panel with the moves in SAN, the clocks and a status bar.
- `chess tui`: bubbletea interface with keyboard and mouse moves, a promotion dialog, clocks, a UCI engine opponent that does not block the interface, and an analysis mode with an evaluation bar and the main line.
- `chess pgn --replay`: step through a game with the arrow keys (lab 1.2-A, `docs/labs/1.2-A-pgn-replay.md`).
- `pkg/chess`: `ErrInvalidNotation`, `ErrIllegalMove` and `ErrAmbiguousMove`, wrapped by `ParseMove`, `ParseSAN` and `Move`; `ParseFENTimed`; `Game.CheckTimeout`; `PGNHeaders.TimeControl`.
- `make ci` runs the CI jobs locally.
- `Game.PerftParallel(depth, workers)`: perft with the root moves shared among goroutines, and `chess perft --workers N` (default: every CPU). About 6.5x faster on 8 cores (lab 2.0-A, `docs/labs/2.0-A-parallel-perft.md`).
- `BenchmarkPerft` and `BenchmarkPerftParallel`, the reference nodes/s measures.
- `bin/chess-engine` (`cmd/engine`), a UCI engine: PeSTO evaluation with game phase (#36), negamax with alpha-beta and iterative deepening (#37), MVV-LVA, killer and history move ordering (#38), quiescence search (#39), a lockless transposition table (#40), time management for `movetime`, clocks and `stop` (#41) and a UCI server (#42).
- Null-move pruning (+86 Elo), late-move reductions (+100) and futility pruning (+35), each switchable as a UCI option and accepted by SPRT against the version before it (lab #44, `docs/labs/3.0-A-pruning.md`).
- `Threads`: Lazy SMP with a shared transposition table, +60 Elo for 2 threads and +119 for 4 (lab #45, `docs/labs/3.0-B-lazy-smp.md`).
- `OwnBook` and `BookFile`: Polyglot opening books, off by default; the gm2001 book measured no gain (lab #46, `docs/labs/3.0-C-polyglot-book.md`).
- `chess match`: engine-versus-engine matches in game pairs with Elo error bars and an SPRT, and `scripts/sprt.sh` to test the working tree against an earlier commit (#43).
- `pkg/chess`: `SearchBoard`, `SearchMove` and the piece-type constants, an experimental make/unmake API for engines (see `docs/API.md`).
- The UCI client gains `SearchGame`, a search after a list of moves, so an engine sees the game's repetitions.

### Changed

- Move generation is about 128x faster (~110k to ~14M perft nodes/s on one core) and a perft search no longer allocates: the board is a `[64]byte` array, moves are generated from precomputed tables and tested with make/unmake instead of a board copy, and mate/stalemate detection stops at the first legal move. `IBoard`, `IPiece` and the rest of the API are unchanged.
- Threefold repetition uses an incremental Zobrist hash instead of a string key rebuilt after every move.
- The perft suite reaches the full reference depths (about 600M nodes, ~15s with `make perft`); the default `go test` covers every depth up to 1M nodes.
- A draw offer in the CLI is answered by the opponent of the side that made it; `accept` used to fail because it used the side to move.
- The module now depends on bubbletea and lipgloss, for the client only. `pkg/chess` still imports nothing outside the standard library, and `go.mod` stays on Go 1.23.

### Fixed

- `NewGame` did not count the starting position for threefold repetition, so returning to it twice did not draw; `ParseFEN(StartingFEN)` already did.

## [1.1.0] - 2026-10-20

Hardening release: no new rules, a smaller and documented public API.

### Added

- Fuzz targets for FEN, SAN and PGN (`FuzzParseFEN`, `FuzzSAN`, `FuzzPGN`), run with `make fuzz`.
- CI jobs for the deep perft suite and a short fuzz run.
- godoc on every exported symbol of `pkg/chess`, a package overview and runnable `Example*` functions.
- `docs/API.md`, which lists the stable surface and the known debt.

### Changed

- **Breaking:** the piece constructors (`NewPawn`, `NewKnight`, `NewBishop`, `NewRook`, `NewQueen`, `NewKing`) and piece types, `NewBoard`, `NewInitialBoard` and `CoordinateConverter` are no longer exported. Use `NewGame` or `ParseFEN` to get a position.

## [1.0.0]

First stable release. The public API lives in `github.com/idylicaro/go-chess/pkg/chess`.

### Added

- Public API in `pkg/chess`: `LegalMoves()`, `Result()`, `DrawReason()`, `IsOver()`, `Undo()`, `Resign()` and draw offers (`OfferDraw`, `AcceptDraw`, `DeclineDraw`, `PendingDrawOffer`).
- Re-exported types (`Color`, `Move`, `TimeConfig`, `Clock`, `SystemClock`), `ParseMove` and the `PromoteTo*` constants, so callers never import `internal/`.
- `ErrGameOver` and `ErrNoMoveToUndo` sentinel errors.
- A timeout against insufficient material is a draw, not a loss.
- FEN parsing and generation, SAN parsing and formatting, PGN export.
- Perft and divide, checked against the six reference positions.
- An injectable clock for timed games.
- CLI commands `moves`, `undo`, `resign`, `draw`, `accept`, `decline`.
- GitHub Actions CI (gofmt, `go vet`, `go test -race`, golangci-lint) and an MIT license.

### Fixed

- Insufficient-material check panicked and mis-read bishop square colors.
- Attack detection and move simulation ignored castling, en passant and promotion.
- The fifty-move rule fired after 25 moves per side instead of 50.
- Timeouts were never enforced, and the timer could deadlock from its callback.
- Checkmate on the move that reached a draw condition was reported as a draw.

### Changed

- **Breaking:** the engine moved from `internal/chess` to `pkg/chess`, and the module path is now `github.com/idylicaro/go-chess`.
- `internal/common` is now `package common`, so imports no longer need an alias.
- Error messages include the offending value and the expected format.
- `Game.Move` rejects moves once the game is over.

[1.1.0]: https://github.com/idylicaro/go-chess/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/idylicaro/go-chess/releases/tag/v1.0.0
