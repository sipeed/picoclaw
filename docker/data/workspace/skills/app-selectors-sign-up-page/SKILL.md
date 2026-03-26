---
name: app-selectors-sign-up-page
description: DOM selectors and component map for the Sign Up Page page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Sign Up Page — Component Map

> Generated: 2026-03-26T07:04:19.075Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Sign Up Page
**URL:** `/signup`


**Input Fields (3):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Email address | `text` | `page.locator('.v-field__input').nth(0)` |
| 2 | Password | `password` | `page.locator('.v-field__input').nth(1)` |
| 3 | Confirm Password | `password` | `page.locator('.v-field__input').nth(2)` |

**Buttons (2):**
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto mb-5 w-100 text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`
- `page.locator('button:has-text("Sign in")')`

**Custom Elements & IDs (13):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Email address * Password * Confirm Password * Sign UpAlready |
| `.toolbar-sign-up` | `header` | `toolbar-sign-up` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.sign-up-container` | `div` | `sign-up-container` | Email address * Password * Confirm Password * Sign UpAlready |
| `.sign-up-card` | `div` | `sign-up-card` | Email address * Password * Confirm Password * Sign UpAlready |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Sign Up |

---

