# Getting Started with GitHub Codespaces

## 📋 What We Just Created

I've created all the Affine integration code and Codespace configuration files. Now you need to:
1. Push these changes to GitHub
2. Open a Codespace
3. Test the integration

## 🚀 Step-by-Step Instructions

### Step 1: Push Changes to GitHub

On your Windows machine, run these commands in PowerShell or Git Bash:

```bash
# Navigate to your picoclaw directory (if not already there)
cd C:\Users\jackwang\Documents\picoclaw\picoclaw

# Check what files were created/modified
git status

# Add all the new files
git add .

# Commit the changes
git commit -m "Add Affine integration with Codespace support"

# Push to GitHub (replace 'main' with your branch name if different)
git push origin main
```

**If you don't have Git configured yet:**

```bash
# Configure Git (first time only)
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# Then run the commands above
```

### Step 2: Open GitHub Codespace

1. **Go to your GitHub repository**
   - Open your browser
   - Navigate to: `https://github.com/YOUR_USERNAME/picoclaw`
   - (Or if it's a fork: `https://github.com/sipeed/picoclaw`)

2. **Create a Codespace**
   - Click the green **"Code"** button (top right)
   - Click the **"Codespaces"** tab
   - Click **"Create codespace on main"**
   
   ![Codespace Button](https://docs.github.com/assets/cb-77061/mw-1440/images/help/codespaces/new-codespace-button.webp)

3. **Wait for Setup** (2-3 minutes)
   - Codespace will automatically:
     - ✅ Install Go 1.23
     - ✅ Download dependencies
     - ✅ Build PicoClaw
     - ✅ Run setup script
     - ✅ Create config directory

### Step 3: Configure Affine in Codespace

Once your Codespace opens, you'll see VS Code in your browser.

**In the Codespace terminal:**

```bash
# 1. Edit the config file
code ~/.picoclaw/config.json

# 2. Find the "affine" section and update it:
```

Update these values in the config:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_ACTUAL_API_KEY_HERE",
      "workspace_id": "YOUR_ACTUAL_WORKSPACE_ID_HERE",
      "timeout_seconds": 30
    }
  }
}
```

**Save the file** (Ctrl+S or Cmd+S)

### Step 4: Get Your Affine Credentials

**While the Codespace is setting up, get your Affine API key:**

1. Open a new browser tab
2. Go to https://app.affine.pro
3. Log in to your account
4. Click your avatar (top right) → **Settings**
5. Go to **"API Keys"** section
6. Click **"Generate New Key"**
7. **Copy the API key** (save it somewhere safe!)
8. **Copy your workspace ID** from the URL:
   - URL looks like: `https://app.affine.pro/workspace/abc123xyz`
   - Your workspace ID is: `abc123xyz`

### Step 5: Test the Integration

In your Codespace terminal:

```bash
# Test 1: Verify PicoClaw is built
./picoclaw version

# Test 2: List your Affine workspaces
./picoclaw agent -m "List my Affine workspaces"

# Test 3: List pages
./picoclaw agent -m "List all pages in my Affine workspace"

# Test 4: Search
./picoclaw agent -m "Search my Affine notes for 'test'"

# Test 5: Create a test note
./picoclaw agent -m "Create a note in Affine titled 'Hello from Codespace!'"
```

## 🎯 Quick Reference

### Files Created

Here's what I created for you:

```
📁 Your Repository
├── 📄 pkg/tools/affine.go              # Main Affine tool implementation
├── 📄 pkg/tools/affine_test.go         # Unit tests
├── 📄 pkg/config/config.go             # Updated with Affine config
├── 📄 pkg/agent/instance.go            # Updated to register Affine tool
├── 📄 config/config.example.json       # Updated with Affine section
├── 📁 .devcontainer/
│   ├── 📄 devcontainer.json            # Codespace configuration
│   └── 📄 setup.sh                     # Automatic setup script
├── 📁 .github/workflows/
│   └── 📄 codespace-test.yml           # CI/CD for testing
├── 📁 docs/
│   └── 📄 AFFINE_INTEGRATION.md        # Complete documentation
├── 📄 AFFINE_QUICKSTART.md             # Quick start guide
├── 📄 AFFINE_IMPLEMENTATION_SUMMARY.md # Technical details
├── 📄 CODESPACE_SETUP.md               # Codespace setup guide
└── 📄 GETTING_STARTED_CODESPACE.md     # This file!
```

### Common Commands in Codespace

```bash
# Build PicoClaw
make build

# Run tests
go test ./pkg/tools -v -run TestAffineTool

# Test Affine integration
./picoclaw agent -m "Your command here"

# View logs
./picoclaw agent -m "Your command" --verbose

# Edit config
code ~/.picoclaw/config.json
```

## 🐛 Troubleshooting

### "Git push failed"

If you get authentication errors:

```bash
# Use GitHub CLI to authenticate
gh auth login

# Or use personal access token
# Go to: https://github.com/settings/tokens
# Generate a token with 'repo' scope
# Use it as your password when pushing
```

### "Codespace won't start"

- Wait a few minutes and try again
- Check GitHub status: https://www.githubstatus.com/
- Try creating a new Codespace

### "Setup script failed"

In the Codespace terminal:

```bash
# Run setup manually
bash .devcontainer/setup.sh

# Or step by step:
go mod download
make build
mkdir -p ~/.picoclaw
cp config/config.example.json ~/.picoclaw/config.json
```

### "Affine API errors"

```bash
# Verify your API key is set correctly
cat ~/.picoclaw/config.json | grep api_key

# Test API connectivity
curl -X POST https://app.affine.pro/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"query":"{ workspaces { id name } }"}'
```

## 📱 Alternative: Use GitHub.dev

If Codespaces isn't available, you can use GitHub.dev (lightweight editor):

1. Go to your repository on GitHub
2. Press `.` (period key) on your keyboard
3. This opens VS Code in the browser
4. You can edit files but can't run/test code

Then you'll need to:
- Clone to a machine with Go installed
- Or wait for Codespaces access

## 💡 What Happens Next?

Once you're in the Codespace and have configured Affine:

1. **Test the integration** with the commands above
2. **Try different operations**:
   - Create notes
   - Search your workspace
   - Update existing pages
   - Organize with tags

3. **Develop further** if needed:
   - Modify `pkg/tools/affine.go`
   - Add new features
   - Run tests: `go test ./pkg/tools -v`

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "Configure Affine integration"
   git push
   ```

## 🎉 You're All Set!

After following these steps, you'll have:
- ✅ All code pushed to GitHub
- ✅ A working Codespace with Go environment
- ✅ PicoClaw built and ready to use
- ✅ Affine integration configured and tested

**Next Steps:**
1. Push the code to GitHub (Step 1 above)
2. Open a Codespace (Step 2 above)
3. Configure Affine (Step 3 above)
4. Start testing! (Step 5 above)

Need help? Check:
- 📖 [CODESPACE_SETUP.md](CODESPACE_SETUP.md) - Detailed Codespace guide
- 📖 [AFFINE_QUICKSTART.md](AFFINE_QUICKSTART.md) - Affine quick start
- 📖 [docs/AFFINE_INTEGRATION.md](docs/AFFINE_INTEGRATION.md) - Complete documentation

---

**Ready to start?** Run the Git commands in Step 1 on your Windows machine! 🚀
