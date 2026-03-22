# 2026-03-23 Picoclaw Weixin Command And Config Alignment

## Summary

Aligned the existing native `weixin` implementation with the workflow used in the desktop-facing `agent/` changes by making the login path and sample configuration directly usable.

## Changes

- Added `picoclaw channels weixin login`
- Added command aliases:
  - `picoclaw onboard wechat`
  - `picoclaw channels wechat login`
- Added the missing `channels.weixin` section to `config/config.example.json`
- Updated `docs/channels/weixin/README.md` to document both login entrypoints and the full config fields used by the native channel

## Why

`picoclaw-clone` already had the native Weixin protocol layer, QR login flow, polling loop, and media support. The remaining gap was usability:

- users could follow docs but not see `channels.weixin` in the shipped example config
- the command surface did not match the newer `channels ... login` workflow
- the full `base_url` / `cdn_base_url` fields were not documented in the main Weixin README

## Result

The repo now exposes a complete Weixin setup path through:

1. `picoclaw onboard weixin`
2. `picoclaw channels weixin login`
3. `config/config.example.json` sample configuration
4. native `pkg/channels/weixin` runtime support already present in the codebase
