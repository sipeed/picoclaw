package openclaw

var migrateableFiles = []string{
	"AGENTS.md",
	"SOUL.md",
	"USER.md",
	"TOOLS.md",
	"HEARTBEAT.md",
}

var migrateableDirs = []string{
	"memory",
	"skills",
}

var supportedChannels = map[string]bool{
	"whatsapp":  true,
	"telegram":  true,
	"feishu":    true,
	"discord":   true,
	"maixcam":   true,
	"qq":        true,
	"dingtalk":  true,
	"slack":     true,
	"line":      true,
	"onebot":    true,
	"wecom":     true,
	"wecom_app": true,
}

var partiallySupportedChannels = map[string]string{
	"signal":      "No PicoClaw adapter available",
	"matrix":      "No PicoClaw adapter available",
	"irc":         "No PicoClaw adapter available",
	"mattermost":  "No PicoClaw adapter available",
	"imessage":    "macOS-only channel - requires manual setup",
	"bluebubbles": "No PicoClaw adapter available - consider iMessage instead",
}
