package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemTool struct{}

func NewSystemTool() *SystemTool {
	return &SystemTool{}
}

func (t *SystemTool) Name() string {
	return "system_stats"
}

func (t *SystemTool) Description() string {
	return "Get current system statistics including CPU, Memory, Disk usage, and Host info."
}

func (t *SystemTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *SystemTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	var sb strings.Builder

	// Host Info
	hInfo, err := host.Info()
	if err == nil {
		sb.WriteString(fmt.Sprintf("Host: %s (%s %s)\n", hInfo.Hostname, hInfo.Platform, hInfo.PlatformVersion))
		sb.WriteString(fmt.Sprintf("Uptime: %s\n", time.Duration(hInfo.Uptime)*time.Second))
	}

	// CPU
	percent, err := cpu.Percent(time.Second, false)
	if err == nil && len(percent) > 0 {
		sb.WriteString(fmt.Sprintf("CPU Usage: %.2f%%\n", percent[0]))
	}

	// Memory
	vMem, err := mem.VirtualMemory()
	if err == nil {
		usedGB := float64(vMem.Used) / 1024 / 1024 / 1024
		totalGB := float64(vMem.Total) / 1024 / 1024 / 1024
		sb.WriteString(fmt.Sprintf("Memory: %.2f GB / %.2f GB (%.2f%%)\n", usedGB, totalGB, vMem.UsedPercent))
	}

	// Disk (Root)
	dUsage, err := disk.Usage("/")
	if err == nil {
		usedGB := float64(dUsage.Used) / 1024 / 1024 / 1024
		totalGB := float64(dUsage.Total) / 1024 / 1024 / 1024
		sb.WriteString(fmt.Sprintf("Disk (/): %.2f GB / %.2f GB (%.2f%%)\n", usedGB, totalGB, dUsage.UsedPercent))
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}
