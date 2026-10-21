# Changelog

## v1.0.0

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
