SHELL := /bin/bash

BIN_DIR := bin

.PHONY: all build test test-unit test-integration test-e2e lint tidy clean install install-user uninstall release-snapshot generate migrate

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

install: build
	@echo "Installing gaxx to /usr/local/bin..."
	sudo cp $(BIN_DIR)/gaxx /usr/local/bin/gaxx
	sudo cp $(BIN_DIR)/gaxx-agent /usr/local/bin/gaxx-agent
	sudo chmod +x /usr/local/bin/gaxx
	sudo chmod +x /usr/local/bin/gaxx-agent
	@echo "Installation complete! You can now use 'gaxx' from anywhere."

install-user: build
	@echo "Installing gaxx to ~/bin..."
	mkdir -p ~/bin
	cp $(BIN_DIR)/gaxx ~/bin/gaxx
	cp $(BIN_DIR)/gaxx-agent ~/bin/gaxx-agent
	chmod +x ~/bin/gaxx
	chmod +x ~/bin/gaxx-agent
	@echo "Installation complete! Make sure ~/bin is in your PATH."
	@echo "Add this to your ~/.zshrc: export PATH=\"\$$HOME/bin:\$$PATH\""

uninstall:
	@echo "Removing gaxx from system..."
	sudo rm -f /usr/local/bin/gaxx
	sudo rm -f /usr/local/bin/gaxx-agent
	rm -f ~/bin/gaxx
	rm -f ~/bin/gaxx-agent
	@echo "Uninstall complete."

release-snapshot:
	goreleaser release --snapshot --clean

generate:
	@echo "no generators yet"

migrate:
	@echo "no migrations yet"


