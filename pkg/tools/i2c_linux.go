package tools

import (
	"encoding/json"
	"fmt"
	"syscall"
	"unsafe"
)

// I2C ioctl constants from Linux kernel headers (<linux/i2c-dev.h>, <linux/i2c.h>)
const (
	i2cSlave = 0x0703 // Set slave address (fails if in use by driver)
	i2cFuncs = 0x0705 // Query adapter functionality bitmask
	i2cSmbus = 0x0720 // Perform SMBus transaction

	// I2C_FUNC capability bits
	i2cFuncSmbusQuick    = 0x00010000
	i2cFuncSmbusReadByte = 0x00020000

	// SMBus transaction types
	i2cSmbusRead  = 0
	i2cSmbusWrite = 1

	// SMBus protocol sizes
	i2cSmbusQuick = 0
	i2cSmbusByte  = 1
)

// i2cSmbusData matches the kernel union i2c_smbus_data (34 bytes max).
// For quick and byte transactions only the first byte is used (if at all).
type i2cSmbusData [34]byte

// i2cSmbusArgs matches the kernel struct i2c_smbus_ioctl_data.
type i2cSmbusArgs struct {
	readWrite uint8
	command   uint8
	size      uint32
	data      *i2cSmbusData
}

// smbusProbe performs a single SMBus probe at the given address.
// Uses SMBus Quick Write (safest) or falls back to SMBus Read Byte for
// EEPROM address ranges where quick write can corrupt AT24RF08 chips.
// This matches i2cdetect's MODE_AUTO behavior.
func smbusProbe(fd int, addr int, hasQuick bool) bool {
	// EEPROM ranges: use read byte (quick write can corrupt AT24RF08)
	useReadByte := (addr >= 0x30 && addr <= 0x37) || (addr >= 0x50 && addr <= 0x5F)

	if !useReadByte && hasQuick {
		// SMBus Quick Write: [START] [ADDR|W] [ACK/NACK] [STOP]
		// Safest probe — no data transferred
		args := i2cSmbusArgs{
			readWrite: i2cSmbusWrite,
			command:   0,
			size:      i2cSmbusQuick,
			data:      nil,
		}
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cSmbus, uintptr(unsafe.Pointer(&args)))
		return errno == 0
	}

	// SMBus Read Byte: [START] [ADDR|R] [ACK/NACK] [DATA] [STOP]
	var data i2cSmbusData
	args := i2cSmbusArgs{
		readWrite: i2cSmbusRead,
		command:   0,
		size:      i2cSmbusByte,
		data:      &data,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cSmbus, uintptr(unsafe.Pointer(&args)))
	return errno == 0
}

// scan probes valid 7-bit addresses on a bus for connected devices.
// Uses the same hybrid probe strategy as i2cdetect's MODE_AUTO:
// SMBus Quick Write for most addresses, SMBus Read Byte for EEPROM ranges.
func (t *I2CTool) scan(args map[string]interface{}) *ToolResult {
	bus, errResult := parseI2CBus(args)
	if errResult != nil {
		return errResult
	}

	devPath := fmt.Sprintf("/dev/i2c-%s", bus)
	fd, err := syscall.Open(devPath, syscall.O_RDWR, 0)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open %s: %v (check permissions and i2c-dev module)", devPath, err))
	}
	defer syscall.Close(fd)

	// Query adapter capabilities to determine available probe methods.
	// I2C_FUNCS writes an unsigned long, which is word-sized on Linux.
	var funcs uintptr
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cFuncs, uintptr(unsafe.Pointer(&funcs)))
	if errno != 0 {
		return ErrorResult(fmt.Sprintf("failed to query I2C adapter capabilities on %s: %v", devPath, errno))
	}

	hasQuick := funcs&i2cFuncSmbusQuick != 0
	hasReadByte := funcs&i2cFuncSmbusReadByte != 0

	if !hasQuick && !hasReadByte {
		return ErrorResult(fmt.Sprintf("I2C adapter %s supports neither SMBus Quick nor Read Byte — cannot probe safely", devPath))
	}

	type deviceEntry struct {
		Address string `json:"address"`
		Status  string `json:"status,omitempty"`
	}

	var found []deviceEntry
	// Scan 0x08-0x77, skipping I2C reserved addresses 0x00-0x07
	for addr := 0x08; addr <= 0x77; addr++ {
		// Set slave address — EBUSY means a kernel driver owns this address
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cSlave, uintptr(addr))
		if errno != 0 {
			if errno == syscall.EBUSY {
				found = append(found, deviceEntry{
					Address: fmt.Sprintf("0x%02x", addr),
					Status:  "busy (in use by kernel driver)",
				})
			}
			continue
		}

		if smbusProbe(fd, addr, hasQuick) {
			found = append(found, deviceEntry{
				Address: fmt.Sprintf("0x%02x", addr),
			})
		}
	}

	if len(found) == 0 {
		return SilentResult(fmt.Sprintf("No devices found on %s. Check wiring and pull-up resistors.", devPath))
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"bus":     devPath,
		"devices": found,
		"count":   len(found),
	}, "", "  ")
	return SilentResult(fmt.Sprintf("Scan of %s:\n%s", devPath, string(result)))
}

// readDevice reads bytes from an I2C device, optionally at a specific register
func (t *I2CTool) readDevice(args map[string]interface{}) *ToolResult {
	bus, errResult := parseI2CBus(args)
	if errResult != nil {
		return errResult
	}

	addr, errResult := parseI2CAddress(args)
	if errResult != nil {
		return errResult
	}

	length := 1
	if l, ok := args["length"].(float64); ok {
		length = int(l)
	}
	if length < 1 || length > 256 {
		return ErrorResult("length must be between 1 and 256")
	}

	devPath := fmt.Sprintf("/dev/i2c-%s", bus)
	fd, err := syscall.Open(devPath, syscall.O_RDWR, 0)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open %s: %v", devPath, err))
	}
	defer syscall.Close(fd)

	// Set slave address
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cSlave, uintptr(addr))
	if errno != 0 {
		return ErrorResult(fmt.Sprintf("failed to set I2C address 0x%02x: %v", addr, errno))
	}

	// If register is specified, write it first
	if regFloat, ok := args["register"].(float64); ok {
		reg := int(regFloat)
		if reg < 0 || reg > 255 {
			return ErrorResult("register must be between 0x00 and 0xFF")
		}
		_, err := syscall.Write(fd, []byte{byte(reg)})
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to write register 0x%02x: %v", reg, err))
		}
	}

	// Read data
	buf := make([]byte, length)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read from device 0x%02x: %v", addr, err))
	}

	// Format as hex bytes
	hexBytes := make([]string, n)
	intBytes := make([]int, n)
	for i := 0; i < n; i++ {
		hexBytes[i] = fmt.Sprintf("0x%02x", buf[i])
		intBytes[i] = int(buf[i])
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"bus":     devPath,
		"address": fmt.Sprintf("0x%02x", addr),
		"bytes":   intBytes,
		"hex":     hexBytes,
		"length":  n,
	}, "", "  ")
	return SilentResult(string(result))
}

// writeDevice writes bytes to an I2C device, optionally at a specific register
func (t *I2CTool) writeDevice(args map[string]interface{}) *ToolResult {
	confirm, _ := args["confirm"].(bool)
	if !confirm {
		return ErrorResult("write operations require confirm: true. Please confirm with the user before writing to I2C devices, as incorrect writes can misconfigure hardware.")
	}

	bus, errResult := parseI2CBus(args)
	if errResult != nil {
		return errResult
	}

	addr, errResult := parseI2CAddress(args)
	if errResult != nil {
		return errResult
	}

	dataRaw, ok := args["data"].([]interface{})
	if !ok || len(dataRaw) == 0 {
		return ErrorResult("data is required for write (array of byte values 0-255)")
	}
	if len(dataRaw) > 256 {
		return ErrorResult("data too long: maximum 256 bytes per I2C transaction")
	}

	data := make([]byte, 0, len(dataRaw)+1)

	// If register is specified, prepend it to the data
	if regFloat, ok := args["register"].(float64); ok {
		reg := int(regFloat)
		if reg < 0 || reg > 255 {
			return ErrorResult("register must be between 0x00 and 0xFF")
		}
		data = append(data, byte(reg))
	}

	for i, v := range dataRaw {
		f, ok := v.(float64)
		if !ok {
			return ErrorResult(fmt.Sprintf("data[%d] is not a valid byte value", i))
		}
		b := int(f)
		if b < 0 || b > 255 {
			return ErrorResult(fmt.Sprintf("data[%d] = %d is out of byte range (0-255)", i, b))
		}
		data = append(data, byte(b))
	}

	devPath := fmt.Sprintf("/dev/i2c-%s", bus)
	fd, err := syscall.Open(devPath, syscall.O_RDWR, 0)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open %s: %v", devPath, err))
	}
	defer syscall.Close(fd)

	// Set slave address
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), i2cSlave, uintptr(addr))
	if errno != 0 {
		return ErrorResult(fmt.Sprintf("failed to set I2C address 0x%02x: %v", addr, errno))
	}

	// Write data
	n, err := syscall.Write(fd, data)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to write to device 0x%02x: %v", addr, err))
	}

	return SilentResult(fmt.Sprintf("Wrote %d byte(s) to device 0x%02x on %s", n, addr, devPath))
}
