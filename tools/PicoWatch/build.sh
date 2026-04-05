#!/bin/bash
set -euo pipefail

APP_NAME="PicoWatch"
BUILD_DIR="build"
APP_DIR="$BUILD_DIR/$APP_NAME.app"

echo "Building $APP_NAME..."

rm -rf "$BUILD_DIR"
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

xcrun swiftc \
  -sdk "$(xcrun --show-sdk-path -sdk macosx)" \
  -target arm64-apple-macosx14.0 \
  -framework AppKit -framework SwiftUI \
  -parse-as-library \
  -O \
  PicoWatch/PicoWatchApp.swift \
  PicoWatch/AppDelegate.swift \
  PicoWatch/MonitorEngine.swift \
  PicoWatch/PopoverView.swift \
  -o "$APP_DIR/Contents/MacOS/$APP_NAME"

cp PicoWatch/Info.plist "$APP_DIR/Contents/Info.plist"

echo "Built: $APP_DIR"
echo "Run with: open \"$APP_DIR\""
