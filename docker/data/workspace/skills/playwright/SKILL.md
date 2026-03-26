---
name: playwright
description: End-to-End (E2E) testing for the Intent Platform. Use when asked to test complex user flows across Dashboard, Flow Designer, and Knowledge Base. Focuses on Vuetify 3 selector strategies, GlobalFormField patterns, and monorepo integration.
---

# Picoclaw E2E Testing Skill

This skill provides specialized guidance for browser-based automation (Playwright) against the Intent Platform monorepo: Vue 3, Vuetify 3, FastAPI. It is based on the actual DOM structure and component patterns in the Dashboard service.

---

## Selector Strategy (Vuetify 3 & GlobalFormField)

Production code does **not** use `data-testid`. Labels from `GlobalFormField` are **not** linked via `for`/id, so `getByLabel()` often fails. Use these strategies in order of preference.

### 1. Anchor-by-label (recommended for forms)

Find the container that holds the label text, then the input inside it. This matches how `GlobalFormField` renders: a `<label>` and a sibling `v-text-field` inside a wrapper div.

```typescript
// Scope to a unique container first (e.g. login card, modal)
const card = page.locator(".login-card");

// Email: div that contains the label "Email address", then the input inside
const emailInput = card
  .locator("div")
  .filter({ hasText: /^Email address/ })
  .locator("input")
  .first();

// Password: same pattern
const passwordInput = card
  .locator("div")
  .filter({ hasText: /^Password/ })
  .locator("input")
  .first();
```

- Use **scoped context** (e.g. `.login-card`, or the modal's custom CSS class from the relevant `app-selectors-*` SKILL.md) so you don't match multiple instances. NEVER use `.v-dialog--active`.
- Match the **exact i18n string** for the locale under test (see Login flow for en).

### 2. ARIA roles and names

Use when the element exposes a proper role/label:

- Buttons: `page.getByRole('button', { name: /Login/i })`
- Links: `page.getByRole('link', { name: /Sign up/i })`

Inputs are often not properly associated with labels (no `for`/id in GlobalFormField), so prefer anchor-by-label for text fields.

### 3. Vuetify 3 structure (fallback)

**Do not rely on `.v-text-field`** as the root selector. In Vuetify 3, `v-text-field` composes `v-input` and `v-field`; the root may have `.v-input`, and the native `<input>` lives under `.v-field__input`. The class `.v-text-field` is not guaranteed on the root.

If anchor-by-label is not possible:

- Scope to a unique parent, then use:
  - `.v-input input` or
  - `.v-field__input` (then the inner `input` if needed)
- Checkboxes/radios: `.v-checkbox-btn input`, `.v-radio-group input`

### 4. Placeholder (when present)

If the field has a placeholder: `page.getByPlaceholder(/enter email/i)`.

---

## Auth & Login Flow

### Route and redirects

- Login page: `/login`.
- Unauthenticated access to `/` redirects to `/login` (router guard). Always **wait for the login page** before querying the form.
- After successful login, the app may redirect to `/?select_org` for organization selection.

### Wait before interacting with the login form

1. Wait for URL: `await page.waitForURL(/\/login/);`
2. Wait for the login card: `await page.locator('.login-card').waitFor({ state: 'visible' });`
3. Optionally wait for the email input: `await emailInput.waitFor({ state: 'visible' });`

### Login form selectors (English locale)

- **Email**: Label text `"Email address"`. Use anchor-by-label scoped to `.login-card` (see above).
- **Password**: Label text `"Password"`. Same pattern.
- **Submit**: `page.getByRole('button', { name: /^Login$/i })`.
- **Forgot password**: Button/link with text "Forgot Password".
- **Sign up**: Button/link with text "Sign up".

### Forgot Password page selectors

The Forgot Password page (`/forgot-password`) also uses `GlobalFormField` with `inputType="text"` for the email field, so **there is no `type="email"` and often no placeholder**. Do **not** rely on:

- `input[type="email"]`
- `input[placeholder*="email" i]`

Any spec that continues to use these generic selectors on the Forgot Password page will keep failing, because no matching element exists in the DOM there; you **must** switch to the label-anchored selector below.

Instead:

- Scope to `.forgot-password-card`.
- Use anchor-by-label with the **forgot password email label** (English: `"Email address"` from `forgotPasswordPage.emailLabel`).
- Use the button text `"Send Email"` (or variants) for the submit button.

Example:

```ts
await page.goto("/forgot-password");
await page.waitForURL(/\/forgot-password/);
await page.locator(".forgot-password-card").waitFor({ state: "visible" });

const forgotCard = page.locator(".forgot-password-card");
const emailInput = forgotCard
  .locator("div")
  .filter({ hasText: /^Email address/ })
  .locator("input")
  .first();

const sendEmailButton = page.getByRole("button", { name: /Send Email|Send/i });

await expect(emailInput).toBeVisible();
await emailInput.fill("test@intnt.ai");
await expect(sendEmailButton).toBeVisible();
await sendEmailButton.click();
```

If you need a locale-agnostic fallback (e.g. different language labels), you can also use the Vuetify structure scoped to the card:

```ts
const emailInput = page.locator(".forgot-password-card .v-input input").first();
```

but **do not** use this unscoped (without `.forgot-password-card`) to avoid accidentally matching other fields.

### Example: full login flow

```typescript
await page.goto("/login"); // or base URL and let redirect
await page.waitForURL(/\/login/);
await page.locator(".login-card").waitFor({ state: "visible" });

const loginCard = page.locator(".login-card");
const emailInput = loginCard
  .locator("div")
  .filter({ hasText: /^Email address/ })
  .locator("input")
  .first();
const passwordInput = loginCard
  .locator("div")
  .filter({ hasText: /^Password/ })
  .locator("input")
  .first();
const loginButton = page.getByRole("button", { name: /^Login$/i });

await expect(emailInput).toBeVisible();
await expect(passwordInput).toBeVisible();
await expect(loginButton).toBeVisible();

await emailInput.fill("user@example.com");
await passwordInput.fill("password");
await loginButton.click();
```

### SSO

- SSO button: `v-btn` with icon `mdi-shield-account` or text like "Sign in with SSO".
- Provider list appears in a modal; select by visible provider name.

---

## Organization Selection

Two contexts:

### 1. Full-page org selection (`/?select_org`)

After login, the app may show `SelectOrganizationPage` when the URL has `?select_org`.

- Container: `.container`
- Title: `.welcome-title`
- Logout: `.logout-button`
- Organization list: `.organization-card` (click to select)
- Create org: button with text matching "Create Organization" / createOrganization
- Modal for creating org: Check the relevant `app-selectors-*` SKILL.md for the actual modal selector (custom CSS class). Do NOT use `.v-dialog--active`.

### 2. In-app org switcher (drawer)

`OrganizationSelector` in the layout uses **Teleport to body** for the dropdown. The menu is **not** inside the drawer DOM.

- Trigger: `.org-dropdown-trigger`
- Menu: `.org-dropdown-menu` (appears on `body`). Wait for it after clicking the trigger: `page.locator('.org-dropdown-menu').waitFor({ state: 'visible' })`
- Items: `.org-dropdown-item`
- Empty state: `.org-dropdown-empty`

---

## Flow Designer

- **Canvas**: VueFlow (`@vue-flow/core`). Main wrapper: `.dnd-flow`, flow root: `.basic-flow`, pane: `.vue-flow__pane`.
- **Nodes**: Rendered inside the flow; identify by type/label or position. Node types come from metadata (e.g. Start, End, RestAPINode shown as "Rest API Node").
- **Sidebar**: Nodes dropdown in a panel; add nodes by click or drag-and-drop from the sidebar to the canvas.
- **Config**: `NodeConfigurationModal`, `EdgeConfigurationModal`; wait for modal to be visible using the actual custom CSS class from the relevant `app-selectors-*` SKILL.md before interacting with form fields.

---

## Knowledge Base

- **File upload**: Components like `DragDropFileUpload`; use Playwright file chooser or drop events as needed.
- **Status**: Use classes such as `.knowledge-base-status-pill` to assert processing state when present.

---

## Modals & global UI

- **NEVER use `.v-dialog--active` or `.v-dialog` as selectors.** These are generic Vuetify wrapper classes that may not exist in the actual DOM. Instead, always use the real custom CSS class from the relevant `app-selectors-*` SKILL.md (e.g. `.modal-content`, `.modal-header`, `.modal-title`).
- **Form modals**: Look up the modal's actual selector in the "Discovered Modals / Dialogs" section of the page's `app-selectors-*` SKILL.md, then scope form fields inside that selector.
- **Snackbars**: Feedback is shown via `GlobalSnackBar`. Assert messages in `.v-snackbar` (e.g. content or role).

---

## Waiting & synchronization

- **Navigation**: After `page.goto()` or a redirect, wait for the target URL (`waitForURL`) and then for a stable container (e.g. `.login-card`, `.container`).
- **API**: When needed, wait for specific responses: `page.waitForResponse(urlOrPredicate)`.
- **Transitions**: Vuetify uses transitions (e.g. `v-fade-transition`). Wait for the element to be visible and stable before clicking.
- **Loading**: Buttons may show loading state (e.g. `.v-btn--loading`). Wait for the action to finish before asserting (e.g. wait for loading to disappear or for next UI state).

---

## Mocking & data

- **MSW**: The platform uses Mock Service Worker for network mocking. For E2E, use `page.addInitScript()` to inject MSW if running without a full backend.
- **Auth/orgs**: Intercept Supabase (or auth) calls to simulate different roles (e.g. Owner vs Member) for permission tests.
- **Files**: Mock signed URLs (e.g. GCS/S3) to test file previews without real bucket access.

---

## Monorepo context

- **Dashboard** (`services/dashboard`): Main UI; E2E targets this app.
- **Python service** (`services/python`): Backend; E2E typically runs against a dev or local docker-compose environment.
- **Supabase**: Auth and real-time; mock or stub as needed (e.g. `auth.loginWithSSO`) to avoid external redirects.

---

## Reporting format

You MUST include console.log statements for every step and a final report summary at the end of the test.

Example for each step:
```typescript
console.log('✅ PASS: Step 5 - Login successful, redirected to organization selection page');
// OR if failed
console.log('❌ FAIL: Step 5 - Login failed, still on login page');
```

Example for the final report result at the end of the test run:
```typescript
  console.log('\\n📍 Step 6: Report PASS or FAIL for each step');
  console.log('\\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Page loaded');
  console.log('✅ Step 2: PASS - Login form visible');
  console.log('✅ Step 3: PASS - Credentials entered');
  console.log('✅ Step 4: PASS - Login button clicked');
  console.log('✅ Step 5: PASS - Login successful, redirected to ?select_org');
  console.log('✅ Step 6: PASS - All steps passed');
  console.log('='.repeat(70));
  console.log('\\n');
```

---

## Debugging checklist

- **Hydration / i18n**: Ensure the app has finished loading and (if applicable) i18n has resolved. Wait for URL and a non-layout element (e.g. `.login-card`) before interacting.
- **Label text**: Does the regex match the rendered label exactly? (e.g. "Email address" vs "Email".) Use the exact strings from `locales/en/pages.json` for the default locale.
- **Teleports**: Menus and dialogs (e.g. `v-menu`) render at the end of `<body>`, not inside the component tree. Query for `.org-dropdown-menu` or the modal's actual custom CSS class from the relevant `app-selectors-*` SKILL.md on `page`, not inside a parent that doesn't contain them. NEVER use `.v-dialog--active`.
- **Hidden inputs**: Vuetify may wrap the real `<input>` for styling. Prefer the visible interaction target; use `.click({ force: true })` only if necessary and document why.
- **Redirects**: If using `page.goto('/')`, wait for the redirect to `/login` to complete before looking for login inputs.
- **Vuetify 3**: Prefer anchor-by-label over `.v-text-field`; use `.v-input` / `.v-field__input` as fallback when needed.

> **Important:** If an auto-generated spec still contains  
> `page.locator('input[type="email"], input[placeholder*="email" i]')`  
> on `/forgot-password`, it is **incorrect for this app** and must be replaced  
> with one of the patterns above. The login page happens to work with  
> `.v-text-field.locator('input')`, but the Forgot Password page does not  
> expose any email-specific attributes, so generic email selectors will  
> always fail there.

## STABILITY RULES FOR VUETIFY UI:

Selector priority:
1. Use getByRole() with accessible names whenever possible
2. Use getByLabel() for text fields
3. Use getByText() for menu items
4. Use Vuetify classes only when necessary (.v-btn, .v-select). NEVER use .v-dialog or .v-dialog--active.

Dialogs:
- NEVER use .v-dialog--active — it is a generic Vuetify wrapper that may not exist in the DOM.
- Instead, find the modal's actual custom CSS class from the relevant app-selectors-* SKILL.md.
- Wait for dialogs using the real selector, e.g.:
  await page.locator('.modal-content').waitFor()

Menus / dropdowns (Vuetify v-select):
- Vuetify dropdowns are teleported to the bottom of the body as `.v-overlay`
- NEVER use `[role="option"]` or `[role="combobox"]` or `getByRole('option')` — Vuetify does NOT use these ARIA roles
- NEVER traverse parent with `.locator('..')` to find a select — it won't work with Vuetify's DOM structure

To select an option from a Vuetify v-select dropdown:

```typescript
// Step 1: Click the v-select to open it (use label from SKILL.md)
await page.locator('.v-select').filter({ hasText: /Frequency/ }).click();
await page.waitForTimeout(500);

// Step 2: Click the list item in the overlay (teleported to body)
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Weekly/ }).click();
await page.waitForTimeout(300);
```

If the app-selectors-* SKILL.md lists dropdown options, use the exact option text from there.
If multiple v-selects exist, use `.filter({ hasText: /LabelText/ })` or `.nth(N)` to target the right one.

IMPORTANT: After clicking a v-list-item, the overlay closes automatically. Do NOT press Escape.

Tables:
- When selecting rows by email:
await page.locator('tr', { hasText: "test2@intnt.ai" })

Notifications:
- Wait for snackbar using:
await expect(page.locator('.v-snackbar')).toContainText('success')

Buttons:
- Always ensure buttons are enabled before clicking
await expect(button).toBeEnabled()

Animations:
- After opening dialogs or menus, wait for the element to become visible before interacting.