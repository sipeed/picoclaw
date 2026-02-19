# USB Device Monitoring

Monitor and manage USB devices connected to your PicoClaw system.

## Overview

PicoClaw can monitor USB devices for:
- Device connection/disconnection events
- Serial port availability
- Hardware health status
- Automated device-specific actions

## Use Cases

- **IoT deployments**: Monitor sensors and actuators
- **Edge devices**: Track connected hardware
- **Automation**: Trigger actions on device events
- **Debugging**: Track device issues in the field

## Device Detection

### List Connected Devices

Use the CLI to check connected devices:

```bash
picoclaw status --devices
```

### USB Device Events

PicoClaw can log USB device changes. Enable in configuration:

```json
{
  "devices": {
    "usb_monitor": {
      "enabled": true,
      "log_events": true,
      "watch": [
        {"vendor_id": "10c4", "product_id": "ea60"},
        {"vendor_id": "0403", "product_id": "6001"}
      ]
    }
  }
}
```

## Serial Port Monitoring

### Configuration

```json
{
  "devices": {
    "serial": {
      "enabled": true,
      "ports": [
        {
          "path": "/dev/ttyUSB0",
          "name": "sensor_node",
          "baud_rate": 115200
        },
        {
          "path": "/dev/ttyACM0",
          "name": "arduino",
          "baud_rate": 9600
        }
      ]
    }
  }
}
```

### Serial Port Options

| Option | Type | Description |
|--------|------|-------------|
| `path` | string | Device path (e.g., `/dev/ttyUSB0`) |
| `name` | string | Friendly name for reference |
| `baud_rate` | int | Communication speed |
| `data_bits` | int | Data bits (default: 8) |
| `stop_bits` | int | Stop bits (default: 1) |
| `parity` | string | Parity: `none`, `even`, `odd` |
| `timeout` | int | Read timeout in seconds |

### Detecting Serial Ports

```bash
# List all serial ports
ls /dev/tty* | grep -E '(USB|ACM)'

# Using picocom to test
picocom -b 115200 /dev/ttyUSB0

# Using screen
screen /dev/ttyUSB0 115200
```

## Device Hotplug

### Linux udev Rules

Create udev rules for consistent device naming:

```bash
# /etc/udev/rules.d/99-picoclaw.rules

# USB serial device with specific vendor/product
SUBSYSTEM=="tty", ATTRS{idVendor}=="10c4", ATTRS{idProduct}=="ea60", SYMLINK+="picoclaw_sensor"

# Arduino boards
SUBSYSTEM=="tty", ATTRS{idVendor}=="2341", SYMLINK+="picoclaw_arduino"

# Set permissions
SUBSYSTEM=="tty", ATTRS{idVendor}=="10c4", MODE="0666"
```

Apply rules:

```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

### Device Permissions

Add your user to required groups:

```bash
# For serial ports
sudo usermod -aG dialout $USER

# For USB devices
sudo usermod -aG plugdev $USER

# Logout and login for changes to take effect
```

## Hardware Health Monitoring

### System Sensors

Monitor system health with lm-sensors:

```bash
# Install lm-sensors
sudo apt install lm-sensors

# Detect sensors
sudo sensors-detect

# View readings
sensors
```

### Agent Integration

Configure PicoClaw to use hardware data:

```
User: "What's the system temperature?"

Agent uses exec tool to run `sensors` and reports:
"Current CPU temperature: 45.2°C"
```

### I2C Device Monitoring

Monitor I2C-connected sensors:

```json
{
  "devices": {
    "i2c": {
      "enabled": true,
      "bus": 1,
      "devices": [
        {
          "address": "0x76",
          "name": "bme280",
          "type": "sensor",
          "interval": "60s"
        }
      ]
    }
  }
}
```

## USB Device Information

### Using lsusb

```bash
# List all USB devices
lsusb

# Verbose output for specific device
lsusb -v -d 10c4:ea60

# Show device tree
lsusb -t
```

### Using usb-devices

```bash
# Detailed device information
usb-devices
```

### Common USB Serial Chips

| Vendor ID | Product ID | Description |
|-----------|------------|-------------|
| `10c4` | `ea60` | CP210x (common USB-UART) |
| `0403` | `6001` | FTDI FT232 |
| `2341` | `0043` | Arduino Uno |
| `2341` | `0010` | Arduino Mega |
| `1a86` | `7523` | CH340 (cheap clones) |

## Troubleshooting

### Device Not Detected

1. Check physical connection
2. Verify driver is loaded:
   ```bash
   lsmod | grep usbserial
   ```
3. Check dmesg for errors:
   ```bash
   dmesg | grep -i usb
   ```

### Permission Denied

```bash
# Check device permissions
ls -la /dev/ttyUSB0

# Temporary fix
sudo chmod 666 /dev/ttyUSB0

# Permanent fix (add to dialout group)
sudo usermod -aG dialout $USER
```

### Device Disconnects Randomly

1. Check power supply (insufficient power can cause disconnects)
2. Check USB cable quality
3. Disable USB autosuspend:
   ```bash
   echo -1 | sudo tee /sys/module/usbcore/parameters/autosuspend
   ```

### Multiple Same Devices

Use udev rules to create persistent symlinks based on serial number:

```bash
# Get device serial
udevadm info -a -n /dev/ttyUSB0 | grep serial

# Create symlink by serial
SUBSYSTEM=="tty", ATTRS{serial}=="12345", SYMLINK+="device_a"
```

## Monitoring Scripts

### Continuous Device Monitor

Create a monitoring script:

```bash
#!/bin/bash
# /usr/local/bin/usb-monitor.sh

while true; do
    echo "=== $(date) ==="
    lsusb
    echo "Serial ports:"
    ls -la /dev/tty* 2>/dev/null | grep -E '(USB|ACM)'
    echo ""
    sleep 60
done
```

### Alert on Device Disconnection

```bash
#!/bin/bash
# Alert when device disappears

DEVICE="/dev/ttyUSB0"

if [ ! -e "$DEVICE" ]; then
    echo "ALERT: Device $DEVICE not found!"
    # Send notification
    # curl -X POST webhook_url -d "Device $DEVICE disconnected"
    exit 1
fi
```

## Integration with PicoClaw

### Reading Sensor Data

```
User: "Read the temperature from the BME280 sensor"

Agent:
1. Uses i2c tool to detect devices
2. Reads from address 0x76
3. Parses sensor data
4. Returns: "Temperature: 23.5°C, Humidity: 65%, Pressure: 1013 hPa"
```

### Controlling Serial Devices

```
User: "Send the reset command to the Arduino"

Agent:
1. Uses exec tool: stty -F /dev/ttyACM0 9600
2. Echoes command: echo "RESET" > /dev/ttyACM0
3. Reads response: cat /dev/ttyACM0
4. Returns: "Arduino reset successfully"
```

### Automated Monitoring

Use cron tools for periodic checks:

```json
{
  "cron": {
    "jobs": [
      {
        "name": "check_sensors",
        "schedule": "*/5 * * * *",
        "prompt": "Check if all I2C sensors are responding and report any issues"
      }
    ]
  }
}
```

## Best Practices

1. **Use udev rules** for consistent device naming
2. **Monitor device health** with periodic checks
3. **Log device events** for troubleshooting
4. **Set up alerts** for critical devices
5. **Document device mappings** in your configuration
6. **Use quality cables** to prevent connection issues
7. **Consider power requirements** for USB devices

## See Also

- [Hardware Tools](../user-guide/tools/hardware.md)
- [I2C and SPI](../user-guide/tools/hardware.md)
- [Cron and Scheduled Tasks](../user-guide/tools/cron.md)
- [MaixCam Integration](../user-guide/channels/maixcam.md)
