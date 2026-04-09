#!/bin/bash
set -e

echo "📦 Installing Node.js and npm..."

# Install Node.js using nvm (Node Version Manager)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash

# Load nvm
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Install Node.js LTS
nvm install --lts
nvm use --lts

# Verify installation
echo ""
echo "✅ Node.js installed:"
node --version
npm --version

echo ""
echo "🎉 Ready to install affine-mcp-server!"
echo "Run: npm install -g affine-mcp-server"
