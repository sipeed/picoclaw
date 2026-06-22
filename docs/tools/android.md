# Android ADB Tool

This document describes the Android tool MVP.

The tool adds a small ADB-backed Android operations surface for explicitly configured devices. It is intended as a first implementation step for the Android Device Automation roadmap item.

## MVP scope

The MVP supports device discovery, basic status inspection, screenshot capture, UI hierarchy summary, and a small set of fixed input primitives.

The tool does not expose arbitrary shell execution. Each operation is implemented as a fixed action with validation.

## Safety model

The tool is designed to be fail-closed:

- it is disabled unless explicitly enabled;
- at least one device must be configured;
- mutating actions require explicit confirmation or dry-run mode;
- current app/package checks can be enforced;
- known sensitive package patterns are blocked by default;
- screenshots have a maximum size;
- every operation has a timeout;
- action rate limiting is applied;
- mutating actions are logged.

## Environment configuration

Initial configuration is environment based:

- `PICOCLAW_TOOLS_ANDROID_ENABLED`
- `PICOCLAW_TOOLS_ANDROID_ADB_PATH`
- `PICOCLAW_TOOLS_ANDROID_DEVICE_ID`
- `PICOCLAW_TOOLS_ANDROID_DEVICE_SERIAL`
- `PICOCLAW_TOOLS_ANDROID_ALLOW_PACKAGES`
- `PICOCLAW_TOOLS_ANDROID_BLOCK_PACKAGES`
- `PICOCLAW_TOOLS_ANDROID_TIMEOUT_MS`
- `PICOCLAW_TOOLS_ANDROID_MAX_ACTIONS_PER_MINUTE`
- `PICOCLAW_TOOLS_ANDROID_SCREENSHOT_MAX_BYTES`

## Follow-up work

A future iteration can add a lightweight Android or Termux agent backend while keeping the same tool interface. That would avoid requiring ADB for all deployment modes and would fit low-power Android devices better.
