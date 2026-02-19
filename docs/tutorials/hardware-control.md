# Hardware Control Tutorial

This tutorial guides you through controlling I2C and SPI hardware devices with PicoClaw.

## Prerequisites

- 25 minutes
- Linux-based system (Raspberry Pi, SBC, etc.)
- PicoClaw installed and configured
- Basic electronics knowledge
- I2C or SPI devices to experiment with

## Overview

PicoClaw can interact with hardware devices through:

| Interface | Use Case | Speed |
|-----------|----------|-------|
| I2C | Sensors, displays, slow devices | 100-400 kHz |
| SPI | Flash memory, fast displays | Up to MHz |

## Part 1: Setup

### Enable I2C (Raspberry Pi)

```bash
# Using raspi-config
sudo raspi-config
# Interface Options -> I2C -> Yes

# Or manually
sudo apt install i2c-tools
sudo modprobe i2c-dev
```

### Enable SPI (Raspberry Pi)

```bash
# Using raspi-config
sudo raspi-config
# Interface Options -> SPI -> Yes

# Or manually
sudo modprobe spidev
```

### Set Permissions

```bash
# Add user to required groups
sudo usermod -aG i2c $USER
sudo usermod -aG spi $USER

# Logout and login for changes to take effect
```

### Verify Setup

```bash
# Check I2C buses
ls /dev/i2c-*

# Check SPI devices
ls /dev/spidev*

# Install tools
sudo apt install i2c-tools spi-tools
```

## Part 2: I2C Device Detection

### Scan for Devices

```bash
# Scan I2C bus 1
i2cdetect -y 1
```

Output:

```
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- --
10: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
30: -- -- -- -- -- -- -- -- 38 39 3a 3b 3c 3d 3e 3f
40: 40 41 42 43 44 45 46 47 -- -- -- -- -- -- -- --
50: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
60: -- -- -- -- -- -- -- -- -- -- -- -- 6c -- -- --
70: -- -- -- -- -- -- -- 76
```

Numbers shown are device addresses in hex (e.g., 0x76, 0x38).

### Using PicoClaw

```
User: "Detect all I2C buses"
```

Agent response:

```
I'll detect the I2C buses on this system.

[Uses i2c tool with action "detect"]

Available I2C buses:
- i2c-1: Broadcom SMBus I2C adapter

User: "Scan bus 1 for devices"

[Uses i2c tool with action "scan", bus 1]

Found devices on bus 1:
- 0x38: Unknown device
- 0x40: Unknown device
- 0x76: Likely BME280/BMP280 sensor
```

## Part 3: Reading Sensor Data

### Example: BME280 Temperature Sensor

The BME280 is a popular temperature/humidity/pressure sensor at address 0x76 or 0x77.

```
User: "Read temperature from the BME280 at address 0x76"
```

Agent actions:

1. First, scan to verify device:

```json
{
  "action": "scan",
  "bus": 1
}
```

2. Read calibration data:

```json
{
  "action": "read",
  "bus": 1,
  "address": "0x76",
  "register": "0x88"
}
```

3. Read sensor registers:

```json
{
  "action": "read",
  "bus": 1,
  "address": "0x76",
  "register": "0xFA"
}
```

4. Calculate and report temperature.

### Example Conversation

```
User: What's the temperature reading?

Agent: I'll read the temperature from the BME280.

[Scans bus, finds device at 0x76]
[Reads register 0xFA for temperature data]

Current temperature: 23.5 degrees Celsius
Humidity: 65%
Pressure: 1013.25 hPa

User: Log this to a file

Agent: I'll append this reading to a log file.

[Writes to sensors.log]

Logged: 2024-01-15 10:30:00 - Temp: 23.5C, Humidity: 65%, Pressure: 1013.25hPa
```

## Part 4: SPI Communication

### List SPI Devices

```
User: "List available SPI devices"
```

Agent response:

```
[Uses spi tool with action "list"]

Available SPI devices:
- /dev/spidev0.0 (CS0)
- /dev/spidev0.1 (CS1)
```

### SPI Transfer

```
User: "Send bytes 0xAA 0xBB 0xCC to /dev/spidev0.0"
```

Agent action:

```json
{
  "action": "transfer",
  "device": "/dev/spidev0.0",
  "data": "0xAA 0xBB 0xCC"
}
```

### Example: Read SPI Flash

```
User: "Read the first 16 bytes from the SPI flash on /dev/spidev0.0"
```

Agent actions:

1. Send read command (0x03) + address (0x00 0x00 0x00):

```json
{
  "action": "transfer",
  "device": "/dev/spidev0.0",
  "data": "0x03 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00"
}
```

2. Parse the response and display hex dump.

## Part 5: Create a Hardware Agent

### Agent Configuration

Create a specialized hardware agent:

```bash
mkdir -p ~/.picoclaw/workspaces/hardware
nano ~/.picoclaw/workspaces/hardware/AGENT.md
```

```markdown
# Hardware Control Agent

## Role
You are a hardware controller for embedded and IoT devices.

## Capabilities
- I2C device detection and communication
- SPI device control
- Sensor reading and logging
- GPIO control (via exec tool)

## Known Devices
| Address | Device | Description |
|---------|--------|-------------|
| 0x76 | BME280 | Temp/Humidity/Pressure |
| 0x68 | DS1307/MPU6050 | RTC or IMU |
| 0x3C | SSD1306 | OLED Display |
| 0x27 | PCF8574 | I/O Expander |

## Behavior
1. Always detect devices before communication
2. Log all readings with timestamps
3. Handle errors gracefully
4. Report sensor values with units

## Safety
- Never write random values to unknown registers
- Verify device presence before operations
- Use appropriate delays between operations
```

### Add to Configuration

```json
{
  "agents": {
    "list": [
      {
        "id": "hardware",
        "workspace": "~/.picoclaw/workspaces/hardware",
        "model": "anthropic/claude-sonnet-4",
        "restrict_to_workspace": false
      }
    ]
  }
}
```

### Use the Hardware Agent

```bash
picoclaw agent -a hardware -m "Scan all I2C buses and identify devices"
```

## Part 6: Practical Projects

### Project 1: Weather Station

Create a monitoring system:

```
User: "Set up a weather monitoring system"

Agent:
1. I'll configure a weather monitoring system using the BME280.

[Scans for device]
[Reads current values]

I'll create a logging script:

[Creates weather_log.sh using write_file]

Now let's set up a cron job to log every 5 minutes:

[Creates cron job using cron tool]

Weather station configured:
- Sensor: BME280 at 0x76
- Log file: ~/workspace/weather.log
- Interval: Every 5 minutes
```

### Project 2: OLED Display

Control an SSD1306 display:

```
User: "Display 'Hello World' on the OLED"

Agent:
I'll write to the SSD1306 display at 0x3C.

[Sends initialization sequence]
[Sends text data]

"Hello World" should now be displayed on the OLED.
```

### Project 3: Relay Control

Control relays via I2C I/O expander:

```
User: "Turn on relay 1"

Agent:
I'll control the relay via the PCF8574 at 0x27.

[Reads current state]
[Writes new state with bit 0 set]

Relay 1 is now ON.
```

## Part 7: Sensor Database

Create a sensor reading database:

### Using Heartbeat

Create `HEARTBEAT.md`:

```markdown
# Sensor Monitoring

Every 1 minute:
1. Read BME280 (temperature, humidity, pressure)
2. Read light sensor (if connected)
3. Append to ~/workspace/sensors.csv in format:
   timestamp,temperature,humidity,pressure,light
4. Alert if any value is out of range
```

### Configure Heartbeat

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": "1m"
  }
}
```

### Run

```bash
picoclaw agent --heartbeat
```

### Query Data

```
User: "What's the average temperature over the last hour?"

Agent:
[Reads sensors.csv]
[Calculates average]

Average temperature over the last hour: 23.2 degrees Celsius
Min: 22.8, Max: 24.1
```

## Part 8: Troubleshooting

### Device Not Detected

```bash
# Check I2C is enabled
ls /dev/i2c-*

# Check kernel modules
lsmod | grep i2c

# Check permissions
groups $USER
```

### Permission Denied

```bash
# Check device permissions
ls -la /dev/i2c-1

# Temporary fix
sudo chmod 666 /dev/i2c-1

# Permanent fix
sudo usermod -aG i2c $USER
# Logout and login
```

### Communication Errors

1. Check wiring (SDA, SCL, VCC, GND)
2. Verify pull-up resistors (4.7K typical)
3. Check device address
4. Try different bus speed

### PicoClaw Debug Mode

```bash
picoclaw agent --debug -a hardware -m "Read sensor"
```

## Common I2C Devices

| Device | Address | Type | Notes |
|--------|---------|------|-------|
| BME280 | 0x76/0x77 | Sensor | Temp/Humidity/Pressure |
| BMP280 | 0x76/0x77 | Sensor | Temp/Pressure only |
| MPU6050 | 0x68 | IMU | Accelerometer/Gyro |
| HMC5883L | 0x1E | Magnetometer | Compass |
| SSD1306 | 0x3C/0x3D | Display | 128x64 OLED |
| PCF8574 | 0x20-0x27 | I/O | 8-bit expander |
| DS1307 | 0x68 | RTC | Real-time clock |
| ADS1115 | 0x48-0x4B | ADC | 16-bit, 4 channels |

## Safety Guidelines

1. **Always verify device presence** before writing
2. **Check voltage levels** (3.3V vs 5V)
3. **Use pull-up resistors** for I2C
4. **Handle interrupts carefully**
5. **Log operations** for debugging
6. **Test commands manually** before automation

## Next Steps

- [Hardware Tools Reference](../user-guide/tools/hardware.md)
- [Multi-Agent Tutorial](multi-agent-setup.md) - Create a dedicated hardware agent
- [Scheduled Tasks](scheduled-tasks.md) - Automate sensor logging
- [MaixCam Integration](../user-guide/channels/maixcam.md) - Vision capabilities

## Summary

You learned:
- How to enable I2C and SPI interfaces
- How to detect and scan for devices
- How to read sensor data
- How to create a hardware agent
- Practical project examples
- Troubleshooting techniques

You can now control hardware devices with PicoClaw!
