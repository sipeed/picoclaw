package cliui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sipeed/picoclaw/pkg/config"
)

// PrintOnboardComplete prints the post-onboard “ready” message and next steps.
func PrintOnboardComplete(logo string, encrypt bool, configPath string) {
	if !UseFancyLayout() {
		printOnboardPlain(logo, encrypt, configPath)
		return
	}
	printOnboardFancy(logo, encrypt, configPath)
}

func printOnboardPlain(logo string, encrypt bool, configPath string) {
	fmt.Printf("\n%s picoclaw is ready!\n", logo)
	fmt.Println("\nNext steps:")
	if encrypt {
		fmt.Println("  1. Set your encryption passphrase before starting picoclaw:")
		fmt.Println("       export PICOCLAW_KEY_PASSPHRASE=<your-passphrase>   # Linux/macOS")
		fmt.Println("       set PICOCLAW_KEY_PASSPHRASE=<your-passphrase>      # Windows cmd")
		fmt.Println("")
		fmt.Println("  2. Add your selected LLM to the `model_name` field in", configPath)
		fmt.Println("  3. Add your API key to", securityPath(configPath))
	} else {
		fmt.Println("  1. Add your selected LLM to the `model_name` field in", configPath)
		fmt.Println("  2. Add your API key to", securityPath(configPath))
	}
	fmt.Println("")
	fmt.Println("     Recommended:")
	fmt.Println("     - OpenRouter: https://openrouter.ai/keys (access 100+ models)")
	fmt.Println("     - Ollama:     https://ollama.com (local, free)")
	fmt.Println("")
	fmt.Println("     See README.md for 17+ supported providers.")
	fmt.Println("")
	if encrypt {
		fmt.Println("  4. Chat: picoclaw agent -m \"Hello!\"")
	} else {
		fmt.Println("  3. Chat: picoclaw agent -m \"Hello!\"")
	}
}

func printOnboardFancy(logo string, encrypt bool, configPath string) {
	inner := InnerWidth()
	box := borderStyle().MaxWidth(inner + 8)

	ready := titleBarStyle().Render(logo+" picoclaw is ready!") + "\n"
	fmt.Println()
	fmt.Println(box.Width(inner).Render(strings.TrimSpace(ready)))
	fmt.Println()

	steps := buildOnboardingSteps(encrypt, configPath)
	rec := recommendedBlock()
	chat := chatStep(encrypt)

	if UseColumnLayout() {
		leftW := min(inner/2-2, 52)
		rightW := inner - leftW - 4
		if rightW < 36 {
			rightW = 36
		}
		leftBlock := borderStyle().MaxWidth(leftW + 8).Width(leftW).
			Render(titleBarStyle().Render("Next steps") + "\n\n" + bodyStyle().Width(leftW).Render(steps))
		rightBlock := borderStyle().MaxWidth(rightW + 8).Width(rightW).
			Render(mutedStyle().Bold(true).Render("Recommended") + "\n\n" + bodyStyle().Width(rightW).Render(rec))
		gap := strings.Repeat(" ", 2)
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, gap, rightBlock))
		fmt.Println()
		full := borderStyle().Width(inner).Render(bodyStyle().Width(inner - 4).Render(chat))
		fmt.Println(full)
		return
	}

	// Same order as plain output: numbered steps → recommended → chat line.
	next := titleBarStyle().Render("Next steps") + "\n\n" +
		bodyStyle().Width(inner-4).Render(steps+"\n\n"+rec+"\n\n"+chat)
	fmt.Println(borderStyle().Width(inner).Render(next))
}

func buildOnboardingSteps(encrypt bool, configPath string) string {
	var b strings.Builder
	if encrypt {
		b.WriteString("1. Set your encryption passphrase before starting picoclaw:\n")
		b.WriteString("   export PICOCLAW_KEY_PASSPHRASE=<your-passphrase>   # Linux/macOS\n")
		b.WriteString("   set PICOCLAW_KEY_PASSPHRASE=<your-passphrase>      # Windows cmd\n\n")
		b.WriteString("2. Add your selected LLM to the `model_name` field in\n   ")
		b.WriteString(configPath)
		b.WriteString("\n")
		b.WriteString("3. Add your API key to\n   ")
		b.WriteString(securityPath(configPath))
		b.WriteString("\n")
	} else {
		b.WriteString("1. Add your selected LLM to the `model_name` field in\n   ")
		b.WriteString(configPath)
		b.WriteString("\n")
		b.WriteString("2. Add your API key to\n   ")
		b.WriteString(securityPath(configPath))
		b.WriteString("\n")
	}
	return b.String()
}

func recommendedBlock() string {
	return "• OpenRouter: https://openrouter.ai/keys\n  (access 100+ models)\n\n" +
		"• Ollama: https://ollama.com\n  (local, free)\n\n" +
		"See README.md for 17+ supported providers."
}

func chatStep(encrypt bool) string {
	if encrypt {
		return "4. Chat:\n   picoclaw agent -m \"Hello!\""
	}
	return "3. Chat:\n   picoclaw agent -m \"Hello!\""
}

// Derive the expected path of the security configuration file
// from the general configuration file path.
// NOTE: Function duplicated from config/security.go#securityPath()
func securityPath(configPath string) string {
	configDir := filepath.Dir(configPath)
	return filepath.Join(configDir, config.SecurityConfigFile)
}
