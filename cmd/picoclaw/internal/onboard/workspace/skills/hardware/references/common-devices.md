# Common I2C/SPI Device Reference

## I2C Devices

### AHT20 — Temperature & Humidity
- **Address:** 0x38
- **Init:** Write `[0xBE, 0x08, 0x00]` then wait 10ms
- **Measure:** Write `[0xAC, 0x33, 0x00]`, wait 80ms, read 6 bytes
- **Parse:** Status=byte[0], Humidity=(byte[1]<<12|byte[2]<<4|byte[3]>>4)/2^20*100, Temp=(byte[3]&0x0F<<16|byte[4]<<8|byte[5])/2^20*200-50
- **Notes:** No register addressing — write command bytes directly (omit `register` param)

### BME280 / BMP280 — Temperature, Humidity, Pressure
- **Address:** 0x76 or 0x77 (SDO pin selects)
- **Chip ID register:** 0xD0 → BMP280=0x58, BME280=0x60
- **Data registers:** 0xF7-0xFE (pressure, temperature, humidity)
- **Config:** Write 0xF2 (humidity oversampling), 0xF4 (temp/press oversampling + mode), 0xF5 (standby, filter)
- **Forced measurement:** Write `[0x25]` to register 0xF4, wait 40ms, read 8 bytes from 0xF7
- **Calibration:** Read 26 bytes from 0x88 and 7 bytes from 0xE1 for compensation formulas
- **Also available via SPI** (mode 0 or 3)

### SSD1306 — 128x64 OLED Display
- **Address:** 0x3C (or 0x3D if SA0 high)
- **Command prefix:** 0x00 (write to register 0x00)
- **Data prefix:** 0x40 (write to register 0x40)
- **Init sequence:** `[0xAE, 0xD5, 0x80, 0xA8, 0x3F, 0xD3, 0x00, 0x40, 0x8D, 0x14, 0x20, 0x00, 0xA1, 0xC8, 0xDA, 0x12, 0x81, 0xCF, 0xD9, 0xF1, 0xDB, 0x40, 0xA4, 0xA6, 0xAF]`
- **Display on:** 0xAF, **Display off:** 0xAE
- **Also available via SPI** (faster, recommended for animations)

### MPU6050 — 6-axis Accelerometer + Gyroscope
- **Address:** 0x68 (or 0x69 if AD0 high)
- **WHO_AM_I:** Register 0x75 → should return 0x68
- **Wake up:** Write `[0x00]` to register 0x6B (clear sleep bit)
- **Read accel:** 6 bytes from register 0x3B (XH,XL,YH,YL,ZH,ZL) — signed 16-bit, default ±2g
- **Read gyro:** 6 bytes from register 0x43 — signed 16-bit, default ±250°/s
- **Read temp:** 2 bytes from register 0x41 — Temp°C = value/340 + 36.53

### DS3231 — Real-Time Clock
- **Address:** 0x68
- **Read time:** 7 bytes from register 0x00 (seconds, minutes, hours, day, date, month, year) — BCD encoded
- **Set time:** Write 7 BCD bytes to register 0x00
- **Temperature:** 2 bytes from register 0x11 (signed, 0.25°C resolution)
- **Status:** Register 0x0F — bit 2 = busy, bit 0 = alarm 1 flag

### INA219 — Current & Power Monitor
- **Address:** 0x40-0x4F (A0,A1 pin selectable)
- **Config:** Register 0x00 — set voltage range, gain, ADC resolution
- **Shunt voltage:** Register 0x01 (signed 16-bit, LSB=10µV)
- **Bus voltage:** Register 0x02 (bits 15:3, LSB=4mV)
- **Power:** Register 0x03 (after calibration)
- **Current:** Register 0x04 (after calibration)
- **Calibration:** Register 0x05 — set based on shunt resistor value

### PCA9685 — 16-Channel PWM / Servo Controller
- **Address:** 0x40-0x7F (A0-A5 selectable, default 0x40)
- **Mode 1:** Register 0x00 — bit 4=sleep, bit 5=auto-increment
- **Set PWM freq:** Sleep → write prescale to 0xFE → wake. Prescale = round(25MHz / (4096 × freq)) - 1
- **Channel N on/off:** Registers 0x06+4*N to 0x09+4*N (ON_L, ON_H, OFF_L, OFF_H)
- **Servo 0°-180°:** ON=0, OFF=150-600 (at 50Hz). Typical: 0°=150, 90°=375, 180°=600

### AT24C256 — 256Kbit EEPROM
- **Address:** 0x50-0x57 (A0-A2 selectable)
- **Read:** Write 2-byte address (high, low), then read N bytes
- **Write:** Write 2-byte address + up to 64 bytes (page write), wait 5ms for write cycle
- **Page size:** 64 bytes. Writes that cross page boundary wrap around.

## SPI Devices

### MCP3008 — 8-Channel 10-bit ADC
- **Interface:** SPI mode 0, max 3.6 MHz @ 5V
- **Read channel N:** Send `[0x01, (0x80 | N<<4), 0x00]`, result in last 10 bits of bytes 1-2
- **Formula:** value = ((byte[1] & 0x03) << 8) | byte[2]
- **Voltage:** value × Vref / 1024

### W25Q128 — 128Mbit SPI Flash
- **Interface:** SPI mode 0 or 3, up to 104 MHz
- **Read ID:** Send `[0x9F, 0, 0, 0]` → manufacturer + device ID
- **Read data:** Send `[0x03, addr_high, addr_mid, addr_low]` + N zero bytes
- **Status:** Send `[0x05, 0]` → bit 0 = BUSY
