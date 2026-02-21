# MaixCAM Deployment

MaixCAM is an AI camera platform from Sipeed that comes with PicoClaw pre-integrated. This guide covers deployment and configuration on MaixCAM for AI-powered camera applications.

## Hardware Specifications

| Specification | Value |
|---------------|-------|
| CPU | AX620E (RISC-V C906 @ 1GHz) |
| NPU | 14.4 TOPS (for AI inference) |
| RAM | 256MB DDR3 |
| Storage | MicroSD card slot |
| Camera | GC2145, support for other modules |
| Display | 2.4" 640x480 touchscreen |
| Network | WiFi 6, Ethernet (via dock) |
| USB | 1x USB 2.0 Type-C |
| Price | ~$50 USD |

## Where to Buy

- [Sipeed Official Store](https://www.aliexpress.com/store/911197660)
- [Seeed Studio](https://www.seeedstudio.com/)
- [MaixCAM Product Page](https://wiki.sipeed.com/maixcam)

## Pre-installed PicoClaw

MaixCAM comes with PicoClaw pre-installed and configured. The built-in MaixCAM channel allows direct interaction through the device.

### Quick Start

1. Power on MaixCAM
2. Connect to WiFi (via touchscreen or web interface)
3. PicoClaw starts automatically
4. Interact via the touchscreen or connect external channels

## Initial Setup

### First Boot

```bash
# Default hostname: maixcam.local
# Default credentials: root / root (or no password)

# SSH access
ssh root@maixcam.local
```

### Network Configuration

#### WiFi Setup (Touchscreen)

1. Tap Settings icon on the home screen
2. Select WiFi
3. Choose your network and enter password

#### WiFi Setup (Command Line)

```bash
# Using nmcli
nmcli dev wifi connect "SSID" password "PASSWORD"

# Or using wpa_supplicant
wpa_passphrase "SSID" "PASSWORD" >> /etc/wpa_supplicant.conf
wpa_supplicant -B -i wlan0 -c /etc/wpa_supplicant.conf
udhcpc -i wlan0
```

### Static IP

```bash
# Configure static IP
cat > /etc/network/interfaces.d/eth0 << 'EOF'
auto eth0
iface eth0 inet static
    address 192.168.1.100
    netmask 255.255.255.0
    gateway 192.168.1.1
    dns-nameservers 8.8.8.8
EOF

# Restart networking
/etc/init.d/networking restart
```

## PicoClaw Configuration

### Configuration File

```bash
# Edit configuration
vi ~/.picoclaw/config.json
# or
vi /etc/picoclaw/config.json
```

### MaixCAM Channel

The MaixCAM channel provides native integration:

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

### Full Configuration Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "openrouter/anthropic/claude-sonnet-4",
      "max_tokens": 2048,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    },
    "zhipu": {
      "api_key": "your-zhipu-key"
    }
  },
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790
    },
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  },
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

## Camera Integration

### Taking Photos

PicoClaw can use the camera through the exec tool:

```bash
# Take a photo
libcamera-still -o /tmp/photo.jpg

# Or using the built-in camera tool
maixcam-capture /tmp/photo.jpg
```

### Camera Skills

Create a skill for camera operations:

```markdown
# Camera Skill

You can use the camera to take photos and analyze them.

## Commands

- Take a photo: `libcamera-still -o /tmp/photo.jpg`
- List photos: `ls -la /tmp/*.jpg`
- Delete photos: `rm /tmp/photo.jpg`

## Example

User: "Take a photo and describe what you see"
Action: Take photo using exec tool, then analyze
```

### Vision Analysis

To use vision capabilities:

1. Set up an image model in config:

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter/anthropic/claude-sonnet-4",
      "image_model": "openrouter/anthropic/claude-sonnet-4"
    }
  }
}
```

2. Take a photo and ask PicoClaw to analyze it

## AI Features

### NPU-Accelerated Inference

MaixCAM has a built-in NPU for AI inference. While PicoClaw uses cloud LLMs by default, you can:

1. Run local vision models on the NPU
2. Use local object detection
3. Process images before sending to cloud

### Local Vision Model

```bash
# Example: Run object detection
# (Specific commands depend on installed packages)
python3 /opt/maixcam/detection.py --input /tmp/photo.jpg
```

### Speech Recognition

MaixCAM supports voice input:

```bash
# Audio recording
arecord -d 5 -f cd /tmp/recording.wav

# Speech-to-text (if configured)
maixcam-asr /tmp/recording.wav
```

## Running PicoClaw

### Starting Manually

```bash
# Start gateway
picoclaw gateway

# With debug output
picoclaw gateway --debug

# Agent mode (one-shot)
picoclaw agent -m "What do you see?"
```

### As a Service

MaixCAM typically runs PicoClaw as a systemd service:

```bash
# Check status
systemctl status picoclaw

# Start/stop/restart
sudo systemctl start picoclaw
sudo systemctl stop picoclaw
sudo systemctl restart picoclaw

# View logs
journalctl -u picoclaw -f
```

### Auto-start on Boot

```bash
# Enable auto-start
sudo systemctl enable picoclaw

# Disable auto-start
sudo systemctl disable picoclaw
```

## Use Cases

### Security Camera

Configure PicoClaw for security monitoring:

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 5
  }
}
```

Create a heartbeat task that:
1. Captures an image
2. Analyzes for anomalies
3. Sends alerts via Telegram

### Smart Doorbell

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN"
    },
    "maixcam": {
      "enabled": true
    }
  }
}
```

Workflow:
1. Doorbell button triggers camera
2. PicoClaw captures and analyzes image
3. Sends notification with photo to Telegram

### Object Recognition Assistant

Use with the camera for object identification:

```
User: "What is on the table?"
PicoClaw: [Takes photo] "I can see a laptop, coffee mug, and notebook on the table."
```

### Plant Monitor

Monitor plant health:

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 60
  }
}
```

## Performance Optimization

### Memory Management

```bash
# Check memory
free -m

# The 256MB RAM should be sufficient for basic operation
# If needed, reduce max_tokens in config
```

### CPU Frequency

```bash
# Check current frequency
cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq

# Set performance governor
echo performance > /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
```

### Storage

```bash
# Check disk space
df -h

# Clean up old sessions
rm -rf ~/.picoclaw/workspace/sessions/*.json.old
```

## Web Interface

MaixCAM includes a web interface for configuration:

```
http://maixcam.local/
```

Features:
- Live camera view
- PicoClaw chat interface
- Settings configuration
- System monitoring

## Troubleshooting

### PicoClaw Not Starting

```bash
# Check if running
ps aux | grep picoclaw

# Check logs
journalctl -u picoclaw -n 100

# Test manually
picoclaw gateway --debug
```

### Camera Issues

```bash
# Check camera device
ls -la /dev/video*

# Test camera
libcamera-hello

# Check for errors
dmesg | grep -i camera
```

### WiFi Connection Issues

```bash
# Check WiFi status
nmcli dev status

# Reconnect
nmcli dev wifi connect "SSID" password "PASSWORD"

# Check signal strength
iwconfig wlan0
```

### Touchscreen Not Responding

```bash
# Check input devices
cat /proc/bus/input/devices

# Calibrate touchscreen
ts_calibrate
```

### API Errors

```bash
# Test API connectivity
curl -I https://api.openai.com

# Check DNS
nslookup api.openai.com

# Verify API key
echo $PICOCLAW_PROVIDERS_OPENROUTER_API_KEY
```

## Updates

### Update PicoClaw

```bash
# Download latest version
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-riscv64
chmod +x picoclaw-linux-riscv64
sudo mv picoclaw-linux-riscv64 /usr/local/bin/picoclaw

# Restart service
sudo systemctl restart picoclaw
```

### System Update

```bash
# Update packages
apt update && apt upgrade -y

# Update firmware (if available)
# Check Sipeed wiki for instructions
```

## Resources

- [MaixCAM Wiki](https://wiki.sipeed.com/maixcam)
- [Sipeed GitHub](https://github.com/sipeed)
- [MaixCAM Examples](https://github.com/sipeed/MaixCAM)
- [PicoClaw GitHub](https://github.com/sipeed/picoclaw)
- [MaixPy Documentation](https://wiki.sipeed.com/maixpy/)
