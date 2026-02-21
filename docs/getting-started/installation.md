# Installation

This guide covers installing PicoClaw on your system. Choose the method that best fits your needs.

> **Current Version:** `v0.1.2` — [View all releases](https://github.com/sipeed/picoclaw/releases)

## Quick Install

Download the precompiled binary for your platform from the [Releases page](https://github.com/sipeed/picoclaw/releases).




## Installation Methods

### Option 1: Precompiled Binary (Recommended)

The easiest way to get started. Download the appropriate binary for your platform:

| Platform | Architecture | Download |
|----------|-------------|----------|
| **Linux** | x86_64 | `picoclaw_Linux_x86_64.tar.gz` |
| **Linux** | ARM64 | `picoclaw_Linux_arm64.tar.gz` |
| **Linux** | ARMv6 | `picoclaw_Linux_armv6.tar.gz` |
| **Linux** | RISC-V 64 | `picoclaw_Linux_riscv64.tar.gz` |
| **Linux** | MIPS64 | `picoclaw_Linux_mips64.tar.gz` |
| **Linux** | s390x | `picoclaw_Linux_s390x.tar.gz` |
| **macOS** | ARM64 (M1/M2/M3/M4) | `picoclaw_Darwin_arm64.tar.gz` |
| **macOS** | x86_64 (Intel) | `picoclaw_Darwin_x86_64.tar.gz` |
| **Windows** | x86_64 | `picoclaw_Windows_x86_64.zip` |
| **Windows** | ARM64 | `picoclaw_Windows_arm64.zip` |
| **FreeBSD** | x86_64 | `picoclaw_Freebsd_x86_64.tar.gz` |
| **FreeBSD** | ARM64 | `picoclaw_Freebsd_arm64.tar.gz` |
| **FreeBSD** | ARMv6 | `picoclaw_Freebsd_armv6.tar.gz` |


**One-liner install Linux/macOS:**

```bash
# Linux/macOS - detect platform and install
VERSION="v0.1.2"
OS=$(uname -s)
ARCH=$(uname -m)
[ "$OS" = "Darwin" ] && OS="Darwin"
[ "$OS" = "Linux" ] && OS="Linux"
[ "$ARCH" = "x86_64" ] && ARCH="x86_64"
[ "$ARCH" = "aarch64" ] && ARCH="arm64"
wget -qO- https://github.com/sipeed/picoclaw/releases/download/${VERSION}/picoclaw_${OS}_${ARCH}.tar.gz | tar xz
sudo mv picoclaw /usr/local/bin/
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

VERSION="v0.1.2"

# Download binary
wget https://github.com/sipeed/picoclaw/releases/download/${VERSION}/picoclaw_Linux_arm64.tar.gz
tar -xzf picoclaw_Linux_arm64.tar.gz
chmod +x picoclaw

# Install proot for chroot environment
pkg install proot

# Initialize
termux-chroot ./picoclaw onboard
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
| macOS | $500+ | M1/M2/M3/M4 native |
| FreeBSD servers | Varies | Server deployment |

## Troubleshooting

### Permission denied

```bash
chmod +x picoclaw
```

### Command not found

Make sure the binary is in your PATH:
```bash
# Add to PATH temporarily
export PATH="$PWD:$PATH"

# Or move to a standard location
sudo mv picoclaw /usr/local/bin/
```

### Go version too old

PicoClaw requires Go 1.21 or later:
```bash
go version
# Upgrade Go if needed
```

### Docker issues

See [Docker Deployment](../deployment/docker.md) for troubleshooting Docker-specific issues.
