# Board Pinout & Pinmux Reference

## LicheeRV Nano (SG2002)

### I2C Buses

| Bus | Pins | Notes |
|-----|------|-------|
| I2C-1 | P18 (SCL), P21 (SDA) | **Shared with WiFi SDIO** — must stop WiFi first |
| I2C-3 | Available on header | Check device tree for pin assignment |
| I2C-5 | Software (BitBang) | Slower but no pin conflicts |

### SPI Buses

| Bus | Pins | Notes |
|-----|------|-------|
| SPI-2 | P18 (CS), P21 (MISO), P22 (MOSI), P23 (SCK) | **Shared with WiFi** — must stop WiFi first |
| SPI-4 | Software (BitBang) | Slower but no pin conflicts |

### Setup Steps for I2C-1

```bash
# 1. Stop WiFi (shares pins with I2C-1)
/etc/init.d/S30wifi stop

# 2. Configure pinmux for I2C-1
devmem 0x030010D0 b 0x2   # P18 → I2C1_SCL
devmem 0x030010DC b 0x2   # P21 → I2C1_SDA

# 3. Load i2c-dev module
modprobe i2c-dev

# 4. Verify
ls /dev/i2c-*
```

### Setup Steps for SPI-2

```bash
# 1. Stop WiFi (shares pins with SPI-2)
/etc/init.d/S30wifi stop

# 2. Configure pinmux for SPI-2
devmem 0x030010D0 b 0x1   # P18 → SPI2_CS
devmem 0x030010DC b 0x1   # P21 → SPI2_MISO
devmem 0x030010E0 b 0x1   # P22 → SPI2_MOSI
devmem 0x030010E4 b 0x1   # P23 → SPI2_SCK

# 3. Verify
ls /dev/spidev*
```

### Max Tested SPI Speed
- SPI-2 hardware: tested up to **93 MHz**
- `spidev_test` is pre-installed on the official image for loopback testing

---

## MaixCAM

### I2C Buses

| Bus | Pins | Notes |
|-----|------|-------|
| I2C-1 | Overlaps with WiFi | Not recommended |
| I2C-3 | Overlaps with WiFi | Not recommended |
| I2C-5 | A15 (SCL), A27 (SDA) | **Recommended** — software I2C, no conflicts |

### Setup Steps for I2C-5

```bash
# Configure pins using pinmap utility
# (MaixCAM uses a pinmap tool instead of devmem)
# Refer to: https://wiki.sipeed.com/hardware/en/maixcam/gpio.html

# Load i2c-dev
modprobe i2c-dev

# Verify
ls /dev/i2c-*
```

---

## MaixCAM2

### I2C Buses

| Bus | Pins | Notes |
|-----|------|-------|
| I2C-6 | A1 (SCL), A0 (SDA) | Available on header |
| I2C-7 | Available | Check device tree |

### Setup Steps

```bash
# Configure pinmap for I2C-6
# A1 → I2C6_SCL, A0 → I2C6_SDA
# Refer to MaixCAM2 documentation for pinmap commands

modprobe i2c-dev
ls /dev/i2c-*
```

---

## NanoKVM

Uses the same SG2002 SoC as LicheeRV Nano. GPIO and I2C access follows the same pinmux procedure. Refer to the LicheeRV Nano section above.

Check NanoKVM-specific pin headers for available I2C/SPI lines:
- https://wiki.sipeed.com/hardware/en/kvm/NanoKVM/introduction.html

---

## Common Issues

### devmem not found
The `devmem` utility may not be in the default image. Options:
- Use `busybox devmem` if busybox is installed
- Download devmem from the Sipeed package repository
- Cross-compile from source (single C file)

### Dynamic bus numbering
I2C adapter numbers can change between boots depending on driver load order. Always use `i2c detect` to find current bus assignments rather than hardcoding bus numbers.

### Permissions
`/dev/i2c-*` and `/dev/spidev*` typically require root access. Options:
- Run picoclaw as root
- Add user to `i2c` and `spi` groups
- Create udev rules: `SUBSYSTEM=="i2c-dev", MODE="0666"`
