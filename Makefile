.PHONY: all build install uninstall clean help test deploy-hostinger deploy-hostinger-setup deploy-hostinger-status deploy-hostinger-rollback

# Build variables
BINARY_NAME=picoclaw
BUILD_DIR=build
CMD_DIR=cmd/$(BINARY_NAME)
MAIN_GO=$(CMD_DIR)/main.go

# Version
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell $(GO) version | awk '{print $$3}')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Go variables
GO?=go
GOFLAGS?=-v

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_MAN_DIR=$(INSTALL_PREFIX)/share/man/man1

# Workspace and Skills
PICOCLAW_HOME?=$(HOME)/.picoclaw
WORKSPACE_DIR?=$(PICOCLAW_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

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
	else ifeq ($(UNAME_M),riscv64)
		ARCH=riscv64
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Darwin)
	PLATFORM=darwin
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)-$(PLATFORM)-$(ARCH)

# Default target
all: build

## generate: Run generate
generate:
	@echo "Run generate..."
	@rm -r ./$(CMD_DIR)/workspace 2>/dev/null || true
	@$(GO) generate ./...
	@echo "Run generate complete"

## build: Build the picoclaw binary for current platform
build: generate
	@echo "Building $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## build-all: Build picoclaw for all platforms
build-all: generate
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=riscv64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-riscv64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "All builds complete"

## install: Install picoclaw to system and copy builtin skills
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)
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
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## vet: Run go vet for static analysis
vet:
	@$(GO) vet ./...

## fmt: Format Go code
test:
	@$(GO) test ./...

## fmt: Format Go code
fmt:
	@$(GO) fmt ./...

## deps: Download dependencies
deps:
	@$(GO) mod download
	@$(GO) mod verify

## update-deps: Update dependencies
update-deps:
	@$(GO) get -u ./...
	@$(GO) mod tidy

## check: Run vet, fmt, and verify dependencies
check: deps fmt vet test

## run: Build and run picoclaw
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# ── Hostinger Deployment ─────────────────────────────
HOSTINGER_HOST?=
HOSTINGER_USER?=root
HOSTINGER_SSH_KEY?=$(HOME)/.ssh/id_rsa
HOSTINGER_SSH_PORT?=22
HOSTINGER_DEPLOY_METHOD?=docker

## deploy-hostinger-setup: Run initial server setup on Hostinger VPS
deploy-hostinger-setup:
	@bash deploy/hostinger/setup-server.sh

## deploy-hostinger: Deploy PicoClaw to Hostinger VPS
deploy-hostinger:
	@bash deploy/hostinger/deploy.sh \
		-h "$(HOSTINGER_HOST)" \
		-u "$(HOSTINGER_USER)" \
		-k "$(HOSTINGER_SSH_KEY)" \
		-p "$(HOSTINGER_SSH_PORT)" \
		-m "$(HOSTINGER_DEPLOY_METHOD)"

## deploy-hostinger-status: Check PicoClaw status on Hostinger VPS
deploy-hostinger-status:
	@bash deploy/hostinger/status.sh \
		-h "$(HOSTINGER_HOST)" \
		-u "$(HOSTINGER_USER)" \
		-k "$(HOSTINGER_SSH_KEY)" \
		-p "$(HOSTINGER_SSH_PORT)" \
		-m "$(HOSTINGER_DEPLOY_METHOD)"

## deploy-hostinger-rollback: Rollback PicoClaw on Hostinger VPS
deploy-hostinger-rollback:
	@bash deploy/hostinger/rollback.sh \
		-h "$(HOSTINGER_HOST)" \
		-u "$(HOSTINGER_USER)" \
		-k "$(HOSTINGER_SSH_KEY)" \
		-p "$(HOSTINGER_SSH_PORT)" \
		-m "$(HOSTINGER_DEPLOY_METHOD)"

## help: Show this help message
help:
	@echo "picoclaw Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make install            # Install to ~/.local/bin"
	@echo "  make uninstall          # Remove from /usr/local/bin"
	@echo ""
	@echo "Hostinger Deployment:"
	@echo "  make deploy-hostinger-setup  HOSTINGER_HOST=1.2.3.4  # Initial server setup"
	@echo "  make deploy-hostinger        HOSTINGER_HOST=1.2.3.4  # Deploy to VPS"
	@echo "  make deploy-hostinger-status HOSTINGER_HOST=1.2.3.4  # Check status"
	@echo "  make deploy-hostinger-rollback HOSTINGER_HOST=1.2.3.4 # Rollback"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX              # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR               # Workspace directory (default: ~/.picoclaw/workspace)"
	@echo "  VERSION                     # Version string (default: git describe)"
	@echo "  HOSTINGER_HOST              # Hostinger VPS IP address"
	@echo "  HOSTINGER_USER              # SSH user (default: root)"
	@echo "  HOSTINGER_SSH_KEY           # SSH key path (default: ~/.ssh/id_rsa)"
	@echo "  HOSTINGER_DEPLOY_METHOD     # Deploy method: docker or binary (default: docker)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
