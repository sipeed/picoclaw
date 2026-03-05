# GitHub Codespaces Setup for PicoClaw

## 🚀 Quick Start with Codespaces

### Option 1: One-Click Setup (Recommended)

1. **Open in Codespaces**
   - Go to your GitHub repository
   - Click the green "Code" button
   - Select "Codespaces" tab
   - Click "Create codespace on main"

2. **Wait for Setup**
   - Codespace will automatically:
     - Install Go 1.23
     - Download dependencies
     - Build PicoClaw
     - Run initial tests
     - Create config directory

3. **Configure API Keys**
   ```bash
   # Edit the config file
   code ~/.picoclaw/config.json
   
   # Add your Affine credentials:
   # - api_key: Your Affine API key
   # - workspace_id: Your workspace ID
   ```

4. **Test It!**
   ```bash
   # Test basic functionality
   ./picoclaw version
   
   # Test Affine integration
   ./picoclaw agent -m "List my Affine workspaces"
   ```

### Option 2: Manual Setup

If you prefer to set up manually:

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Install dependencies
go mod download

# Build
make build

# Create config
mkdir -p ~/.picoclaw
cp config/config.example.json ~/.picoclaw/config.json

# Edit config with your API keys
nano ~/.picoclaw/config.json
```

## 📝 Configure Affine Integration

### 1. Get Affine API Key

**For Affine Cloud:**
1. Visit https://app.affine.pro
2. Click your avatar → Settings
3. Go to "API Keys"
4. Click "Generate New Key"
5. Copy the API key
6. Copy workspace ID from URL: `https://app.affine.pro/workspace/YOUR_ID`

**For Self-Hosted:**
1. Access your Affine instance
2. Settings → API Keys
3. Generate key
4. Note your GraphQL endpoint

### 2. Update Config

Edit `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_AFFINE_API_KEY_HERE",
      "workspace_id": "YOUR_WORKSPACE_ID_HERE",
      "timeout_seconds": 30
    }
  }
}
```

Or use environment variables:

```bash
export PICOCLAW_TOOLS_AFFINE_ENABLED=true
export PICOCLAW_TOOLS_AFFINE_API_URL="https://app.affine.pro/graphql"
export PICOCLAW_TOOLS_AFFINE_API_KEY="your-api-key"
export PICOCLAW_TOOLS_AFFINE_WORKSPACE_ID="your-workspace-id"
```

## 🧪 Testing

### Run Unit Tests

```bash
# Test Affine tool
go test ./pkg/tools -v -run TestAffineTool

# Test all tools
go test ./pkg/tools -v

# Test with coverage
go test ./pkg/tools -cover
```

### Test Live Integration

```bash
# List workspaces
./picoclaw agent -m "Show my Affine workspaces"

# List pages
./picoclaw agent -m "List all pages in my Affine workspace"

# Search
./picoclaw agent -m "Search my Affine notes for 'test'"

# Create a test note
./picoclaw agent -m "Create a test note in Affine titled 'Codespace Test'"

# Read a page (replace with actual page ID)
./picoclaw agent -m "Read page PAGE_ID from Affine"
```

## 🔧 Development Workflow

### Build and Test Cycle

```bash
# Make changes to code
code pkg/tools/affine.go

# Build
make build

# Run tests
go test ./pkg/tools -v -run TestAffineTool

# Test manually
./picoclaw agent -m "Test command"
```

### Debug Mode

```bash
# Run with verbose logging
./picoclaw agent -m "Your message" --verbose

# Or set log level
export LOG_LEVEL=debug
./picoclaw agent -m "Your message"
```

### Hot Reload Development

```bash
# Install air for hot reload (optional)
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## 📦 Codespace Features

### Pre-installed Tools

- ✅ Go 1.23
- ✅ Git
- ✅ GitHub CLI
- ✅ VS Code Extensions (Go, GitLens)
- ✅ Make
- ✅ Development tools (gopls, dlv, staticcheck)

### Port Forwarding

Codespaces automatically forwards these ports:
- **18790**: PicoClaw Gateway
- **18791**: LINE Webhook
- **18792**: WeCom App Webhook
- **18793**: WeCom Bot Webhook

### Persistent Storage

Your `~/.picoclaw` directory is mounted from your local machine (if available), so your config persists across Codespace sessions.

## 🐛 Troubleshooting

### "Go not found"

```bash
# Verify Go installation
go version

# If not found, reload the terminal
source ~/.bashrc
```

### "Module not found"

```bash
# Download dependencies
go mod download
go mod tidy
```

### "Build failed"

```bash
# Clean and rebuild
make clean
make build

# Or manually
go build -o picoclaw ./cmd/picoclaw
```

### "Config not found"

```bash
# Create config directory
mkdir -p ~/.picoclaw

# Copy example config
cp config/config.example.json ~/.picoclaw/config.json

# Edit with your keys
code ~/.picoclaw/config.json
```

### "Affine API errors"

```bash
# Verify API key
echo $PICOCLAW_TOOLS_AFFINE_API_KEY

# Test API connectivity
curl -X POST https://app.affine.pro/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"query":"{ workspaces { id name } }"}'
```

## 🎯 Quick Test Script

Create a test script to verify everything works:

```bash
#!/bin/bash
# test-affine.sh

echo "🧪 Testing Affine Integration..."

# Test 1: List workspaces
echo "Test 1: List workspaces"
./picoclaw agent -m "List my Affine workspaces"

# Test 2: List pages
echo "Test 2: List pages"
./picoclaw agent -m "List pages in my Affine workspace"

# Test 3: Search
echo "Test 3: Search"
./picoclaw agent -m "Search my Affine notes for 'test'"

# Test 4: Create note
echo "Test 4: Create note"
./picoclaw agent -m "Create a note in Affine titled 'Codespace Test $(date)'"

echo "✅ Tests complete!"
```

Make it executable and run:

```bash
chmod +x test-affine.sh
./test-affine.sh
```

## 📚 Additional Resources

- **Affine Integration Guide**: [docs/AFFINE_INTEGRATION.md](docs/AFFINE_INTEGRATION.md)
- **Quick Start**: [AFFINE_QUICKSTART.md](AFFINE_QUICKSTART.md)
- **Implementation Details**: [AFFINE_IMPLEMENTATION_SUMMARY.md](AFFINE_IMPLEMENTATION_SUMMARY.md)
- **PicoClaw Docs**: [README.md](README.md)

## 💡 Tips

1. **Save Your Config**: Commit your config template (without secrets) to a private repo
2. **Use Secrets**: Store API keys in GitHub Secrets for CI/CD
3. **Test Locally First**: Test in Codespace before deploying
4. **Monitor Usage**: Check Affine API usage limits
5. **Version Control**: Create a branch for your changes

## 🤝 Contributing

Found an issue or want to improve the Affine integration?

1. Create a branch in Codespace
2. Make your changes
3. Run tests: `go test ./pkg/tools -v`
4. Commit and push
5. Create a pull request

## 🎉 You're Ready!

Your Codespace is now set up for PicoClaw development with Affine integration. Happy coding! 🚀
