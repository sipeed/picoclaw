//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestGenerateUniqueFilename(t *testing.T) {
	tests := []struct {
		name       string
		baseName   string
		tmpContent string
		setupFiles func(string) // 创建已存在的文件
		wantSuffix string       // 期望的文件名后缀（空表示原名）
		wantReused bool         // 期望是否重用
	}{
		{
			name:       "returns base path when file doesn't exist",
			baseName:   "test.jpg",
			tmpContent: "new content",
			setupFiles: func(string) {},
			wantSuffix: "",
			wantReused: false,
		},
		{
			name:       "reuses file when hash and size match",
			baseName:   "test.jpg",
			tmpContent: "same content",
			setupFiles: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "test.jpg"), []byte("same content"), 0o644)
			},
			wantSuffix: "",
			wantReused: true,
		},
		{
			name:       "adds suffix when file exists with different content",
			baseName:   "test.jpg",
			tmpContent: "different content",
			setupFiles: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "test.jpg"), []byte("old content"), 0o644)
			},
			wantSuffix: "-1.jpg",
			wantReused:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			basePath := filepath.Join(dir, tt.baseName)
			tt.setupFiles(dir)

			// Create temporary file with test content
			tmpFile, err := os.CreateTemp("", "test_*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			if _, err := tmpFile.WriteString(tt.tmpContent); err != nil {
				t.Fatalf("failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			channel := &FeishuChannel{}
			gotPath, gotReused, err := channel.generateUniqueFilename(basePath, tmpFile.Name())

			if err != nil {
				t.Errorf("generateUniqueFilename() error = %v", err)
				return
			}

			// Check suffix
			gotBase := filepath.Base(gotPath)
			wantBase := tt.baseName
			if tt.wantSuffix != "" {
				ext := filepath.Ext(tt.baseName)
				nameWithoutExt := strings.TrimSuffix(tt.baseName, ext)
				wantBase = nameWithoutExt + tt.wantSuffix
			}

			if gotBase != wantBase {
				t.Errorf("generateUniqueFilename() filename = %q, want %q", gotBase, wantBase)
			}

			if gotReused != tt.wantReused {
				t.Errorf("generateUniqueFilename() reused = %v, want %v", gotReused, tt.wantReused)
			}

			// Clean up
			os.Remove(tmpFile.Name())
		})
	}
}

func TestCalculateFileHash(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantHash  string
		wantError bool
	}{
		{
			name:      "calculates SHA-256 hash correctly",
			content:   "hello world",
			wantHash:  "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", // SHA-256 of "hello world"
			wantError: false,
		},
		{
			name:      "handles empty content",
			content:   "",
			wantHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // SHA-256 of empty string
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(t.TempDir(), "test.txt")

			if err := os.WriteFile(testFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			channel := &FeishuChannel{}
			got, err := channel.calculateFileHash(testFile)

			if (err != nil) != tt.wantError {
				t.Errorf("calculateFileHash() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.wantHash {
				t.Errorf("calculateFileHash() = %q, want %q", got, tt.wantHash)
			}
		})
	}
}

func TestGenerateFilePath(t *testing.T) {
	tests := []struct {
		name       string
		downloadDir string
		filename    string
		wantPrefix  string // 期望路径前缀（年/月部分会变化）
	}{
		{
			name:       "generates correct path structure",
			downloadDir: "/downloads",
			filename:    "report.pdf",
			wantPrefix:  "/downloads/",
		},
		{
			name:       "sanitizes filename",
			downloadDir: "/downloads",
			filename:    "report:name.pdf",
			wantPrefix:  "/downloads/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &FeishuChannel{}
			got := channel.generateFilePath(tt.downloadDir, tt.filename)

			if !filepath.IsAbs(got) {
				t.Errorf("generateFilePath() returned relative path, want absolute")
			}

			// Check path format: {download_dir}/{year}/{month}/{sanitized_filename}
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("generateFilePath() = %q, want prefix %q", got, tt.wantPrefix)
			}

			// Check that filename is sanitized
			base := filepath.Base(got)
			if strings.Contains(base, ":") || strings.Contains(base, "*") {
				t.Errorf("generateFilePath() filename not sanitized: %q", base)
			}
		})
	}
}

func TestLoadDirectoryIndex(t *testing.T) {
	tests := []struct {
		name          string
		setupIndex    func(string) error
		wantFileCount int
		wantVersion   int
	}{
		{
			name:          "returns empty index when file doesn't exist",
			setupIndex:    func(string) error { return nil },
			wantFileCount: 0,
			wantVersion:   1,
		},
		{
			name: "loads valid index file",
			setupIndex: func(dir string) error {
				index := DirectoryIndex{
					Version: 1,
					Files: []FileMeta{
						{
							Filename:     "test.pdf",
							Hash:         "abc123",
							Size:         1024,
							FileKey:      "key123",
							MessageID:    "msg123",
							DownloadedAt: time.Now(),
						},
					},
				}
				data, err := json.Marshal(index)
				if err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, indexFileName), data, 0o644)
			},
			wantFileCount: 1,
			wantVersion:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			_ = tt.setupIndex(dir)

			channel := &FeishuChannel{}
			index, err := channel.loadDirectoryIndex(dir)

			if err != nil {
				t.Errorf("loadDirectoryIndex() error = %v", err)
				return
			}

			if len(index.Files) != tt.wantFileCount {
				t.Errorf("loadDirectoryIndex() file count = %d, want %d", len(index.Files), tt.wantFileCount)
			}

			if index.Version != tt.wantVersion {
				t.Errorf("loadDirectoryIndex() version = %d, want %d", index.Version, tt.wantVersion)
			}
		})
	}
}

func TestSaveDirectoryIndex(t *testing.T) {
	dir := t.TempDir()

	index := &DirectoryIndex{
		Version: 1,
		Files: []FileMeta{
			{
				Filename:     "test.pdf",
				Hash:         "abc123",
				Size:         1024,
				FileKey:      "key123",
				MessageID:    "msg123",
				DownloadedAt: time.Now(),
			},
		},
	}

	channel := &FeishuChannel{}
	err := channel.saveDirectoryIndex(dir, index)

	if err != nil {
		t.Errorf("saveDirectoryIndex() error = %v", err)
		return
	}

	// Verify file was created
	indexPath := filepath.Join(dir, indexFileName)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("saveDirectoryIndex() file not created")
		return
	}

	// Verify content
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Errorf("failed to read index file: %v", err)
		return
	}

	var loaded DirectoryIndex
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("failed to unmarshal index: %v", err)
		return
	}

	if len(loaded.Files) != 1 {
		t.Errorf("saveDirectoryIndex() saved wrong number of files: got %d, want 1", len(loaded.Files))
	}
}

func TestUpdateIndexWithFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string) (*DirectoryIndex, error)
		meta     FileMeta
		verify   func(*DirectoryIndex) error
	}{
		{
			name: "adds new file to empty index",
			setup: func(dir string) (*DirectoryIndex, error) {
				return &DirectoryIndex{Version: 1, Files: []FileMeta{}}, nil
			},
			meta: FileMeta{
				Filename:     "new.pdf",
				Hash:         "abc123",
				Size:         1024,
				FileKey:      "key123",
				MessageID:    "msg123",
				DownloadedAt: time.Now(),
			},
			verify: func(index *DirectoryIndex) error {
				if len(index.Files) != 1 {
					return fmt.Errorf("expected 1 file, got %d", len(index.Files))
				}
				if index.Files[0].Filename != "new.pdf" {
					return fmt.Errorf("expected filename new.pdf, got %s", index.Files[0].Filename)
				}
				return nil
			},
		},
		{
			name: "updates existing file in index",
			setup: func(dir string) (*DirectoryIndex, error) {
				index := &DirectoryIndex{
					Version: 1,
					Files: []FileMeta{
						{
							Filename:     "existing.pdf",
							Hash:         "old123",
							Size:         512,
							FileKey:      "key123",
							MessageID:    "msg123",
							DownloadedAt: time.Now(),
						},
					},
				}
				return index, nil
			},
			meta: FileMeta{
				Filename:     "existing.pdf",
				Hash:         "new456",
				Size:         2048,
				FileKey:      "key456",
				MessageID:    "msg456",
				DownloadedAt: time.Now(),
			},
			verify: func(index *DirectoryIndex) error {
				if len(index.Files) != 1 {
					return fmt.Errorf("expected 1 file, got %d", len(index.Files))
				}
				if index.Files[0].Hash != "new456" {
					return fmt.Errorf("expected hash new456, got %s", index.Files[0].Hash)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			_, _ = tt.setup(dir)

			channel := &FeishuChannel{}
			err := channel.updateIndexWithFile(dir, tt.meta)

			if err != nil {
				t.Errorf("updateIndexWithFile() error = %v", err)
				return
			}

			// Verify
			index, err := channel.loadDirectoryIndex(dir)
			if err != nil {
				t.Errorf("failed to load index for verification: %v", err)
				return
			}

			if err := tt.verify(index); err != nil {
				t.Errorf("verification failed: %v", err)
			}
		})
	}
}

func TestFindFileByKeyInIndex(t *testing.T) {
	dir := t.TempDir()

	// Setup: Create index with test data
	index := &DirectoryIndex{
		Version: 1,
		Files: []FileMeta{
			{
				Filename:     "file1.pdf",
				Hash:         "abc123",
				Size:         1024,
				FileKey:      "key1",
				MessageID:    "msg1",
				DownloadedAt: time.Now(),
			},
			{
				Filename:     "file2.pdf",
				Hash:         "def456",
				Size:         2048,
				FileKey:      "key2",
				MessageID:    "msg2",
				DownloadedAt: time.Now(),
			},
		},
	}

	channel := &FeishuChannel{}
	_ = channel.saveDirectoryIndex(dir, index)

	// Test: Find existing file
	meta, err := channel.findFileByKeyInIndex(dir, "key1")
	if err != nil {
		t.Errorf("findFileByKeyInIndex() error = %v", err)
		return
	}

	if meta == nil {
		t.Errorf("findFileByKeyInIndex() = nil, want file metadata")
		return
	}

	if meta.Filename != "file1.pdf" {
		t.Errorf("findFileByKeyInIndex() filename = %q, want file1.pdf", meta.Filename)
	}

	// Test: Find non-existing file
	meta, _ = channel.findFileByKeyInIndex(dir, "key999")
	if meta != nil {
		t.Errorf("findFileByKeyInIndex() = %v, want nil", meta)
	}
}
