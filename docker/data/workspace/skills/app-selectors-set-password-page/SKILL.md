---
name: app-selectors-set-password-page
description: DOM selectors and component map for the Set Password Page page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Set Password Page — Component Map

> Generated: 2026-03-27T09:36:39.194Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Set Password Page
**URL:** `/auth/set-password`

**Headings:**
- `h2` — "Set Password"


**Text Content (1):**
- [p] "Please set your password to activate your account."

**Input Fields (2):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Password | `password` | `.custom-drawer input[type="password"]` |
| 2 | Confirm Password | `password` | `.custom-drawer input[type="password"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Buttons (1):**
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto mb-5 w-100 text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`

**Custom Elements & IDs (11):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Set PasswordPlease set your password to activate your accoun |
| `.toolbar-set-password` | `header` | `toolbar-set-password` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.set-password-container` | `div` | `set-password-container` | Set PasswordPlease set your password to activate your accoun |
| `.set-password-card` | `div` | `set-password-card` | Set PasswordPlease set your password to activate your accoun |
| `` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Set Password |

---

