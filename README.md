# Go Chess  

## Directory Overview

```bash
├── cmd/
│   ├── server/
│   │    └── main.go
│   └── cli/
│       └── main.go
├── internal/
│   ├── chess/
│   │   ├── board.go
│   │   ├── piece.go
│   │   ├── game.go
│   │   └── rules.go
│   └── uci/
│       └── protocol.go
├── pkg/
│   └── game_engine/
│       └── engine.go
└── go.mod
```

- **cmd/**  
    Contains the executables for the project. Each subdirectory is a separate program that can be built and run.

    - **cmd/server/**  
        Backend server responsible for managing matches, multiplayer features, and communication with the AI. The entry point is `main.go`.

    - **cmd/cli/**  
        Command-line interface for interacting with the chess engine.

- **internal/**  
    Core logic of the project, intended for internal use only and not meant to be imported by other projects.

    - **internal/chess/**  
        Implements the chess engine logic:
        - `board.go`: Board representation.
        - `piece.go`: Piece definitions and properties.
        - `game.go`: Game state management (turns, check, etc.).
        - `rules.go`: Movement rules, validation, castling, and more.

    - **internal/uci/**  
        Handles communication with Stockfish using the UCI protocol.
        - `protocol.go`: Functions for sending commands and parsing responses.

- **pkg/**  
    Contains libraries that are safe to use in external projects. If you develop a generic chess engine library for reuse, place it here.

    - **pkg/game_engine/**  
        Encapsulates the chess engine logic and provides a public interface. Useful for sharing the engine across multiple projects.

- **go.mod**  
    Go module definition file.



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
- [ ] Problema: Roque não verifica se o caminho está livre ou se o rei passa por xeque.
- [ ] Parser para notação algébrica padrão, Geração de notação algébrica para output, Ambiguidade resolution (Nbd2 vs Nfd2).
- [ ] CLI with charmbracelet/bubbletea.



## Game Tests
To run script to run some scenarios, you need:
- execute `go build -o bin/chess cmd/cli/*.go`
- run `go run run_tests.go `