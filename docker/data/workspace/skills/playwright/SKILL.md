---
name: playwright
description: E2E testing for the Intent Platform (Vue 3, Vuetify 3). Selector strategies, modal patterns, and dropdown interaction for the Dashboard service.
---

# Picoclaw E2E Testing Skill

## Selector Strategy

Production code does **not** use `data-testid`. Labels from `GlobalFormField` are **not** linked via `for`/id, so `getByLabel()` often fails.

### Input fields

```typescript
// By placeholder (preferred when available)
const nameInput = drawer.locator('input[placeholder="Enter knowledge group name"]');

// By index inside container
const firstInput = drawer.locator('input[type="text"]').nth(0);

// getByPlaceholder
const hourInput = page.getByPlaceholder(/HH/i).first();

// For login card — anchor-by-label works here because labels are inline
const emailInput = page.locator('.login-card')
  .locator('div').filter({ hasText: /^Email address/ })
  .locator('input').first();
```

**Do NOT** use `.filter({ hasText: /placeholder text/ })` on a `div` — placeholder is an HTML attribute, not visible text, so `hasText` won't match it.

**Do NOT** use `.filter({ hasAttribute: ... })` — not a valid Playwright API. Valid filter options: `hasText`, `hasNotText`, `has`, `hasNot`.

Always scope to a container (`.custom-drawer-overlay`, `.login-card`, `.v-overlay--active .v-card`, etc.).

### Buttons and roles

- Buttons: `page.getByRole('button', { name: /Login/i })`
- Links: `page.getByRole('link', { name: /Sign up/i })`

### Dropdowns (Vuetify v-select)

Vuetify dropdowns are **teleported** to the bottom of `<body>` as `.v-overlay`. Never use `[role="option"]`, `getByRole('option')`, or `.locator('..')` parent traversal.

```typescript
// Step 1: Click the v-select to open it
await modal.locator('.v-select').nth(0).click();
await page.waitForTimeout(500);

// Step 2: Click the option in the teleported overlay
await page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /Monthly/ }).click();
await page.waitForTimeout(300);
```

- Use the exact dropdown index (`.nth(N)`) or label from the app-selectors-* SKILL.md.
- Do NOT use `.filter({ hasText: /LabelText/ })` on `.v-select` when label text is in a sibling `<label>` (common in GlobalFormField).
- After clicking a `.v-list-item`, the overlay closes automatically. Do NOT press Escape.

### Radio groups and button toggles

Use the selectors from the app-selectors-* SKILL.md. Do NOT guess component types like `.v-radio-group`, `.v-btn-toggle`, or `.v-tabs` if they are not documented there.

---

## Auth & Login Flow

- Login page: `/login`. Unauthenticated `/` redirects to `/login`.
- After login, may redirect to `/?select_org` for org selection.

Before interacting:
1. `await page.waitForURL(/\/login/);`
2. `await page.locator('.login-card').waitFor({ state: 'visible' });`

Login form selectors:
- **Email**: `.login-card` → `div` filter `hasText: /^Email address/` → `input`
- **Password**: `.login-card` → `div` filter `hasText: /^Password/` → `input`
- **Submit**: `page.getByRole('button', { name: /^Login$/i })`

---

## Organization Selection

### Full-page (`/?select_org`)

- Organization list: `.organization-card` (click to select)
- Container: `.container`, title: `.welcome-title`

### In-app org switcher

- Trigger: `.org-dropdown-trigger`
- Menu: `.org-dropdown-menu` (teleported to `body`)
- Items: `.org-dropdown-item`

---

## Modals & Overlays

- **NEVER** use `.v-dialog--active`, `.v-dialog`, or `[role="dialog"]` as selectors. Always use the real CSS class from the app-selectors-* SKILL.md.
- For Vuetify overlay modals: use `.v-overlay--active .v-card` scoped with the custom class documented in the SKILL.md.
- Look up the modal's actual selector in the "Discovered Modals / Dialogs" section, then scope form fields inside it.
- **Snackbars**: `await expect(page.locator('.v-snackbar')).toContainText('success')`

---

## Waiting & Synchronization

- **Navigation**: After `page.goto()` or redirect, `waitForURL` then wait for a stable container.
- **Transitions**: Wait for elements to be visible before clicking.
- **Button loading**: Wait for `.v-btn--loading` to disappear before asserting.
- **Modal loading**: After a modal opens, wait for loading to finish before asserting content:
  ```typescript
  await modal.locator('text=/loading/i')
    .waitFor({ state: 'hidden', timeout: 10000 }).catch(() => {});
  ```

---

## Reporting Format

Include `console.log` for every step and a summary at the end. Print `✅ PASS` only AFTER the assertion succeeds.

```typescript
console.log('📍 Step 5: Verify redirect');
await expect(page).toHaveURL(/dashboard/);
console.log('✅ PASS: Step 5 - Redirected to dashboard');
```

Summary block at the end:

```typescript
console.log('\n' + '='.repeat(70));
console.log('📊 TEST SUMMARY');
console.log('='.repeat(70));
console.log('✅ Step 1: PASS - Page loaded');
// ... one line per step ...
console.log('='.repeat(70));
```

---

## Debugging Checklist

- **Teleports**: Menus/dialogs render at end of `<body>`, not inside parent components. Query on `page`, not inside a scoped parent.
- **Hidden inputs**: Vuetify wraps `<input>` for styling. Use `.click({ force: true })` only if necessary.
- **Redirects**: If `page.goto('/')`, wait for redirect to `/login` before querying inputs.
