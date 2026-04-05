# ✅ Affine Integration Setup Checklist

Follow this checklist to get everything working!

## 📍 Where You Are Now

You're on your Windows machine with all the code files created. Now you need to push to GitHub and test in Codespace.

---

## Part 1: Push to GitHub (On Your Windows Machine)

### ☐ Step 1: Check Git Status

Open PowerShell or Git Bash in your picoclaw directory:

```bash
cd C:\Users\jackwang\Documents\picoclaw\picoclaw
git status
```

You should see many new/modified files listed.

### ☐ Step 2: Stage All Changes

```bash
git add .
```

### ☐ Step 3: Commit Changes

```bash
git commit -m "Add Affine integration with Codespace support

- Add Affine tool implementation (pkg/tools/affine.go)
- Add Affine configuration support
- Add Codespace development environment
- Add comprehensive documentation
- Add unit tests"
```

### ☐ Step 4: Push to GitHub

```bash
# If you're working on main branch:
git push origin main

# Or if you're on a different branch:
git push origin YOUR_BRANCH_NAME
```

**✅ Checkpoint:** Visit your GitHub repository in a browser. You should see all the new files!

---

## Part 2: Get Affine Credentials (While Waiting)

### ☐ Step 5: Log in to Affine

Open browser and go to: https://app.affine.pro

### ☐ Step 6: Generate API Key

1. Click your avatar (top right)
2. Click **Settings**
3. Go to **API Keys** section
4. Click **Generate New Key**
5. **Copy and save the API key** (you'll need this soon!)

### ☐ Step 7: Get Workspace ID

Look at your browser URL:
```
https://app.affine.pro/workspace/abc123xyz456
                                 ^^^^^^^^^^^^^^
                                 This is your workspace ID
```

**Copy and save your workspace ID!**

**✅ Checkpoint:** You should have:
- ✅ API Key (looks like: `affine_xxxxxxxxxxxxx`)
- ✅ Workspace ID (looks like: `abc123xyz456`)

---

## Part 3: Open GitHub Codespace

### ☐ Step 8: Navigate to Your Repository

Go to: `https://github.com/YOUR_USERNAME/picoclaw`

(Or if it's a fork: `https://github.com/sipeed/picoclaw`)

### ☐ Step 9: Create Codespace

1. Click the green **"Code"** button (top right)
2. Click the **"Codespaces"** tab
3. Click **"Create codespace on main"**

### ☐ Step 10: Wait for Setup (2-3 minutes)

You'll see a progress screen. The setup script will:
- Install Go
- Download dependencies
- Build PicoClaw
- Create config directory

**✅ Checkpoint:** You should see VS Code in your browser with a terminal at the bottom.

---

## Part 4: Configure Affine in Codespace

### ☐ Step 11: Open Config File

In the Codespace terminal, run:

```bash
code ~/.picoclaw/config.json
```

### ☐ Step 12: Find Affine Section

Scroll down to find the `"affine"` section (around line 50-60).

### ☐ Step 13: Update Credentials

Replace these values:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "PASTE_YOUR_API_KEY_HERE",
      "workspace_id": "PASTE_YOUR_WORKSPACE_ID_HERE",
      "timeout_seconds": 30
    }
  }
}
```

### ☐ Step 14: Save Config

Press `Ctrl+S` (or `Cmd+S` on Mac) to save.

**✅ Checkpoint:** Your config file should have your real API key and workspace ID.

---

## Part 5: Test the Integration

### ☐ Step 15: Verify Build

In the Codespace terminal:

```bash
./picoclaw version
```

You should see version information.

### ☐ Step 16: Test List Workspaces

```bash
./picoclaw agent -m "List my Affine workspaces"
```

**Expected output:** List of your workspaces with names and IDs.

### ☐ Step 17: Test List Pages

```bash
./picoclaw agent -m "List all pages in my Affine workspace"
```

**Expected output:** List of pages with titles and tags.

### ☐ Step 18: Test Search

```bash
./picoclaw agent -m "Search my Affine notes for 'test'"
```

**Expected output:** Search results (or "No results found").

### ☐ Step 19: Test Create Page

```bash
./picoclaw agent -m "Create a note in Affine titled 'Codespace Test' with content 'Hello from PicoClaw!'"
```

**Expected output:** Confirmation that page was created with an ID.

### ☐ Step 20: Verify in Affine

1. Go back to https://app.affine.pro
2. Check if you see the new "Codespace Test" page
3. Open it to verify the content

**✅ Checkpoint:** You should see your new page in Affine!

---

## Part 6: Run Unit Tests

### ☐ Step 21: Run Affine Tests

```bash
go test ./pkg/tools -v -run TestAffineTool
```

**Expected output:** All tests should pass (PASS).

### ☐ Step 22: Run All Tool Tests

```bash
go test ./pkg/tools -v
```

**Expected output:** All tests pass.

---

## 🎉 Success Criteria

You've successfully completed the setup if:

- ✅ All code is pushed to GitHub
- ✅ Codespace is running
- ✅ PicoClaw is built
- ✅ Config has your Affine credentials
- ✅ `./picoclaw agent -m "List my Affine workspaces"` works
- ✅ You can create a test note in Affine
- ✅ Unit tests pass

---

## 🐛 Troubleshooting

### Problem: Git push fails with authentication error

**Solution:**
```bash
# Use GitHub CLI
gh auth login

# Or create a personal access token:
# 1. Go to https://github.com/settings/tokens
# 2. Generate new token with 'repo' scope
# 3. Use token as password when pushing
```

### Problem: Codespace setup fails

**Solution:**
```bash
# Run setup manually
bash .devcontainer/setup.sh

# Or build manually
go mod download
make build
```

### Problem: "Affine API error: 401 Unauthorized"

**Solution:**
- Check your API key is correct (no extra spaces)
- Verify the key hasn't expired
- Generate a new key if needed

### Problem: "Workspace not found"

**Solution:**
- Verify your workspace ID is correct
- Check you have access to the workspace
- Try listing workspaces first to see available IDs

### Problem: Tests fail

**Solution:**
```bash
# Some tests may fail without real API credentials
# That's OK! The important tests are:
go test ./pkg/tools -v -run TestAffineTool_Name
go test ./pkg/tools -v -run TestAffineTool_Parameters
```

---

## 📚 Next Steps After Setup

Once everything works:

1. **Explore Features:**
   - Try different search queries
   - Create notes with tags
   - Update existing pages
   - Get workspace structure

2. **Read Documentation:**
   - [AFFINE_QUICKSTART.md](AFFINE_QUICKSTART.md) - Quick examples
   - [docs/AFFINE_INTEGRATION.md](docs/AFFINE_INTEGRATION.md) - Complete guide
   - [CODESPACE_SETUP.md](CODESPACE_SETUP.md) - Codespace details

3. **Customize:**
   - Modify the tool if needed
   - Add new features
   - Contribute back to the project!

---

## 📞 Need Help?

- **Discord**: https://discord.gg/V4sAZ9XWpN
- **GitHub Issues**: https://github.com/sipeed/picoclaw/issues
- **Documentation**: Check the docs/ folder

---

**Current Step:** Start with Part 1, Step 1 (Check Git Status) on your Windows machine! 🚀
