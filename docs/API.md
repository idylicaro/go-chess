# Superfície pública de `pkg/chess`

Registro das decisões da revisão feita na v1.1 (#15). A API pública é o que o `go doc ./pkg/chess` lista; o que não está aqui abaixo como "estável" pode mudar sem aviso.

## Estável

| Grupo | Símbolos | Por quê |
|---|---|---|
| Partida | `Game`, `NewGame`, `NewTimedGame`, `Move`, `Undo`, `LegalMoves`, `IsOver`, `Result`, `DrawReason`, `GetCurrentTurn`, `GetState` | Núcleo do uso da biblioteca. |
| Encerramento | `Resign`, `OfferDraw`, `AcceptDraw`, `DeclineDraw`, `PendingDrawOffer` | Regras de fim de partida. |
| Notação | `ParseFEN`, `FEN`, `StartingFEN`, `SAN`, `ParseSAN`, `ParseMove`, `PGN`, `PGNHeaders` | Entrada e saída de partidas. |
| Verificação | `Perft`, `Divide` | Usados pelos testes de referência e por ferramentas externas. |
| Relógio | `StartTimer`, `PauseTimer`, `ResumeTimer`, `HasTimer`, `GetRemainingTime`, `TimeConfig`, `Clock`, `SystemClock`, `Blitz3Plus2`, `Rapid10Plus0`, `Classical90Plus30` | Partidas com tempo e relógio injetável. |
| Tipos | `Color`, `White`, `Black`, `PieceType`, `PromoteTo*`, `Result*`, `GameState`, `CastlingRights`, `ErrGameOver`, `ErrNoMoveToUndo`, `ErrInvalidNotation`, `ErrIllegalMove`, `ErrAmbiguousMove` | Tipos que aparecem nas assinaturas acima. |

## Removido do export na v1.1

| Símbolo | Decisão | Motivo |
|---|---|---|
| `NewPawn`, `NewKnight`, `NewBishop`, `NewRook`, `NewQueen`, `NewKing` e os tipos `Pawn`, `Knight`, `Bishop`, `Rook`, `Queen`, `King` | Agora `newX` e `xPiece`, internos ao pacote | Ninguém fora do pacote precisa montar peças; quem lê uma peça usa `IPiece`. |
| `NewBoard`, `NewInitialBoard` | Agora `newBoard`, `newInitialBoard` | Um tabuleiro avulso permite estados que `Game` não garante (reis faltando, direitos de roque incoerentes). Para partir de uma posição, use `ParseFEN`. |
| `CoordinateConverter` | Agora `coordinateConverter` | Detalhe de implementação da conversão `e4` ↔ índices. |

Isso quebra quem importava esses nomes, e o CHANGELOG registra a mudança. Nenhum código do repositório (CLI, testes) dependia deles fora do pacote.

## Dívida conhecida

Estes símbolos continuam exportados porque o CLI os usa via `Game.GetBoard()`, e removê-los pede um desenho novo de leitura do tabuleiro:

- `IBoard`, `IPiece`, `Board`, `Piece`, `Position`: `GetBoard()` devolve o tabuleiro vivo, com métodos que o alteram (`SetPiece`, `MovePiece`, `RemovePiece`...). Alterar o tabuleiro por fora de `Game.Move` corrompe o estado da partida.
- `GetDebugInfo`: existe para a interface do pacote `internal/debug`.

Plano: substituir `GetBoard()` por uma visão somente leitura (por exemplo `PieceAt(square)`) e então mover o tabuleiro para `internal`. Como isso é uma quebra de API, entra na v2 (ver `docs/ROADMAP.md`), e não numa v1.x.
