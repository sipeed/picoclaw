# JANE Evolution Plan: Active Autonomous Task Execution

**Objective:** Evolve JANE from a passive information retriever into an active, first-principles agent capable of autonomous task execution.

## 1. Audit of Current Workflows & Friction Points
After analyzing JANE (PicoClaw)'s current workflow implementation and tools available in `pkg/tools` and `pkg/agent`, the following friction points have been identified where JANE can transition from "answering" to "doing":

*   **Action Limitation (The "Answering" Trap):** JANE's existing toolset is heavily geared toward data fetching (`web_search`, `web_fetch`, `read_file`) and answering user queries based on that fetched context. Actions are largely constrained to simple single-step primitives (`write_file`, `exec`, `browser_action`).
*   **Web Automation Rigidity:** The `browser_action` tool provides primitive Playwright-based navigation, clicking, typing, and extraction. However, it lacks continuous, scriptable control logic inside the sandbox, requiring a round-trip to the LLM for every single atomic browser operation.
*   **Complex Task Execution:** `go_eval` provides an excellent mechanism for dynamic Go code execution (via Yaegi), but it's isolated. The AI cannot easily build long-running daemons or multi-step orchestrations securely without running into iteration limits or blocking the main event loop.
*   **Approval & Flow Friction:** The lack of robust "Human-in-the-Loop" features (Phase 4 of `AGENT_LOOP_IMPROVEMENTS.md`) creates a barrier for high-risk system-level tasks.

## 2. Actionable Enhancements for Autonomous Skill Acquisition
To transition JANE into an active agent, we prioritize the following features:

### A. Advanced Sandboxed Scripting
*   **Enhance `go_eval` with Contextual Bindings:** JANE currently uses `yaegi` for sandboxed Go execution. We should inject pre-configured bindings for internal APIs (like `mcp`, `bus`, `browser`) directly into the Yaegi interpreter context. This allows JANE to write a *single script* that performs a complete workflow (e.g., "Navigate to site, scrape data, format it, and send a message") rather than invoking tools 20 times in a loop.

### B. High-level Automation & Orchestration
*   **Agent-Directed CI/CD execution:** Incorporate a lightweight pipeline execution engine (similar to Dagger) where JANE can dynamically generate and run a directed acyclic graph (DAG) of containerized tasks for build, test, and deployment workflows natively.
*   **Scriptable Browser Automation:** Instead of sending atomic commands (`click`, `type`) sequentially, enable JANE to generate a full automation script and run it through a lightweight embedded scripting language to improve latency and reliability.

### C. System-level Sandboxing
*   **Linux Landlock Integration:** For secure execution of external binaries and shell scripts, enhance the `exec` tool by utilizing Linux Landlock (via `go-landlock`) or lightweight virtualization (like `gVisor`). This will provide strict filesystem and network isolation, allowing JANE to safely run untrusted code without relying solely on simple directory restrictions.

## 3. Technical Intelligence: High-Momentum Go Repositories
Research of high-momentum Go-based repositories on GitHub has surfaced the following modular tools that can be integrated into JANE's core architecture for sandboxing, automation, and tool-calling:

### Sandboxing & Execution Isolation
*   **[gVisor](https://github.com/google/gvisor) (17.9K ⭐):** An application kernel that provides an isolation boundary between the application and the host kernel. Ideal for providing deep container-level sandboxing when JANE needs to execute complex, untrusted code autonomously.
*   **[go-landlock](https://github.com/landlock-lsm/go-landlock):** A Go library for the Linux Landlock sandboxing feature. This is a lightweight alternative to gVisor that can be directly integrated into the `shell`/`exec` tool to sandbox file system access at a kernel level without full virtualization.

### Automation & Scripting
*   **[Dagger](https://github.com/dagger/dagger) (15.5K ⭐):** An automation engine for building, testing, and shipping code. Integrating Dagger's Go SDK would allow JANE to write robust, containerized CI/CD pipelines natively as a set of autonomous skills.
*   **[go-rod](https://github.com/go-rod/rod) (6.8K ⭐):** A Chrome DevTools Protocol driver for web automation. It's lighter and more Go-native than Playwright, making it an excellent candidate to replace or augment `browser_action` for high-performance, scriptable browser tasks.
*   **[Tengo](https://github.com/d5/tengo) (3.7K ⭐):** A fast, lightweight script language for Go. While JANE prefers `yaegi` for Go syntax, Tengo could serve as a secure, sandboxed scripting layer for user-defined lightweight automation if Yaegi is too heavy for certain contexts.
*   **[Watchtower](https://github.com/containrrr/watchtower) (24.5K ⭐):** A process for automating Docker container base image updates. Its container management logic could inform how JANE manages containerized tools or handles self-updating capabilities.

### Action Plan Summary
1.  **Phase 1:** Upgrade the `go_eval` (Yaegi) tool by exposing internal JANE bindings (HTTP clients, browser contexts, file system access), empowering JANE to execute multi-step scripts in one go.
2.  **Phase 2:** Integrate `go-rod` to replace/enhance `browser_action` for deeper, more stable autonomous web navigation and interaction.
3.  **Phase 3:** Introduce `go-landlock` into the `exec` and `shell` tools to harden the existing workspace-level restrictions, allowing JANE to safely run complex bash scripts or binaries downloaded during autonomous task execution.