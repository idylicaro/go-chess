# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

CLI and TUI release (milestone v1.2). The library only gains a few additions.

### Added

- `chess` subcommands: `play`, `tui`, `analyze`, `perft`, `fen` and `pgn`, each with `--help`.
- Moves in SAN or UCI with distinct errors for a malformed, illegal or ambiguous move.
- `play` flags `--fen`, `--time 5+3`, `--engine`, `--level`, `--color`, `--no-color`, `--ascii` and `--pgn-out`, and the in-game commands `undo`, `resign`, `draw`, `accept`, `decline`, `fen`, `pgn`, `flip`, `moves [square]` and `help`.
- A colored Unicode board with an ASCII fallback, highlights for the last move, check and legal destinations, captured pieces and material, and a side panel with the moves in SAN, the clocks and a status bar.
- `chess tui`: bubbletea interface with keyboard and mouse moves, a promotion dialog, clocks, a UCI engine opponent that does not block the interface, and an analysis mode with an evaluation bar and the main line.
- `chess pgn --replay`: step through a game with the arrow keys (lab 1.2-A, `docs/labs/1.2-A-pgn-replay.md`).
- `pkg/chess`: `ErrInvalidNotation`, `ErrIllegalMove` and `ErrAmbiguousMove`, wrapped by `ParseMove`, `ParseSAN` and `Move`; `ParseFENTimed`; `Game.CheckTimeout`; `PGNHeaders.TimeControl`.
- `make ci` runs the CI jobs locally.

### Changed

- A draw offer in the CLI is answered by the opponent of the side that made it; `accept` used to fail because it used the side to move.
- The module now depends on bubbletea and lipgloss, for the client only. `pkg/chess` still imports nothing outside the standard library, and `go.mod` stays on Go 1.23.

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
