.PHONY: dev dev-frontend dev-backend build test lint clean

# Go variables
GO?=CGO_ENABLED=0 go
WEB_GO?=$(GO)
GOFLAGS?=-v -tags stdjson

# Build variables
BUILD_DIR=build

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
	LDFLAGS=-H=windowsgui $(LDFLAGS)
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

# Run both frontend and backend dev servers
dev:
	@if [ ! -f $(BUILD_DIR)/picoclaw-launcher ] || [ ! -d backend/dist ]; then \
		echo "Build artifacts not found, building..."; \
		$(MAKE) build; \
	fi
	@echo "Starting backend and frontend dev servers..."
	@$(MAKE) dev-backend & $(MAKE) dev-frontend

# Start frontend dev server (Vite, with proxy to backend)
dev-frontend:
	cd frontend && pnpm dev

# Start backend dev server
dev-backend:
	cd backend && ${WEB_GO} run -ldflags "$(LDFLAGS)" .

# Build frontend and embed into Go binary
build:
	cd frontend && pnpm build:backend
	${WEB_GO} build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/picoclaw-launcher ./backend/

# Run all tests
test:
	cd backend && ${WEB_GO} test ./...
	cd frontend && pnpm lint

# Lint and format
lint:
	cd backend && ${WEB_GO} vet ./...
	cd frontend && pnpm check

# Clean build artifacts
clean:
	rm -rf frontend/dist backend/dist $(BUILD_DIR)
	mkdir -p backend/dist && touch backend/dist/.gitkeep
