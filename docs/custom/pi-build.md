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

Run this single command in PowerShell from the repository root:
```powershell
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="arm"; $env:GOARM="7"; go build -o picoclaw-custom ./cmd/picoclaw
```

This instructs the Go compiler to generate a standalone Linux executable tailored for the Pi's architecture.

## Step B: Transfer to Pi

Use `scp` (Secure Copy Protocol), which transfers files over an encrypted SSH connection.

```powershell
scp picoclaw-custom tim@picoclaw.local:/tmp/
```

*This copies your newly compiled `picoclaw-custom` file from Windows up to the `/tmp/` folder on your Pi.*

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
