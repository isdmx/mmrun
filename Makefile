BINARY      := mmrun
PKG         := github.com/isdmx/mmrun
VERSION_PKG := $(PKG)/internal/version

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).Date=$(DATE)

GO      ?= go
GOBIN   ?= $(shell $(GO) env GOPATH)/bin

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

## tidy: sync go.mod/go.sum
.PHONY: tidy
tidy:
	$(GO) mod tidy

## tidy-check: fail if go.mod/go.sum are not tidy
.PHONY: tidy-check
tidy-check:
	$(GO) mod tidy
	@git diff --exit-code -- go.mod go.sum || (echo "go.mod/go.sum not tidy; run 'make tidy'" && exit 1)

## fmt: format code (gofumpt + goimports via golangci-lint)
.PHONY: fmt
fmt:
	golangci-lint fmt ./...

## fmt-check: fail if code is not formatted
.PHONY: fmt-check
fmt-check:
	golangci-lint fmt --diff ./...

## vet: run go vet
.PHONY: vet
vet:
	$(GO) vet ./...

## lint: run golangci-lint
.PHONY: lint
lint:
	golangci-lint run ./...

## test: run tests
.PHONY: test
test:
	$(GO) test ./...

## test-race: run tests with the race detector and coverage
.PHONY: test-race
test-race:
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...

## cover: show coverage summary (run test-race first)
.PHONY: cover
cover: test-race
	$(GO) tool cover -func=coverage.out | tail -1

## build: build the binary with version metadata into ./bin
.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

## install: install the binary into GOBIN with version metadata
.PHONY: install
install:
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags '$(LDFLAGS)' .

## run: build and run (use ARGS="..." to pass arguments)
.PHONY: run
run: build
	./bin/$(BINARY) $(ARGS)

## snapshot: build cross-platform snapshot binaries with goreleaser
.PHONY: snapshot
snapshot:
	goreleaser build --snapshot --clean

## ci: run the full set of checks (what CI runs)
.PHONY: ci
ci: tidy-check fmt-check vet lint test-race

## tools: install developer tooling
.PHONY: tools
tools:
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/goreleaser/goreleaser/v2@latest

## hooks: install git hooks via prek (pre-commit compatible)
.PHONY: hooks
hooks:
	prek install --install-hooks

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf bin/ dist/ coverage.out
