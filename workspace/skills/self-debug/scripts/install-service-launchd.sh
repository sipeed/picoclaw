#!/bin/bash

create_plist_file() {
    cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${service_name}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${exec_path}</string>
        <string>agent</string>
        <string>--config</string>
        <string>${picoclaw_config}</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PICOCLAW_HOME</key>
        <string>${picoclaw_home}</string>
        <key>PICOCLAW_CONFIG</key>
        <string>${picoclaw_config}</string>
        <key>PICOCLAW_SERVICE_NAME</key>
        <string>${service_name}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>${picoclaw_home}</string>
    <key>StandardOutPath</key>
    <string>~/Library/logs/${service_name}.log</string>
    <key>StandardErrorPath</key>
    <string>~/Library/logs/${service_name}.err.log</string>
</dict>
</plist>
EOF
}

# Variable setup to match your Linux logic
service_name=${1:-picoclaw}

# macOS specific absolute paths
exec_path=$(which picoclaw)
picoclaw_home="$HOME/.${service_name}"
picoclaw_config="${picoclaw_home}/config.json"
plist_path="$HOME/Library/LaunchAgents/${service_name}.plist"

# Ensure directory exists
mkdir -p "${picoclaw_home}"

# Use the heredoc function to write the file
create_plist_file > "${plist_path}"

# Load it (macOS equivalent of systemctl enable --now)
echo "Enable using:"
echo "launchctl bootstrap gui/$(id -u) '${plist_path}'"

echo "Service ${service_name} installed at ${plist_path}"