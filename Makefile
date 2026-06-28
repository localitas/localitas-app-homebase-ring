.PHONY: build test install uninstall start stop restart status logs logs-err \
       build-docker start-docker stop-docker restart-docker logs-docker \
       docker-push lint strict-lint

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

# ── Docker ────────────────────────────────────────────────────

build-docker: build-linux
	docker build -t homebase-ring:latest .

start-docker: build-docker stop-docker
	@docker run -d -p $(PORT):8000 --name homebase-ring \
		--log-opt max-size=10m --log-opt max-file=7 \
		homebase-ring:latest
	@echo "homebase-ring running in Docker on port $(PORT)"

stop-docker:
	@docker rm -f homebase-ring 2>/dev/null || true

restart-docker: stop-docker start-docker

logs-docker:
	@docker logs -f homebase-ring

# ── Release & Registry ────────────────────────────────────────

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GHCR_IMAGE := ghcr.io/localitas/localitas-app-$(APP_NAME)

docker-push: test build-docker
	docker tag $(APP_NAME):latest $(GHCR_IMAGE):latest
	docker tag $(APP_NAME):latest $(GHCR_IMAGE):$(VERSION)
	docker push $(GHCR_IMAGE):latest
	docker push $(GHCR_IMAGE):$(VERSION)
	@echo "✅ Pushed $(GHCR_IMAGE):latest and $(GHCR_IMAGE):$(VERSION)"

build-release: lint
	@mkdir -p dist
	@echo "Building $(APP_NAME) $(VERSION) ($(COMMIT))..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -trimpath \
		-o dist/homebase-ring-server-darwin-arm64 ./cmd/homebase-ring-server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -trimpath \
		-o dist/homebase-ring-server-darwin-amd64 ./cmd/homebase-ring-server
	@echo "Built: dist/homebase-ring-server-darwin-arm64, dist/homebase-ring-server-darwin-amd64"

release: build-release
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then echo "Set VERSION=vX.Y.Z"; exit 1; fi
	@echo "Creating release $(VERSION) on GitHub..."
	gh release create $(VERSION) \
		dist/homebase-ring-server-darwin-arm64 \
		dist/homebase-ring-server-darwin-amd64 \
		--title "$(VERSION)" --generate-notes
	@echo "✅ Released $(VERSION)"
