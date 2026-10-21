# Testing Guide

This guide covers testing, benchmarking, and debugging for the chess engine.

## Testing

### Running Tests

```bash
# Run all integration tests
go test -v ./pkg/chess -run "Integration"

# Run specific test
go test -v ./pkg/chess -run "TestGameWithTimerIntegration"

# Run with coverage
go test -v -cover ./pkg/chess
```

The test suite includes:
- **TestGameIntegration** - Complete game scenarios
- **TestGameWithTimerIntegration** - Timer functionality 
- **TestCastlingIntegration** - Castling validation
- **TestEnPassantIntegration** - En passant validation

### Terminal client

```bash
go test ./internal/...               # CLI, render, TUI, engine client, PGN reader
go test ./internal/tui -update       # rewrite the golden snapshots after a deliberate layout change
```

- `internal/render` and `internal/tui` compare their output with golden files in `testdata/`. Read the diff of `testdata/` before committing an update.
- `internal/tui/flow_test.go` runs the model in a real `tea.Program` with `teatest`. The tests that need a real engine run only when `stockfish` is in the `PATH` (and not with `-short`).
- The CLI tests use the test binary itself as a scripted UCI engine (`GO_CHESS_FAKE_UCI`), and `execute` takes the environment, so the output does not depend on the machine's locale or terminal.
- `make ci` runs the same jobs as `.github/workflows/ci.yml` on your machine: gofmt, vet, `-race` tests, golangci-lint, the deep perft and the fuzz targets.

### Perft (move generation correctness)

Perft counts the leaf nodes of the legal move tree and compares them with
the [reference results](https://www.chessprogramming.org/Perft_Results) for
six positions (initial, Kiwipete, positions 3–6).

```bash
# Depths up to 1M nodes run with the normal suite
go test ./pkg/chess -run Perft

# Every reference depth (~600M nodes, ~15s on 8 cores)
make perft            # same as: go test -tags perft -run Perft ./pkg/chess
```

When a count is wrong, split it by first move and diff against a reference
engine (e.g. Stockfish's `go perft N`):

```go
game, _ := chess.ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
for move, nodes := range game.Divide(3) {
    fmt.Println(move, nodes) // e.g. "e1g1 2059"
}
```

## Benchmarking

### Manual Benchmarks

```bash
# Basic benchmark
go test -bench=. ./pkg/chess

# With memory stats
go test -bench=. -benchmem ./pkg/chess

# Specific benchmark
go test -bench=BenchmarkMoveGeneration ./pkg/chess
```

### Automated Script

The `scripts/benchmark.sh` script provides comprehensive analysis:

```bash
# Basic run
./scripts/benchmark.sh

# With integration tests
./scripts/benchmark.sh --integration

# Multiple runs for accuracy
./scripts/benchmark.sh -r 5

# With profiling
./scripts/benchmark.sh --cpu-profile --mem-profile

# Compare with previous results
./scripts/benchmark.sh -c old_benchmark.txt
```

### Available Benchmarks

- **BenchmarkMoveGeneration** - Legal move generation
- **BenchmarkGameStateUpdate** - Game state updates
- **BenchmarkFullGame** - Complete game simulation
- **BenchmarkTimerOperations** - Timer operations
- **BenchmarkBoardCopy** - Board copying
- **BenchmarkLegalMoveValidation** - Move validation
- **BenchmarkEnPassantValidation** - En passant validation
- **BenchmarkCastlingValidation** - Castling validation

### Reading Results

```
BenchmarkMoveGeneration-8     100000    10050 ns/op    1024 B/op    15 allocs/op
```

- **100000**: iterations run
- **10050 ns/op**: nanoseconds per operation
- **1024 B/op**: bytes allocated per operation  
- **15 allocs/op**: allocations per operation

Target performance:
- Move Generation: under 1ms
- Game State Update: under 100μs
- Board Copy: under 50μs

## Debugging

### Environment Variables

```bash
# Basic debug info
export CHESS_DEBUG_LEVEL=basic
export CHESS_DEBUG_FILE=debug.log  # Optional: save to file

# Detailed debugging (includes timings)
export CHESS_DEBUG_LEVEL=detailed

# Verbose debugging (everything)
export CHESS_DEBUG_LEVEL=verbose

# Run the game
go run cmd/cli/*.go
```

### Programmatic Setup

```go
package main

import "github.com/idylicaro/go-chess/pkg/chess"

func main() {
    // Initialize debug
    err := chess.InitDebug(chess.DebugDetailed, true, "chess_debug.log")
    if err != nil {
        log.Fatal(err)
    }
    
    // Your code here...
}
```

### Debug Levels

- **DebugNone** - No debugging
- **DebugBasic** - Moves and basic errors
- **DebugDetailed** - Game state, timings, board
- **DebugVerbose** - Everything including legal moves

### Debug Output Example

```
[DEBUG] [INFO] Move: (4,6)->(4,4) by White, Game State: InProgress
[DEBUG] [DETAIL] Game.Move took 2.345ms
[DEBUG] [DETAIL] === Game State Debug ===
[DEBUG] [DETAIL] Current Turn: Black
[DEBUG] [DETAIL] Game State: InProgress
[DEBUG] [DETAIL] Half Move Clock: 0
[DEBUG] [DETAIL] Timer - White: 2m59s, Black: 3m0s, Current: Black
```

## Performance Analysis

### Regular Monitoring

Run benchmarks regularly to track performance:

```bash
# Daily benchmarks
./scripts/benchmark.sh > daily_benchmark_$(date +%Y%m%d).txt

# Compare with baseline
./scripts/benchmark.sh -c baseline_benchmark.txt
```

Watch these metrics:
- Move generation time
- Memory usage per operation
- Number of allocations

### Profiling

When you need deeper analysis:

```bash
# CPU profiling
./scripts/benchmark.sh --cpu-profile
go tool pprof .reports/cpu.prof

# Memory profiling  
./scripts/benchmark.sh --mem-profile
go tool pprof .reports/mem.prof
```

### Performance Indicators

**Critical Issues:**
- Move generation over 10ms
- Memory leaks
- Timer drift over 100ms

**Watch Out:**
- Move generation over 1ms
- Board copy over 100μs
- Too many small allocations

**Good Performance:**
- Move generation under 1ms
- Stable allocations
- Timer precision under 10ms

## Development Workflow

### Before Making Changes

```bash
# 1. Baseline benchmark
./scripts/benchmark.sh -o baseline/

# 2. Make your changes

# 3. Run tests
go test ./pkg/chess

# 4. Run benchmarks
./scripts/benchmark.sh --integration -c baseline/benchmark_results.txt

# 5. Check HTML report
open performance_report.html
```

### CI/CD Integration
TODO: add CI workflow
```bash
# Add to pipeline
go test -v ./pkg/chess
./scripts/benchmark.sh --integration -o artifacts/
```

