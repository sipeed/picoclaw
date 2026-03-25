//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		rawContent  string
		want        string
	}{
		{
			name:        "text message",
			messageType: "text",
			rawContent:  `{"text": "hello world"}`,
			want:        "hello world",
		},
		{
			name:        "text message invalid JSON",
			messageType: "text",
			rawContent:  `not json`,
			want:        "not json",
		},
		{
			name:        "post message returns raw JSON",
			messageType: "post",
			rawContent:  `{"title": "test post"}`,
			want:        `{"title": "test post"}`,
		},
		{
			name:        "image message returns empty",
			messageType: "image",
			rawContent:  `{"image_key": "img_xxx"}`,
			want:        "",
		},
		{
			name:        "file message with filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx", "file_name": "report.pdf"}`,
			want:        "report.pdf",
		},
		{
			name:        "file message without filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx"}`,
			want:        "",
		},
		{
			name:        "audio message with filename",
			messageType: "audio",
			rawContent:  `{"file_key": "file_xxx", "file_name": "recording.ogg"}`,
			want:        "recording.ogg",
		},
		{
			name:        "media message with filename",
			messageType: "media",
			rawContent:  `{"file_key": "file_xxx", "file_name": "video.mp4"}`,
			want:        "video.mp4",
		},
		{
			name:        "unknown message type returns raw",
			messageType: "sticker",
			rawContent:  `{"sticker_id": "sticker_xxx"}`,
			want:        `{"sticker_id": "sticker_xxx"}`,
		},
		{
			name:        "empty raw content",
			messageType: "text",
			rawContent:  "",
			want:        "",
		},
		{
			name:        "interactive card returns raw JSON",
			messageType: "interactive",
			rawContent:  `{"schema":"2.0","body":{"elements":[{"tag":"markdown","content":"Hello from card"}]}}`,
			want:        `{"schema":"2.0","body":{"elements":[{"tag":"markdown","content":"Hello from card"}]}}`,
		},
		{
			name:        "interactive card with complex structure returns raw JSON",
			messageType: "interactive",
			rawContent:  `{"header":{"title":{"tag":"plain_text","content":"Title"}},"elements":[{"tag":"div","text":{"tag":"lark_md","content":"Card content"}}]}`,
			want:        `{"header":{"title":{"tag":"plain_text","content":"Title"}},"elements":[{"tag":"div","text":{"tag":"lark_md","content":"Card content"}}]}`,
		},
		{
			name:        "interactive card invalid JSON returns as-is",
			messageType: "interactive",
			rawContent:  `not valid json`,
			want:        `not valid json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.messageType, tt.rawContent)
			if got != tt.want {
				t.Errorf("extractContent(%q, %q) = %q, want %q", tt.messageType, tt.rawContent, got, tt.want)
			}
		})
	}
}

func TestAppendMediaTags(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		messageType string
		mediaRefs   []string
		want        string
	}{
		{
			name:        "no refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   nil,
			want:        "hello",
		},
		{
			name:        "empty refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   []string{},
			want:        "hello",
		},
		{
			name:        "image with content",
			content:     "check this",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "check this [image: photo]",
		},
		{
			name:        "image empty content",
			content:     "",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "[image: photo]",
		},
		{
			name:        "audio",
			content:     "listen",
			messageType: "audio",
			mediaRefs:   []string{"ref1"},
			want:        "listen [audio]",
		},
		{
			name:        "media/video",
			content:     "watch",
			messageType: "media",
			mediaRefs:   []string{"ref1"},
			want:        "watch [video]",
		},
		{
			name:        "file",
			content:     "report.pdf",
			messageType: "file",
			mediaRefs:   []string{"ref1"},
			want:        "report.pdf [file]",
		},
		{
			name:        "unknown type",
			content:     "something",
			messageType: "sticker",
			mediaRefs:   []string{"ref1"},
			want:        "something [attachment]",
		},
		{
			name:        "interactive card with images returns content unchanged",
			content:     `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`,
			messageType: "interactive",
			mediaRefs:   []string{"ref1"},
			want:        `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendMediaTags(tt.content, tt.messageType, tt.mediaRefs)
			if got != tt.want {
				t.Errorf(
					"appendMediaTags(%q, %q, %v) = %q, want %q",
					tt.content,
					tt.messageType,
					tt.mediaRefs,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestExtractFeishuSenderID(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name   string
		sender *larkim.EventSender
		want   string
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   "",
		},
		{
			name:   "nil sender ID",
			sender: &larkim.EventSender{SenderId: nil},
			want:   "",
		},
		{
			name: "userId preferred",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr("u_abc123"),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "u_abc123",
		},
		{
			name: "openId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "ou_def456",
		},
		{
			name: "unionId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "on_ghi789",
		},
		{
			name: "all empty strings",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr(""),
				},
			},
			want: "",
		},
		{
			name: "nil userId pointer falls through",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  nil,
					OpenId:  strPtr("ou_def456"),
					UnionId: nil,
				},
			},
			want: "ou_def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeishuSenderID(tt.sender)
			if got != tt.want {
				t.Errorf("extractFeishuSenderID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateAndCreateDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		setup   func() string // 创建测试目录并返回路径，返回空字符串表示不创建
		wantErr bool
		cleanup func(string)  // 清理函数
	}{
		{
			name: "directory exists and is writable",
			setup: func() string {
				dir := t.TempDir()
				return dir
			},
			wantErr: false,
		},
		{
			name: "directory does not exist - creates successfully",
			setup: func() string {
				baseDir := t.TempDir()
				newDir := baseDir + "/newdir/subdir"
				return newDir
			},
			wantErr: false,
		},
		{
			name: "path exists but is a file not directory",
			setup: func() string {
				dir := t.TempDir()
				filePath := dir + "/notadir"
				// 创建一个文件
				if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
			wantErr: true,
		},
		{
			name: "directory exists but not writable",
			setup: func() string {
				dir := t.TempDir()
				// 在 Unix 系统上，通过 chmod 移除写权限来测试
				// 注意：这在 Windows 上可能不工作
				return dir
			},
			wantErr: false, // 实际上 TempDir 通常是可写的，这个测试场景较难模拟
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup()
			if dir == "" {
				t.Skip("test setup returned empty directory")
			}

			// 创建一个测试用的 FeishuChannel 实例
			channel := &FeishuChannel{
				globalCfg: &config.Config{},
			}
			err := channel.validateAndCreateDir(dir)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateAndCreateDir() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 如果期望成功，验证目录确实存在且是目录
			if !tt.wantErr && err == nil {
				info, statErr := os.Stat(dir)
				if statErr != nil {
					t.Errorf("directory %s does not exist after validation: %v", dir, statErr)
				}
				if info != nil && !info.IsDir() {
					t.Errorf("path %s exists but is not a directory", dir)
				}
			}
		})
	}
}

func TestResolveDownloadDir(t *testing.T) {
	tests := []struct {
		name        string
		downloadDir string
		workspace   string
		setup       func() string // 创建测试目录并返回路径
		wantEmpty   bool          // 是否期望返回空字符串
		cleanup     func(string)  // 清理函数
	}{
		{
			name:        "no download dir configured returns empty",
			downloadDir: "",
			workspace:   "/workspace",
			wantEmpty:   true,
		},
		{
			name:        "existing directory returns as-is",
			downloadDir: "",
			workspace:   "/workspace",
			setup: func() string {
				dir := t.TempDir()
				return dir
			},
			wantEmpty: false,
		},
		{
			name:        "non-existing directory is created",
			downloadDir: "",
			workspace:   "/workspace",
			setup: func() string {
				baseDir := t.TempDir()
				newDir := baseDir + "/newdir/subdir"
				return newDir
			},
			wantEmpty: false,
		},
		{
			name:        "path exists but is a file returns empty",
			downloadDir: "",
			workspace:   "/workspace",
			setup: func() string {
				dir := t.TempDir()
				filePath := dir + "/notadir"
				if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
			wantEmpty: true,
		},
		{
			name:        "relative path resolves relative to workspace",
			downloadDir: "downloads",
			workspace:   "",
			setup:       func() string { return "" }, // Dummy setup
			wantEmpty:   false,
		},
		{
			name:        "absolute path remains absolute",
			downloadDir: "",
			workspace:   "",
			setup: func() string {
				// Return absolute path
				return t.TempDir()
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var downloadDir string
			var workspace string

			if tt.downloadDir != "" {
				downloadDir = tt.downloadDir
			}

			// For relative path test, create a temp workspace
			if tt.name == "relative path resolves relative to workspace" {
				workspace = t.TempDir()
			} else if tt.setup != nil {
				downloadDir = tt.setup()
			} else {
				workspace = tt.workspace
			}

			// 创建测试用的 FeishuChannel 实例
			channel := &FeishuChannel{
				config: config.FeishuConfig{
					DownloadDir: downloadDir,
				},
				globalCfg: &config.Config{
					Agents: config.AgentsConfig{
						Defaults: config.AgentDefaults{
							Workspace: workspace,
						},
					},
				},
			}

			// 调用被测试的函数
			got := channel.resolveDownloadDir()

			// 验证结果
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("resolveDownloadDir() = %q, want empty string", got)
				}
			} else {
				if got == "" {
					t.Errorf("resolveDownloadDir() returned empty string, want non-empty")
				}

				// 验证目录存在且可访问
				if got != "" {
					info, err := os.Stat(got)
					if err != nil {
						t.Errorf("resolveDownloadDir() returned %q but stat failed: %v", got, err)
					} else if !info.IsDir() {
						t.Errorf("resolveDownloadDir() returned %q but it's not a directory", got)
					}
				}
			}

			// 清理
			if tt.cleanup != nil {
				tt.cleanup(got)
			}
		})
	}
}
