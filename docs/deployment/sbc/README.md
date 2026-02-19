# Single-Board Computer Deployment

PicoClaw is optimized for resource-constrained environments, making it ideal for single-board computers (SBCs). This section covers deployment on popular low-cost hardware platforms.

## Why PicoClaw on SBCs?

- **Ultra-lightweight**: Single binary, minimal dependencies
- **Low resource usage**: Runs comfortably on 256MB RAM
- **Native ARM64 support**: No emulation required
- **Offline capable**: Works with local LLM providers (Ollama, vLLM)
- **Always-on**: Low power consumption for 24/7 operation

## Supported Hardware

| Hardware | Price | Architecture | RAM | Recommended For |
|----------|-------|--------------|-----|-----------------|
| [Raspberry Pi 4/5](rpi.md) | $35-80 | ARM64 | 2-8GB | Home automation, general use |
| [LicheeRV Nano](licheerv-nano.md) | $9.9 | RISC-V | 64MB | Ultra-low-cost edge AI |
| [MaixCAM](maixcam.md) | $50 | RISC-V | 256MB | AI camera applications |
| Orange Pi | $15-50 | ARM64 | 1-4GB | Budget alternative |
| Rock Pi | $35-75 | ARM64 | 1-4GB | More I/O options |
| NanoPi | $15-60 | ARM64 | 512MB-4GB | Compact form factor |

## Architecture Support

PicoClaw provides precompiled binaries for all major SBC architectures:

| Architecture | Binary Name | Devices |
|--------------|-------------|---------|
| ARM64 | `picoclaw-linux-arm64` | Raspberry Pi 4/5, Orange Pi, most SBCs |
| RISC-V 64 | `picoclaw-linux-riscv64` | LicheeRV Nano, MaixCAM |
| LoongArch | `picoclaw-linux-loong64` | Loongson-based devices |

## Quick Start

### Generic Installation

Most SBCs running Linux can use this approach:

```bash
# Download the appropriate binary for your architecture
# For ARM64 (most SBCs):
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-arm64

# For RISC-V:
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-riscv64

# Make executable
chmod +x picoclaw-linux-*

# Initialize
./picoclaw-linux-* onboard

# Configure
nano ~/.picoclaw/config.json

# Run
./picoclaw-linux-* gateway
```

### Check Your Architecture

```bash
uname -m
# aarch64 = ARM64 (use picoclaw-linux-arm64)
# riscv64 = RISC-V (use picoclaw-linux-riscv64)
# loongarch64 = LoongArch (use picoclaw-linux-loong64)
```

## Common Configuration

### Minimal Memory Configuration

For devices with limited RAM:

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter/anthropic/claude-sonnet-4",
      "max_tokens": 2048,
      "max_tool_iterations": 10
    }
  },
  "heartbeat": {
    "enabled": false
  },
  "devices": {
    "enabled": false
  }
}
```

### Local LLM Integration

Use with Ollama for offline operation:

```json
{
  "agents": {
    "defaults": {
      "model": "ollama/llama3.2:1b"
    }
  },
  "providers": {
    "ollama": {
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

### Power Management

For battery-powered or solar setups:

```bash
# Reduce CPU frequency
echo powersave | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Disable unnecessary services
sudo systemctl disable bluetooth
sudo systemctl disable avahi-daemon
```

## Performance Tuning

### Memory Optimization

```bash
# Create swap file (if needed)
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Add to /etc/fstab for persistence
/swapfile none swap sw 0 0
```

### CPU Optimization

```bash
# Check CPU info
cat /proc/cpuinfo

# Set performance governor
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### Storage Optimization

```bash
# Use tmpfs for temporary files
sudo mount -t tmpfs -o size=100M tmpfs /tmp

# Disable access time updates in /etc/fstab
# Add noatime to your root partition options
```

## Systemd Service Setup

For always-on operation, set up as a systemd service:

```bash
# Create service file
sudo tee /etc/systemd/system/picoclaw.service << 'EOF'
[Unit]
Description=PicoClaw AI Assistant
After=network.target

[Service]
Type=simple
User=picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable picoclaw
sudo systemctl start picoclaw
```

See the [Systemd Guide](../systemd.md) for detailed configuration.

## Network Configuration

### Static IP

Set up a static IP for reliable access:

```bash
# Using nmcli
nmcli con mod "Wired connection 1" ipv4.addresses 192.168.1.100/24
nmcli con mod "Wired connection 1" ipv4.gateway 192.168.1.1
nmcli con mod "Wired connection 1" ipv4.dns "8.8.8.8"
nmcli con mod "Wired connection 1" ipv4.method manual
nmcli con up "Wired connection 1"
```

### WiFi Setup

```bash
# Using nmcli
nmcli dev wifi connect "SSID" password "PASSWORD"
```

### Access Point Mode

Turn your SBC into a WiFi access point:

```bash
# Install hostapd
sudo apt install hostapd dnsmasq

# Configure access point
sudo tee /etc/hostapd/hostapd.conf << 'EOF'
interface=wlan0
driver=nl80211
ssid=PicoClaw
hw_mode=g
channel=7
wpa=2
wpa_passphrase=yourpassword
wpa_key_mgmt=WPA-PSK
EOF

# Enable
sudo systemctl enable hostapd
sudo systemctl start hostapd
```

## Monitoring

### Resource Monitoring

```bash
# Install monitoring tools
sudo apt install htop iotop

# Monitor resources
htop

# Check temperature (Raspberry Pi)
vcgencmd measure_temp

# Check memory
free -h
```

### Health Monitoring Script

```bash
#!/bin/bash
# health-check.sh

# Check if PicoClaw is running
if ! pgrep -f "picoclaw gateway" > /dev/null; then
    echo "PicoClaw not running, restarting..."
    systemctl restart picoclaw
fi

# Check memory usage
MEM_USED=$(free | grep Mem | awk '{print int($3/$2 * 100)}')
if [ "$MEM_USED" -gt 90 ]; then
    echo "Memory usage high: ${MEM_USED}%"
fi

# Check disk space
DISK_USED=$(df -h / | tail -1 | awk '{print int($5)}')
if [ "$DISK_USED" -gt 90 ]; then
    echo "Disk usage high: ${DISK_USED}%"
fi
```

## Troubleshooting

### Out of Memory

```bash
# Check memory
free -h

# Add swap
sudo fallocate -l 512M /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Reduce memory in config
# Set lower max_tokens value
```

### Slow Performance

```bash
# Check CPU governor
cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Set performance mode
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Check for thermal throttling
cat /sys/class/thermal/thermal_zone*/temp
```

### Network Issues

```bash
# Check network interface
ip addr

# Test connectivity
ping google.com

# Check DNS
nslookup api.openai.com
```

### Binary Not Executing

```bash
# Check architecture
uname -m

# Verify binary architecture
file picoclaw-linux-*

# Check permissions
ls -la picoclaw-linux-*

# Check for missing libraries
ldd picoclaw-linux-*
```

## Hardware-Specific Guides

- [Raspberry Pi 4/5](rpi.md) - Full guide for the most popular SBC
- [LicheeRV Nano](licheerv-nano.md) - Ultra-low-cost RISC-V board
- [MaixCAM](maixcam.md) - AI camera with built-in PicoClaw support

## Community Resources

- [PicoClaw GitHub](https://github.com/sipeed/picoclaw)
- [Sipeed Wiki](https://wiki.sipeed.com/)
- [Raspberry Pi Forums](https://forums.raspberrypi.com/)
