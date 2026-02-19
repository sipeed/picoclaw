# LicheeRV Nano Deployment

LicheeRV Nano is an ultra-low-cost RISC-V single-board computer at just $9.9, making it the most affordable way to run PicoClaw. This guide covers setup and deployment on this remarkable little board.

## Hardware Specifications

| Specification | Value |
|---------------|-------|
| CPU | SG2002 (RISC-V C906 @ 1GHz) |
| RAM | 64MB DDR2 |
| Storage | MicroSD card slot |
| Network | 10/100 Ethernet |
| USB | 1x USB 2.0 |
| Video | HDMI output |
| Price | $9.9 USD |

## Where to Buy

- [Sipeed Official Store](https://www.aliexpress.com/store/911197660)
- [Seeed Studio](https://www.seeedstudio.com/)

## Operating System Setup

### Download Image

LicheeRV Nano uses Buildroot-based Linux images:

```bash
# Download from Sipeed
# https://github.com/sipeed/LicheeRV-Nano-Build/releases

# Choose the latest release
wget https://github.com/sipeed/LicheeRV-Nano-Build/releases/download/v0.2/licheervnano-0.2.img.gz

# Decompress
gunzip licheervnano-0.2.img.gz
```

### Flash to SD Card

```bash
# Using dd (Linux/macOS)
sudo dd if=licheervnano-0.2.img of=/dev/sdX bs=4M status=progress
sync

# Or use balenaEtcher (cross-platform)
# https://www.balena.io/etcher/
```

### First Boot

1. Insert the microSD card into LicheeRV Nano
2. Connect HDMI monitor (optional) or serial console
3. Connect Ethernet
4. Power on via USB-C

Default login:
- Username: `root`
- Password: `licheepi` (or no password)

### Serial Console Setup

If you don't have HDMI:

```bash
# Connect USB-TTL adapter to GPIO pins:
# - TX -> GPIO 12 (RX)
# - RX -> GPIO 13 (TX)
# - GND -> GND

# Use screen or minicom
screen /dev/ttyUSB0 115200
# or
minicom -D /dev/ttyUSB0
```

## Network Configuration

### DHCP (Default)

The system should automatically obtain an IP address via DHCP:

```bash
# Check IP address
ip addr show eth0

# Or check DHCP leases on your router
```

### Static IP

```bash
# Edit network configuration
cat > /etc/network/interfaces.d/eth0 << 'EOF'
auto eth0
iface eth0 inet static
    address 192.168.1.100
    netmask 255.255.255.0
    gateway 192.168.1.1
    dns-nameservers 8.8.8.8 8.8.4.4
EOF

# Restart networking
/etc/init.d/networking restart
```

### SSH Access

```bash
# Start SSH daemon (if not running)
/etc/init.d/ssh start

# Enable on boot
update-rc.d ssh defaults

# Connect from another machine
ssh root@192.168.1.100
```

## Installing PicoClaw

### Download RISC-V Binary

```bash
# Download RISC-V 64-bit binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-riscv64

# Make executable
chmod +x picoclaw-linux-riscv64

# Move to system path
mv picoclaw-linux-riscv64 /usr/local/bin/picoclaw
```

### Initialize Configuration

```bash
# Initialize
picoclaw onboard

# Edit configuration
vi ~/.picoclaw/config.json
```

### Minimal Configuration

Due to limited memory (64MB), use minimal settings:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "openrouter/anthropic/claude-sonnet-4",
      "max_tokens": 1024,
      "max_tool_iterations": 5,
      "temperature": 0.7
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    }
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  },
  "heartbeat": {
    "enabled": false
  },
  "devices": {
    "enabled": false
  }
}
```

## Memory Optimization

The 64MB RAM limitation requires careful optimization:

### Create Swap File

```bash
# Create 128MB swap file
dd if=/dev/zero of=/swapfile bs=1M count=128
mkswap /swapfile
swapon /swapfile

# Add to fstab for persistence
echo "/swapfile none swap sw 0 0" >> /etc/fstab
```

### Disable Unnecessary Services

```bash
# List running services
ps aux

# Stop and disable unnecessary services
/etc/init.d/crond stop
update-rc.d -f crond remove

# If not using GUI
/etc/init.d/lightdm stop
update-rc.d -f lightdm remove
```

### Use tmpfs

```bash
# Mount /tmp as tmpfs
mount -t tmpfs -o size=16M tmpfs /tmp

# Add to fstab
echo "tmpfs /tmp tmpfs defaults,size=16M 0 0" >> /etc/fstab
```

## Running PicoClaw

### Interactive Testing

```bash
# Test agent mode
picoclaw agent -m "Hello"

# Debug mode
picoclaw agent --debug -m "What is 2+2?"
```

### Gateway Mode

```bash
# Start gateway
picoclaw gateway

# In background
nohup picoclaw gateway > picoclaw.log 2>&1 &

# View logs
tail -f picoclaw.log
```

### As a Service

Create init script for Buildroot:

```bash
cat > /etc/init.d/picoclaw << 'EOF'
#!/bin/sh

DAEMON=/usr/local/bin/picoclaw
DAEMON_ARGS="gateway"
PIDFILE=/var/run/picoclaw.pid

case "$1" in
    start)
        echo "Starting PicoClaw..."
        start-stop-daemon -S -b -m -p $PIDFILE -x $DAEMON -- $DAEMON_ARGS
        ;;
    stop)
        echo "Stopping PicoClaw..."
        start-stop-daemon -K -p $PIDFILE -x $DAEMON
        rm -f $PIDFILE
        ;;
    restart)
        $0 stop
        sleep 1
        $0 start
        ;;
    status)
        if [ -f $PIDFILE ]; then
            echo "PicoClaw is running"
        else
            echo "PicoClaw is not running"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac

exit 0
EOF

chmod +x /etc/init.d/picoclaw
update-rc.d picoclaw defaults
```

## Use Cases

### Telegram Bot

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

### Home Automation Controller

Use with GPIO for home automation:

```bash
# Control GPIO (example)
echo 17 > /sys/class/gpio/export
echo out > /sys/class/gpio/gpio17/direction
echo 1 > /sys/class/gpio/gpio17/value
```

### Edge AI Integration

Process sensor data and make decisions:

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": ["rm -rf", "reboot"]
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 60
  }
}
```

## Performance Tips

### Optimize Network

```bash
# Use wired Ethernet for stability
# Keep connections minimal

# Reduce DNS lookups
echo "nameserver 8.8.8.8" > /etc/resolv.conf
```

### Reduce Logging

```bash
# Run without debug output
picoclaw gateway

# Limit log file size
picoclaw gateway 2>&1 | rotatelogs /var/log/picoclaw-%Y%m%d.log 86400 &
```

### Monitor Resources

```bash
# Check memory
free -m

# Check CPU
top

# Check temperature (if available)
cat /sys/class/thermal/thermal_zone*/temp
```

## Troubleshooting

### Out of Memory

```bash
# Check memory usage
free -m

# If OOM, increase swap
swapoff /swapfile
dd if=/dev/zero of=/swapfile bs=1M count=256
mkswap /swapfile
swapon /swapfile

# Reduce max_tokens in config
```

### Binary Not Executing

```bash
# Check architecture
uname -m
# Should show: riscv64

# Verify binary
file /usr/local/bin/picoclaw

# Check if executable
ls -la /usr/local/bin/picoclaw

# Check dependencies
ldd /usr/local/bin/picoclaw
```

### Network Issues

```bash
# Check interface
ip addr

# Test connectivity
ping -c 3 google.com

# Test DNS
nslookup api.openai.com

# Check routes
ip route
```

### Service Won't Start

```bash
# Check if already running
ps aux | grep picoclaw

# Check logs
cat picoclaw.log

# Test manually
picoclaw gateway
```

## Advanced Configuration

### Watchdog

Set up hardware watchdog for automatic reboot on failure:

```bash
# Load watchdog module
modprobe dw_wdt

# Configure watchdog
cat > /etc/watchdog.conf << 'EOF'
watchdog-device = /dev/watchdog
interval = 10
retry-timeout = 60
repair-max = 1
EOF

# Start watchdog daemon
watchdog
```

### Auto-Restart Script

```bash
#!/bin/sh
# /usr/local/bin/picoclaw-watchdog.sh

while true; do
    if ! pgrep -f "picoclaw gateway" > /dev/null; then
        echo "PicoClaw not running, restarting..."
        picoclaw gateway >> /var/log/picoclaw.log 2>&1 &
        sleep 10
    fi
    sleep 30
done
```

## Resources

- [LicheeRV Nano Wiki](https://wiki.sipeed.com/hardware/zh/lichee/RV_Nano/1_intro.html)
- [Sipeed GitHub](https://github.com/sipeed)
- [PicoClaw Releases](https://github.com/sipeed/picoclaw/releases)
- [RISC-V International](https://riscv.org/)
