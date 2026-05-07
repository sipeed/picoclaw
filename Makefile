.PHONY: all build install uninstall clean help test build-all lint-docs

# Build variables
BINARY_NAME=picoclaw
BUILD_DIR=build
CMD_DIR=cmd/$(BINARY_NAME)
MAIN_GO=$(CMD_DIR)/main.go
EXT=

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

# Go variables
GO?=go
WEB_GO?=$(GO)
CGO_ENABLED?=0
GO_BUILD_TAGS?=goolm,stdjson
GOFLAGS?=-v -tags $(GO_BUILD_TAGS)
GOCACHE?=$(CURDIR)/.cache/go-build
GOMODCACHE?=$(CURDIR)/.cache/go-mod
GOTOOLCHAIN?=local
export CGO_ENABLED
export GOCACHE
export GOMODCACHE
export GOTOOLCHAIN
comma:=,
empty:=
space:=$(empty) $(empty)
GO_BUILD_TAGS_NO_GOOLM:=$(subst $(space),$(comma),$(strip $(filter-out goolm,$(subst $(comma),$(space),$(GO_BUILD_TAGS)))))
GOFLAGS_NO_GOOLM?=-v -tags $(GO_BUILD_TAGS_NO_GOOLM)

# Patch MIPS LE ELF e_flags (offset 36) for NaN2008-only kernels (e.g. Ingenic X2600).
#
# Bytes (octal): \004 \024 \000 \160  →  little-endian 0x70001404
#   0x70000000  EF_MIPS_ARCH_32R2   MIPS32 Release 2
#   0x00001000  EF_MIPS_ABI_O32     O32 ABI
#   0x00000400  EF_MIPS_NAN2008     IEEE 754-2008 NaN encoding
#   0x00000004  EF_MIPS_CPIC        PIC calling sequence
#
# Go's GOMIPS=softfloat emits no FP instructions, so the NaN mode is irrelevant
# at runtime — this is purely an ELF metadata fix to satisfy the kernel's check.
# patchelf cannot modify e_flags; dd at a fixed offset is the most portable way.
#
# Ref: https://codebrowser.dev/linux/linux/arch/mips/include/asm/elf.h.html
define PATCH_MIPS_FLAGS
	@if [ -f "$(1)" ]; then \
		printf '\004\024\000\160' | dd of=$(1) bs=1 seek=36 count=4 conv=notrunc 2>/dev/null || \
		{ echo "Error: failed to patch MIPS e_flags for $(1)"; exit 1; }; \
	else \
		echo "Error: $(1) not found, cannot patch MIPS e_flags"; exit 1; \
	fi
endef

# Patch creack/pty for loong64 support (upstream doesn't have ztypes_loong64.go)
PTY_PATCH_LOONG64=pty_dir=$$(go env GOMODCACHE)/github.com/creack/pty@v1.1.9; \
	if [ -d "$$pty_dir" ] && [ ! -f "$$pty_dir/ztypes_loong64.go" ]; then \
		chmod +w "$$pty_dir" 2>/dev/null || true; \
		printf '//go:build linux && loong64\npackage pty\ntype (_C_int int32; _C_uint uint32)\n' > "$$pty_dir/ztypes_loong64.go"; \
	fi

# Golangci-lint
GOLANGCI_LINT?=golangci-lint

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_MAN_DIR=$(INSTALL_PREFIX)/share/man/man1
INSTALL_TMP_SUFFIX=.new

# Workspace and Skills
PICOCLAW_HOME?=$(HOME)/.picoclaw
WORKSPACE_DIR?=$(PICOCLAW_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

LNCMD=ln -sf

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
	UNAME_S?=$(shell uname -s)
	UNAME_M?=$(shell uname -m)
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
	WEB_GO=CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go
	ifeq ($(UNAME_M),x86_64)
		ARCH?=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH?=arm64
	else
		ARCH?=$(UNAME_M)
	endif
else
	PLATFORM=$(UNAME_S)
	ifeq ($(UNAME_M),x86_64)
		ARCH?=amd64
	else
	    ARCH?=$(UNAME_M)
	endif
	# Detect Windows (Git Bash / MSYS2)
    IS_WINDOWS:=$(if $(findstring MINGW,$(UNAME_S)),yes,$(if $(findstring MSYS,$(UNAME_S)),yes,$(if $(findstring CYGWIN,$(UNAME_S)),yes,no)))
	ifeq ($(IS_WINDOWS),yes)
	    EXT=.exe
	    LNCMD=cp
	else ifeq ($(UNAME_S),windows) # failsafe for force windows build in other OS using UNAME_S=windows
		EXT=.exe
	endif

endif

ifeq ($(OS),Windows_NT)
	PLATFORM=windows
	ifeq ($(UNAME_M),x86_64)
		ARCH?=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH?=arm64
	else
		ARCH?=$(UNAME_M)
	endif
	EXT=.exe
endif

ifneq ($(strip $(GOOS)),)
	PLATFORM:=$(GOOS)
endif

ifneq ($(strip $(GOARCH)),)
	ARCH:=$(GOARCH)
endif

ifeq ($(PLATFORM),windows)
	EXT=.exe
endif

BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)-$(PLATFORM)-$(ARCH)

# Default target
all: build

## generate: Run generate
generate:
	@echo "Run generate..."
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "if (Test-Path -LiteralPath './$(CMD_DIR)/workspace') { Remove-Item -LiteralPath './$(CMD_DIR)/workspace' -Recurse -Force }"
	@$(POWERSHELL) "$$env:GOOS=''; $$env:GOARCH=''; $(GO) generate ./..."
else
	@rm -r ./$(CMD_DIR)/workspace 2>/dev/null || true
	@GOOS=$$($(GO) env GOHOSTOS) GOARCH=$$($(GO) env GOHOSTARCH) $(GO) generate ./...
endif
	@echo "Run generate complete"

## build: Build the picoclaw binary for current platform
build: generate
	@echo "Building $(BINARY_NAME)$(EXT) for $(PLATFORM)/$(ARCH)..."
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path '$(BUILD_DIR)' | Out-Null"
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH)$(EXT) ./$(CMD_DIR)
	@$(POWERSHELL) "Copy-Item -LiteralPath '$(BINARY_PATH)$(EXT)' -Destination '$(BUILD_DIR)/$(BINARY_NAME)$(EXT)' -Force"
else
	@mkdir -p $(BUILD_DIR)
	@GOOS=$(PLATFORM) GOARCH=$(ARCH) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH)$(EXT) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)$(EXT)"
	@$(LNCMD) $(BINARY_NAME)-$(PLATFORM)-$(ARCH)$(EXT) $(BUILD_DIR)/$(BINARY_NAME)$(EXT)
endif
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(EXT)"

## build-launcher: Build the picoclaw-launcher (web console) binary
build-launcher:
	@echo "Building picoclaw-launcher for $(PLATFORM)/$(ARCH)..."
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "New-Item -ItemType Directory -Force -Path '$(BUILD_DIR)' | Out-Null"
	@$(MAKE) -C web build PLATFORM="$(PLATFORM)" ARCH="$(ARCH)" EXT="$(EXT)" OUTPUT="$(CURDIR)/$(BUILD_DIR)/picoclaw-launcher-$(PLATFORM)-$(ARCH)$(EXT)" GO_BUILD_TAGS="$(GO_BUILD_TAGS)"
	@$(POWERSHELL) "Copy-Item -LiteralPath '$(BUILD_DIR)/picoclaw-launcher-$(PLATFORM)-$(ARCH)$(EXT)' -Destination '$(BUILD_DIR)/picoclaw-launcher$(EXT)' -Force"
else
	@mkdir -p $(BUILD_DIR)
	@GOOS=$(PLATFORM) GOARCH=$(ARCH) $(MAKE) -C web build \
		OUTPUT="$(CURDIR)/$(BUILD_DIR)/picoclaw-launcher-$(PLATFORM)-$(ARCH)$(EXT)" \
		WEB_GO='$(WEB_GO)' \
		GO_BUILD_TAGS='$(GO_BUILD_TAGS)' \
		LDFLAGS='$(LDFLAGS)'
	@$(LNCMD) picoclaw-launcher-$(PLATFORM)-$(ARCH)$(EXT) $(BUILD_DIR)/picoclaw-launcher$(EXT)
endif
	@echo "Build complete: $(BUILD_DIR)/picoclaw-launcher$(EXT)"

build-launcher-frontend:
	@$(MAKE) -C web build-frontend

## build-whatsapp-native: Build with WhatsApp native (whatsmeow) support; larger binary
build-whatsapp-native: generate
## @echo "Building $(BINARY_NAME) with WhatsApp native for $(PLATFORM)/$(ARCH)..."
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=loong64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-loong64 ./$(CMD_DIR)
	GOOS=linux GOARCH=riscv64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-riscv64 ./$(CMD_DIR)
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build -tags $(GO_BUILD_TAGS_NO_GOOLM),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle ./$(CMD_DIR)
	$(call PATCH_MIPS_FLAGS,$(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle)
	GOOS=darwin GOARCH=arm64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -tags $(GO_BUILD_TAGS),whatsapp_native -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
## @$(GO) build $(GOFLAGS) -tags whatsapp_native -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete"
##	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## build-linux-arm: Build for Linux ARMv7 (e.g. Raspberry Pi Zero 2 W 32-bit)
build-linux-arm: generate
	@echo "Building for linux/arm (GOARM=7)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm"

## build-linux-arm64: Build for Linux ARM64 (e.g. Raspberry Pi Zero 2 W 64-bit)
build-linux-arm64: generate
	@echo "Building for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

## build-linux-mipsle: Build for Linux MIPS32 LE
build-linux-mipsle: generate
	@echo "Building for linux/mipsle (softfloat)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build $(GOFLAGS_NO_GOOLM) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle ./$(CMD_DIR)
	$(call PATCH_MIPS_FLAGS,$(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle"

## build-android-arm64: Build core for Android ARM64
build-android-arm64: generate
	@echo "Building for android/arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-android-arm64 ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-android-arm64"

## build-launcher-android-arm64: Build launcher for Android ARM64
build-launcher-android-arm64:
	@echo "Building picoclaw-launcher for android/arm64..."
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) -C web build-android-arm64 \
		OUTPUT_ANDROID_ARM64="$(CURDIR)/$(BUILD_DIR)/picoclaw-launcher-android-arm64" \
		GO='$(GO)' \
		LDFLAGS='$(LDFLAGS)'
	@echo "Build complete: $(BUILD_DIR)/picoclaw-launcher-android-arm64"

## build-android-bundle: Build core and launcher for all Android architectures and package as universal zip
build-android-bundle: generate
	@echo "Building core for all Android architectures..."
	@mkdir -p $(BUILD_DIR)
	GOOS=android GOARCH=arm64 $(GO) build -tags stdjson -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-android-arm64 ./$(CMD_DIR)
	@echo "Building launcher for Android arm64..."
	@$(MAKE) build-launcher-android-arm64
	@echo "Staging JNI libs..."
	@rm -rf $(BUILD_DIR)/android-staging
	@mkdir -p $(BUILD_DIR)/android-staging/arm64-v8a
	@cp $(BUILD_DIR)/$(BINARY_NAME)-android-arm64 $(BUILD_DIR)/android-staging/arm64-v8a/libpicoclaw.so
	@cp $(BUILD_DIR)/picoclaw-launcher-android-arm64 $(BUILD_DIR)/android-staging/arm64-v8a/libpicoclaw-web.so
	@cd $(BUILD_DIR)/android-staging && zip -r ../picoclaw-android-universal.zip .
	@rm -rf $(BUILD_DIR)/android-staging
	@echo "All Android builds complete: $(BUILD_DIR)/picoclaw-android-universal.zip"

## build-pi-zero: Build for Raspberry Pi Zero 2 W (32-bit and 64-bit)
build-pi-zero: build-linux-arm build-linux-arm64
	@echo "Pi Zero 2 W builds: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm (32-bit), $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 (64-bit)"

## build-all: Build the picoclaw core binary for all Makefile-managed platforms
build-all: generate
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@$(PTY_PATCH_LOONG64)
	GOOS=linux GOARCH=loong64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-loong64 ./$(CMD_DIR)
	GOOS=linux GOARCH=riscv64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-riscv64 ./$(CMD_DIR)
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build $(GOFLAGS_NO_GOOLM) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle ./$(CMD_DIR)
	$(call PATCH_MIPS_FLAGS,$(BUILD_DIR)/$(BINARY_NAME)-linux-mipsle)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	GOOS=netbsd GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-amd64 ./$(CMD_DIR)
	GOOS=netbsd GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-arm64 ./$(CMD_DIR)
	@echo "Core builds complete"

## install: Install picoclaw to system and copy builtin skills
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	# Copy binary with temporary suffix to ensure atomic update
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX)
	@mv -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installed binary to $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Installation complete!"

## uninstall: Remove picoclaw from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Removed binary from $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Note: Only the executable file has been deleted."
	@echo "If you need to delete all configurations (config.json, workspace, etc.), run 'make uninstall-all'"

## uninstall-all: Remove picoclaw and all data
uninstall-all:
	@echo "Removing workspace and skills..."
	@rm -rf $(PICOCLAW_HOME)
	@echo "Removed workspace: $(PICOCLAW_HOME)"
	@echo "Complete uninstallation done!"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
ifeq ($(OS),Windows_NT)
	@$(POWERSHELL) "if (Test-Path -LiteralPath '$(BUILD_DIR)') { Remove-Item -LiteralPath '$(BUILD_DIR)' -Recurse -Force }"
else
	@rm -rf $(BUILD_DIR)
endif
	@echo "Clean complete"

## vet: Run go vet for static analysis
vet: generate
	@packages="$$($(GO) list $(GOFLAGS) ./...)" && \
		$(GO) vet $(GOFLAGS) $$(printf '%s\n' "$$packages" | grep -v '^github.com/sipeed/picoclaw/web/')
	@cd web/backend && $(WEB_GO) vet ./...

## test: Test Go code
test: generate
	@$(GO) test $(GOFLAGS) $$($(GO) list $(GOFLAGS) ./... | grep -v github.com/sipeed/picoclaw/web/)
	@cd web && make test

## fmt: Format Go code
fmt:
	@$(GOLANGCI_LINT) fmt

## lint-docs: Check common documentation layout and naming conventions
lint-docs:
	@./scripts/lint-docs.sh

## lint: Run linters
lint:
	@$(GOLANGCI_LINT) run --build-tags $(GO_BUILD_TAGS)
	@./scripts/lint-docs.sh

## fix: Fix linting issues
fix:
	@$(GOLANGCI_LINT) run --fix --build-tags $(GO_BUILD_TAGS)

## deps: Download dependencies
deps:
	@$(GO) mod download
	@$(GO) mod verify

## update-deps: Update dependencies
update-deps:
	@$(GO) get -u ./...
	@$(GO) mod tidy

## check: Run deps, fmt, vet, tests, and docs consistency checks
check: deps fmt vet test lint-docs

## run: Build and run picoclaw
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## docker-build: Build Docker image (minimal Alpine-based)
docker-build:
	@echo "Building minimal Docker image (Alpine-based)..."
	docker compose -f docker/docker-compose.yml build picoclaw-agent picoclaw-gateway

## docker-build-full: Build Docker image with full MCP support (Node.js 24)
docker-build-full:
	@echo "Building full-featured Docker image (Node.js 24)..."
	docker compose -f docker/docker-compose.full.yml build picoclaw-agent picoclaw-gateway

## docker-test: Test MCP tools in Docker container
docker-test:
	@echo "Testing MCP tools in Docker..."
	@chmod +x scripts/test-docker-mcp.sh
	@./scripts/test-docker-mcp.sh

## docker-run: Run picoclaw gateway in Docker (Alpine-based)
docker-run:
	docker compose -f docker/docker-compose.yml --profile gateway up

## docker-run-full: Run picoclaw gateway in Docker (full-featured)
docker-run-full:
	docker compose -f docker/docker-compose.full.yml --profile gateway up

## docker-run-agent: Run picoclaw agent in Docker (interactive, Alpine-based)
docker-run-agent:
	docker compose -f docker/docker-compose.yml run --rm picoclaw-agent

## docker-run-agent-full: Run picoclaw agent in Docker (interactive, full-featured)
docker-run-agent-full:
	docker compose -f docker/docker-compose.full.yml run --rm picoclaw-agent

## docker-clean: Clean Docker images and volumes
docker-clean:
	docker compose -f docker/docker-compose.yml down -v
	docker compose -f docker/docker-compose.full.yml down -v
	docker rmi picoclaw:latest picoclaw:full 2>/dev/null || true


## build-macos-app: Build PicoClaw macOS .app bundle (no terminal window)
build-macos-app:build-launcher
	@echo "Building macOS .app bundle..."
	@if [ "$(UNAME_S)" != "Darwin" ]; then \
		echo "Error: This target is only available on macOS"; \
		exit 1; \
	fi
	@./scripts/build-macos-app.sh $(PLATFORM)-$(ARCH)
	@echo "macOS .app bundle created: $(BUILD_DIR)/PicoClaw.app"

## mem: Build membench, download LOCOMO data (if needed), run benchmark, and show results
mem:
	@echo "Building membench..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/membench ./cmd/membench
	@echo "Build complete: $(BUILD_DIR)/membench"
	@if [ ! -f $(BUILD_DIR)/memdata/locomo10.json ]; then \
		echo "Downloading LOCOMO dataset..."; \
		mkdir -p $(BUILD_DIR)/memdata; \
		curl -sfL "https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json" \
			-o $(BUILD_DIR)/memdata/locomo10.json && [ -s $(BUILD_DIR)/memdata/locomo10.json ] || { echo "Error: LOCOMO download failed"; exit 1; }; \
		echo "Download complete"; \
	else \
		echo "LOCOMO dataset already exists, skipping download"; \
	fi
	@echo "Running benchmark..."
	@rm -rf $(BUILD_DIR)/memout
	@$(BUILD_DIR)/membench run --data $(BUILD_DIR)/memdata --out $(BUILD_DIR)/memout --budget 4000

## help: Show this help message
help:
	@echo "picoclaw Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sort | awk -F': ' '{printf "  %-16s %s\n", substr($$1, 4), $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make install            # Install to ~/.local/bin"
	@echo "  make uninstall          # Remove from /usr/local/bin"
	@echo "  make install-skills     # Install skills to workspace"
	@echo "  make docker-build       # Build minimal Docker image"
	@echo "  make docker-test        # Test MCP tools in Docker"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX          # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR           # Workspace directory (default: ~/.picoclaw/workspace)"
	@echo "  VERSION                 # Version string (default: git describe)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
