# syntax=docker/dockerfile:1.7
#
# Imagem de DESENVOLVIMENTO. O código NÃO é copiado: entra por bind mount.
# Contém Go, golangci-lint, govulncheck e o xmllint (validação contra os esquemas da SEFAZ, pacote xsd).
# O go.mod pede Go 1.23 como mínimo; a integração contínua confere 1.23, 1.24 e a estável.

ARG GO_VERSION=1.26.11
FROM golang:${GO_VERSION}-alpine

ARG UID=1000
ARG GID=1000
ARG GOLANGCI_LINT_VERSION=v2.14.0
ARG GOVULNCHECK_VERSION=v1.9.0

RUN apk add --no-cache bash git make libxml2-utils

# As ferramentas ficam fora de /go, que recebe o volume do cache de módulos.
ENV GOBIN=/usr/local/bin
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} \
 && go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
 && rm -rf /root/.cache /go/pkg/mod/*

# Usuário com o UID/GID do host: o que o container gera pertence a você.
RUN set -eux; \
    group="$(getent group "${GID}" | cut -d: -f1 || true)"; \
    if [ -z "$group" ]; then addgroup -g "${GID}" dev; group=dev; fi; \
    adduser -D -u "${UID}" -G "$group" -s /bin/bash dev; \
    mkdir -p /src /go/pkg/mod /home/dev/.cache/go-build /home/dev/.cache/golangci-lint; \
    chown -R "${UID}:${GID}" /src /go /home/dev

ENV GOPATH=/go \
    GOCACHE=/home/dev/.cache/go-build \
    GOLANGCI_LINT_CACHE=/home/dev/.cache/golangci-lint \
    GOFLAGS=-buildvcs=false \
    CGO_ENABLED=0

USER dev
WORKDIR /src
