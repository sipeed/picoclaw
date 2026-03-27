#!/usr/bin/env bash
# picoclaw-uninstall
# This script uninstalls PicoClaw and all of its associated data.

set -e

# Default installation paths (matching the Makefile)
INSTALL_PREFIX="${HOME}/.local"
INSTALL_BIN_DIR="${INSTALL_PREFIX}/bin"
BINARY_NAME="picoclaw"
PICOCLAW_HOME="${HOME}/.picoclaw"

echo "Uninstalling PicoClaw..."

# Remove the executable binary
if [ -f "${INSTALL_BIN_DIR}/${BINARY_NAME}" ]; then
  rm -f "${INSTALL_BIN_DIR}/${BINARY_NAME}"
  echo "Removed executable: ${INSTALL_BIN_DIR}/${BINARY_NAME}"
else
  echo "Executable ${INSTALL_BIN_DIR}/${BINARY_NAME} not found. Skipping."
fi

# Remove the launcher binary
if [ -f "${INSTALL_BIN_DIR}/picoclaw-launcher" ]; then
  rm -f "${INSTALL_BIN_DIR}/picoclaw-launcher"
  echo "Removed executable: ${INSTALL_BIN_DIR}/picoclaw-launcher"
else
  echo "Executable ${INSTALL_BIN_DIR}/picoclaw-launcher not found. Skipping."
fi

# Remove the workspace and configurations
if [ -d "${PICOCLAW_HOME}" ]; then
  rm -rf "${PICOCLAW_HOME}"
  echo "Removed all workspace and configuration data: ${PICOCLAW_HOME}"
else
  echo "Data directory ${PICOCLAW_HOME} not found. Skipping."
fi

# Remove the uninstaller script itself
SCRIPT_PATH="${INSTALL_BIN_DIR}/picoclaw-uninstall"
if [ -f "${SCRIPT_PATH}" ]; then
  rm -f "${SCRIPT_PATH}"
  echo "Removed uninstaller script: ${SCRIPT_PATH}"
fi

echo "PicoClaw uninstallation complete!"
