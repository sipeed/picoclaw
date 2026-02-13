//go:build !linux

package tools

// transfer is a stub for non-Linux platforms.
func (t *SPITool) transfer(args map[string]interface{}) *ToolResult {
	return ErrorResult("SPI is only supported on Linux")
}

// readDevice is a stub for non-Linux platforms.
func (t *SPITool) readDevice(args map[string]interface{}) *ToolResult {
	return ErrorResult("SPI is only supported on Linux")
}
