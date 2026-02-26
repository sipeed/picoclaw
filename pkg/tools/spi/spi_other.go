//go:build !linux

package spi

import "github.com/sipeed/picoclaw/pkg/tools/common"

// transfer is a stub for non-Linux platforms.
func (t *SPITool) transfer(args map[string]any) *common.ToolResult {
	return common.ErrorResult("SPI is only supported on Linux")
}

// readDevice is a stub for non-Linux platforms.
func (t *SPITool) readDevice(args map[string]any) *common.ToolResult {
	return common.ErrorResult("SPI is only supported on Linux")
}
