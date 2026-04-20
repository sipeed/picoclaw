# Profile Page Reference Document

**Generated:** 2026-04-07 13:30  
**Status:** Complete

---

## 0. Files Read

✅ **All 8 files read successfully:**

1. `/src/pages/ProfilePage.vue` — Main profile page with name update and avatar
2. `/src/pages/auth/ChangeEmailPage.vue` — Change email page
3. `/src/pages/auth/ChangePasswordPage.vue` — Change password page
4. `/src/components/GlobalFormField.vue` — Generic form field component
5. `/src/components/AlertModal.vue` — Confirmation modal component
6. `/src/components/UserAvatar.vue` — User avatar component with logout
7. `/src/stores/profile.store.ts` — Profile Pinia store
8. `/src/stores/snackbar.store.ts` — Snackbar notification store

---

## 1. Profile Page Navigation

### Sidebar or Avatar Selector to Open Profile Page
```typescript
// From sidebar menu
await page.locator('a:has-text("Profile")').click();

// Or from user avatar menu
const userAvatar = page.locator('.v-avatar').first();
await userAvatar.click();

// Then click Profile option (if available in menu)
await page.locator('button').filter({ hasText: /Profile/ }).click();

// Or navigate directly
await page.goto('https://dashboard.int3nt.info/profile');
```

### Page URL or Route Identifier
```typescript
// Expected URL after navigation
// URL: https://dashboard.int3nt.info/profile (or /profile route)

// Verify page loaded
await page.waitForURL(/\/profile/);
```

### How to Verify Profile Page is Loaded
```typescript
// Wait for page title
const title = page.locator('h2').first();
await expect(title).toContainText('Profile Settings');

// Wait for profile container
const profileContainer = page.locator('.profile-container');
await profileContainer.waitFor({ state: 'visible', timeout: 10000 });

// Wait for breadcrumb
const breadcrumb = page.locator('a').filter({ hasText: /Profile Settings/ });
await breadcrumb.waitFor({ state: 'visible', timeout: 10000 });

// Wait for avatar section
const avatar = page.locator('.v-avatar').first();
await avatar.waitFor({ state: 'visible', timeout: 10000 });

// Wait for form fields
const firstNameInput = page.locator('.v-text-field').nth(0);
await firstNameInput.waitFor({ state: 'visible', timeout: 10000 });
```

---

## 2. Update Profile Name

### Selector for the Name/Display Name Input Field
```typescript
// First name input (first GlobalFormField)
const firstNameInput = page.locator('.v-text-field').nth(0).locator('input');

// Last name input (second GlobalFormField)
const lastNameInput = page.locator('.v-text-field').nth(1).locator('input');

// Or by placeholder
const firstNameInput = page.getByPlaceholder(/first name/i).first();
const lastNameInput = page.getByPlaceholder(/last name/i).first();
```

### Selector for the Save/Update Button
```typescript
// Save Changes button
const saveBtn = page.locator('button').filter({ hasText: /Save Changes|save changes/i });

// Or by class
const saveBtn = page.locator('.bg-btn-primary').filter({ hasText: /Save/ });
```

### Success Notification Selector and Text
```typescript
// Success snackbar after saving
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('Profile updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 3. Change Email Flow

### How to Navigate to the Change Email Page/Section
```typescript
// From Profile page, click "Change Email" link
const changeEmailLink = page.locator('button').filter({ hasText: /Change Email|change email/i });
await changeEmailLink.click();

// Or navigate directly
await page.goto('https://dashboard.int3nt.info/change-email');

// Verify URL
await page.waitForURL(/\/change-email/);
```

### Selector for the Current Password Input (if Required)
```typescript
// Note: ChangeEmailPage.vue does NOT require current password
// Only new email and confirm email are needed

// This field does NOT exist on the page
// Skip this step if testing ChangeEmailPage
```

### Selector for the New Email Input
```typescript
// New email input (first GlobalFormField)
const newEmailInput = page.locator('.v-text-field').nth(0).locator('input');

// Or by label
const newEmailInput = page.locator('label').filter({ hasText: /New Email/ })
  .locator('..').locator('input').first();
```

### Selector for the Confirm/Save Button
```typescript
// Confirm button
const confirmBtn = page.locator('button').filter({ hasText: /^Confirm$|^Save$/i });

// Or by class
const confirmBtn = page.locator('button[color="#E57E1F"]').filter({ hasText: /Confirm/ });
```

### Success Notification Selector and Text
```typescript
// Success snackbar after email change
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('Email updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

### Any Redirect or Confirmation Step After Saving
```typescript
// After confirming email change, AlertModal appears first
// User must confirm in the modal before email is updated

// Confirmation modal selector
const confirmModal = page.locator('.v-overlay--active').filter({ hasText: /confirm/i });

// Confirm button in modal
const confirmModalBtn = confirmModal.locator('button').filter({ hasText: /^Confirm$/i });
await confirmModalBtn.click();

// Then success snackbar appears and user stays on page
```

---

## 4. Change Password Flow

### How to Navigate to the Change Password Page/Section
```typescript
// From Profile page, click "Change Password" link
const changePasswordLink = page.locator('button').filter({ hasText: /Change Password|change password/i });
await changePasswordLink.click();

// Or navigate directly
await page.goto('https://dashboard.int3nt.info/change-password');

// Verify URL
await page.waitForURL(/\/change-password/);
```

### Selector for the Current Password Input
```typescript
// Note: ChangePasswordPage.vue does NOT require current password
// Only new password and confirm password are needed

// This field does NOT exist on the page
// Skip this step if testing ChangePasswordPage
```

### Selector for the New Password Input
```typescript
// New password input (first GlobalFormField with isPass="true")
const newPasswordInput = page.locator('.v-text-field').nth(0).locator('input');

// Or by label
const newPasswordInput = page.locator('label').filter({ hasText: /New Password/ })
  .locator('..').locator('input').first();
```

### Selector for the Confirm New Password Input
```typescript
// Confirm new password input (second GlobalFormField with isPass="true")
const confirmPasswordInput = page.locator('.v-text-field').nth(1).locator('input');

// Or by label
const confirmPasswordInput = page.locator('label').filter({ hasText: /Confirm New Password/ })
  .locator('..').locator('input').first();
```

### Selector for the Submit Button
```typescript
// Confirm button
const submitBtn = page.locator('button').filter({ hasText: /^Confirm$|^Submit$/i });

// Or by class
const submitBtn = page.locator('button[color="#E57E1F"]').filter({ hasText: /Confirm/ });
```

### Success Notification Selector and Text
```typescript
// Success snackbar after password change
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('Password updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

### Any Redirect or Logout Step After Saving
```typescript
// After confirming password change, AlertModal appears first
// User must confirm in the modal before password is updated

// Confirmation modal selector
const confirmModal = page.locator('.v-overlay--active').filter({ hasText: /confirm/i });

// Confirm button in modal
const confirmModalBtn = confirmModal.locator('button').filter({ hasText: /^Confirm$/i });
await confirmModalBtn.click();

// Then success snackbar appears
// User stays on page (no automatic logout)
```

---

## 5. User Avatar

### Selector for the Avatar/Profile Picture
```typescript
// Avatar on profile page
const avatar = page.locator('.v-avatar').first();

// Avatar with initials
const avatarWithInitials = page.locator('.v-avatar').filter({ hasText: /[A-Z]{1,2}/ });

// Avatar in header (UserAvatar component)
const headerAvatar = page.locator('v-avatar').first();
```

### How to Upload a New Avatar (File Input Selector for setInputFiles())
```typescript
// File input on ProfilePage
const fileInput = page.locator('input[type="file"][accept="image/*"]').first();

// Upload button
const uploadBtn = page.locator('button').filter({ hasText: /Upload New Picture|upload/i });
await uploadBtn.click();

// Then set file
await fileInput.setInputFiles('/path/to/avatar.png');

// Preview appears in avatar
// Then save changes to persist
```

### Success Notification Selector and Text
```typescript
// Success snackbar after avatar upload
const snackbar = page.locator('.v-snackbar');

// Check for success message (appears after saving profile changes)
await expect(snackbar).toContainText('Profile updated successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 6. Form Field Patterns

### How GlobalFormField Renders in the DOM
```typescript
// GlobalFormField structure:
// <div>
//   <label>Field Label <span class="text-red">*</span></label>
//   <v-text-field
//     v-model="localValue"
//     variant="outlined"
//     :type="isPasswordMasked ? 'password' : 'text'"
//     :append-inner-icon="isPass ? icon : ''"
//     ...
//   >
//   </v-text-field>
// </div>

const fieldContainer = page.locator('label').filter({ hasText: /First Name/ })
  .locator('..');
```

### Correct Selector to Target Text Inputs
```typescript
// ✅ CORRECT: Target the v-text-field by index, then the input inside it
const firstInput = page.locator('.v-text-field').nth(0).locator('input');
const secondInput = page.locator('.v-text-field').nth(1).locator('input');

// ❌ WRONG: Trying to fill v-text-field directly (it's a div, not an input)
await page.locator('.v-text-field').nth(0).fill('value'); // FAILS

// ✅ CORRECT: Fill the input inside
await page.locator('.v-text-field').nth(0).locator('input').fill('value');

// Alternative: By placeholder
const emailInput = page.getByPlaceholder(/email/i).first();
await emailInput.fill('newemail@example.com');
```

### Correct Selector to Target Validation Error Messages
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

---

## 7. Notifications

### Selector for Success Snackbar/Toast
```typescript
// Snackbar appears at bottom of screen
const snackbar = page.locator('.v-snackbar');

// Wait for snackbar to appear
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Check for success message
await expect(snackbar).toContainText('success|updated|changed');

// Check for specific messages
await expect(snackbar).toContainText('Profile updated successfully');
await expect(snackbar).toContainText('Email updated successfully');
await expect(snackbar).toContainText('Password updated successfully');
```

### Selector for Error Snackbar/Toast
```typescript
// Error snackbar uses same selector
const errorSnackbar = page.locator('.v-snackbar');

// Check for error message
await expect(errorSnackbar).toContainText('error|failed|invalid');

// Check for specific error messages
await expect(errorSnackbar).toContainText('Profile update failed');
await expect(errorSnackbar).toContainText('Email update failed');
await expect(errorSnackbar).toContainText('Password update failed');
```

### Typical Wait Timeout
```typescript
// Snackbars appear quickly: 3-5 seconds
const timeout = 5000; // 5 seconds

// Wait for snackbar to appear
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });

// Snackbar auto-dismisses after 3 seconds (default timeout in snackbar.store.ts)
await page.locator('.v-snackbar').waitFor({ state: 'hidden', timeout: 10000 });
```

---

## 8. Common Workflows

### Update Profile Name
```typescript
// Step 1: Navigate to profile page
await page.goto('https://dashboard.int3nt.info/profile');
await page.waitForURL(/\/profile/);

// Step 2: Wait for page to load
await page.locator('.profile-container').waitFor({ state: 'visible' });

// Step 3: Fill first name
await page.locator('.v-text-field').nth(0).locator('input').fill('John');

// Step 4: Fill last name
await page.locator('.v-text-field').nth(1).locator('input').fill('Doe');

// Step 5: Click Save Changes
await page.locator('button').filter({ hasText: /Save Changes/i }).click();

// Step 6: Confirm in modal
const modal = page.locator('.v-overlay--active');
await modal.waitFor({ state: 'visible' });
await modal.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 7: Verify success
await expect(page.locator('.v-snackbar')).toContainText('Profile updated successfully');
```

### Change Email
```typescript
// Step 1: Navigate to change email page
await page.goto('https://dashboard.int3nt.info/change-email');
await page.waitForURL(/\/change-email/);

// Step 2: Fill new email
await page.locator('.v-text-field').nth(0).locator('input').fill('newemail@example.com');

// Step 3: Confirm new email
await page.locator('.v-text-field').nth(1).locator('input').fill('newemail@example.com');

// Step 4: Click Confirm
await page.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 5: Confirm in modal
const modal = page.locator('.v-overlay--active');
await modal.waitFor({ state: 'visible' });
await modal.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 6: Verify success
await expect(page.locator('.v-snackbar')).toContainText('Email updated successfully');
```

### Change Password
```typescript
// Step 1: Navigate to change password page
await page.goto('https://dashboard.int3nt.info/change-password');
await page.waitForURL(/\/change-password/);

// Step 2: Fill new password
await page.locator('.v-text-field').nth(0).locator('input').fill('NewPassword123!');

// Step 3: Confirm new password
await page.locator('.v-text-field').nth(1).locator('input').fill('NewPassword123!');

// Step 4: Click Confirm
await page.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 5: Confirm in modal
const modal = page.locator('.v-overlay--active');
await modal.waitFor({ state: 'visible' });
await modal.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 6: Verify success
await expect(page.locator('.v-snackbar')).toContainText('Password updated successfully');
```

### Upload Avatar
```typescript
// Step 1: Navigate to profile page
await page.goto('https://dashboard.int3nt.info/profile');
await page.waitForURL(/\/profile/);

// Step 2: Wait for page to load
await page.locator('.profile-container').waitFor({ state: 'visible' });

// Step 3: Click Upload New Picture button
const uploadBtn = page.locator('button').filter({ hasText: /Upload New Picture/i });
await uploadBtn.click();

// Step 4: Set file
const fileInput = page.locator('input[type="file"][accept="image/*"]');
await fileInput.setInputFiles('/path/to/avatar.png');

// Step 5: Verify preview appears
const avatar = page.locator('.v-avatar').first();
await expect(avatar).toBeVisible();

// Step 6: Save changes
await page.locator('button').filter({ hasText: /Save Changes/i }).click();

// Step 7: Confirm in modal
const modal = page.locator('.v-overlay--active');
await modal.waitFor({ state: 'visible' });
await modal.locator('button').filter({ hasText: /^Confirm$/i }).click();

// Step 8: Verify success
await expect(page.locator('.v-snackbar')).toContainText('Profile updated successfully');
```

---

## 9. Test Credentials & URLs

```typescript
const testCredentials = {
  email: 'heidi@intnt.ai',
  password: 'testing2026!',
  organization: 'Testing2026!'
};

const urls = {
  login: 'https://dashboard.int3nt.info/login',
  profile: 'https://dashboard.int3nt.info/profile',
  changeEmail: 'https://dashboard.int3nt.info/change-email',
  changePassword: 'https://dashboard.int3nt.info/change-password',
  dashboard: 'https://dashboard.int3nt.info'
};
```

---

## 10. Key Selectors Summary

| Element | Selector |
|---------|----------|
| Profile page container | `.profile-container` |
| First name input | `.v-text-field:nth(0) input` |
| Last name input | `.v-text-field:nth(1) input` |
| New email input | `.v-text-field:nth(0) input` (on change-email page) |
| New password input | `.v-text-field:nth(0) input` (on change-password page) |
| Avatar | `.v-avatar:nth(0)` |
| Upload button | `button:has-text("Upload New Picture")` |
| File input | `input[type="file"][accept="image/*"]` |
| Save Changes button | `button:has-text("Save Changes")` |
| Confirm button | `button:has-text("Confirm")` |
| Change Email link | `button:has-text("Change Email")` |
| Change Password link | `button:has-text("Change Password")` |
| Success snackbar | `.v-snackbar:has-text("success\|updated")` |
| Error snackbar | `.v-snackbar:has-text("error\|failed")` |
| Confirmation modal | `.v-overlay--active:has-text("confirm")` |

---

**End of Reference Document**
