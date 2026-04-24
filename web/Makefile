.PHONY: dev dev-frontend dev-backend build build-frontend build-dev-picoclaw test lint clean \
	build-android-arm64 build-android-bundle

# Go variables
GO?=go
WEB_GO?=$(GO)
CGO_ENABLED?=0
GO_BUILD_TAGS?=goolm,stdjson
GOFLAGS?=-v -tags $(GO_BUILD_TAGS)
GOCACHE?=$(abspath ../.cache/go-build)
GOMODCACHE?=$(abspath ../.cache/go-mod)
GOTOOLCHAIN?=local
export CGO_ENABLED
export GOCACHE
export GOMODCACHE
export GOTOOLCHAIN

# Build variables
BUILD_DIR=build
EXT=
OUTPUT?=$(BUILD_DIR)/picoclaw-launcher$(EXT)
OUTPUT_ANDROID_ARM64?=$(BUILD_DIR)/picoclaw-launcher-android-arm64$(EXT)
FRONTEND_DIR=frontend
FRONTEND_INSTALL_STAMP=$(FRONTEND_DIR)/node_modules/.picoclaw-install-stamp
BACKEND_DIR=backend
BACKEND_DIST=$(BACKEND_DIR)/dist
PICOCLAW_BINARY_NAME=picoclaw
PICOCLAW_BINARY?=$(abspath ../build/$(PICOCLAW_BINARY_NAME))
LAUNCHER_GUI_LDFLAG=

ifeq ($(OS),Windows_NT)
	POWERSHELL=powershell -NoProfile -Command
	WINDOWS_GOARCH_RAW:=$(strip $(shell go env GOARCH 2>NUL))
endif

# Version
ifeq ($(OS),Windows_NT)
	VERSION_RAW:=$(strip $(shell git describe --tags --always --dirty 2>NUL))
	GIT_COMMIT_RAW:=$(strip $(shell git rev-parse --short=8 HEAD 2>NUL))
	BUILD_TIME_RAW:=$(strip $(shell powershell -NoProfile -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK'"))
	GO_VERSION_RAW:=$(strip $(shell go env GOVERSION 2>NUL))
else
	VERSION_RAW:=$(strip $(shell git describe --tags --always --dirty 2>/dev/null))
	GIT_COMMIT_RAW:=$(strip $(shell git rev-parse --short=8 HEAD 2>/dev/null))
	BUILD_TIME_RAW:=$(strip $(shell date +%FT%T%z))
	GO_VERSION_RAW:=$(strip $(shell go env GOVERSION 2>/dev/null))
endif
VERSION?=$(if $(VERSION_RAW),$(VERSION_RAW),dev)
GIT_COMMIT=$(if $(GIT_COMMIT_RAW),$(GIT_COMMIT_RAW),dev)
BUILD_TIME=$(if $(BUILD_TIME_RAW),$(BUILD_TIME_RAW),dev)
GO_VERSION=$(if $(GO_VERSION_RAW),$(GO_VERSION_RAW),unknown)
CONFIG_PKG=github.com/sipeed/picoclaw/pkg/config
LDFLAGS=-X $(CONFIG_PKG).Version=$(VERSION) -X $(CONFIG_PKG).GitCommit=$(GIT_COMMIT) -X $(CONFIG_PKG).BuildTime=$(BUILD_TIME) -X $(CONFIG_PKG).GoVersion=$(GO_VERSION) -s -w


# OS detection
ifeq ($(OS),Windows_NT)
	UNAME_S=Windows
	ifeq ($(WINDOWS_GOARCH_RAW),amd64)
		UNAME_M=x86_64
	else ifeq ($(WINDOWS_GOARCH_RAW),arm64)
		UNAME_M=arm64
	else ifeq ($(WINDOWS_GOARCH_RAW),386)
		UNAME_M=x86
	else
		UNAME_M=$(if $(WINDOWS_GOARCH_RAW),$(WINDOWS_GOARCH_RAW),x86_64)
	endif
else
	UNAME_S:=$(shell uname -s)
	UNAME_M:=$(shell uname -m)
endif

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
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
	EXT=.exe
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
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path (Split-Path -Parent '$(OUTPUT)') | Out-Null"
else
	@mkdir -p "$$(dirname "$(OUTPUT)")"
endif
	${WEB_GO} build $(GOFLAGS) -ldflags "$(LAUNCHER_LDFLAGS)" -o "$(OUTPUT)" ./$(BACKEND_DIR)/

# Build launcher for Android ARM64 (frontend must already be built)
build-android-arm64: build-frontend
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path '$(BUILD_DIR)' | Out-Null"
else
	@mkdir -p $(BUILD_DIR)
endif
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o "$(OUTPUT_ANDROID_ARM64)" ./$(BACKEND_DIR)/

# Build launcher for all Android architectures
build-android-bundle: build-frontend
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path '$(BUILD_DIR)' | Out-Null"
else
	@mkdir -p $(BUILD_DIR)
endif
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/picoclaw-launcher-android-arm64" ./$(BACKEND_DIR)/
	@echo "All Android launcher builds complete"

build-frontend:
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "if ((-not (Test-Path -LiteralPath '$(FRONTEND_DIR)/node_modules')) -or (-not (Test-Path -LiteralPath '$(FRONTEND_DIR)/node_modules/.bin/tsc')) -or (-not (Test-Path -LiteralPath '$(FRONTEND_INSTALL_STAMP)')) -or ((Get-Content -LiteralPath '$(FRONTEND_INSTALL_STAMP)' -Raw).Trim() -ne (((Get-FileHash -LiteralPath '$(FRONTEND_DIR)/package.json' -Algorithm SHA256).Hash + ':' + (Get-FileHash -LiteralPath '$(FRONTEND_DIR)/pnpm-lock.yaml' -Algorithm SHA256).Hash)))) { Write-Host 'Installing frontend dependencies...'; Push-Location '$(FRONTEND_DIR)'; try { pnpm install --frozen-lockfile } finally { Pop-Location }; Set-Content -LiteralPath '$(FRONTEND_INSTALL_STAMP)' -Value (((Get-FileHash -LiteralPath '$(FRONTEND_DIR)/package.json' -Algorithm SHA256).Hash + ':' + (Get-FileHash -LiteralPath '$(FRONTEND_DIR)/pnpm-lock.yaml' -Algorithm SHA256).Hash)) -NoNewline }"
else
	@expected_stamp="$$(cat $(FRONTEND_DIR)/package.json $(FRONTEND_DIR)/pnpm-lock.yaml | cksum | awk '{print $$1 ":" $$2}')"; \
	if [ ! -d $(FRONTEND_DIR)/node_modules ] || \
		[ ! -x $(FRONTEND_DIR)/node_modules/.bin/tsc ] || \
		[ ! -f $(FRONTEND_INSTALL_STAMP) ] || \
		[ "$$(cat $(FRONTEND_INSTALL_STAMP) 2>/dev/null)" != "$$expected_stamp" ]; then \
		echo "Installing frontend dependencies..."; \
		(cd $(FRONTEND_DIR) && CI=true pnpm install --frozen-lockfile) && \
		printf '%s\n' "$$expected_stamp" > $(FRONTEND_INSTALL_STAMP); \
	fi
endif
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm build:backend

build-dev-picoclaw:
	@echo "Building picoclaw for launcher development..."
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path (Split-Path -Parent '$(PICOCLAW_BINARY)') | Out-Null"
else
	@mkdir -p "$$(dirname "$(PICOCLAW_BINARY)")"
endif
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
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "$$paths=@('$(FRONTEND_DIR)/dist','$(BACKEND_DIST)','$(BUILD_DIR)'); foreach($$p in $$paths){ if (Test-Path -LiteralPath $$p) { Remove-Item -LiteralPath $$p -Recurse -Force } }"
	@node $(FRONTEND_DIR)/scripts/ensure-backend-gitkeep.cjs
else
	rm -rf $(FRONTEND_DIR)/dist $(BACKEND_DIST) $(BUILD_DIR)
	node $(FRONTEND_DIR)/scripts/ensure-backend-gitkeep.cjs
endif
