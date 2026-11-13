# A interface única do projeto. Tudo roda no container de desenvolvimento: `make help` lista.
.DEFAULT_GOAL := help
.PHONY: help image shell go tidy fmt fmt-check vet lint vuln test test-short cover mutate ci

GREMLINS_VERSION := v0.5.1
IMAGE := gonfe-dev

# O código entra por bind mount; os módulos e o cache de compilação ficam em volumes.
RUN = docker run --rm -i -v "$(CURDIR)":/src -v gonfe-dev-mod:/go/pkg/mod -v gonfe-dev-cache:/home/dev/.cache $(IMAGE)

help: ## Lista os comandos
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ── Ambiente ──

image: ## Constrói a imagem de desenvolvimento (Go, golangci-lint, govulncheck, xmllint)
	docker build -t $(IMAGE) --build-arg UID=$$(id -u) --build-arg GID=$$(id -g) docker/dev

shell: ## Abre um shell no container
	docker run --rm -it -v "$(CURDIR)":/src -v gonfe-dev-mod:/go/pkg/mod -v gonfe-dev-cache:/home/dev/.cache $(IMAGE) bash

go: ## Roda o go no container: make go args="test ./cte/"
	$(RUN) go $(args)

## ── Código ──

tidy: ## go mod tidy
	$(RUN) go mod tidy

fmt: ## Formata o código
	$(RUN) gofmt -w .

fmt-check: ## Falha se houver arquivo sem formatar
	$(RUN) sh -c 'out=$$(gofmt -l .); test -z "$$out" || { echo "$$out"; exit 1; }'

vet: ## go vet
	$(RUN) go vet ./...

lint: ## golangci-lint
	$(RUN) golangci-lint run ./...

vuln: ## govulncheck
	$(RUN) govulncheck ./...

test: ## Todos os testes, com a validação contra os esquemas oficiais (xmllint)
	$(RUN) go test -count=1 ./...

test-short: ## Os testes sem os lentos
	$(RUN) go test -short ./...

cover: ## Cobertura por pacote; detalhe em tmp/cover.html
	mkdir -p tmp
	$(RUN) go test -count=1 -coverprofile=tmp/cover.out ./... >/dev/null
	$(RUN) go tool cover -html=tmp/cover.out -o tmp/cover.html
	$(RUN) go tool cover -func=tmp/cover.out | tail -1

mutate: ## Mutação por demanda, fora do gate: make mutate pkg=./validacao
	@test -n "$(pkg)" || (echo "Informe o pacote: make mutate pkg=./validacao" && exit 2)
	$(RUN) sh -c 'GOBIN=/tmp/mutation-tools go install github.com/go-gremlins/gremlins/cmd/gremlins@$(GREMLINS_VERSION) && /tmp/mutation-tools/gremlins unleash --workers $(or $(workers),4) --timeout-coefficient 10 -S lt $(pkg)'

ci: fmt-check vet lint test ## O gate: o mesmo que `make finish` exige

include scripts/git-flow.mk
