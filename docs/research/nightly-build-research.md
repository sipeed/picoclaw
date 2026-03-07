# Picoclaw Nightly Build Research

## Repository Overview

**Repository**: https://github.com/sipeed/picoclaw
**Current Branch**: `main`
**Latest Commit**: `66e6fb6`

### Project Structure

```
picoclaw/
├── .github/workflows/       # GitHub Actions workflows
│   ├── build.yml           # CI build on push to main
│   ├── pr.yml              # PR validation
│   ├── release.yml         # Release workflow (manual trigger)
│   └── upload-tos.yml      # Upload to Volcengine TOS
├── .goreleaser.yaml        # GoReleaser configuration
├── Makefile                # Build targets
├── go.mod                  # Go 1.25.7
├── cmd/                    # Main entry points
│   ├── picoclaw/          # Main binary
│   ├── picoclaw-launcher/ # Web launcher
│   └── picoclaw-launcher-tui/ # TUI launcher
├── pkg/                    # Core packages
│   ├── agent/             # Agent logic
│   ├── auth/              # Authentication
│   ├── channels/          # Communication channels
│   ├── config/            # Configuration
│   ├── memory/            # Memory management
│   ├── providers/         # LLM providers
│   ├── skills/            # Skills system
│   └── tools/             # Tool functions
├── docker/                 # Docker configurations
└── docs/                   # Documentation
```

## Existing Workflows

### 1. build.yml
- **Trigger**: Push to `main`
- **Action**: Builds picoclaw using `make build-all`
- **Platform**: Ubuntu latest

### 2. release.yml
- **Trigger**: Manual workflow dispatch
- **Actions**:
  - Creates git tag
  - Runs GoReleaser to build for all platforms
  - Uploads to GitHub Releases
  - Optionally uploads to TOS
- **Platforms**: Linux, Windows, Darwin, FreeBSD (amd64, arm64, riscv64, loong64, arm)
- **Docker**: Multi-platform builds to GHCR and DockerHub

### 3. pr.yml
- **Trigger**: Pull requests
- **Action**: PR validation (details not visible in current session)

### 4. upload-tos.yml
- **Trigger**: Called from release workflow
- **Action**: Uploads artifacts to Volcengine TOS

## Build System

### Makefile Targets
| Target | Description |
|--------|-------------|
| `build` | Build for current platform |
| `build-all` | Build for all platforms (8 variants) |
| `build-whatsapp-native` | Build with WhatsApp native support |
| `build-linux-arm` | Build for Linux ARMv7 |
| `build-linux-arm64` | Build for Linux ARM64 |
| `build-pi-zero` | Build for Raspberry Pi Zero 2 W |
| `install` | Install to `~/.local/bin` |
| `test` | Run tests |
| `lint` | Run golangci-lint |
| `docker-build` | Build minimal Docker image (Alpine) |
| `docker-build-full` | Build full Docker image (Node.js 24) |

### GoReleaser Configuration

**Builds:**
- `picoclaw` - Main binary
- `picoclaw-launcher` - Web launcher
- `picoclaw-launcher-tui` - TUI launcher

**Target Platforms:**
- OS: Linux, Windows, Darwin, FreeBSD
- Arch: amd64, arm64, riscv64, loong64, arm (v6, v7)

**Docker Images:**
- `ghcr.io/<owner>/picoclaw`
- `docker.io/<repository>`
- Platforms: linux/amd64, linux/arm64, linux/riscv64

## Nightly Build Requirements

### Goals
1. Build daily snapshots of the `main` branch
2. Upload artifacts to GitHub Releases (as pre-release)
3. Optionally build and push Docker images
4. Keep limited number of nightly builds (cleanup old ones)

### Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Cron Schedule**: `0 2 * * *` (2 AM UTC) | Off-peak hours, allows testing during business day |
| **Release Type**: Pre-release | Clearly distinguishes from stable releases |
| **Tag Format**: `nightly-YYYYMMDD` | Easy to identify and sort by date |
| **Artifact Retention**: 30 days | Balance between availability and storage |
| **Docker Tags**: `nightly`, `nightly-YYYYMMDD` | Allow both latest and pinned versions |

## Proposed Nightly.yml Workflow

```yaml
name: Nightly Build

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC
  workflow_dispatch:      # Allow manual trigger

jobs:
  nightly:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write
    steps:
      - name: Checkout
        uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - name: Set nightly tag
        id: tag
        run: |
          NIGHTLY_TAG="nightly-$(date +%Y%m%d)"
          echo "tag=$NIGHTLY_TAG" >> $GITHUB_OUTPUT

      - name: Create/Update nightly tag
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git tag -f "${{ steps.tag.outputs.tag }}"
          git push -f origin "${{ steps.tag.outputs.tag }}"

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Run GoReleaser (snapshot)
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: ~> v2
          args: release --clean --snapshot
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          NIGHTLY_TAG: ${{ steps.tag.outputs.tag }}
          DOCKERHUB_IMAGE_NAME: ${{ vars.DOCKERHUB_REPOSITORY }}

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: nightly-artifacts
          path: dist/

      - name: Clean old nightly releases
        run: |
          # Keep only last 30 nightly releases
          gh release list \
            --query "[startsWith(name, 'nightly-')] | [30:].[].name" \
            --json name \
            -t '{{ range . }}{{ .name }}{{ "\n" }}{{ end }}' \
            | xargs -I {} gh release delete {} --yes || true
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Notes

1. **Security**: No secrets required for basic nightly builds
2. **Storage**: GitHub provides 2GB free storage; implement cleanup if needed
3. **Testing**: Nightly builds serve as canary builds for testing
4. **Docker**: Can optionally push `nightly` Docker tags
5. **Notifications**: Could add Slack/discord notifications on failure

## References

- GoReleaser docs: https://goreleaser.com/
- GitHub Actions docs: https://docs.github.com/en/actions
- Picoclaw docs: /docs/tools_configuration.md
