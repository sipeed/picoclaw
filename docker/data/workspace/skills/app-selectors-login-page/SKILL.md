---
name: app-selectors-login-page
description: DOM selectors and component map for the Login Page page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Login Page — Component Map

> Generated: 2026-03-27T09:36:27.299Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Login Page
**URL:** `/login`


**Input Fields (2):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Email address | `text` | `.custom-drawer input[type="text"]` |
| 2 | Password | `password` | `.custom-drawer input[type="password"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Buttons (4):**
- `page.locator('button:has-text("Forgot Password")')`
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto mb-5 w-100 text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-outlined m-auto mb-5 w-100 text-capitalize font-weight-bold`
- `page.locator('button:has-text("Sign up")')`

**Custom Elements & IDs (11):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `.toolbar-login` | `header` | `toolbar-login` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.login-container` | `div` | `login-container` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `.login-card` | `div` | `login-card` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Login |

---

