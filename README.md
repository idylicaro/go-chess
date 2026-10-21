# Go Chess
A chess engine written in Go. Implements board representation, piece logic, and game state management. Communicates with Stockfish via the UCI protocol. Includes both a CLI interface and a backend server for multiplayer support.

## Directory Overview

```bash
├── cmd/
│   ├── server/
│   │    └── main.go
│   └── cli/
│       └── main.go
├── pkg/
│   └── chess/
│       ├── board.go
│       ├── piece.go
│       ├── game.go
│       └── outcome.go
├── internal/
│   ├── uci/
│   │   └── protocol.go
│   ├── common/
│   │   └── piece.go
│   ├── debug/
│   │   ├── debug.go
│   │   └── chess.go
│   └── timer/
│       └── timer.go
```

- **cmd/**  
   Contains the executables for the project. Each subdirectory is a separate program that can be built and run.

  - **cmd/server/**  
     Backend server responsible for managing matches, multiplayer features, and communication with the AI. The entry point is `main.go`.

  - **cmd/cli/**  
     Command-line interface for interacting with the chess engine.

- **pkg/chess/**  
   The public, importable API (`idylicaro/go-chess/pkg/chess`). It re-exports the types it needs from `internal/` (`Color`, `Move`, `TimeConfig`, ...), so callers never import `internal/`.

  - `board.go`: Board representation.
  - `piece.go`: Piece definitions and properties.
  - `game.go`: Game state management (turns, check, etc.).
  - `outcome.go`: `Result()`, `DrawReason()`, `LegalMoves()`, resignation and draw offers.
  - `undo.go`: `Undo()`.

- **internal/**  
   Supporting code, not meant to be imported by other projects.

  - **internal/common/**
    Handles common utilities and types used across the project.

  - **internal/debug/**  
    Provides debugging utilities and logging for the chess engine.

  - **internal/timer/**  
    Implements timing and time control features for the chess engine.

  - **internal/uci/**  
    Handles communication with Stockfish using the UCI protocol.

    - `protocol.go`: Functions for sending commands and parsing responses.

  - **scripts/**  
    Contains utility scripts for development and testing.

## UCI Command Processing

The UCI format for a move is a string like "e2e4".

```
       a b c d e f g h
    8 [ ][ ][ ][ ][ ][ ][ ][ ]
    7 [ ][ ][ ][ ][ ][ ][ ][ ]
    6 [ ][ ][ ][ ][ ][ ][ ][ ]
    5 [ ][ ][ ][ ][ ][ ][ ][ ]
    4 [ ][ ][ ][ ][ ][ ][ ][ ]
    3 [ ][ ][ ][ ][ ][ ][ ][ ]
    2 [ ][ ][ ][ ][ ][ ][ ][ ]
    1 [ ][ ][ ][ ][ ][ ][ ][ ]
```

---

## Internal board Overview

```
       0 1 2 3 4 5 6 7
    0 [ ][ ][ ][ ][ ][ ][ ][ ]
    1 [ ][ ][ ][ ][ ][ ][ ][ ]
    2 [ ][ ][ ][ ][ ][ ][ ][ ]
    3 [ ][ ][ ][ ][ ][ ][ ][ ]
    4 [ ][ ][ ][ ][ ][ ][ ][ ]
    5 [ ][ ][ ][ ][ ][ ][ ][ ]
    6 [ ][ ][ ][ ][ ][ ][ ][ ]
    7 [ ][ ][ ][ ][ ][ ][ ][ ]
```

---

## TODO

- [ ] Parser para notação algébrica padrão, Geração de notação algébrica para output, Ambiguidade resolution (Nbd2 vs Nfd2).
- [ ] CLI with charmbracelet/bubbletea.

## How to build

To run script to run some scenarios, you need:

- to build `go build -o bin/chess cmd/cli/*.go`
- to run `./bin/chess`
