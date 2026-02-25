package service

import "fmt"

func renderSystemdUnit(exePath, pathEnv string) string {
	return fmt.Sprintf(`[Unit]
Description=PicoClaw Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s gateway
Restart=always
RestartSec=3
Environment=HOME=%%h
Environment=PATH=%s
WorkingDirectory=%%h

[Install]
WantedBy=default.target
`, exePath, pathEnv)
}

func renderLaunchdPlist(label, exePath, stdoutPath, stderrPath, pathEnv string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>

  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>gateway</string>
  </array>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <true/>

  <key>StandardOutPath</key>
  <string>%s</string>

  <key>StandardErrorPath</key>
  <string>%s</string>

  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>%s</string>
  </dict>
</dict>
</plist>
`, label, exePath, stdoutPath, stderrPath, pathEnv)
}
