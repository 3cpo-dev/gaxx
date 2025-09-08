SHELL := /bin/bash

BIN_DIR := bin

.PHONY: all build test test-unit test-integration test-e2e lint tidy clean release-snapshot generate migrate

all: build

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/gaxx ./cmd/gaxx
	CGO_ENABLED=0 go build -o $(BIN_DIR)/gaxx-agent ./cmd/gaxx-agent

test: test-unit

test-unit:
	go test ./internal/... ./pkg/...

test-integration: build
	go test -v -run TestFullWorkflow ./integration_test.go

test-e2e: build
	./test/e2e_test.sh

test-monitoring: build
	./test_monitoring.sh

test-monitoring-full: build
	./final_monitoring_demo.sh

test-all: test-unit test-integration test-e2e test-monitoring

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


