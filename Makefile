SHELL := /bin/bash

BIN_DIR := bin

.PHONY: all build test lint tidy clean release-snapshot generate migrate

all: build

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/gaxx ./cmd/gaxx
	CGO_ENABLED=0 go build -o $(BIN_DIR)/gaxx-agent ./cmd/gaxx-agent

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf $(BIN_DIR) dist

release-snapshot:
	goreleaser release --snapshot --clean

generate:
	@echo "no generators yet"

migrate:
	@echo "no migrations yet"


