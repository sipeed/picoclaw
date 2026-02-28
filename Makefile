.PHONY: all build install uninstall clean help test build-android build-android-arm64 build-android-x86_64 build-android-arm clean-android

# Build variables
BINARY_NAME=clawdroid
BUILD_DIR=build
CMD_DIR=cmd/$(BINARY_NAME)
MAIN_GO=$(CMD_DIR)/main.go

# Version
VERSION?=$(shell cat VERSION 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell $(GO) version | awk '{print $$3}')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Go variables
GO?=go
GOFLAGS?=-v
export GOTOOLCHAIN?=local

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_MAN_DIR=$(INSTALL_PREFIX)/share/man/man1

# Android
ANDROID_JNILIBS_DIR=android/app/src/embedded/jniLibs

# Workspace and Skills
CLAWDROID_HOME?=$(HOME)/.clawdroid
WORKSPACE_DIR?=$(CLAWDROID_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

# OS detection
UNAME_S:=$(shell uname -s)
UNAME_M:=$(shell uname -m)

# Platform-specific settings (Linux only)
PLATFORM=linux
ifeq ($(UNAME_M),x86_64)
	ARCH=amd64
else ifeq ($(UNAME_M),aarch64)
	ARCH=arm64
else
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

## build: Build the clawdroid binary for current platform
build: generate
	@echo "Building $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## build-all: Build clawdroid for all platforms
build-all: generate
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	@echo "All builds complete"

## build-android: Build clawdroid for Android (all architectures)
build-android: generate
	@echo "Building $(BINARY_NAME) for Android (all architectures)..."
	@mkdir -p $(ANDROID_JNILIBS_DIR)/arm64-v8a
	@mkdir -p $(ANDROID_JNILIBS_DIR)/x86_64
	@mkdir -p $(ANDROID_JNILIBS_DIR)/armeabi-v7a
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/arm64-v8a/libclawdroid.so ./$(CMD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/x86_64/libclawdroid.so ./$(CMD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/armeabi-v7a/libclawdroid.so ./$(CMD_DIR)
	@echo "Android build complete (all architectures)"

## build-android-arm64: Build clawdroid for Android (arm64-v8a only)
build-android-arm64: generate
	@echo "Building $(BINARY_NAME) for Android (arm64-v8a)..."
	@mkdir -p $(ANDROID_JNILIBS_DIR)/arm64-v8a
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/arm64-v8a/libclawdroid.so ./$(CMD_DIR)
	@echo "Android build complete (arm64-v8a)"

## build-android-x86_64: Build clawdroid for Android (x86_64 only)
build-android-x86_64: generate
	@echo "Building $(BINARY_NAME) for Android (x86_64)..."
	@mkdir -p $(ANDROID_JNILIBS_DIR)/x86_64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/x86_64/libclawdroid.so ./$(CMD_DIR)
	@echo "Android build complete (x86_64)"

## build-android-arm: Build clawdroid for Android (armeabi-v7a only)
build-android-arm: generate
	@echo "Building $(BINARY_NAME) for Android (armeabi-v7a)..."
	@mkdir -p $(ANDROID_JNILIBS_DIR)/armeabi-v7a
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build -trimpath $(LDFLAGS) \
		-o $(ANDROID_JNILIBS_DIR)/armeabi-v7a/libclawdroid.so ./$(CMD_DIR)
	@echo "Android build complete (armeabi-v7a)"

## clean-android: Remove Android jniLibs build artifacts
clean-android:
	@echo "Cleaning Android build artifacts..."
	@rm -rf $(ANDROID_JNILIBS_DIR)
	@echo "Android clean complete"

## install: Install clawdroid to system and copy builtin skills
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installed binary to $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Installation complete!"

## uninstall: Remove clawdroid from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Removed binary from $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Note: Only the executable file has been deleted."
	@echo "If you need to delete all configurations (config.json, workspace, etc.), run 'make uninstall-all'"

## uninstall-all: Remove clawdroid and all data
uninstall-all:
	@echo "Removing workspace and skills..."
	@rm -rf $(CLAWDROID_HOME)
	@echo "Removed workspace: $(CLAWDROID_HOME)"
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

## run: Build and run clawdroid
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## help: Show this help message
help:
	@echo "clawdroid Makefile"
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
	@echo "  make install-skills     # Install skills to workspace"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX          # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR           # Workspace directory (default: ~/.clawdroid/workspace)"
	@echo "  VERSION                 # Version string (default: git describe)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
