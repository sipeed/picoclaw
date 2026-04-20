# Authentication Flow Reference Document

**Generated:** 2026-04-07 12:00  
**Status:** Complete

---

## 0. Files Read

✅ **All 10 files read successfully:**

1. `/src/pages/auth/LoginPage.vue` — Login form with email/password inputs
2. `/src/pages/auth/ForgotPasswordPage.vue` — Forgot password flow
3. `/src/pages/auth/ChangeEmailPage.vue` — Change email form
4. `/src/pages/auth/ChangePasswordPage.vue` — Change password form
5. `/src/components/organization-selector/OrganizationSelector.vue` — Org dropdown selector
6. `/src/components/organization-selector/SelectOrganizationPage.vue` — Org selection page
7. `/src/components/GlobalFormField.vue` — Form field component (text, password, select, etc.)
8. `/src/components/AlertModal.vue` — Modal component for confirmations
9. `/src/stores/auth.store.ts` — Pinia auth store with login, logout, org switching
10. `/src/stores/profile.store.ts` — Profile store for email/password updates

---

## 1. Login Page

**URL:** `https://dashboard.int3nt.info/login`

### Email Field Selector
```typescript
// Using GlobalFormField component
// The email input is inside a v-text-field wrapper
const emailInput = page.locator('.v-text-field').nth(0).locator('input');

// Alternative: By label (if label is visible in DOM)
const emailInput = page.locator('label').filter({ hasText: /Email address/ })
  .locator('..').locator('input').first();

// Most reliable: Direct v-text-field index
await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
```

### Password Field Selector
```typescript
// Password field is second v-text-field, also inside GlobalFormField
const passwordInput = page.locator('.v-text-field').nth(1).locator('input');

// Fill password (note: inputType="text" with isPass="true" means it renders as type="password" when masked)
await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
```

### Login Button Selector
```typescript
// Button text is "Login" (ONE word, no space)
// Use getByRole for maximum reliability
await page.getByRole('button', { name: /login/i }).click();

// Alternative: Exact text match
await page.locator('button').filter({ hasText: /^Login$/ }).click();
```

### Post-Login Redirect Behavior
```typescript
// After successful login, redirect is to /?select_org (organization selection page)
// NOT directly to dashboard
await page.waitForURL(/\?select_org/, { timeout: 20000 });

// Then user must select an organization before accessing dashboard
// After selecting org, redirect is to /dashboard (or home page)
await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

### Complete Login Flow (Copy-Paste Ready)
```typescript
// Step 1: Navigate to login
await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

// Step 2: Wait for page to load
await page.locator('.login-card').waitFor({ state: 'visible', timeout: 10000 });

// Step 3: Fill credentials
await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');

// Step 4: Click login
await page.getByRole('button', { name: /login/i }).click();

// Step 5: Wait for redirect to org selection
await page.waitForURL(/\?select_org/, { timeout: 20000 });
```

---

## 2. Organization Selection

### Organization Card Selector
```typescript
// On /?select_org page, org cards are rendered with class "organization-card"
const orgCard = page.locator('.organization-card');

// To select by organization name
const orgCard = page.locator('.organization-card').filter({ hasText: 'Testing2026!' });
```

### Filter by Organization Name
```typescript
// Use hasText to filter by name
const card = page.locator('.organization-card').filter({ hasText: 'Testing2026!' });

// Click to select
await card.click();
```

### Organization Card Page/Overlay Selector
```typescript
// The org selection page is at /?select_org
// Container class: .container
// Welcome title: .welcome-title
// Info text: .info-text
// Organizations grid: .organizations-grid

const container = page.locator('.container');
const welcomeTitle = page.locator('.welcome-title');
```

### Full Organization Selection Flow
```typescript
// After login redirect to /?select_org
await page.waitForURL(/\?select_org/, { timeout: 20000 });

// Wait for loader to disappear
const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
if (await loader.first().isVisible().catch(() => false)) {
  await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
}

// Wait for org cards to appear
await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });

// Click the organization card
await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();

// Wait for redirect to dashboard
await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

---

## 3. Forgot Password Page

### Navigation to Forgot Password
```typescript
// From login page, click "Forgot Password" button
const forgotPasswordLink = page.locator('button').filter({ hasText: /Forgot Password|Forgot password/ });
await forgotPasswordLink.click();

// Or navigate directly
await page.goto('https://dashboard.int3nt.info/forgot-password');
```

### Email Input Selector
```typescript
// Email input inside GlobalFormField component
const emailInput = page.locator('.v-text-field').nth(0).locator('input');

// Fill email
await emailInput.fill('heidi@intnt.ai');
```

### Submit Button Selector
```typescript
// Button text is "Send Email"
const submitButton = page.locator('button').filter({ hasText: /Send Email|send email/ });
await submitButton.click();

// Or by role
await page.getByRole('button', { name: /send email/i }).click();
```

### Success/Error Notification Selectors
```typescript
// Success snackbar
await expect(page.locator('.v-snackbar')).toContainText('success');

// Error snackbar
await expect(page.locator('.v-snackbar')).toContainText('error');

// Wait for snackbar to appear
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

### Success Page Elements
```typescript
// After successful password reset, page shows success state
// Check icon container: .check-icon
// Success message: h2 (contains success title)
// Resend email button: button with "Resend" text
// Countdown timer: span inside resend button container

const checkIcon = page.locator('.check-icon');
const successTitle = page.locator('h2');
const resendButton = page.locator('button').filter({ hasText: /Resend/ });
```

---

## 4. Change Email Flow

### Navigation Path
```typescript
// Direct URL: https://dashboard.int3nt.info/change-email
await page.goto('https://dashboard.int3nt.info/change-email');

// Or from profile: /profile → click "Change Email"
```

### Current Email Input Selector
```typescript
// Note: There is NO "current email" field in ChangeEmailPage.vue
// Only "new email" and "confirm new email" fields

// New email input (first GlobalFormField)
const newEmailInput = page.locator('.v-text-field').nth(0).locator('input');

// Confirm email input (second GlobalFormField)
const confirmEmailInput = page.locator('.v-text-field').nth(1).locator('input');
```

### New Email Input Selector
```typescript
const newEmailInput = page.locator('.v-text-field').nth(0).locator('input');
await newEmailInput.fill('newemail@example.com');
```

### Confirm Email Input Selector
```typescript
const confirmEmailInput = page.locator('.v-text-field').nth(1).locator('input');
await confirmEmailInput.fill('newemail@example.com');
```

### Confirm/Save Button Selector
```typescript
// Button text is "Confirm"
const confirmButton = page.locator('button').filter({ hasText: /^Confirm$/ });
await confirmButton.click();

// Or by v-btn with color
const confirmButton = page.locator('.confirm-btn');
await confirmButton.click();
```

### Success Notification Selector and Text
```typescript
// Success snackbar appears after email is updated
await expect(page.locator('.v-snackbar')).toContainText('Email updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });

// Success message text (from i18n): 'changeEmailPage.emailUpdatedSuccessfully'
```

### Confirmation Modal
```typescript
// After clicking Confirm, AlertModal appears
// Modal selector: .v-dialog (standard Vuetify dialog)
// Modal title: "Confirm Email Change"
// Modal description: Shows new email

const modal = page.locator('.v-overlay--active').filter({ hasText: /Confirm Email Change/ });
await modal.waitFor({ state: 'visible', timeout: 5000 });

// Primary button in modal: "Confirm"
await modal.locator('button').filter({ hasText: /^Confirm$/ }).click();
```

---

## 5. Change Password Flow

### Navigation Path
```typescript
// Direct URL: https://dashboard.int3nt.info/change-password
await page.goto('https://dashboard.int3nt.info/change-password');

// Or from profile: /profile → click "Change Password"
```

### Current Password Input Selector
```typescript
// Note: There is NO "current password" field in ChangePasswordPage.vue
// Only "new password" and "confirm new password" fields

// New password input (first GlobalFormField with isPass="true")
const newPasswordInput = page.locator('.v-text-field').nth(0).locator('input');
```

### New Password Input Selector
```typescript
const newPasswordInput = page.locator('.v-text-field').nth(0).locator('input');

// Note: isPass="true" means the input type toggles between "password" and "text"
// There's an eye icon to toggle visibility
// Fill the input directly (Playwright handles hidden inputs)
await newPasswordInput.fill('NewPassword123!');
```

### Confirm Password Input Selector
```typescript
const confirmPasswordInput = page.locator('.v-text-field').nth(1).locator('input');
await confirmPasswordInput.fill('NewPassword123!');
```

### Submit Button Selector
```typescript
// Button text is "Confirm"
const confirmButton = page.locator('.confirm-btn');
await confirmButton.click();

// Or by filter
await page.locator('button').filter({ hasText: /^Confirm$/ }).click();
```

### Success Notification Selector and Text
```typescript
// Success snackbar appears after password is updated
await expect(page.locator('.v-snackbar')).toContainText('Password updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });

// Success message text (from i18n): 'changePasswordPage.passwordUpdatedSuccessfully'
```

### Confirmation Modal
```typescript
// After clicking Confirm, AlertModal appears
// Modal selector: .v-overlay--active (standard Vuetify dialog)
// Modal title: "Confirm Password Change"
// Modal description: "Are you sure you want to change your password?"

const modal = page.locator('.v-overlay--active').filter({ hasText: /Confirm Password Change/ });
await modal.waitFor({ state: 'visible', timeout: 5000 });

// Primary button in modal: "Confirm"
await modal.locator('button').filter({ hasText: /^Confirm$/ }).click();
```

---

## 6. Form Field Patterns

### GlobalFormField Component Structure

**What it renders in the DOM:**

```html
<div>
  <label>
    Field Label
    <span class="text-red" v-if="isRequired">* </span>
  </label>
  
  <!-- For inputType="text" -->
  <v-text-field
    v-model="localValue"
    variant="outlined"
    :type="isPasswordMasked ? 'password' : 'text'"
    :append-inner-icon="isPass ? (!maskPasswordField ? 'mdi-eye' : 'mdi-eye-off') : ''"
    @click:append-inner="toggleMaskPassword"
    :placeholder="placeholder"
    :disabled="disabled"
    :autocomplete="autocomplete"
  ></v-text-field>
</div>
```

### Correct Selector to Target Text Inputs Inside GlobalFormField

```typescript
// ✅ CORRECT: Target the v-text-field by index, then the input inside it
const firstInput = page.locator('.v-text-field').nth(0).locator('input');
const secondInput = page.locator('.v-text-field').nth(1).locator('input');

// ❌ WRONG: Trying to fill v-text-field directly (it's a div, not an input)
await page.locator('.v-text-field').nth(0).fill('value'); // FAILS

// ✅ CORRECT: Fill the input inside
await page.locator('.v-text-field').nth(0).locator('input').fill('value');

// Alternative: By placeholder (if available)
const emailInput = page.getByPlaceholder(/email/i).first();
await emailInput.fill('heidi@intnt.ai');
```

### Correct Selector to Target Error Messages

```typescript
// GlobalFormField uses Vuetify's validate-on="input"
// Error messages appear inside the v-text-field component

// Wait for error message to appear
const errorMessage = page.locator('.v-messages__message');
await errorMessage.waitFor({ state: 'visible', timeout: 3000 });

// Check error text
await expect(errorMessage).toContainText('Email must be valid');

// Or check if field has error class
const field = page.locator('.v-text-field');
await expect(field).toHaveClass(/error/);
```

### Password Field with Eye Icon

```typescript
// For password fields (isPass="true"), there's an eye icon to toggle visibility
const passwordInput = page.locator('.v-text-field').nth(1).locator('input');
const eyeIcon = page.locator('.v-text-field').nth(1).locator('[class*="append-inner"]');

// Click eye icon to toggle visibility
await eyeIcon.click();

// Input type changes from "password" to "text"
// But Playwright can fill hidden inputs directly, so no need to toggle
await passwordInput.fill('testing2026!');
```

### Required Field Indicator

```typescript
// Required fields show a red asterisk
const requiredLabel = page.locator('label').filter({ hasText: /\*/ });

// Check if field is required
const label = page.locator('label').filter({ hasText: /Email address/ });
const hasAsterisk = await label.locator('.text-red').isVisible();
console.log('Field is required:', hasAsterisk);
```

---

## 7. Notifications

### Success Snackbar Selector
```typescript
// Snackbar appears at bottom of screen
const snackbar = page.locator('.v-snackbar');

// Wait for it to appear
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Check for success message
await expect(snackbar).toContainText('success');
await expect(snackbar).toContainText('Email updated successfully');
```

### Error Snackbar Selector
```typescript
// Error snackbar uses same selector
const errorSnackbar = page.locator('.v-snackbar');

// Check for error message
await expect(errorSnackbar).toContainText('error');
await expect(errorSnackbar).toContainText('Email update failed');
```

### Typical Wait Timeout
```typescript
// Snackbar appears quickly, 3-5 seconds is typical
const snackbar = page.locator('.v-snackbar');
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Snackbar auto-dismisses after 3-5 seconds
await snackbar.waitFor({ state: 'hidden', timeout: 10000 });
```

### Snackbar Content Structure

```typescript
// The snackbar contains text content
// Selector for the message text
const messageText = page.locator('.v-snackbar__content');

// Or just check the entire snackbar
await expect(page.locator('.v-snackbar')).toContainText('Your message here');
```

---

## 8. AlertModal Component

### Modal Selector
```typescript
// AlertModal uses v-dialog, which renders as .v-overlay--active when open
// Best practice: scope by modal title

const modal = page.locator('.v-overlay--active').filter({ hasText: /Modal Title/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });
```

### Modal Elements

```typescript
// Close button (X in top right)
const closeBtn = modal.locator('button[icon="mdi-close"]');

// Icon (if provided)
const icon = modal.locator('.v-icon').first();

// Title (h2 element)
const title = modal.locator('h2');

// Description (p element)
const description = modal.locator('p');

// Buttons
const primaryBtn = modal.locator('button').filter({ hasText: /Confirm|Yes/ });
const secondaryBtn = modal.locator('button').filter({ hasText: /Cancel|No/ });
```

### Confirmation Modal Example (Change Email)

```typescript
// After filling email and clicking "Confirm", modal appears
const modal = page.locator('.v-overlay--active').filter({ hasText: /Confirm Email Change/ });
await modal.waitFor({ state: 'visible', timeout: 5000 });

// Modal shows new email address
const description = modal.locator('p');
await expect(description).toContainText('newemail@example.com');

// Click confirm button
const confirmBtn = modal.locator('button').filter({ hasText: /^Confirm$/ });
await confirmBtn.click();

// Wait for modal to close
await modal.waitFor({ state: 'hidden', timeout: 5000 });

// Check for success snackbar
await expect(page.locator('.v-snackbar')).toContainText('Email updated successfully');
```

---

## 9. Test Credentials

```typescript
const testCredentials = {
  email: 'heidi@intnt.ai',
  password: 'testing2026!',
  organization: 'Testing2026!'
};
```

---

## 10. Common Playwright Patterns for This App

### ✅ Correct Patterns

```typescript
// 1. Fill input inside v-text-field
await page.locator('.v-text-field').nth(0).locator('input').fill('value');

// 2. Click button by role
await page.getByRole('button', { name: /login/i }).click();

// 3. Wait for modal by filtering on text
const modal = page.locator('.v-overlay--active').filter({ hasText: /Title/ });
await modal.waitFor({ state: 'visible' });

// 4. Check snackbar content
await expect(page.locator('.v-snackbar')).toContainText('success');

// 5. Wait for redirect
await page.waitForURL(/expected-url/);
```

### ❌ Wrong Patterns

```typescript
// WRONG: Filling v-text-field directly (it's a div)
await page.locator('.v-text-field').nth(0).fill('value');

// WRONG: Using v-dialog as selector (Vue component, not CSS class)
await page.locator('v-dialog').waitFor();

// WRONG: Using [role="dialog"] (Vuetify doesn't use this)
await page.locator('[role="dialog"]').waitFor();

// WRONG: Using .v-dialog class (doesn't exist)
await page.locator('.v-dialog').waitFor();

// WRONG: Trying to get by label on GlobalFormField (labels aren't linked)
await page.getByLabel('Email address').fill('value');
```

---

**End of Reference Document**
