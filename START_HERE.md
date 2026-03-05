# 🚀 START HERE - Complete Setup Guide

## 📍 Current Situation

✅ **What's Done:**
- All Affine integration code is created
- Codespace configuration is ready
- Documentation is complete
- Tests are written

❌ **What You Need to Do:**
1. Push code to GitHub (from your Windows machine)
2. Open GitHub Codespace (in your browser)
3. Configure Affine credentials
4. Test it!

---

## 🎯 Three Simple Steps

### Step 1️⃣: Push to GitHub (5 minutes)

**On your Windows machine**, open PowerShell or Git Bash:

```bash
# Navigate to your picoclaw folder
cd C:\Users\jackwang\Documents\picoclaw\picoclaw

# Add all files
git add .

# Commit
git commit -m "Add Affine integration"

# Push to GitHub
git push origin main
```

**Done?** ✅ Go to your GitHub repo in browser to verify files are there.

---

### Step 2️⃣: Open Codespace (3 minutes)

**In your browser:**

1. Go to your GitHub repository
2. Click green **"Code"** button
3. Click **"Codespaces"** tab
4. Click **"Create codespace on main"**
5. Wait 2-3 minutes for setup

**Done?** ✅ You should see VS Code in your browser.

---

### Step 3️⃣: Configure & Test (5 minutes)

**In the Codespace terminal:**

```bash
# 1. Get your Affine API key first!
# Go to: https://app.affine.pro → Settings → API Keys → Generate

# 2. Edit config
code ~/.picoclaw/config.json

# 3. Update the affine section with your credentials:
#    - api_key: "your-key-here"
#    - workspace_id: "your-workspace-id"

# 4. Save (Ctrl+S)

# 5. Test it!
./picoclaw agent -m "List my Affine workspaces"
```

**Done?** ✅ You should see your Affine workspaces listed!

---

## 🎉 That's It!

If all three steps worked, you're done! 

Try these commands:

```bash
# List pages
./picoclaw agent -m "List my Affine pages"

# Search
./picoclaw agent -m "Search my notes for 'test'"

# Create a note
./picoclaw agent -m "Create a note titled 'Hello from PicoClaw!'"
```

---

## 📚 Detailed Guides

Need more help? Check these:

| Guide | What's Inside |
|-------|---------------|
| **[CHECKLIST.md](CHECKLIST.md)** | Step-by-step checklist with checkboxes |
| **[GETTING_STARTED_CODESPACE.md](GETTING_STARTED_CODESPACE.md)** | Detailed Codespace instructions |
| **[AFFINE_QUICKSTART.md](AFFINE_QUICKSTART.md)** | Quick examples and use cases |
| **[docs/AFFINE_INTEGRATION.md](docs/AFFINE_INTEGRATION.md)** | Complete documentation |
| **[CODESPACE_SETUP.md](CODESPACE_SETUP.md)** | Codespace troubleshooting |

---

## 🆘 Quick Troubleshooting

### "Git push failed"
```bash
gh auth login
# Or use personal access token from github.com/settings/tokens
```

### "Can't find Codespaces"
- Make sure you're logged into GitHub
- Check if your account has Codespaces access
- Try refreshing the page

### "Affine API error"
- Double-check your API key (no spaces!)
- Verify workspace ID is correct
- Try generating a new API key

### "Command not found"
```bash
# Rebuild
make build

# Or run setup again
bash .devcontainer/setup.sh
```

---

## 🎯 Visual Workflow

```
┌─────────────────────────────────────────────────────────────┐
│  YOUR WINDOWS MACHINE                                       │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. git add .                                          │ │
│  │ 2. git commit -m "Add Affine integration"            │ │
│  │ 3. git push origin main                              │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  GITHUB.COM (in browser)                                    │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. Go to your repository                             │ │
│  │ 2. Click "Code" → "Codespaces"                       │ │
│  │ 3. Click "Create codespace on main"                  │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  GITHUB CODESPACE (VS Code in browser)                     │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 1. code ~/.picoclaw/config.json                      │ │
│  │ 2. Add your Affine API key & workspace ID            │ │
│  │ 3. Save (Ctrl+S)                                     │ │
│  │ 4. ./picoclaw agent -m "List my Affine workspaces"  │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                    ✅ SUCCESS! 🎉
```

---

## 💡 What You'll Be Able to Do

Once setup is complete, you can:

✅ List all your Affine workspaces
✅ Browse pages with tags
✅ Search across all your notes
✅ Read full page content
✅ Create new notes with tags
✅ Update existing pages
✅ View workspace structure

All through natural language commands to PicoClaw!

---

## 🚀 Ready to Start?

**Right now, on your Windows machine:**

1. Open PowerShell or Git Bash
2. Navigate to: `C:\Users\jackwang\Documents\picoclaw\picoclaw`
3. Run: `git status`
4. Follow Step 1️⃣ above

**You got this!** 💪

---

**Questions?** Check [CHECKLIST.md](CHECKLIST.md) for detailed steps with troubleshooting.
