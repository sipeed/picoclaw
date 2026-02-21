# Hardware Tools

Interact with hardware devices via I2C and SPI buses. (Linux only)

## Prerequisites

- Linux operating system
- Appropriate permissions (usually root or i2c/spi group)
- Enabled hardware interfaces

## I2C Tool

### i2c

Interact with I2C bus devices.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `action` | string | Yes | Action: `detect`, `scan`, `read`, `write` |
| `bus` | int | Conditional | Bus number |
| `address` | string | Conditional | Device address (hex) |
| `register` | string | Conditional | Register address (hex) |
| `data` | string | Conditional | Data to write (hex) |

**Actions:**

### detect

List available I2C buses:

```json
{
  "action": "detect"
}
```

### scan

Scan bus for devices:

```json
{
  "action": "scan",
  "bus": 1
}
```

### read

Read from device register:

```json
{
  "action": "read",
  "bus": 1,
  "address": "0x76",
  "register": "0x00"
}
```

### write

Write to device register:

```json
{
  "action": "write",
  "bus": 1,
  "address": "0x76",
  "register": "0x00",
  "data": "0xAA"
}
```

## SPI Tool

### spi

Interact with SPI bus devices.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `action` | string | Yes | Action: `list`, `transfer`, `read` |
| `device` | string | Conditional | SPI device path |
| `data` | string | Conditional | Data to transfer (hex) |

**Actions:**

### list

List available SPI devices:

```json
{
  "action": "list"
}
```

### transfer

Send and receive data:

```json
{
  "action": "transfer",
  "device": "/dev/spidev0.0",
  "data": "0xAA 0xBB 0xCC"
}
```

### read

Read data from device:

```json
{
  "action": "read",
  "device": "/dev/spidev0.0"
}
```

## Setup (Linux)

### Enable I2C

```bash
# Raspberry Pi
sudo raspi-config
# Interface Options -> I2C -> Enable

# Add user to i2c group
sudo usermod -aG i2c $USER
```

### Enable SPI

```bash
# Raspberry Pi
sudo raspi-config
# Interface Options -> SPI -> Enable

# Add user to spi group
sudo usermod -aG spi $USER
```

### Install Tools

```bash
# Debian/Ubuntu
sudo apt install i2c-tools spi-tools
```

## Common Devices

### I2C Sensors

| Device | Address | Description |
|--------|---------|-------------|
| BME280 | 0x76, 0x77 | Temperature, humidity, pressure |
| MPU6050 | 0x68 | Accelerometer, gyroscope |
| SSD1306 | 0x3C | OLED display |

### SPI Devices

| Device | Description |
|--------|-------------|
| W25Q32 | Flash memory |
| MFRC522 | RFID reader |
| ILI9341 | TFT display |

## Example: Read BME280

```
User: "Read temperature from the BME280 sensor"

Agent uses i2c tool:
{
  "action": "scan",
  "bus": 1
}

Finds device at 0x76, then:

{
  "action": "read",
  "bus": 1,
  "address": "0x76",
  "register": "0xFA"
}

Agent: "Temperature reading: 23.5°C"
```

## See Also

- [Tools Overview](README.md)
- [MaixCam Channel](../channels/maixcam.md)
