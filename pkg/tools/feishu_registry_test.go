package tools

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestRegisterFeishuToolsWithNilInputs(t *testing.T) {
	RegisterFeishuTools(nil, &config.Config{})
	RegisterFeishuTools(NewToolRegistry(), nil)
	RegisterFeishuToolsWithClient(nil, &config.Config{}, &mockFeishuRemoteClient{})
}
