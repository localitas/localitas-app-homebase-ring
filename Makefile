.PHONY: build test install uninstall start stop restart status logs logs-err \
       lint strict-lint

APP_NAME := homebase-ring
PORT ?= 9223
BASE_PATH ?= /apps/ext/homebase-ring/
PLIST_NAME := com.localitas.app.homebase-ring
PLIST_FILE := $(HOME)/Library/LaunchAgents/$(PLIST_NAME).plist
LOG_DIR := $(HOME)/.localitas/logs/homebase-ring
BIN_PATH := $(shell pwd)/bin/homebase-ring-server
WORK_DIR := $(shell pwd)

# ── Build & Test ──────────────────────────────────────────────

build: lint

build-linux: lint
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath \
		-o homebase-ring-server-linux-amd64 ./cmd/homebase-ring-server
	@mkdir -p bin
	go build -o bin/homebase-ring-server ./cmd/homebase-ring-server

test: lint
	go test -v ./...

lint:
	@echo "Running gofmt..."
	@gofmt -w .
	@echo "Running go vet..."
	@go vet ./...

strict-lint: lint
	@echo "Running staticcheck..."
	@if ! command -v staticcheck > /dev/null 2>&1; then \
		echo "Installing staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@staticcheck ./...
	@echo "staticcheck passed"

# ── Native (launchd) ─────────────────────────────────────────

install: build
	@mkdir -p $(LOG_DIR)
	@sed 's|$${BIN_PATH}|$(BIN_PATH)|g; s|$${PORT}|$(PORT)|g; s|$${BASE_PATH}|$(BASE_PATH)|g; s|$${LOG_DIR}|$(LOG_DIR)|g; s|$${WORK_DIR}|$(WORK_DIR)|g' \
		plist.template > $(PLIST_FILE)
	@echo "Installed launchd service: $(PLIST_NAME)"

uninstall: stop
	@rm -f $(PLIST_FILE)
	@echo "Uninstalled launchd service: $(PLIST_NAME)"

start: install
	@launchctl load $(PLIST_FILE) 2>/dev/null || true
	@echo "Started $(PLIST_NAME) on port $(PORT)"

stop:
	@launchctl unload $(PLIST_FILE) 2>/dev/null || true
	@echo "Stopped $(PLIST_NAME)"

restart: stop start

status:
	@launchctl list | grep $(PLIST_NAME) || echo "$(PLIST_NAME) is not running"

logs:
	@tail -f $(LOG_DIR)/stdout.log

logs-err:
	@tail -f $(LOG_DIR)/stderr.log
