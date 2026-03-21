#!/bin/bash
# Build macOS .app bundle for Piconomous Launcher

set -e

EXECUTABLE=$1

if [ -z "$EXECUTABLE" ]; then
    echo "Usage: $0 <executable>"
    exit 1
fi

echo "executable: $EXECUTABLE"

APP_NAME="Piconomous Launcher"
APP_PATH="./build/${APP_NAME}.app"
APP_CONTENTS="${APP_PATH}/Contents"
APP_MACOS="${APP_CONTENTS}/MacOS"
APP_RESOURCES="${APP_CONTENTS}/Resources"
APP_EXECUTABLE="piconomous-launcher"
ICON_SOURCE="./scripts/icon.icns"

# Clean up existing .app
if [ -d "$APP_PATH" ]; then
    echo "Removing existing ${APP_PATH}"
    rm -rf "$APP_PATH"
fi

# Create directory structure
echo "Creating .app bundle structure..."
mkdir -p "$APP_MACOS"
mkdir -p "$APP_RESOURCES"

# Copy executable
echo "Copying executable..."
if [ -f "./web/build/${APP_EXECUTABLE}" ]; then
    cp "./web/build/${APP_EXECUTABLE}" "${APP_MACOS}/"
else
    echo "Error: ./web/build/${APP_EXECUTABLE} not found. Please build the web backend first."
    echo "Run: make build in web dir"
    exit 1
fi
if [ -f "./build/piconomous" ]; then
    cp "./build/piconomous" "${APP_MACOS}/"
else
    echo "Error: ./build/piconomous not found. Please build the main file first."
    echo "Run: make build"
    exit 1
fi
chmod +x "${APP_MACOS}/"*

# Create Info.plist
echo "Creating Info.plist..."
cat > "${APP_CONTENTS}/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>piconomous-launcher</string>
    <key>CFBundleIdentifier</key>
    <string>com.piconomous.launcher</string>
    <key>CFBundleName</key>
    <string>Piconomous Launcher</string>
    <key>CFBundleDisplayName</key>
    <string>Piconomous Launcher</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
    <key>LSRequiresCarbon</key>
    <true/>
    <key>LSUIElement</key>
    <string>1</string>
</dict>
</plist>
EOF

#sips -z 128 128 "$ICON_SOURCE" --out "${ICONSET_PATH}/icon_128x128.png" > /dev/null 2>&1
#
## Create icns file
#iconutil -c icns "$ICONSET_PATH" -o "$ICON_OUTPUT" 2>/dev/null || {
#    echo "Warning: iconutil failed"
#}

cp $ICON_SOURCE "${APP_RESOURCES}/icon.icns"

echo ""
echo "=========================================="
echo "Successfully created: ${APP_PATH}"
echo "=========================================="
echo ""
echo "To launch Piconomous:"
echo "  1. Double-click ${APP_NAME}.app in Finder"
echo "  2. Or use: open ${APP_PATH}"
echo ""
echo "Note: The app will run in the menu bar (systray) without a terminal window."
echo ""
