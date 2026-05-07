package tools

import hardwaretools "github.com/sipeed/picoclaw/pkg/tools/hardware"

type (
	I2CTool    = hardwaretools.I2CTool
	SerialTool = hardwaretools.SerialTool
	SPITool    = hardwaretools.SPITool
)

func NewI2CTool() *I2CTool {
	return hardwaretools.NewI2CTool()
}

func NewSPITool() *SPITool {
	return hardwaretools.NewSPITool()
}

func NewSerialTool() *SerialTool {
	return hardwaretools.NewSerialTool()
}
