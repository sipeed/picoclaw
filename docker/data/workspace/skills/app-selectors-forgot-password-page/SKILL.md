---
name: app-selectors-forgot-password-page
description: DOM selectors and component map for the Forgot Password Page page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Forgot Password Page — Component Map

> Generated: 2026-03-26T12:23:48.380Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Forgot Password Page
**URL:** `/forgot-password`

**Headings:**
- `h2` — "Forgot your Password"


**Text Content (2):**
- [p] "Please provide the email address that the invitation was sent to"
- [text-center] "The link will be sent to your email"

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Email address | `text` | `.custom-drawer input[type="text"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Buttons (2):**
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto mb-5 w-100 text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`
- `page.locator('.sign-in-button')`
  classes: `sign-in-button`

**Custom Elements & IDs (9):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Forgot your PasswordPlease provide the email address that th |
| `.toolbar-forgot-password` | `header` | `toolbar-forgot-password` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.forgot-password-container` | `div` | `forgot-password-container` | Forgot your PasswordPlease provide the email address that th |
| `.forgot-password-card` | `div` | `forgot-password-card` | Forgot your PasswordPlease provide the email address that th |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Send Email |
| `.sign-in-button` | `button` | `sign-in-button` | Back to sign in |

---

