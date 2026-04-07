# Git Workflow Guide

This guide breaks down how to manage your custom `picoclaw` logic alongside upstream official releases.

## Core Concepts
- **Forking** is for **Taking Ownership**: It creates an independent copy of an entire project repository on your GitHub account so you have full read/write admin controls.
- **Branching** is for **Making Changes**: It isolates a specific line of code history (e.g., `feature/wrapper`) so you can experiment without breaking your working `main` branch.

## The "One Custom Branch" Strategy (Best & Lowest Effort)

For maintaining a personal Raspberry Pi deployment, the cleanest, least confusing, and absolute lowest effort method is to literally never switch branches again. Instead of juggling dozens of feature branches, you maintain a single dedicated branch (e.g., `custom-pi`) that holds all your custom logic. 

**Initial Setup:**
1. **The Fork**: You forked `sipeed/picoclaw` to `TimZickenrott/picoclaw`.
2. **The Clone**: You downloaded your fork to your PC.
3. **The Branch**: You started on `feature/wrapper`. Let's rename this to be your permanent deploy branch:
   ```bash
   git branch -m feature/wrapper custom-pi
   ```

From now on, you always work on and deploy from `custom-pi`. 

### How to Save Your Work (Pushing)
When you write new custom code or edit features, just commit and push directly to `custom-pi` to save it:
```bash
git add .
git commit -m "Added my new custom logic"
git push origin custom-pi
```

## How To Pull Official Updates (Merging)

When the official Sipeed developers release a new version (e.g., bugfixes or new capabilities), you want to pull those into your `custom-pi` branch so you get the newest features without losing your wrapper code. 

Here is exactly how you do it, all from your `custom-pi` branch:
*(Assuming you already added the original repo as an 'upstream' remote)*
```bash
# 1. Download the newest official code
git fetch upstream

# 2. Smoothly merge the newest official features directly into your custom code
git merge upstream/main
```

### How Git Resolves This
- **The Safe Merge:** If Sipeed changed completely different files than you did (e.g., they fixed a bug in `telegram.go`, but you modified `loop.go`), Git will automatically splice the files together perfectly silently. Your wrapper stays intact.
- **The Merge Conflict:** If the developers modified the *exact same* lines of code in `loop.go` that you modified, Git will pause the merge and declare a "Merge Conflict."
  - You open `loop.go` in your IDE. You will see markers (`<<<<<<`) showing their new code alongside your custom code. 
  - You manually edit that block to make sure your wrapper surrounds their new logic, save it, and run `git merge --continue`.

Since your custom wrapper is self-contained and small, adjusting the code during a conflict takes barely a minute!
