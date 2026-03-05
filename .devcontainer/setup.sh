#!/bin/bash
set -e

echo "🚀 Setting up PicoClaw development environment..."

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download

# Install development tools
echo "🔧 Installing development tools..."
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# Build PicoClaw
echo "🔨 Building PicoClaw..."
make build

# Create config directory if it doesn't exist
echo "📁 Setting up config directory..."
mkdir -p ~/.picoclaw

# Copy example config if config doesn't exist
if [ ! -f ~/.picoclaw/config.json ]; then
    echo "📝 Creating example config..."
    cp config/config.example.json ~/.picoclaw/config.json
    echo "✅ Config created at ~/.picoclaw/config.json"
    echo "⚠️  Please update with your API keys!"
fi

# Run tests to verify everything works
echo "🧪 Running tests..."
go test ./pkg/tools -v -run TestAffineTool || echo "⚠️  Some tests may fail without API credentials"

echo ""
echo "✅ Setup complete!"
echo ""
echo "📚 Quick Start:"
echo "  1. Edit ~/.picoclaw/config.json with your API keys"
echo "  2. Run: ./picoclaw onboard"
echo "  3. Test: ./picoclaw agent -m 'Hello!'"
echo ""
echo "🧪 Test Affine Integration:"
echo "  1. Add Affine credentials to ~/.picoclaw/config.json"
echo "  2. Run: ./picoclaw agent -m 'List my Affine workspaces'"
echo ""
echo "📖 Documentation:"
echo "  - Affine Integration: docs/AFFINE_INTEGRATION.md"
echo "  - Quick Start: AFFINE_QUICKSTART.md"
echo ""
