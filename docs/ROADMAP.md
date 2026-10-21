# Roadmap

Estado: **v1 concluída** (núcleo correto e testado: lances legais, FEN, perft, SAN/PGN, undo, relógio injetável, cliente UCI, API pública em `pkg/chess`).

Convenções:
- Os números de issue são os reais do GitHub (milestones e issues criadas). As #11 e #12 já existiam e cobrem partes da v1.1 (CI, README, LICENSE) e da v2 (performance); por isso as linhas sobrepostas apontam para elas.
- Branches seguem o padrão `feat/N-slug`, `fix/N-slug`, `lab/N-slug`.
- Um **lab** é um experimento curto com critério de sucesso medido. "Não vale a pena" é um resultado válido e deve ser registrado em `docs/labs/`.

## Visão geral

| Milestone | Tema | Depende de |
|---|---|---|
| v1.1 | Hardening | v1 |
| v1.2 | CLI/TUI | v1 |
| v2 | Performance | v1.1 |
| v3 | Engine própria | v2 |
| v4 | Servidor multiplayer | v1.1 |
| v5 | Clientes web | v4 (WASM só precisa da v1.1) |
| v6 | Extras | v3, v4 |

As trilhas **engine** (v2 → v3), **produto** (v4 → v5) e **interface** (v1.2) são independentes entre si depois da v1.1.

---

## v1.1 — Hardening

Fechar a base antes de construir em cima dela.

**Pronto quando:** fuzz rodando em CI sem achados, API documentada, repositório limpo e tag `v1.1.0` publicada.

| # | Issue | Branch |
|---|---|---|
| #11 | Limpar a migração `internal/chess` → `pkg/chess`: README com o layout real, remover `coverage.out` do versionamento, `.gitignore` (já coberta pela #11) | — |
| #13 | Fuzz de FEN, SAN e PGN com `go test -fuzz` (round-trip e ausência de panic) | `feat/13-fuzz-parsers` |
| #14 | godoc em todos os símbolos exportados de `pkg/chess` e `Example*` executáveis | `feat/14-godoc-examples` |
| #15 | Revisar a superfície pública: o que precisa ser exportado, o que volta para `internal` | `feat/15-api-surface` |
| #11 | CI (GitHub Actions): `go vet`, `staticcheck`, testes com `-race`, perft rápido; perft profundo em job separado (já coberta pela #11) | — |
| #16 | Versionamento semântico e `CHANGELOG.md`; tag `v1.1.0` | `feat/16-release-process` |

**Lab 1.1-A — Teste diferencial contra Stockfish.** (#17) Gerar posições aleatórias por partidas aleatórias, comparar a lista de lances legais e o `perft` com o Stockfish via UCI. Sucesso: 100 mil posições sem divergência, ou divergências catalogadas como bugs.

---

## v1.2 — CLI/TUI

Melhorar o CLI e o display. Só consome `pkg/chess`, então pode andar em paralelo com qualquer outra trilha.

**Pronto quando:** dá para jogar uma partida completa só pelo teclado, retomar a partir de FEN ou PGN, rodar sem cores em CI, e o render tem testes de snapshot.

### Fase 1 — CLI

| # | Issue | Branch |
|---|---|---|
| #18 | Estruturar subcomandos: `play`, `analyze`, `perft`, `fen`, `pgn` | `feat/18-cli-commands` |
| #19 | Entrada de lances em SAN e UCI, com erros claros para lance ilegal ou ambíguo | `feat/19-cli-move-input` |
| #20 | Comandos de partida: `undo`, `resign`, `draw`, `fen`, `pgn`, `flip`, `help` | `feat/20-cli-game-commands` |
| #21 | Flags: `--fen`, `--time 5+3`, `--engine`, `--level`, `--color`, `--no-color`, `--ascii` | `feat/21-cli-flags` |
| #22 | Exportar PGN ao fim da partida (arquivo ou stdout) | `feat/22-cli-pgn-export` |

### Fase 2 — Display

| # | Issue | Branch |
|---|---|---|
| #23 | Tabuleiro com cores alternadas, peças Unicode e fallback ASCII, detecção de capacidade do terminal | `feat/23-board-render` |
| #24 | Destaques: último lance, rei em xeque, destinos legais da peça selecionada | `feat/24-board-highlights` |
| #25 | Tabuleiro virado, coordenadas, peças capturadas e diferença de material | `feat/25-board-orientation-material` |
| #26 | Lista de lances em SAN, relógios e barra de status ao lado do tabuleiro | `feat/26-side-panel` |

### Fase 3 — TUI com bubbletea

| # | Issue | Branch |
|---|---|---|
| #27 | Esqueleto Elm (`Model`/`Update`/`View`) com `bubbletea` e `lipgloss` | `feat/27-tui-skeleton` |
| #28 | Mover por seleção (teclado e mouse), diálogo de promoção | `feat/28-tui-move-selection` |
| #29 | Relógio com tick e fim de partida por tempo | `feat/29-tui-clock` |
| #30 | Jogar contra engine em `tea.Cmd` assíncrono (UI não trava) | `feat/30-tui-vs-engine` |
| #31 | Modo análise: barra de avaliação e linha principal a partir de `info score`/`pv` | `feat/31-tui-analysis` |
| #32 | Testes de snapshot do `View` (golden files) e `teatest` | `feat/32-tui-tests` |

**Lab 1.2-A — Replay de PGN.** (#33) Navegar partidas importadas lance a lance (←/→). Sucesso: abre um PGN de 100+ lances e navega sem recalcular a partida inteira a cada passo (usa o undo).

**Lab 1.2-B — Terminal no navegador.** (#34) Reaproveitar o `View` com xterm.js via WASM. Só vale se o front web (v5) for confirmado.

---

## v2 — Performance

Tornar a geração de lances rápida e preparar a base para busca.

**Pronto quando:** perft(6) da posição inicial roda dentro da meta definida no lab 2-A, com os testes de perft da v1 passando sem alteração.

| # | Issue | Branch |
|---|---|---|
| #12 | Baseline de benchmarks salvo em `docs/bench/` (perft nodes/s, alocações por lance) (já coberta pela #12) | — |
| #12 | Zobrist hashing incremental em `move_application` e no undo (já coberta pela #12) | — |
| #12 | Repetição de posição e regra das três repetições usando o hash (já coberta pela #12) | — |
| #12 | Geração pseudo-legal + filtro de legalidade (evita simular todo lance) (já coberta pela #12) | — |
| #12 | Redução de alocações: slices reaproveitados, lance como struct compacto (já coberta pela #12) | — |
| #12 | Bitboards e magic bitboards, caso o lab 2-A aprove (já coberta pela #12) | — |

**Lab 2-A — Bitboards vs. board 8x8.** (#12) Implementar atrás da mesma interface e comparar com o perft e o `benchmark_test.go`. Métrica: nodes/s em perft profundo. Decide se a etapa de bitboards da #12 entra.

**Lab 2-B — Perft paralelo.** (#35) Goroutines por lance da raiz. Métrica: speedup por número de núcleos.

---

## v3 — Engine própria

**Pronto quando:** a engine joga uma partida completa por UCI em uma GUI externa, vence o Stockfish em `skill level` baixo e tem Elo estimado por SPRT.

| # | Issue | Branch |
|---|---|---|
| #36 | Avaliação material + tabelas de posição (PST), fase de jogo | `feat/36-evaluation` |
| #37 | Busca negamax com alpha-beta e aprofundamento iterativo | `feat/37-search` |
| #38 | Ordenação de lances: MVV-LVA, killer moves, history heuristic | `feat/38-move-ordering` |
| #39 | Quiescence search | `feat/39-quiescence` |
| #40 | Tabela de transposição (usa o Zobrist da v2) | `feat/40-transposition-table` |
| #41 | Controle de tempo da busca (movetime, wtime/btime, stop) | `feat/41-time-management` |
| #42 | Servidor UCI: `cmd/engine` responde `uci`, `isready`, `position`, `go`, `stop` | `feat/42-uci-server` |
| #43 | Harness de torneio (Cute Chess) e SPRT para validar mudanças | `feat/43-tournament-harness` |

**Lab 3-A — Null move, LMR e futility pruning.** (#44) Cada um medido por SPRT contra a versão anterior.

**Lab 3-B — Lazy SMP.** (#45) Busca em várias goroutines com TT compartilhada. Métrica: ganho de profundidade e Elo por núcleo.

**Lab 3-C — Livro de aberturas (Polyglot).** (#46) Sucesso: ganho de Elo mensurável contra a engine sem livro.

**Lab 3-D — Syzygy.** (#47) Integração com biblioteca externa para finais. Muito experimental.

---

## v4 — Servidor multiplayer

**Pronto quando:** duas pessoas jogam uma partida completa por clientes diferentes, com relógio controlado pelo servidor, e a partida sobrevive a uma reconexão.

| # | Issue | Branch |
|---|---|---|
| #48 | Protocolo de mensagens (JSON sobre WebSocket) e versão do protocolo | `feat/48-protocol` |
| #49 | Servidor WebSocket em `cmd/server`, ciclo de vida de conexões | `feat/49-websocket-server` |
| #50 | Salas e criação de partida (convite por link/código, escolha de cor e controle de tempo) | `feat/50-rooms` |
| #51 | Validação de lances e relógio autoritativos no servidor (usa `timer` com clock injetado) | `feat/51-authoritative-game` |
| #52 | Reconexão e retomada de partida | `feat/52-reconnect` |
| #53 | Ofertas de empate, desistência, abandono por timeout | `feat/53-draw-resign-abandon` |
| #54 | Persistência de partidas em PGN (SQLite ou arquivos) | `feat/54-persistence` |
| #55 | Jogar contra a engine no servidor (Stockfish agora, engine própria depois) | `feat/55-vs-engine` |
| #56 | Rate limit, limites de mensagem e testes de carga | `feat/56-limits-load-tests` |

**Lab 4-A — Cliente de teste headless.** (#57) Dois bots jogam milhares de partidas contra o servidor para achar condições de corrida e vazamentos de goroutine.

---

## v5 — Clientes web

**Pronto quando:** dá para jogar pelo navegador contra outra pessoa ou contra a engine, a partir de uma URL.

| # | Issue | Branch |
|---|---|---|
| #58 | Compilar `pkg/chess` para WASM e expor uma API JS mínima | `feat/58-wasm-core` |
| #59 | Front-end do tabuleiro: arrastar peças, destinos legais, promoção, relógio | `feat/59-web-board` |
| #60 | Cliente do protocolo da v4 (lobby, partida, reconexão) | `feat/60-web-client` |
| #61 | Tela de análise e replay de PGN | `feat/61-web-analysis` |

**Lab 5-A — Validação local no navegador.** (#62) Usar o WASM para validar lances antes de enviar ao servidor. Sucesso: latência percebida menor, sem divergência com o servidor.

---

## v6 — Extras

Itens independentes entre si, escolhidos conforme o interesse.

| # | Issue | Branch |
|---|---|---|
| #63 | Chess960: FEN com X-FEN/Shredder, roque generalizado e perft próprio | `feat/63-chess960` |
| #64 | Rating Elo/Glicko e matchmaking | `feat/64-rating-matchmaking` |
| #65 | Análise de partida importada: avaliação por lance, classificação (erro, gafe) e gráfico | `feat/65-game-analysis` |
| #66 | Puzzles táticos a partir de partidas analisadas | `feat/66-puzzles` |
| #67 | Outras variantes (Three-check, King of the Hill) | `feat/67-variants` |

---

## Ordem sugerida

1. **v1.1 Hardening**, curto, para fechar a base.
2. **Lab 1.1-A** (diferencial) em paralelo, e **v1.2 Fase 1** (CLI), que deixa o projeto utilizável.
3. **v2**, começando pelo Zobrist (#33) e pelo Lab 2-A, que decide o rumo da performance.
4. Bifurcar: **v3** (engine) para quem quer aprender chess programming, **v4** (servidor) para quem quer algo jogável.
5. **v1.2 Fase 3** (TUI) e **v5** (web) entram quando a trilha escolhida tiver o que mostrar.
6. **v6** por último.
