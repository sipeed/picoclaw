package agent

import (
	"fmt"
	"net/url"
	"path/filepath"
	"unicode/utf8"
)

// statusLabel generates a human-readable Japanese status label for a tool call.
func statusLabel(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "web_search":
		if q := strArg(args, "query"); q != "" {
			return fmt.Sprintf("検索中...（%s）", truncLabel(q, 20))
		}
		return "検索中..."
	case "web_fetch":
		if u := strArg(args, "url"); u != "" {
			return fmt.Sprintf("ページ取得中...（%s）", hostFromURL(u))
		}
		return "ページ取得中..."
	case "read_file":
		return fileStatusLabel("ファイル読み取り中...", args)
	case "write_file":
		return fileStatusLabel("ファイル書き込み中...", args)
	case "edit_file":
		return fileStatusLabel("ファイル編集中...", args)
	case "append_file":
		return fileStatusLabel("ファイル追記中...", args)
	case "list_dir":
		if p := strArg(args, "path"); p != "" {
			return fmt.Sprintf("フォルダ確認中...（%s）", filepath.Base(p)+"/")
		}
		return "フォルダ確認中..."
	case "exec":
		if c := strArg(args, "command"); c != "" {
			return fmt.Sprintf("コマンド実行中...（%s）", truncLabel(c, 30))
		}
		return "コマンド実行中..."
	case "memory":
		return memoryStatusLabel(args)
	case "skill":
		return skillStatusLabel(args)
	case "cron":
		return cronStatusLabel(args)
	case "message":
		return "メッセージ送信中..."
	case "spawn":
		if l := strArg(args, "label"); l != "" {
			return fmt.Sprintf("サブタスク開始中...（%s）", truncLabel(l, 20))
		}
		return "サブタスク開始中..."
	case "subagent":
		if l := strArg(args, "label"); l != "" {
			return fmt.Sprintf("サブタスク実行中...（%s）", truncLabel(l, 20))
		}
		return "サブタスク実行中..."
	case "android":
		return androidStatusLabel(args)
	case "mcp":
		return mcpStatusLabel(args)
	case "i2c":
		return i2cStatusLabel(args)
	case "spi":
		return spiStatusLabel(args)
	default:
		return "処理中..."
	}
}

func fileStatusLabel(base string, args map[string]interface{}) string {
	if p := strArg(args, "path"); p != "" {
		return fmt.Sprintf("%s（%s）", base, filepath.Base(p))
	}
	return base
}

func memoryStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "read_long_term":
		return "メモリ読み込み中..."
	case "read_daily":
		return "今日のメモ読み込み中..."
	case "write_long_term":
		return "メモリ書き込み中..."
	case "append_daily":
		return "今日のメモ追記中..."
	default:
		return "メモリ操作中..."
	}
}

func skillStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "skill_list":
		return "スキル一覧取得中..."
	case "skill_read":
		if n := strArg(args, "name"); n != "" {
			return fmt.Sprintf("スキル読み込み中...（%s）", n)
		}
		return "スキル読み込み中..."
	default:
		return "スキル操作中..."
	}
}

func cronStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "add":
		return "リマインダー設定中..."
	case "list":
		return "スケジュール一覧取得中..."
	case "remove":
		return "スケジュール削除中..."
	default:
		return "スケジュール変更中..."
	}
}

func androidStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "list_apps":
		return "アプリ一覧取得中..."
	case "app_info":
		if p := strArg(args, "package_name"); p != "" {
			return fmt.Sprintf("アプリ情報取得中...（%s）", truncLabel(p, 25))
		}
		return "アプリ情報取得中..."
	case "launch_app":
		if p := strArg(args, "package_name"); p != "" {
			return fmt.Sprintf("アプリ起動中...（%s）", truncLabel(p, 25))
		}
		return "アプリ起動中..."
	case "current_activity":
		return "画面情報取得中..."
	case "tap":
		return "タップ中..."
	case "swipe":
		return "スワイプ中..."
	case "text":
		return "テキスト入力中..."
	case "keyevent":
		if k := strArg(args, "key"); k != "" {
			return fmt.Sprintf("キー操作中...（%s）", k)
		}
		return "キー操作中..."
	case "broadcast":
		return "ブロードキャスト送信中..."
	case "intent":
		return "インテント送信中..."
	default:
		return "デバイス操作中..."
	}
}

func mcpStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "mcp_list":
		return "MCPサーバー一覧取得中..."
	case "mcp_tools":
		if s := strArg(args, "server"); s != "" {
			return fmt.Sprintf("MCPツール取得中...（%s）", s)
		}
		return "MCPツール取得中..."
	case "mcp_call":
		if t := strArg(args, "tool"); t != "" {
			if s := strArg(args, "server"); s != "" {
				return fmt.Sprintf("MCPツール実行中...（%s/%s）", s, t)
			}
			return fmt.Sprintf("MCPツール実行中...（%s）", t)
		}
		return "MCPツール実行中..."
	default:
		return "MCP操作中..."
	}
}

func i2cStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "detect":
		return "I2Cバス検出中..."
	case "scan":
		if b := strArg(args, "bus"); b != "" {
			return fmt.Sprintf("I2Cデバイススキャン中...（bus %s）", b)
		}
		return "I2Cデバイススキャン中..."
	case "read":
		if addr := intArg(args, "address"); addr > 0 {
			return fmt.Sprintf("センサー読み取り中...（0x%02X）", addr)
		}
		return "センサー読み取り中..."
	case "write":
		if addr := intArg(args, "address"); addr > 0 {
			return fmt.Sprintf("デバイス書き込み中...（0x%02X）", addr)
		}
		return "デバイス書き込み中..."
	default:
		return "I2C操作中..."
	}
}

func spiStatusLabel(args map[string]interface{}) string {
	switch strArg(args, "action") {
	case "list":
		return "SPIデバイス一覧取得中..."
	case "transfer":
		if d := strArg(args, "device"); d != "" {
			return fmt.Sprintf("SPI通信中...（%s）", d)
		}
		return "SPI通信中..."
	case "read":
		if d := strArg(args, "device"); d != "" {
			return fmt.Sprintf("SPI読み取り中...（%s）", d)
		}
		return "SPI読み取り中..."
	default:
		return "SPI操作中..."
	}
}

// strArg extracts a string argument from a tool arguments map.
func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// intArg extracts an integer argument from a tool arguments map.
func intArg(args map[string]interface{}, key string) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

// truncLabel truncates a string to maxRunes runes, appending "..." if truncated.
func truncLabel(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "..."
}

// hostFromURL extracts the hostname from a URL string.
func hostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return truncLabel(rawURL, 30)
	}
	return u.Host
}
