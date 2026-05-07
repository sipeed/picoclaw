package vault

import (
	"os"
	"path/filepath"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/memory"
)

type VaultStore struct {
	rootPath string
}

func NewVaultStore(rootPath string) *VaultStore {
	return &VaultStore{rootPath: rootPath}
}

func (vs *VaultStore) CreateNote(name string, frontmatter map[string]interface{}, content string) error {
	if err := os.MkdirAll(vs.rootPath, 0755); err != nil {
		return err
	}
	note := "---\n"
	if fm, err := yaml.Marshal(frontmatter); err == nil {
		note += string(fm)
	}
	note += "---\n\n" + content
	notePath := filepath.Join(vs.rootPath, name+".md")
	return os.WriteFile(notePath, []byte(note), 0644)
}

// ReadNote reads a note file and parses frontmatter
func (vs *VaultStore) ReadNote(name string) (map[string]interface{}, string, error) {
	notePath := filepath.Join(vs.rootPath, name+".md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		return nil, "", err
	}
	return memory.ParseFrontmatter(string(content))
}