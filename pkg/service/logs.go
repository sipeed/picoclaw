package service

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func tailFileLines(path string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	txt := strings.TrimRight(string(b), "\n")
	if txt == "" {
		return "", nil
	}
	all := strings.Split(txt, "\n")
	if len(all) <= lines {
		return strings.Join(all, "\n") + "\n", nil
	}
	return strings.Join(all[len(all)-lines:], "\n") + "\n", nil
}

func combineLogSections(sections map[string]string) string {
	out := ""
	keys := make([]string, 0, len(sections))
	for name := range sections {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		text := sections[name]
		if strings.TrimSpace(text) == "" {
			continue
		}
		if out != "" {
			out += "\n"
		}
		out += fmt.Sprintf("==> %s <==\n%s", name, text)
	}
	return out
}
