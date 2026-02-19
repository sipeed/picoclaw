# Installation

This guide covers installing PicoClaw on your system. Choose the method that best fits your needs.

## Quick Install

Download the precompiled binary for your platform from the [Releases page](https://github.com/sipeed/picoclaw/releases).

```bash
# Example: Download for Linux ARM64
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
./picoclaw-linux-arm64 onboard
```

## Installation Methods

### Option 1: Precompiled Binary (Recommended)

The easiest way to get started. Download the appropriate binary for your platform:

| Platform | Architecture | Binary Name |
|----------|-------------|-------------|
| Linux | x86_64 | `picoclaw-linux-amd64` |
| Linux | ARM64 | `picoclaw-linux-arm64` |
| Linux | RISC-V | `picoclaw-linux-riscv64` |
| Linux | LoongArch | `picoclaw-linux-loong64` |
| macOS | ARM64 (M1/M2/M3) | `picoclaw-darwin-arm64` |
| Windows | x86_64 | `picoclaw-windows-amd64.exe` |

```bash
# Download and make executable
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64
chmod +x picoclaw-linux-amd64

# Move to PATH (optional)
sudo mv picoclaw-linux-amd64 /usr/local/bin/picoclaw
```

### Option 2: Build from Source

Building from source gives you the latest features and is recommended for development.

**Prerequisites:**
- Go 1.21 or later
- Git

**Steps:**

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Download dependencies
make deps

# Build for current platform
make build

# The binary will be in ./build/picoclaw
./build/picoclaw onboard
```

**Build for all platforms:**

```bash
make build-all
# Binaries will be in ./build/ for each platform
```

**Install to system:**

```bash
# Install to ~/.local/bin
make install

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"
```

### Option 3: Docker

Run PicoClaw in a container without installing anything locally.

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Configure
cp config/config.example.json config/config.json
# Edit config/config.json with your API keys

# Start gateway mode
docker compose --profile gateway up -d

# Or run agent mode (one-shot)
docker compose run --rm picoclaw-agent -m "Hello!"

# Interactive mode
docker compose run --rm picoclaw-agent
```

See [Docker Deployment](../deployment/docker.md) for more details.

### Option 4: Android/Termux

Run PicoClaw on old Android phones using Termux.

```bash
# Install Termux from F-Droid or Google Play
# Open Termux and run:

# Download binary
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64

# Install proot for chroot environment
pkg install proot

# Initialize
termux-chroot ./picoclaw-linux-arm64 onboard
```

See [Termux Deployment](../deployment/termux.md) for more details.

## Verify Installation

After installing, verify PicoClaw is working:

```bash
# Check version
picoclaw version

# Check status
picoclaw status
```

## Next Steps

1. [Quick Start](quick-start.md) - Configure and run your first chat
2. [Configuration Basics](configuration-basics.md) - Understand the config structure
3. [Get API Keys](quick-start.md#get-api-keys) - Set up your LLM provider

## Supported Platforms

PicoClaw runs on a wide range of hardware:

| Hardware | Cost | Notes |
|----------|------|-------|
| Any Linux x86_64 | Varies | Most common |
| Raspberry Pi | $35+ | ARM64 support |
| LicheeRV Nano | $9.9 | RISC-V, ultra-low cost |
| MaixCAM | $50 | AI camera |
| NanoKVM | $30-100 | Server maintenance |
| Android phones | Free (old) | Via Termux |
| macOS | $500+ | M1/M2/M3 native |

## Troubleshooting

### Permission denied

```bash
chmod +x picoclaw-*
```

### Command not found

Make sure the binary is in your PATH:
```bash
# Add to PATH temporarily
export PATH="$PWD:$PATH"

# Or move to a standard location
sudo mv picoclaw-* /usr/local/bin/picoclaw
```

### Go version too old

PicoClaw requires Go 1.21 or later:
```bash
go version
# Upgrade Go if needed
```

### Docker issues

See [Docker Deployment](../deployment/docker.md) for troubleshooting Docker-specific issues.
