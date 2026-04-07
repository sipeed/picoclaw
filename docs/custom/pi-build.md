# Raspberry Pi Build Guide

This guide details the step-by-step process for building a custom version of Picoclaw and transferring it to a Raspberry Pi Zero 2W running locally.

## Prerequisite: On Vulcan (Windows)

1. **Install Go**: Ensure Go is installed on your Windows machine (`https://go.dev/dl/`). Add it to your `PATH` or invoke it directly.
2. **Clone repo**: Clone your custom fork of Picoclaw:
   ```bash
   git clone https://github.com/<your-username>/picoclaw.git
   ```
3. **Modify code**: Make your custom logic changes (e.g., in `pkg/agent/metrics.go` and `pkg/agent/loop.go`).

## Step A: Compile for Raspberry Pi (ARM v7)

Since the Raspberry Pi Zero 2W is an ARM-based environment (running a 32-bit `armhf` OS usually), we use Go's powerful cross-compilation features natively on Windows.

Run this dynamic command in PowerShell from the repository root to automatically bake in the current Version and Commit Hash:

```powershell
# 1. Capture dynamic build metadata
$v = (git describe --tags --always --dirty 2>$null) -join ""; if (!$v) { $v = "dev" }
$c = (git rev-parse --short HEAD 2>$null) -join ""; if (!$c) { $c = "unknown" }
$t = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"

# 2. Compile with metadata injection
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="arm"; $env:GOARM="7"
go build -v -tags "goolm,stdjson" `
  -ldflags "-X github.com/sipeed/picoclaw/pkg/config.Version=$v -X github.com/sipeed/picoclaw/pkg/config.GitCommit=$c -X github.com/sipeed/picoclaw/pkg/config.BuildTime=$t" `
  -o picoclaw-custom ./cmd/picoclaw
```

This ensures your Telegram header shows the correct version and commit info instead of just `dev`.

## Step B: Transfer to Pi

Use `scp` (Secure Copy Protocol), which transfers files over an encrypted SSH connection.

```powershell
scp picoclaw-custom tim@picoclaw.local:/tmp/
```

*This copies your newly compiled `picoclaw-custom` executable file from Windows up to the `/tmp/` folder on your Pi.*

>**Note:** You are **not** copying the entire project folder or any `.go` code! Go's superpower is compiling all your files and logic down into a single, standalone binary file. That one compressed file contains the entire bot engine and is the absolute only thing the Pi needs to run perfectly.

## Step C: Update the Raspberry Pi Service

Log into your Raspberry Pi terminal via SSH.

1. **Stop the current running service**:
   ```bash
   sudo systemctl stop picoclaw
   ```
2. **Replace the old binary with the new custom one**:
   ```bash
   sudo mv /tmp/picoclaw-custom /usr/local/bin/picoclaw
   ```
   *(Ensure the binary is executable: `sudo chmod +x /usr/local/bin/picoclaw`)*
3. **Restart the service to utilize the new brain**:
   ```bash
   sudo systemctl start picoclaw
   ```

### Why Cross-Compile?
Cross-compiling on your powerful Windows desktop ("Vulcan") saves the Raspberry Pi Zero 2W from the massive heat, CPU stress, and time-consumption of downloading the Go SDK and compiling code with 512MB of RAM.

## Troubleshooting

### "The new code changes aren't showing up!"
If you replaced the binary in `/usr/local/bin/picoclaw` but the bot is still running old logic, your `systemd` service is likely pointing to a different folder.

1. SSH into the Pi and run: `sudo systemctl status picoclaw`
2. Look specifically at the line that says:
   `├─23319 /usr/local/bin/picoclaw gateway` (Under the CGroup tree).
3. If the path listed there is something else (like `/home/tim/picoclaw`), you must `mv` your `picoclaw-custom` binary into *that* specific folder instead!
