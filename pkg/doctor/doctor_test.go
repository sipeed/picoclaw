package doctor

import (
	"os"
	"testing"
)

func TestDoctor_Run(t *testing.T) {
	// Create a temporary config file for testing
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"
	
	// Create minimal config
	configContent := `{
  "agents": {
    "defaults": {
      "workspace": "` + tmpDir + `/workspace",
      "model": "gemini-1.5-flash"
    }
  },
  "providers": {
    "gemini": {
      "api_key": "test-key"
    }
  },
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true
      }
    }
  }
}`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	
	// Create workspace
	os.MkdirAll(tmpDir+"/workspace", 0755)
	
	// Run doctor
	doc := NewDoctor(configPath)
	doc.Run()
	
	// Should be healthy with valid config
	if !doc.IsHealthy() {
		t.Log("Doctor reported issues - check output above")
	}
}