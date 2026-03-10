---
name: playwright-e2e
description: Run browser-based E2E tests on web apps. Use when asked to test
UI flows, search features, login, or any user interaction without pre-written test files.
---

# Playwright E2E Testing Skill

Use this skill for any browser-based testing without pre-written test files.

## General Selector Strategy
- Prefer `[data-testid='...']` attributes when available
- Fall back to semantic selectors: `button`, `input[type='text']`, `a[href='...']`
- Avoid brittle selectors like CSS classes or XPath unless necessary

## SPA Navigation Pattern
1. Set up `waitForResponse()` or `waitForSelector()` BEFORE `goto()`
2. Navigate with `goto()`
3. Await the response/selector promise
4. Only then interact with elements

## General Test Flow
1. Launch browser and navigate to the target URL
2. Identify the elements to interact with using browser DevTools or page source
3. Write and run the test steps sequentially
4. Assert expected outcomes after each key interaction

## Reporting Format
✅ PASS  Step N: description
❌ FAIL  Step N: reason
Result: X/Y PASSED