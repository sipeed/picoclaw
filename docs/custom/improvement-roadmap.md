# Picoclaw Improvement Roadmap: Tools, Skills & Cronjobs

This document outlines how to expand Picoclaw's capabilities based on your specific homelab environment and agent roles (Alpha, Pulse, Forge, Atlas).

---

## 1. The Decision Matrix: Which one do I use?

To choose the right method, ask yourself **"When and how should this happen?"**

| Method | The Use Case | The "Why" | Example |
| :--- | :--- | :--- | :--- |
| **Tools** | **Real-time action/data.** When you ask "What's the price of ASML?" | Gives the LLM **active hands**. It allows the model to fetch fresh facts that weren't in its training data. | `fetch_ticker_price(symbol)` |
| **Skills** | **Consistent Logic.** When you want the LLM to analyze the *quality* of a stock using your specific methodology. | Shapes the **agent's brain**. It ensures the bot always follows your "investment thesis" framework every time it thinks. | `ibkr_analysis_skill` |
| **Cronjobs** | **Scheduled Automation.** When you want a summary at 8 AM every morning without you having to ask for it. | Provides **proactive duty**. It ensures the system is working for you 24/7, even when you aren't chatting with it. | `daily_backup.sh` |

---

## 2. Elaborating on Use Cases

### **Scenario A: "I want to know if my internet is slow right now."**
- **Winning Method:** **Tool**
- **Why?** You are in a live chat and need a data-point *right now*. A `run_speedtest` tool triggered by your question is the most efficient.

### **Scenario B: "I want to track my long-term weight trend and get a weekly report every Sunday."**
- **Winning Method:** **Cronjob + Tool**
- **Why?** Use a **Cronjob** to trigger the analysis every Sunday. The cronjob then uses a **SQLite Tool** to pull the data and send the report to your Telegram.

### **Scenario C: "I want my agent to follow the 'Getting Things Done' (GTD) method for my task list."**
- **Winning Method:** **Skill**
- **Why?** This isn't about an API or a schedule; it's about *how* the agent processes information. A "GTD Skill" contains the rules (folders, next actions, contexts) that the agent must ALWAYS follow when you talk about tasks.

### **Scenario D: "I want to be alerted if ASML drops below €800."**
- **Winning Method:** **Cronjob**
- **Why?** A "Tool" only works when you are talking. You need a background **Cronjob** that runs every 10 minutes, checks the price, and "pokes" you if the condition is met.

---

## 2. Recommended Additions by Agent

### **Alpha (Finance)**
*   **Tool:** `tavily_search` — Real-time news searching for your ticker watchlist.
*   **Skill:** `fundamental_analysis` — A set of instructions for reading IBKR data and flagging 10-K risk factors.
*   **Cronjob:** `portfolio_fetch` — Automatically download your IBKR Flex Query CSV every day at 6 PM.

### **Pulse (Fitness)**
*   **Tool:** `sqlite_query` — Query your local health database for long-term HRV or weight trends.
*   **Skill:** `recovery_logic` — Specific rules for when to recommend a "deload week" based on biometric inputs.
*   **Cronjob:** `morning_brief` — At 7 AM, analyze last night's sleep data and generate a summary waiting for you in Telegram.

### **Forge (Systems)**
*   **Tool:** `ssh_run` — Allow the agent to run commands directly on **Argus** or **Vulcan** for remote debugging.
*   **Skill:** `refactor_patterns` — Knowledge of Go/Python best practices to improve code quality during review.
*   **Cronjob:** `heartbeat_monitor` — Every hour, ping all devices in the lab and alert you if one goes offline.

### **Atlas (Org/Local)**
*   **Tool:** `calendar_api` — Link specifically to an `ical` or Google Calendar to add/remove appointments.
*   **Cronjob:** `daily_planner` — Every Sunday evening, summarize your upcoming week based on your notes.

---

## 3. Tool Discovery: Web Search
**Great news:** Picoclaw already has a powerful `web_search` tool built into `pkg/tools/web.go`. It supports:
- **Tavily** & **Brave Search** (Recommended for pure data)
- **Perplexity** (Best for analytical answers)
- **DuckDuckGo** (Free, no API key needed)

To enable these, you simply need to add your API keys to the `config.json` file we built earlier.

---

## 4. Setup Guide: The "Morning Briefing"
To turn Picoclaw into a proactive assistant that messages you every morning with market and macro news, follow this framework:

### **Phase 1: The Command Script**
Create a script on your Pi (e.g., `~/scripts/morning_brief.sh`):
```bash
#!/bin/bash
# Trigger Alpha to generate a briefing and send it to your Telegram ID
picoclaw --agent alpha --cmd "Search for overnight news on ASML, NVDA and the 10Y Treasury yield. Summarize the macro sentiment for today. Check ~/picoclaw-data/workspace/ibkr for any updates. Send result as a Morning Briefing."
```

### **Phase 2: The Cron Trigger**
Add this to your Pi's crontab (`crontab -e`):
```bash
# Run the briefing every weekday at 7:30 AM
30 07 * * 1-5 /home/tim/scripts/morning_brief.sh
```

### **Why this works:**
1.  **Proactive:** You don't have to remember to check the news; it hits your phone while you're having coffee.
2.  **Context-Aware:** Because it uses the **Alpha** agent, it already knows your watchlist and your portfolio goals.
3.  **Low Latency:** By the time you wake up, the "heavy lifting" (searching and summarizing) is already done.

---

## 5. More Brainstorming Ideas

### **Forge (Systems)**
- **Tool:** `log_audit` — Automatically scan `journalctl` for Picoclaw errors and summarize them.
- **Skill:** `pi_safety_rules` — Explicit rules to never exceed 400MB RAM in generated code.
- **Cronjob:** `vulcan_health_check` — Ping your desktop every hour to ensure the Ollama host is reachable.

### **Pulse (Health)**
- **Skill:** `hrv_interpreter` — A framework to tell you *exactly* how to adjust your workout based on last night's HRV.
- **Cronjob:** `evening_reminder` — At 9 PM, check if you logged your supplements; if not, send a gentle nudge.
- **Model Override:** Force a higher-tier model for specific Pulse tasks (e.g., weekly health audit) using the `model` payload field.

---

## 6. Model Tiering & Routing Overrides

Picoclaw uses a multi-tier routing system to balance performance and cost on the Pi Zero 2W.

### **The Model Tiers**
| Tier | Model | Best For |
| :--- | :--- | :--- |
| **Light** | `gemini-1.5-flash-lite` | Simple commands, data retrieval, basic chat. (Default for most Pulse/Atlas agents) |
| **Mid** | `gemini-1.5-flash` | Reasoning tasks, complex tool orchestration, small code refactors. |
| **High** | `claude-3-5-sonnet` | Critical decision making, complex coding, high-stakes analysis. |

### **The Override Mechanism**
While the `RuleClassifier` automatically promotes tasks to higher tiers based on complexity, you can **force** a specific model for repeated tasks (like Cronjobs) to ensure consistent quality.

#### **Cron Override Example:**
In your `agent.yaml` or cron payload, add the `model` field:
```yaml
payload:
  message: "Run a deep audit of the system logs and suggest security improvements."
  model: "gemini-1.5-flash" # Forces Mid-tier logic for this specific task
```
