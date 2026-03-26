---
name: app-selectors-public
description: >
  Comprehensive DOM selectors and component map for public pages on dashboard.int3nt.info.
  Use this skill when writing or fixing Playwright tests. Contains verified
  selectors for every page: headings, text, inputs, buttons, dropdowns,
  checkboxes, switches, tables, cards, chips, tabs, modals, sheets, expansion
  panels, navigation, alerts, and more.
---

# dashboard.int3nt.info — Public Pages Component Map

> Generated: 2026-03-26T01:24:58.963Z
> Never use auto-generated IDs (`input-v-N`) — they change on re-render.
> Never match dynamic-text buttons by exact text — use patterns or position.

## Global Login Flow

```
Step 1 — Navigate to login:
  await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

Step 2 — Fill credentials:
  await page.locator('.v-field__input').nth(0).fill('EMAIL');
  await page.locator('.v-field__input').nth(1).fill('PASSWORD');
  await page.locator('button:has-text("Login")').click();

Step 3 — Wait for redirect (REQUIRED):
  await page.waitForURL(/\?select_org/, { timeout: 15000 });

Step 4 — Select organization:
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

---

## Public Pages

### Login Page
**URL:** `/login`


**Input Fields (2):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Email address | `text` | `page.locator('.v-field__input').nth(0)` |
| 2 | Password | `password` | `page.locator('.v-field__input').nth(1)` |

**Buttons (4):**
- `page.locator('button:has-text("Forgot Password")')`
- `page.locator('button:has-text("Login")')`
- `page.locator('button:has-text("Sign in with SSO")')`
- `page.locator('button:has-text("Sign up")')`

**Custom Elements & IDs (11):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `.toolbar-login` | `header` | `toolbar-login` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.login-container` | `div` | `login-container` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `.login-card` | `div` | `login-card` | Email address * Password * Forgot PasswordLoginOR Sign in wi |
| `#input-v-1` | `input` | `` |  |
| `#input-v-1-messages` | `div` | `` |  |
| `#input-v-4` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#input-v-4-messages` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Login |

---

### Sign Up Page
**URL:** `/signup`


**Input Fields (3):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Email address | `text` | `page.locator('.v-field__input').nth(0)` |
| 2 | Password | `password` | `page.locator('.v-field__input').nth(1)` |
| 3 | Confirm Password | `password` | `page.locator('.v-field__input').nth(2)` |

**Buttons (2):**
- `page.locator('button:has-text("Sign Up")')`
- `page.locator('button:has-text("Sign in")')`

**Custom Elements & IDs (13):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Email address * Password * Confirm Password * Sign UpAlready |
| `.toolbar-sign-up` | `header` | `toolbar-sign-up` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.sign-up-container` | `div` | `sign-up-container` | Email address * Password * Confirm Password * Sign UpAlready |
| `.sign-up-card` | `div` | `sign-up-card` | Email address * Password * Confirm Password * Sign UpAlready |
| `#input-v-1` | `input` | `` |  |
| `#input-v-1-messages` | `div` | `` |  |
| `#input-v-4` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#input-v-4-messages` | `div` | `` |  |
| `#input-v-7` | `input` | `` |  |
| `#input-v-7-messages` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Sign Up |

---

### Forgot Password Page
**URL:** `/forgot-password`

**Headings:**
- `h2` — "Forgot your Password"


**Text Content (2):**
- [p] "Please provide the email address that the invitation was sent to"
- [text-center] "The link will be sent to your email"

**Input Fields (1):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Email address | `text` | `page.locator('.v-field__input').nth(0)` |

**Buttons (2):**
- `page.locator('button:has-text("Send Email")')`
- `page.locator('button:has-text("Back to sign in")')`

**Custom Elements & IDs (9):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Forgot your PasswordPlease provide the email address that th |
| `.toolbar-forgot-password` | `header` | `toolbar-forgot-password` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.forgot-password-container` | `div` | `forgot-password-container` | Forgot your PasswordPlease provide the email address that th |
| `.forgot-password-card` | `div` | `forgot-password-card` | Forgot your PasswordPlease provide the email address that th |
| `#input-v-1` | `input` | `` |  |
| `#input-v-1-messages` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Send Email |
| `.sign-in-button` | `button` | `sign-in-button` | Back to sign in |

---

### Set Password Page
**URL:** `/auth/set-password`

**Headings:**
- `h2` — "Set Password"


**Text Content (1):**
- [p] "Please set your password to activate your account."

**Input Fields (2):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Password | `password` | `page.locator('.v-field__input').nth(0)` |
| 2 | Confirm Password | `password` | `page.locator('.v-field__input').nth(1)` |

**Buttons (1):**
- `page.locator('button:has-text("Set Password")')`

**Custom Elements & IDs (11):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Set PasswordPlease set your password to activate your accoun |
| `.toolbar-set-password` | `header` | `toolbar-set-password` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.set-password-container` | `div` | `set-password-container` | Set PasswordPlease set your password to activate your accoun |
| `.set-password-card` | `div` | `set-password-card` | Set PasswordPlease set your password to activate your accoun |
| `#input-v-1` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#input-v-1-messages` | `div` | `` |  |
| `#input-v-4` | `input` | `` |  |
| `#input-v-4-messages` | `div` | `` |  |
| `.m-auto` | `button` | `m-auto` | Set Password |

---

### Register Page
**URL:** `/register`

**Headings:**
- `h2` — "Registration Required"


**Text Content (1):**
- [text-medium-emphasis] "Additional steps are required to complete your registration."

**Buttons (1):**
- `page.locator('button:has-text("Back to Login")')`

**Custom Elements & IDs (6):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | Registration RequiredAdditional steps are required to comple |
| `.toolbar-register` | `header` | `toolbar-register` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.register-container` | `div` | `register-container` | Registration RequiredAdditional steps are required to comple |
| `.register-card` | `div` | `register-card` | Registration RequiredAdditional steps are required to comple |
| `.mdi` | `i` | `mdi notranslate` |  |

---

