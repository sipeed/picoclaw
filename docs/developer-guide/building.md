# Building from Source

This guide explains how to build PicoClaw from source code.

## Prerequisites

### Go

PicoClaw requires Go 1.21 or later. Check your Go version:

```bash
go version
```

If Go is not installed, download it from [golang.org](https://golang.org/dl/).

### Make

PicoClaw uses a Makefile for build automation. Ensure `make` is installed:

```bash
make --version
```

### Git

For cloning the repository:

```bash
git --version
```

## Getting the Source Code

Clone the repository:

```bash
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw
```

## Dependencies

Download dependencies:

```bash
make deps
```

This runs `go mod download` to fetch all required modules.

## Building

### Build for Current Platform

Build for your current operating system and architecture:

```bash
make build
```

This will:
1. Run `go generate` to generate any required code
2. Build the binary to `build/picoclaw`

The output binary will be at `build/picoclaw` (or `build/picoclaw.exe` on Windows).

### Build for All Platforms

Build for all supported platforms:

```bash
make build-all
```

This creates binaries for:
- `linux-amd64`
- `linux-arm64`
- `linux-loong64`
- `linux-riscv64`
- `darwin-arm64` (macOS Apple Silicon)
- `windows-amd64`

Binaries are placed in the `build/` directory with platform-specific naming.

### Manual Build

You can also build manually with Go commands:

```bash
# Generate code (if needed)
go generate ./...

# Build for current platform
go build -o build/picoclaw ./cmd/picoclaw

# Build with optimizations
go build -ldflags="-s -w" -o build/picoclaw ./cmd/picoclaw
```

### Cross-Compilation

Build for a specific platform:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o build/picoclaw-linux-amd64 ./cmd/picoclaw

# Linux ARM64 (Raspberry Pi, etc.)
GOOS=linux GOARCH=arm64 go build -o build/picoclaw-linux-arm64 ./cmd/picoclaw

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o build/picoclaw-darwin-arm64 ./cmd/picoclaw

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o build/picoclaw-windows-amd64.exe ./cmd/picoclaw
```

## Installation

### Install to Default Location

Install to `~/.local/bin`:

```bash
make install
```

Ensure `~/.local/bin` is in your PATH:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Add to your shell profile for persistence:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
# or
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
```

### Custom Installation

Install to a custom location:

```bash
go build -o /usr/local/bin/picoclaw ./cmd/picoclaw
```

## Verification

Verify the build:

```bash
./build/picoclaw --version
```

Run a quick test:

```bash
# Interactive mode
./build/picoclaw agent

# Single message
./build/picoclaw agent -m "Hello, PicoClaw!"
```

## Development Build

For development, you may want to build with debug symbols:

```bash
go build -gcflags="all=-N -l" -o build/picoclaw-debug ./cmd/picoclaw
```

## Build Options

### Makefile Targets

| Target | Description |
|--------|-------------|
| `deps` | Download dependencies |
| `build` | Build for current platform |
| `build-all` | Build for all platforms |
| `install` | Install to ~/.local/bin |
| `test` | Run tests |
| `vet` | Run linter |
| `fmt` | Format code |
| `check` | Run all checks (deps, fmt, vet, test) |
| `clean` | Remove build artifacts |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GOOS` | Target operating system |
| `GOARCH` | Target architecture |
| `CGO_ENABLED` | Enable CGO (default: 0) |

## Troubleshooting

### "go: command not found"

Install Go from [golang.org](https://golang.org/dl/).

### "go: cannot find main module"

Ensure you're in the project root directory:

```bash
cd picoclaw
```

### Build Errors

1. Clean and rebuild:

```bash
go clean -cache
make build
```

2. Update dependencies:

```bash
go mod tidy
make deps
```

3. Check Go version:

```bash
go version  # Should be 1.21+
```

### Hardware Tool Build Errors

I2C and SPI tools require Linux-specific system calls. On non-Linux platforms, these tools return errors at runtime but the build should succeed.

If you see build errors related to these tools:

```bash
# The build should still work - these are stub implementations
make build
```

## Docker Build (Optional)

Build with Docker:

```bash
# Build image
docker build -t picoclaw .

# Run container
docker run -it picoclaw agent -m "Hello"
```

## Next Steps

After building:

1. [Configure PicoClaw](../getting-started/configuration-basics.md)
2. [Run your first chat](../getting-started/first-chat.md)
3. [Set up channels](../user-guide/channels/README.md)
