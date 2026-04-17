package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/tools"
)

func main() {
	workspace, _ := os.Getwd()
	maxReadFileSize := 64 * 1024

	fmt.Println("=== 测试 diff_files 工具 ===\n")

	tool := tools.NewDiffTool(workspace, false, maxReadFileSize)

	fmt.Printf("工具名称: %s\n", tool.Name())
	fmt.Printf("工具描述: %s\n", tool.Description())
	fmt.Println()

	testUnifiedDiff(tool)
	testSideBySideDiff(tool)
	testIgnoreWhitespace(tool)

	fmt.Println("\n=== 测试完成 ===")
}

func testUnifiedDiff(tool *tools.DiffTool) {
	fmt.Println("--- 测试 1: 统一差异格式 (unified diff) ---")
	fmt.Println()

	args := map[string]any{
		"file_a":        "test_file_a.txt",
		"file_b":        "test_file_b.txt",
		"format":        "unified",
		"context_lines": 3,
	}

	result := tool.Execute(context.Background(), args)
	fmt.Println("结果:")
	fmt.Println(result.ForLLM)
	fmt.Println()
}

func testSideBySideDiff(tool *tools.DiffTool) {
	fmt.Println("--- 测试 2: 并排对比格式 (side_by_side) ---")
	fmt.Println()

	args := map[string]any{
		"file_a": "test_file_a.txt",
		"file_b": "test_file_b.txt",
		"format": "side_by_side",
	}

	result := tool.Execute(context.Background(), args)
	fmt.Println("结果:")
	fmt.Println(result.ForLLM)
	fmt.Println()
}

func testIgnoreWhitespace(tool *tools.DiffTool) {
	fmt.Println("--- 测试 3: 忽略空白字符差异 (ignore_whitespace) ---")
	fmt.Println()

	args := map[string]any{
		"file_a":            "test_file_a.txt",
		"file_b":            "test_file_whitespace.txt",
		"format":            "unified",
		"ignore_whitespace": false,
	}

	fmt.Println("不忽略空白差异:")
	result1 := tool.Execute(context.Background(), args)
	fmt.Println(result1.ForLLM)
	fmt.Println()

	args["ignore_whitespace"] = true
	fmt.Println("忽略空白差异:")
	result2 := tool.Execute(context.Background(), args)
	fmt.Println(result2.ForLLM)
	fmt.Println()
}
