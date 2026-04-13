.PHONY: dev dev-frontend dev-backend build build-frontend build-dev-picoclaw test lint clean \
	build-android-arm64 build-android-bundle

# Go variables
GO?=CGO_ENABLED=0 go
WEB_GO?=$(GO)
GO_BUILD_TAGS?=goolm,stdjson
GOFLAGS?=-v -tags $(GO_BUILD_TAGS)

# Build variables
BUILD_DIR=build
OUTPUT?=$(BUILD_DIR)/picoclaw-launcher
OUTPUT_ANDROID_ARM64?=$(BUILD_DIR)/picoclaw-launcher-android-arm64
FRONTEND_DIR=frontend
BACKEND_DIR=backend
BACKEND_DIST=$(BACKEND_DIR)/dist
PICOCLAW_BINARY_NAME=picoclaw
PICOCLAW_BINARY?=$(abspath ../build/$(PICOCLAW_BINARY_NAME))
LAUNCHER_GUI_LDFLAG=

# Version
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell $(WEB_GO) version | awk '{print $$3}')
CONFIG_PKG=github.com/sipeed/picoclaw/pkg/config
LDFLAGS=-X $(CONFIG_PKG).Version=$(VERSION) -X $(CONFIG_PKG).GitCommit=$(GIT_COMMIT) -X $(CONFIG_PKG).BuildTime=$(BUILD_TIME) -X $(CONFIG_PKG).GoVersion=$(GO_VERSION) -s -w


# OS detection
UNAME_S:=$(shell uname -s)
UNAME_M:=$(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	PLATFORM=linux
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),aarch64)
		ARCH=arm64
	else ifeq ($(UNAME_M),armv81)
		ARCH=arm64
	else ifeq ($(UNAME_M),loongarch64)
		ARCH=loong64
	else ifeq ($(UNAME_M),riscv64)
		ARCH=riscv64
	else ifeq ($(UNAME_M),mipsel)
		ARCH=mipsle
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Darwin)
	PLATFORM=darwin
	WEB_GO=CGO_ENABLED=1 go
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Windows)
	PLATFORM=windows
	ARCH=$(UNAME_M)
	PICOCLAW_BINARY_NAME=picoclaw.exe
	LAUNCHER_GUI_LDFLAG=-H=windowsgui
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

LAUNCHER_LDFLAGS=$(strip $(LAUNCHER_GUI_LDFLAG) $(LDFLAGS))

# Run both frontend and backend dev servers
dev: build-dev-picoclaw
	@if [ ! -f "$(BACKEND_DIST)/index.html" ]; then \
		echo "Embedded frontend not found, building..."; \
		$(MAKE) build-frontend; \
	fi
	@echo "Starting backend and frontend dev servers..."
	@$(MAKE) dev-backend BACKEND_ARGS='-no-browser' & $(MAKE) dev-frontend

# Start frontend dev server (Vite, with proxy to backend)
dev-frontend:
	cd $(FRONTEND_DIR) && pnpm dev

# Start backend dev server
dev-backend:
	cd $(BACKEND_DIR) && PICOCLAW_BINARY="$(PICOCLAW_BINARY)" ${WEB_GO} run -ldflags "$(LAUNCHER_LDFLAGS)" . $(BACKEND_ARGS)

# Build frontend and embed into Go binary
build: build-frontend
	@mkdir -p "$$(dirname "$(OUTPUT)")"
	${WEB_GO} build $(GOFLAGS) -ldflags "$(LAUNCHER_LDFLAGS)" -o "$(OUTPUT)" ./$(BACKEND_DIR)/

# Build launcher for Android ARM64 (frontend must already be built)
build-android-arm64: build-frontend
	@mkdir -p $(BUILD_DIR)
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o "$(OUTPUT_ANDROID_ARM64)" ./$(BACKEND_DIR)/

# Build launcher for all Android architectures
build-android-bundle: build-frontend
	@mkdir -p $(BUILD_DIR)
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/picoclaw-launcher-android-arm64" ./$(BACKEND_DIR)/
	@echo "All Android launcher builds complete"

build-frontend:
	@if [ ! -d $(FRONTEND_DIR)/node_modules ] || \
		[ $(FRONTEND_DIR)/package.json -nt $(FRONTEND_DIR)/node_modules ] || \
		[ $(FRONTEND_DIR)/pnpm-lock.yaml -nt $(FRONTEND_DIR)/node_modules ]; then \
		echo "Installing frontend dependencies..."; \
		cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile; \
	fi
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm build:backend

build-dev-picoclaw:
	@echo "Building picoclaw for launcher development..."
	@mkdir -p "$$(dirname "$(PICOCLAW_BINARY)")"
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o "$(PICOCLAW_BINARY)" ../cmd/picoclaw

# Run all tests
test:
	cd $(BACKEND_DIR) && ${WEB_GO} test ./...
	cd $(FRONTEND_DIR) && pnpm lint

# Lint and format
lint:
	cd $(BACKEND_DIR) && ${WEB_GO} vet ./...
	cd $(FRONTEND_DIR) && pnpm check

# Clean build artifacts
clean:
	rm -rf $(FRONTEND_DIR)/dist $(BACKEND_DIST) $(BUILD_DIR)
	node $(FRONTEND_DIR)/scripts/ensure-backend-gitkeep.cjs
