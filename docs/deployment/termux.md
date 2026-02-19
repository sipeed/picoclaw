# Android/Termux Deployment

PicoClaw can run on Android devices using Termux, allowing you to repurpose old phones or tablets as AI assistants. This is an excellent way to give new life to unused Android devices.

## Prerequisites

- Android 7.0 or later
- Termux app (from F-Droid recommended)
- ARM64 Android device (most devices from 2015+)

## Installing Termux

### Option 1: F-Droid (Recommended)

The F-Droid version is more up-to-date and doesn't have Google Play restrictions:

1. Install [F-Droid](https://f-droid.org/)
2. Search for "Termux" in F-Droid
3. Install Termux, Termux:API (optional)

### Option 2: GitHub Releases

Download directly from [Termux GitHub releases](https://github.com/termux/termux-app/releases):

1. Download the latest APK
2. Enable "Install from unknown sources" in Android settings
3. Install the APK

### Important Note

**Do not install Termux from Google Play Store.** That version is outdated and has compatibility issues. Use F-Droid or GitHub releases instead.

## Initial Setup

### Step 1: Update Termux Packages

Open Termux and run:

```bash
# Update package lists
pkg update && pkg upgrade -y

# Install essential packages
pkg install wget curl git proot -y
```

### Step 2: Download PicoClaw

```bash
# Navigate to home directory
cd ~

# Download the ARM64 binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-arm64

# Make executable
chmod +x picoclaw-linux-arm64
```

### Step 3: Initialize with proot

Termux has a different filesystem layout than standard Linux. Use `proot` to run PicoClaw:

```bash
# Install proot if not already installed
pkg install proot -y

# Initialize PicoClaw using termux-chroot
termux-chroot ./picoclaw-linux-arm64 onboard
```

### Step 4: Configure PicoClaw

```bash
# Edit configuration
termux-chroot nano ~/.picoclaw/config.json
```

Add your API keys:

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter/anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    }
  }
}
```

## Running PicoClaw

### Agent Mode (One-Shot Queries)

```bash
# Run a single query
termux-chroot ./picoclaw-linux-arm64 agent -m "What is the weather today?"

# Interactive mode
termux-chroot ./picoclaw-linux-arm64 agent

# Debug mode
termux-chroot ./picoclaw-linux-arm64 agent --debug -m "Hello"
```

### Gateway Mode (Always-On Bot)

```bash
# Start gateway
termux-chroot ./picoclaw-linux-arm64 gateway
```

## Creating a Shortcut Script

Create a helper script for easier execution:

```bash
# Create script
cat > $PREFIX/bin/picoclaw << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
# PicoClaw wrapper for Termux

BINARY="$HOME/picoclaw-linux-arm64"

if [ ! -f "$BINARY" ]; then
    echo "PicoClaw binary not found at $BINARY"
    exit 1
fi

termux-chroot "$BINARY" "$@"
EOF

# Make executable
chmod +x $PREFIX/bin/picoclaw
```

Now you can run PicoClaw directly:

```bash
picoclaw agent -m "Hello"
picoclaw gateway
```

## Keeping PicoClaw Running

### Using tmux

Run PicoClaw in a persistent session:

```bash
# Install tmux
pkg install tmux -y

# Create new session
tmux new -s picoclaw

# Start PicoClaw
picoclaw gateway

# Detach: Press Ctrl+B then D

# Reattach later
tmux attach -t picoclaw
```

### Using nohup

Run in background:

```bash
# Start in background
nohup picoclaw gateway > picoclaw.log 2>&1 &

# View logs
tail -f picoclaw.log

# Find process ID
pgrep -f picoclaw

# Stop
pkill -f picoclaw
```

### Auto-Start on Boot

Create a boot script using Termux:Boot (requires Termux:API):

```bash
# Install Termux:API and Termux:Boot from F-Droid

# Create boot directory
mkdir -p ~/.termux/boot

# Create boot script
cat > ~/.termux/boot/picoclaw.sh << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
# Auto-start PicoClaw on boot

# Wait for network
sleep 30

# Start in tmux session
tmux new -d -s picoclaw "picoclaw gateway"
EOF

# Make executable
chmod +x ~/.termux/boot/picoclaw.sh
```

## Battery Optimization

### Disable Battery Optimization

To prevent Android from killing Termux in the background:

1. Go to Settings > Battery > Battery Optimization
2. Find Termux and select "Don't optimize"
3. Also disable battery optimization for Termux:API and Termux:Boot if installed

### Acquire Wakelock

Prevent the device from sleeping:

```bash
# Install Termux:API first
pkg install termux-api -y

# Acquire wakelock
termux-wake-lock

# Release wakelock when done
termux-wake-unlock
```

### Notification (Keep Alive)

Termux shows a persistent notification when running:

```bash
# The notification is shown automatically
# To start foreground service:
termux-notification --ongoing --title "PicoClaw" --content "Running..."
```

## Storage Access

### Access Shared Storage

To access files in your Android shared storage:

```bash
# Request storage permission
termux-setup-storage

# This creates a symlink at ~/storage
# Access Downloads: ~/storage/downloads
# Access DCIM: ~/storage/dcim
# Access Documents: ~/storage/documents
```

### Use Shared Storage for Workspace

Configure PicoClaw to use shared storage:

```json
{
  "agents": {
    "defaults": {
      "workspace": "/data/data/com.termux/files/home/storage/documents/picoclaw"
    }
  }
}
```

## Notifications

Send notifications when PicoClaw completes tasks:

```bash
# Simple notification
termux-notification --title "PicoClaw" --content "Task completed"

# With sound
termux-notification --title "PicoClaw" --content "Task completed" --sound

# With action button
termux-notification --title "PicoClaw" --content "New message" --button1 "View" --button1-action "termux-open-url https://example.com"
```

## Hardware Integration

### Camera Access

Use Termux:API to access the camera:

```bash
# Take a photo
termux-camera-photo ~/photo.jpg

# Use in PicoClaw skills or tools
```

### Sensors

Access device sensors:

```bash
# Get sensor list
termux-sensor -l

# Get sensor data
termux-sensor -s "accelerometer"
```

### Location

Get GPS location:

```bash
# Get current location
termux-location
```

### Text-to-Speech

Have PicoClaw speak responses:

```bash
# Install TTS
pkg install espeak -y

# Speak text
espeak "Hello from PicoClaw"
```

Or use Termux:API TTS:

```bash
termux-tts-speak "Hello from PicoClaw"
```

## Performance Optimization

### Memory Management

Older Android devices may have limited RAM:

```bash
# Check available memory
free -h

# Kill unnecessary processes
pkill -f zygote

# Reduce memory usage in config
# Set lower max_tokens
```

### CPU Throttling

Prevent thermal throttling:

1. Remove phone case for better cooling
2. Reduce screen brightness
3. Disable unnecessary apps

### Network Optimization

For stable network on mobile data:

```bash
# Keep connection alive with periodic ping
while true; do ping -c 1 google.com; sleep 60; done &
```

## Troubleshooting

### Binary Won't Execute

```bash
# Check architecture
uname -m
# Should show aarch64

# Verify binary is for ARM64
file picoclaw-linux-arm64

# Check permissions
ls -la picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
```

### proot Errors

```bash
# Update proot
pkg upgrade proot

# Try alternative proot command
proot ./picoclaw-linux-arm64 onboard
```

### Network Issues

```bash
# Check network
ping google.com

# Check DNS
nslookup api.openai.com

# Test with curl
curl -I https://api.openai.com
```

### Permission Denied

```bash
# Reset Termux permissions in Android settings
# Settings > Apps > Termux > Permissions

# Re-run setup
termux-setup-storage
```

### App Killed in Background

1. Disable battery optimization for Termux
2. Use `termux-wake-lock`
3. Keep the notification visible
4. Use tmux for persistent sessions

### Clear Everything and Start Fresh

```bash
# Remove PicoClaw
rm -rf ~/.picoclaw
rm picoclaw-linux-arm64

# Re-download and setup
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
termux-chroot ./picoclaw-linux-arm64 onboard
```

## Use Cases

### Home Automation Hub

Run PicoClaw as a Telegram bot to control smart home devices:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN"
    }
  }
}
```

### Always-On Assistant

Keep PicoClaw running 24/7 on an old phone:

1. Enable wake lock
2. Disable battery optimization
3. Use tmux for persistence
4. Connect to charger

### Mobile AI Companion

Use PicoClaw with voice input/output:

```bash
# Create voice assistant script
cat > ~/voice-assist.sh << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
echo "Listening..."
RECORDING=$(mktemp /tmp/recording.XXXXXX.wav)
termux-speech-to-text > /tmp/text.txt
TEXT=$(cat /tmp/text.txt)
echo "You said: $TEXT"
RESPONSE=$(picoclaw agent -m "$TEXT")
echo "Response: $RESPONSE"
termux-tts-speak "$RESPONSE"
EOF
chmod +x ~/voice-assist.sh
```

## Resources

- [Termux Wiki](https://wiki.termux.com/)
- [Termux GitHub](https://github.com/termux)
- [PicoClaw Releases](https://github.com/sipeed/picoclaw/releases)
