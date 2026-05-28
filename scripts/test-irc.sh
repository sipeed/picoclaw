#!/bin/sh
# Starts a local Ergo IRC server for testing the IRC channel.
#
# Requirements: docker
# Usage: ./scripts/test-irc.sh

set -e

CONTAINER_NAME="picoclaw-test-ergo"
IRC_PORT=6667

# Clean up any previous instance
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

echo "Starting Ergo IRC server on port $IRC_PORT..."
docker run -d \
    --name "$CONTAINER_NAME" \
    -p "$IRC_PORT:6667" \
    ghcr.io/ergochat/ergo:stable

for i in $(seq 1 10); do
    if nc -z localhost "$IRC_PORT" 2>/dev/null; then
        break
    fi
    if [ "$i" -eq 10 ]; then
        echo "ERROR: Server did not start within 10s"
        exit 1
    fi
    sleep 1
done

echo ""
echo "IRC server ready on localhost:$IRC_PORT"
echo ""
echo "Add this to your ~/.picoclaw/config.json under \"channels\":"
echo ""
echo '  "irc": {'
echo '    "enabled": true,'
echo '    "server": "localhost:6667",'
echo '    "tls": false,'
echo '    "nick": "picobot",'
echo '    "channels": ["#test"],'
echo '    "allow_from": [],'
echo '    "group_trigger": { "mention_only": true }'
echo '  }'
echo ""
echo "Then run picoclaw:"
echo "  cd packages/picoclaw && go run ./cmd/picoclaw gateway"
echo ""
echo "Connect with an IRC client:"
echo "  irssi:   /connect localhost $IRC_PORT"
echo "  weechat: /server add test localhost/$IRC_PORT && /connect test"
echo "  Join #test, then: picobot: hello"
echo ""
echo "To stop the IRC server:"
echo "  docker rm -f $CONTAINER_NAME"
