# Raspberry Pi Deployment

Raspberry Pi is the most popular single-board computer for running PicoClaw. This guide covers deployment on Raspberry Pi 4 and 5, with notes for older models.

## Prerequisites

- Raspberry Pi 4 or 5 (recommended) or Pi 3B+ (minimum)
- 2GB RAM minimum, 4GB+ recommended
- microSD card (16GB minimum, 32GB+ recommended)
- Raspberry Pi OS (64-bit recommended)

## Operating System Setup

### Download Raspberry Pi OS

```bash
# On your computer, download Raspberry Pi OS Lite (64-bit)
# https://www.raspberrypi.com/software/operating-systems/

# Or use Raspberry Pi Imager
# https://www.raspberrypi.com/software/
```

### Flash and Configure

Using Raspberry Pi Imager (recommended):

1. Insert microSD card
2. Open Raspberry Pi Imager
3. Select "Raspberry Pi OS Lite (64-bit)"
4. Click gear icon for advanced options:
   - Set hostname: `picoclaw.local`
   - Enable SSH
   - Set username/password
   - Configure WiFi (if using wireless)
5. Write to card

### First Boot

```bash
# SSH into your Pi
ssh picoclaw@picoclaw.local

# Update system
sudo apt update && sudo apt upgrade -y

# Set timezone
sudo raspi-config
# Navigate to Localisation Options > Timezone
```

## Installing PicoClaw

### Option 1: Precompiled Binary (Recommended)

```bash
# Download ARM64 binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-arm64

# Make executable
chmod +x picoclaw-linux-arm64

# Move to system path
sudo mv picoclaw-linux-arm64 /usr/local/bin/picoclaw

# Initialize configuration
picoclaw onboard
```

### Option 2: Build from Source

```bash
# Install Go 1.21+
wget https://go.dev/dl/go1.23.0.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw
make deps
make build

# Install
sudo make install
```

## Configuration

### Basic Configuration

```bash
# Edit config file
nano ~/.picoclaw/config.json
```

Example configuration:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "openrouter/anthropic/claude-sonnet-4",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

### Environment Variables

```bash
# Add to ~/.bashrc
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-your-key-here"
export PICOCLAW_AGENTS_DEFAULTS_MODEL="openrouter/anthropic/claude-sonnet-4"
```

## Running as a Service

### Create Dedicated User

```bash
sudo useradd -r -s /bin/false -d /var/lib/picoclaw picoclaw
sudo mkdir -p /var/lib/picoclaw/workspace
sudo chown -R picoclaw:picoclaw /var/lib/picoclaw
```

### Create Systemd Service

```bash
sudo tee /etc/systemd/system/picoclaw.service << 'EOF'
[Unit]
Description=PicoClaw AI Assistant
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
WorkingDirectory=/var/lib/picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Restart=on-failure
RestartSec=10

# Environment
Environment=PICOCLAW_CONFIG=/etc/picoclaw/config.json

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/picoclaw
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
```

### Configure for Service

```bash
# Move config to system location
sudo mkdir -p /etc/picoclaw
sudo cp ~/.picoclaw/config.json /etc/picoclaw/
sudo chown picoclaw:picoclaw /etc/picoclaw/config.json
sudo chmod 600 /etc/picoclaw/config.json

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable picoclaw
sudo systemctl start picoclaw

# Check status
sudo systemctl status picoclaw
```

## Hardware-Specific Features

### GPIO Access

Enable GPIO tools for home automation:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": []
    }
  }
}
```

Use with gpiod:

```bash
# Install gpiod tools
sudo apt install gpiod

# Example: Read GPIO pin
gpioget gpiochip0 17

# Example: Set GPIO pin
gpioset gpiochip0 18=1
```

### Temperature Monitoring

```bash
# CPU temperature
vcgencmd measure_temp

# Add to PicoClaw skills or use in exec commands
```

### Camera Module

```bash
# Enable camera
sudo raspi-config
# Navigate to Interface Options > Camera

# Test camera
libcamera-hello

# Capture image
libcamera-still -o image.jpg
```

### Sense HAT

```bash
# Install Sense HAT library
sudo apt install sense-hat

# Python script for sensor data
python3 -c "from sense_hat import SenseHat; s = SenseHat(); print(s.get_temperature())"
```

## Performance Optimization

### Memory Allocation

For Pi 4 with 4GB+ RAM:

```bash
# Check current memory split
vcgencmd get_mem gpu

# Reduce GPU memory (headless)
echo "gpu_mem=16" | sudo tee -a /boot/config.txt
```

### CPU Governor

```bash
# Check current governor
cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Set to performance (may increase power usage)
echo "performance" | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Or set to powersave (reduces performance)
echo "powersave" | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### Disable Unused Services

```bash
# Disable Bluetooth if not used
sudo systemctl disable bluetooth

# Disable audio if not used
sudo systemctl disable alsa-state

# Disable printing
sudo systemctl disable cups
```

### Overclocking (Pi 5)

```bash
# Edit /boot/firmware/config.txt (Pi 5)
# Add overclock settings (use with caution)
# over_voltage=6
# arm_freq=3000
```

## Network Configuration

### Static IP

```bash
# Using nmcli
sudo nmcli con mod "Wired connection 1" ipv4.addresses 192.168.1.100/24
sudo nmcli con mod "Wired connection 1" ipv4.gateway 192.168.1.1
sudo nmcli con mod "Wired connection 1" ipv4.dns "8.8.8.8,8.8.4.4"
sudo nmcli con mod "Wired connection 1" ipv4.method manual
sudo nmcli con up "Wired connection 1"
```

### WiFi

```bash
# Connect to WiFi
sudo nmcli dev wifi connect "SSID" password "PASSWORD"

# Or configure manually
sudo tee /etc/wpa_supplicant/wpa_supplicant.conf << 'EOF'
country=US
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="YourSSID"
    psk="YourPassword"
}
EOF
```

### Access Point Mode

Turn your Pi into a WiFi hotspot:

```bash
# Install required packages
sudo apt install hostapd dnsmasq

# Configure static IP for AP
sudo tee -a /etc/dhcpcd.conf << 'EOF'
interface wlan0
    static ip_address=192.168.4.1/24
    nohook wpa_supplicant
EOF

# Configure DNSMasq
sudo tee /etc/dnsmasq.conf << 'EOF'
interface=wlan0
dhcp-range=192.168.4.2,192.168.4.20,255.255.255.0,24h
domain=picoclaw.local
address=/picoclaw.local/192.168.4.1
EOF

# Configure hostapd
sudo tee /etc/hostapd/hostapd.conf << 'EOF'
interface=wlan0
driver=nl80211
ssid=PicoClaw
hw_mode=g
channel=7
wpa=2
wpa_passphrase=yourpassword
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP CCMP
EOF

# Point to config
sudo tee /etc/default/hostapd << 'EOF'
DAEMON_CONF="/etc/hostapd/hostapd.conf"
EOF

# Enable services
sudo systemctl unmask hostapd
sudo systemctl enable hostapd dnsmasq
sudo reboot
```

## Local LLM Integration

### Ollama Setup

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a small model
ollama pull llama3.2:1b

# Configure PicoClaw to use Ollama
# In config.json:
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

### Performance Notes

For running local LLMs on Raspberry Pi:
- Pi 5 with 8GB RAM can run small models (1-3B parameters)
- Use quantized models (q4, q5) for better performance
- Consider using external GPU accelerators for larger models

## Monitoring and Maintenance

### Health Check Script

```bash
#!/bin/bash
# /usr/local/bin/picoclaw-health.sh

# Check service
if ! systemctl is-active --quiet picoclaw; then
    echo "PicoClaw service not running"
    exit 1
fi

# Check health endpoint
if ! curl -sf http://localhost:18790/health > /dev/null; then
    echo "Health check failed"
    exit 1
fi

# Check temperature
TEMP=$(vcgencmd measure_temp | grep -oP '\d+')
if [ "$TEMP" -gt 80 ]; then
    echo "High temperature: ${TEMP}C"
fi

echo "All checks passed"
exit 0
```

### Log Monitoring

```bash
# View logs
sudo journalctl -u picoclaw -f

# Check for errors
sudo journalctl -u picoclaw -p err
```

### Backup

```bash
# Backup configuration and workspace
sudo tar czf picoclaw-backup-$(date +%Y%m%d).tar.gz \
    /etc/picoclaw \
    /var/lib/picoclaw
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u picoclaw -n 100

# Verify binary
which picoclaw
picoclaw version

# Test manually
sudo -u picoclaw /usr/local/bin/picoclaw gateway
```

### Memory Issues

```bash
# Check memory
free -h

# Add swap
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=100/CONF_SWAPSIZE=512/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

### Network Issues

```bash
# Check interface
ip addr

# Test connectivity
ping google.com

# Check DNS
nslookup api.openai.com
```

### Thermal Throttling

```bash
# Check temperature
vcgencmd measure_temp

# Check throttling
vcgencmd get_throttled

# If throttled, improve cooling
# - Add heatsink
# - Add fan
# - Ensure good airflow
```

## Resources

- [Raspberry Pi Documentation](https://www.raspberrypi.com/documentation/)
- [Raspberry Pi Forums](https://forums.raspberrypi.com/)
- [PicoClaw GitHub](https://github.com/sipeed/picoclaw)
