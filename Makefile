# Makefile for tmux-yule-log
#
# A tmux screensaver plugin written in Go.

.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/sh
.SUFFIXES:
.DEFAULT_GOAL := all

# ---- Configuration

BINARY  := yule-log
BINDIR  := bin
GO      ?= go
GOFLAGS ?=

# ---- Targets

.PHONY: all build run test clean fmt vet lint tidy install help

all: build

build: ## Build binary to bin/yule-log
	$(GO) build $(GOFLAGS) -o $(BINDIR)/$(BINARY) .

run: ## Run directly with go run (accepts ARGS, e.g., make run ARGS="--help")
	$(GO) run $(GOFLAGS) . $(ARGS)

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

fmt: ## Format all Go source files
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint (if available)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run

tidy: ## Run go mod tidy
	$(GO) mod tidy

install: ## Install binary to GOPATH/bin
	$(GO) install $(GOFLAGS) .

clean: ## Remove build artifacts
	rm -rf $(BINDIR)

help: ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
