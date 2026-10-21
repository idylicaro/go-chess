BIN_DIR     := bin
CLI_BIN     := $(BIN_DIR)/chess
COVER_FILE  := coverage.out
PKGS        := ./...
PERFT_TEST  := go test -tags perft -run 'Perft' -v ./pkg/chess

.DEFAULT_GOAL := help

.PHONY: ci fuzz help build run test test-race test-short perft perft-nocache cover cover-html bench bench-report \
        fmt fmt-check vet lint tidy check clean

help: ## Lista os comandos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

## --- Build / execução -------------------------------------------------------

build: ## Compila a CLI em bin/chess
	go build -o $(CLI_BIN) ./cmd/cli

run: build ## Compila e roda a CLI
	./$(CLI_BIN)

## --- Testes ------------------------------------------------------------------

test: ## Roda todos os testes
	go test $(PKGS)

test-race: ## Roda os testes com o detector de race
	go test -race $(PKGS)

test-short: ## Roda os testes pulando os marcados como longos (testing.Short)
	go test -short $(PKGS)

perft: ## Roda o perft completo nas posições de referência (~600M nós, ~15s; usa cache do go test)
	$(PERFT_TEST)

perft-nocache: ## Igual ao perft, mas ignora o cache e sempre executa de novo
	$(PERFT_TEST) -count=1

cover: ## Gera relatório de cobertura no terminal
	go test -coverprofile=$(COVER_FILE) $(PKGS)
	go tool cover -func=$(COVER_FILE) | tail -1

cover-html: cover ## Abre a cobertura em HTML
	go tool cover -html=$(COVER_FILE)

bench: ## Roda os benchmarks do pacote chess
	go test -run=^$$ -bench=. -benchmem ./pkg/chess

bench-report: ## Gera relatório HTML de benchmarks em .reports/
	./scripts/benchmark.sh

## --- Qualidade ---------------------------------------------------------------

fmt: ## Formata o código com gofmt
	gofmt -w .

fmt-check: ## Falha se algum arquivo não estiver formatado
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "Arquivos sem gofmt:"; echo "$$out"; exit 1; fi

vet: ## Roda go vet
	go vet $(PKGS)

lint: ## Roda golangci-lint (precisa estar instalado)
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint não encontrado. Instale: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run

tidy: ## Sincroniza go.mod/go.sum
	go mod tidy

check: fmt-check vet test-race ## Tudo que o CI deve rodar: formatação, vet e testes com race

## --- Limpeza -----------------------------------------------------------------

clean: ## Remove binários, cobertura e relatórios
	rm -rf $(BIN_DIR) $(COVER_FILE) .reports

fuzz: ## Roda cada alvo de fuzz por FUZZTIME (padrão 30s)
	@for t in FuzzParseFEN FuzzSAN FuzzPGN; do \
		go test ./pkg/chess -run '^$$' -fuzz "^$$t\$$" -fuzztime $(or $(FUZZTIME),30s) || exit 1; done
	go test ./internal/pgnread -run '^$$' -fuzz '^FuzzParse$$' -fuzztime $(or $(FUZZTIME),30s)

ci: fmt-check vet test-race lint ## Os jobs do CI, localmente: formatação, vet, race, lint, perft profundo e fuzz
	$(PERFT_TEST) -count=1
	$(MAKE) fuzz
