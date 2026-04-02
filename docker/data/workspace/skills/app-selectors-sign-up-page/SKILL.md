---
name: app-selectors-sign-up-page
description: DOM selectors and component map for the Sign Up Page page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Sign Up Page — Component Map

> Generated: 2026-03-27T09:36:31.271Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Sign Up Page
**URL:** `/signup`


**Input Fields (3):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Email address | `text` | `.custom-drawer input[type="text"]` |
| 2 | Password | `password` | `.custom-drawer input[type="password"]` |
| 3 | Confirm Password | `password` | `.custom-drawer input[type="password"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

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

