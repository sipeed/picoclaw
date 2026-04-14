package cliui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PrintVersion prints version, optional build info, and Go toolchain line.
func PrintVersion(logo, versionLine string, build, goVer string) {
	if !UseFancyLayout() {
		fmt.Printf("%s %s\n", logo, versionLine)
		if build != "" {
			fmt.Printf("  Build: %s\n", build)
		}
		if goVer != "" {
			fmt.Printf("  Go: %s\n", goVer)
		}
		return
	}

	inner := InnerWidth()
	box := borderStyle().Width(inner)

	if UseColumnLayout() {
		leftCol := kvKeyStyle().Width(12).Align(lipgloss.Right)
		rightW := inner - 16
		rightStyle := kvValStyle().Width(rightW)

		rows := [][]string{
			{leftCol.Render("Version"), rightStyle.Render(versionLine)},
		}
		if build != "" {
			rows = append(rows, []string{leftCol.Render("Build"), rightStyle.Render(build)})
		}
		if goVer != "" {
			rows = append(rows, []string{leftCol.Render("Go"), rightStyle.Render(goVer)})
		}
		var body strings.Builder
		for _, r := range rows {
			body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, r[0], "  ", r[1]))
			body.WriteString("\n")
		}
		header := titleBarStyle().Render(logo+" picoclaw") + "\n\n"
		fmt.Println(box.Render(header + body.String()))
		return
	}

	var lines []string
	lines = append(lines, titleBarStyle().Render(logo+" picoclaw"))
	lines = append(lines, "")
	lines = append(lines, kvKeyStyle().Render("Version")+"  "+kvValStyle().Render(versionLine))
	if build != "" {
		lines = append(lines, kvKeyStyle().Render("Build")+"     "+kvValStyle().Render(build))
	}
	if goVer != "" {
		lines = append(lines, kvKeyStyle().Render("Go")+"        "+kvValStyle().Render(goVer))
	}
	fmt.Println(box.Render(strings.Join(lines, "\n")))
}
