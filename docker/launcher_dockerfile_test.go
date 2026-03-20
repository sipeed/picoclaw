package docker

import (
	"os"
	"strings"
	"testing"
)

func TestLauncherDockerfileIncludesNodeRuntime(t *testing.T) {
	data, err := os.ReadFile("Dockerfile.goreleaser.launcher")
	if err != nil {
		t.Fatalf("read Dockerfile.goreleaser.launcher: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "FROM node:") {
		t.Fatalf("launcher Dockerfile should use a Node.js runtime base image, got:\n%s", content)
	}
	if !strings.Contains(content, "COPY $TARGETPLATFORM/picoclaw-launcher /usr/local/bin/picoclaw-launcher") {
		t.Fatal("launcher Dockerfile should still copy the launcher binary")
	}
	if !strings.Contains(content, "ENTRYPOINT [\"picoclaw-launcher\"]") {
		t.Fatal("launcher Dockerfile should keep picoclaw-launcher as the entrypoint")
	}
}
