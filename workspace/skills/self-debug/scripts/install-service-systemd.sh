#!/bin/bash

SERVICE_NAME=${1:-picoclaw}
TEMPLATE=${2:-default}

# Get the directory of the script and the repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. Detect picoclaw installation
echo "Detecting picoclaw installation..."
PICOCLAW_PATH=$(command -v picoclaw)
MISE_BIN=$(command -v mise)

if [ -n "$MISE_BIN" ] && $MISE_BIN which picoclaw &>/dev/null; then
    # If mise is managing picoclaw, use it to ensure the environment is correct.
    # This works whether picoclaw is installed via a tool spec or locally.
    EXEC_START="$MISE_BIN exec -- picoclaw gateway"
    echo "  - Detected mise-managed picoclaw, using: $EXEC_START"
elif [ -n "$PICOCLAW_PATH" ]; then
    # Use the absolute path if it's not managed by mise
    EXEC_START="$PICOCLAW_PATH gateway"
    echo "  - Using binary path: $EXEC_START"
else
    echo "Error: picoclaw not found. Please install it first."
    exit 1
fi

service_template__default() {
    local service_name="$1"
    local exec_start="$2"
    cat <<EOF
[Unit]
Description=${service_name^} Agent (picoclaw)
After=network.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
WorkingDirectory=%h/.${service_name}
Environment="PATH=%h/.local/bin:/usr/local/bin:/usr/bin:/bin"
Environment="PICOCLAW_SERVICE_NAME=${service_name}"
Environment="PICOCLAW_CONFIG=%h/.${service_name}/config.json"
Environment="PICOCLAW_HOME=%h/.${service_name}"
EnvironmentFile=-%h/.${service_name}/.env
ExecStart=${exec_start}
Restart=always
RestartSec=10
SyslogIdentifier=${service_name}

# Logging
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
EOF
}

service_template__multi() {
    local service_name="$1"
    local exec_start="$2"
    cat <<EOF
[Unit]
Description=${service_name^} Agent Instance %i (picoclaw)
After=network.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
WorkingDirectory=%h/.config/${service_name}-%i
Environment="PATH=%h/.local/bin:/usr/local/bin:/usr/bin:/bin"
Environment="PICOCLAW_SERVICE_NAME=${service_name}@%i"
Environment="PICOCLAW_CONFIG=%h/.config/${service_name}-%i/config.json"
Environment="PICOCLAW_HOME=%h/.config/${service_name}-%i/"
EnvironmentFile=-%h/.config/${service_name}-%i/.env
ExecStart=${exec_start}
Restart=always
RestartSec=10
SyslogIdentifier=${service_name}-%i

# Logging
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
EOF

SERVICE_NAME="${SERVICE_NAME}@"
}

show_usage__default() {
    local service_name="$1"
    local output_file="$2"
    cat <<EOF

Finalizing installation...
  - Service installed to $output_file

To start the service:
  $ systemctl --user enable ${service_name}
  $ systemctl --user start ${service_name}

Manage the service with:
  $ systemctl --user status ${service_name}    # Check status
  $ journalctl --user-unit ${service_name} -f  # View logs (real-time)

Picoclaw self-debug skill is now wired to be aware of its own logs!
You can ask it:
  'Check your logs for the last 15 minutes' or 'Why did you fail to connect?'

To ensure the service starts on boot and keeps running after logout:
  $ sudo loginctl enable-linger $(whoami)

Note: If journalctl doesn't show logs, you may need to join the systemd-journal group:
  $ sudo usermod -a -G systemd-journal $(whoami)
EOF
}

show_usage__multi() {
    local service_name="${1}"
    local output_file="$2"
    cat <<EOF

Template installed successfully.
  - Service installed to $output_file

To start an instance (e.g., 'test'):
  $ systemctl --user enable ${service_name}@test
  $ systemctl --user start  ${service_name}@test

Manage the service with:
  $ systemctl  --user status ${service_name}@test     # Check status
  $ journalctl --user-unit   ${service_name}@test -f  # View logs (real-time)

To ensure the service starts on boot and keeps running after logout:
  $ sudo loginctl enable-linger $(whoami)

Note: If journalctl doesn't show logs, you may need to join the systemd-journal group:
  $ sudo usermod -a -G systemd-journal $(whoami)
EOF
}

OUTPUT_FILE="$HOME/.config/systemd/user/${SERVICE_NAME}.service"

# 2. Ensure the directory exists
echo "Ensuring the systemd user directory exists..."
echo "  $ mkdir -p ${OUTPUT_FILE%/*}"
mkdir -p "${OUTPUT_FILE%/*}"

# 3. Generate service file
echo "Generating service file..."
if ! declare -f "service_template__$TEMPLATE" > /dev/null; then
    echo "Error: Template '$TEMPLATE' not found."
    exit 1
fi

"service_template__$TEMPLATE" "$SERVICE_NAME" "$EXEC_START" > "$OUTPUT_FILE"

# 4. Reload
echo "  $ systemctl --user daemon-reload"
systemctl --user daemon-reload

"show_usage__$TEMPLATE" "${SERVICE_NAME%@}" "$OUTPUT_FILE"
