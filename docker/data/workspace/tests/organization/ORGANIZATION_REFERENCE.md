# Organization Page Reference Document

**Generated:** 2026-04-07 13:00  
**Status:** Complete

---

## 0. Files Read

✅ **All 9 files read successfully:**

1. `/src/pages/OrganizationPage.vue` — Main organization page with member management and bot icons
2. `/src/components/organization-selector/OrganizationSelector.vue` — Org selector dropdown component
3. `/src/components/organization-selector/SelectOrganizationPage.vue` — Org selection page (full-page)
4. `/src/components/LogoUploadDialog.vue` — Logo upload dialog with drag-drop
5. `/src/components/DragDropFileUpload.vue` — Generic drag-drop file upload component
6. `/src/components/FormModal.vue` — Form modal with dynamic fields
7. `/src/components/AlertModal.vue` — Alert/confirmation modal
8. `/src/stores/organization.store.ts` — Organization Pinia store
9. `/src/stores/auth.store.ts` — Auth store with org switching

---

## 1. Organization Page Navigation

### Sidebar Selector to Navigate to Organization Page
```typescript
// From any page with sidebar, click Organization link
await page.locator('a:has-text("Organization")').click();

// Alternative: By role
await page.getByRole('link', { name: /Organization/i }).click();

// Alternative: By href
await page.locator('a[href="/organization"]').click();
```

### Page URL / Route Identifier
```typescript
// Expected URL after navigation
// URL: https://dashboard.int3nt.info/organization (or /organization route)

// Verify page loaded
await page.waitForURL(/\/organization/);
```

### How to Verify Organization Page is Loaded
```typescript
// Wait for page title
const title = page.locator('h2').first();
await expect(title).toContainText('Organization Team');

// Wait for members section
const membersSection = page.locator('.organization-container');
await membersSection.waitFor({ state: 'visible', timeout: 10000 });

// Wait for Add Member button
const addMemberBtn = page.locator('button').filter({ hasText: /Add Member|add member/ });
await addMemberBtn.waitFor({ state: 'visible', timeout: 10000 });

// Wait for data table to load
const dataTable = page.locator('.organization-table');
await dataTable.waitFor({ state: 'visible', timeout: 10000 });
```

---

## 2. Member Management

### Selector for the Members Tab/Section
```typescript
// Members section (no separate tabs, all on one page)
const membersSection = page.locator('.organization-container');

// Table header
const tableHeader = page.locator('.organization-table');

// Table rows
const tableRows = page.locator('.v-data-table__tr');
```

### Selector for the Invite Member Button
```typescript
// Add Member button at top of page
const addMemberBtn = page.locator('button').filter({ hasText: /Add Member|add member/i });

// Or by class
const addBtn = page.locator('.text-capitalize.font-weight-bold.bg-btn-primary');
```

### Selector for the Invite Email Input Field
```typescript
// Inside FormModal for adding member
// Email input (first GlobalFormField in the form)
const emailInput = page.locator('.v-text-field').nth(0).locator('input');

// Or by placeholder
const emailInput = page.getByPlaceholder(/email/i).first();
```

### Selector for the Invite Role Selector/Dropdown
```typescript
// Role dropdown (second field in FormModal)
const roleSelect = page.locator('.v-select').nth(0);

// Or by label
const roleSelect = page.locator('label').filter({ hasText: /Role/ })
  .locator('..').locator('.v-select').first();
```

### Selector for the Confirm Invite Button
```typescript
// Inside FormModal
// Confirm/Add button
const confirmBtn = page.locator('button').filter({ hasText: /^Add$|^Confirm$/i });

// Or by class
const confirmBtn = page.locator('.bg-btn-primary').filter({ hasText: /Add|Confirm/ });
```

### How to Locate a Member Row by Email/Name
```typescript
// Find row by email
const memberRow = page.locator('.v-data-table__tr')
  .filter({ hasText: /john@example.com/ });

// Find row by name
const memberRow = page.locator('.v-data-table__tr')
  .filter({ hasText: /John Doe/ });

// Get specific cell value
const emailCell = memberRow.locator('.v-data-table__td').nth(2); // Email is 3rd column
```

### Selector for the Activate/Deactivate Member Button
```typescript
// Actions menu (three dots) in each row
const actionsMenu = page.locator('.v-data-table__tr')
  .filter({ hasText: /member@example.com/ })
  .locator('v-icon').filter({ hasText: /mdi-dots-vertical/ });

// Click to open menu
await actionsMenu.click();

// Deactivate option
const deactivateOption = page.locator('v-list-item').filter({ hasText: /Deactivate/ });

// Activate option
const activateOption = page.locator('v-list-item').filter({ hasText: /Activate/ });
```

### Selector for the Deactivate Confirmation Modal and Confirm Button
```typescript
// Deactivate confirmation modal (AlertModal)
const deactivateModal = page.locator('.v-overlay--active').filter({ hasText: /deactivate/ });

// Confirm button in modal
const confirmBtn = deactivateModal.locator('button').filter({ hasText: /^Confirm$|^Yes$/i });

// Cancel button
const cancelBtn = deactivateModal.locator('button').filter({ hasText: /^Cancel$|^No$/i });
```

### Success Notification Selector and Text
```typescript
// Success snackbar after member action
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('Member');
await expect(snackbar).toContainText('deactivated|activated|invited|role changed');

// Common success messages:
// - "Member invite sent"
// - "Member deactivated"
// - "Member activated"
// - "Role changed"
```

---

## 3. Role Management

### Selector for the Roles Tab/Section
```typescript
// Roles are managed inline with members (no separate tab)
// Role is displayed in the table
const roleCell = page.locator('.v-data-table__tr')
  .filter({ hasText: /member@example.com/ })
  .locator('.v-data-table__td').nth(1); // Role is 2nd column
```

### How to Locate a Role by Name
```typescript
// Find role in table
const roleCell = page.locator('.v-data-table__tr')
  .filter({ hasText: /admin|agent|developer/i })
  .locator('.v-data-table__td').nth(1);

// Find member with specific role
const memberWithRole = page.locator('.v-data-table__tr')
  .filter({ hasText: /admin/ });
```

### Selector for Assigning/Changing a Role
```typescript
// Open actions menu for member
const actionsMenu = page.locator('.v-data-table__tr')
  .filter({ hasText: /member@example.com/ })
  .locator('v-icon').filter({ hasText: /mdi-dots-vertical/ });
await actionsMenu.click();

// Click "Change Role"
const changeRoleOption = page.locator('v-list-item').filter({ hasText: /Change Role/ });
await changeRoleOption.click();

// FormModal opens with role selector
const roleSelect = page.locator('.v-select');
```

### Selector for Role Dropdown Options
```typescript
// After clicking role select, dropdown opens
const roleDropdown = page.locator('.v-overlay--active .v-list-item');

// Select specific role
const adminRole = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /admin/i });
const agentRole = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /agent/i });
const developerRole = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /developer/i });
```

---

## 4. Logo and Bot Icon Upload

### How to Open the Logo Upload Dialog
```typescript
// Note: Logo upload is NOT on OrganizationPage.vue directly
// It's typically accessed from a different page or modal
// But LogoUploadDialog.vue shows the pattern

// If there's a "Upload Logo" button on the page:
const uploadLogoBtn = page.locator('button').filter({ hasText: /Upload Logo|upload logo/i });
await uploadLogoBtn.click();

// Dialog opens via v-model
const logoDialog = page.locator('.v-dialog').filter({ hasText: /logo|Logo/ });
await logoDialog.waitFor({ state: 'visible' });
```

### Selector for the File Input (for setInputFiles())
```typescript
// Inside LogoUploadDialog
const fileInput = page.locator('input[type="file"]').first();

// Set file for upload
await fileInput.setInputFiles('/path/to/logo.png');
```

### Selector for the Upload/Save Button
```typescript
// Upload button in LogoUploadDialog
const uploadBtn = page.locator('button').filter({ hasText: /Upload Logo|upload/i });

// Or by class
const uploadBtn = page.locator('.bg-btn-primary').filter({ hasText: /Upload/ });
```

### How to Verify the Upload Was Successful
```typescript
// Check for success snackbar
const snackbar = page.locator('.v-snackbar');
await expect(snackbar).toContainText('success');
await expect(snackbar).toContainText('Logo|logo');

// Or check that dialog closed
const logoDialog = page.locator('.v-dialog').filter({ hasText: /logo/ });
await expect(logoDialog).toHaveCount(0);
```

### How to Open the Bot Icon Upload
```typescript
// Bot icons section on OrganizationPage
// "Upload Icon" button
const uploadIconBtn = page.locator('button').filter({ hasText: /Upload Icon|upload icon/i });
await uploadIconBtn.click();

// File input for icon upload
const fileInput = page.locator('input[type="file"][accept*="image"]').nth(1);
```

---

## 5. Organization Switching

### How to Access Org Switching (Dropdown in Header or Sidebar)
```typescript
// Organization selector trigger (in header/sidebar)
const orgTrigger = page.locator('.org-dropdown-trigger');

// Click to open dropdown
await orgTrigger.click();
await page.waitForTimeout(300);
```

### Selector for the Org Switcher Trigger
```typescript
// Org dropdown trigger
const orgDropdown = page.locator('.org-dropdown-trigger');

// Or by text (shows current org name)
const orgTrigger = page.locator('.org-dropdown-trigger')
  .filter({ hasText: /Testing2026!/ });
```

### Selector for Org Items in the Switcher Dropdown
```typescript
// After clicking trigger, menu opens (teleported to body)
const orgItems = page.locator('.org-dropdown-item');

// Select specific org by name
const orgItem = page.locator('.org-dropdown-item')
  .filter({ hasText: /Testing2026!/ });

// Or by icon + name
const orgItem = page.locator('.org-dropdown-item')
  .filter({ hasText: /Te/ }); // First 2 letters of org name
```

### How to Verify the Org Was Switched
```typescript
// After clicking org item, page reloads
// Wait for page reload
await page.waitForLoadState('networkidle');

// Verify org name in trigger
const orgName = page.locator('.org-dropdown-trigger .org-name');
await expect(orgName).toContainText('Testing2026!');

// Or check URL still contains /organization
await page.waitForURL(/\/organization/);
```

---

## 6. FormModal Pattern

### How FormModal Renders in the DOM (Overlay Selector)
```typescript
// FormModal uses v-dialog, renders as .v-overlay--active
const formModal = page.locator('.v-overlay--active').filter({ hasText: /Add Member|Change Role/ });

// Or by specific title
const modal = page.locator('.v-dialog').filter({ hasText: /Add Member Modal/ });
```

### Selector for the Modal Title
```typescript
// Modal title (h2 inside v-card)
const title = page.locator('.v-overlay--active h2');

// Or filter by text
const title = page.locator('.v-overlay--active h2').filter({ hasText: /Add Member/ });
```

### Selector for Inputs Inside FormModal
```typescript
// Email input (first GlobalFormField)
const emailInput = page.locator('.v-text-field').nth(0).locator('input');

// Role select (second field)
const roleSelect = page.locator('.v-select').nth(0);

// Or by label
const emailInput = page.locator('label').filter({ hasText: /Email/ })
  .locator('..').locator('input').first();
```

### Selector for the Save/Confirm Button Inside FormModal
```typescript
// Primary button (Add/Confirm)
const saveBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /^Add$|^Confirm$|^Apply$/i });

// Or by class
const saveBtn = page.locator('.bg-btn-primary').filter({ hasText: /Add|Confirm|Apply/ });
```

### Selector for the Cancel/Close Button
```typescript
// Close button (X in top right)
const closeBtn = page.locator('.v-overlay--active button[icon="mdi-close"]');

// Secondary button (Cancel)
const cancelBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /^Cancel$/i });
```

---

## 7. AlertModal (Confirmation Dialog)

### Selector for the Alert/Confirmation Modal
```typescript
// AlertModal uses v-dialog, renders as .v-overlay--active
const alertModal = page.locator('.v-overlay--active').filter({ hasText: /deactivate|confirm|alert/i });

// Or by icon
const alertModal = page.locator('.v-overlay--active').filter({ has: page.locator('.v-icon').filter({ hasText: /mdi-alert/ }) });
```

### Selector for the Confirm/Yes Button
```typescript
// Primary button (Confirm/Yes)
const confirmBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /^Confirm$|^Yes$|^Delete$/i });

// Or by class
const confirmBtn = page.locator('.bg-btn-primary').filter({ hasText: /Confirm|Yes|Delete/ });
```

### Selector for the Cancel/No Button
```typescript
// Secondary button (Cancel/No)
const cancelBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /^Cancel$|^No$/i });

// Or by class
const cancelBtn = page.locator('button').filter({ hasText: /Cancel|No/ })
  .filter({ not: page.locator('.bg-btn-primary') });
```

---

## 8. Notifications

### Selector for Success Snackbar/Toast
```typescript
// Snackbar appears at bottom of screen
const snackbar = page.locator('.v-snackbar');

// Wait for it to appear
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Check for success message
await expect(snackbar).toContainText('success|updated|created|deleted|activated|deactivated');
```

### Selector for Error Snackbar/Toast
```typescript
// Error snackbar uses same selector
const errorSnackbar = page.locator('.v-snackbar');

// Check for error message
await expect(errorSnackbar).toContainText('error|failed|invalid');
```

### Typical Wait Timeout
```typescript
// Snackbars appear quickly: 3-5 seconds
const timeout = 5000; // 5 seconds

// Wait for snackbar to appear
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });

// Snackbar auto-dismisses after 3 seconds (default)
await page.locator('.v-snackbar').waitFor({ state: 'hidden', timeout: 10000 });
```

---

## 9. Common Workflows

### Add a New Member
```typescript
// Step 1: Click Add Member button
await page.locator('button').filter({ hasText: /Add Member/i }).click();

// Step 2: Wait for FormModal
const modal = page.locator('.v-overlay--active').filter({ hasText: /Add Member/ });
await modal.waitFor({ state: 'visible' });

// Step 3: Fill email
await page.locator('.v-text-field').nth(0).locator('input').fill('newuser@example.com');

// Step 4: Select role
await page.locator('.v-select').nth(0).click();
await page.waitForTimeout(300);
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /admin/i }).click();

// Step 5: Click Confirm
await page.locator('.bg-btn-primary').filter({ hasText: /Add/i }).click();

// Step 6: Verify success
await expect(page.locator('.v-snackbar')).toContainText('invite sent');
```

### Deactivate a Member
```typescript
// Step 1: Find member row
const memberRow = page.locator('.v-data-table__tr').filter({ hasText: /member@example.com/ });

// Step 2: Click actions menu
await memberRow.locator('v-icon').filter({ hasText: /mdi-dots-vertical/ }).click();

// Step 3: Click Deactivate
await page.locator('v-list-item').filter({ hasText: /Deactivate/i }).click();

// Step 4: Confirm in AlertModal
const confirmBtn = page.locator('.v-overlay--active button').filter({ hasText: /^Confirm$/i });
await confirmBtn.click();

// Step 5: Verify success
await expect(page.locator('.v-snackbar')).toContainText('deactivated');
```

### Change Member Role
```typescript
// Step 1: Click actions menu for member
const memberRow = page.locator('.v-data-table__tr').filter({ hasText: /member@example.com/ });
await memberRow.locator('v-icon').filter({ hasText: /mdi-dots-vertical/ }).click();

// Step 2: Click Change Role
await page.locator('v-list-item').filter({ hasText: /Change Role/i }).click();

// Step 3: Wait for FormModal
const modal = page.locator('.v-overlay--active').filter({ hasText: /Change Role/ });
await modal.waitFor({ state: 'visible' });

// Step 4: Select new role
await page.locator('.v-select').nth(0).click();
await page.waitForTimeout(300);
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /developer/i }).click();

// Step 5: Click Apply
await page.locator('.bg-btn-primary').filter({ hasText: /Apply/i }).click();

// Step 6: Confirm in AlertModal
const confirmBtn = page.locator('.v-overlay--active button').filter({ hasText: /^Confirm$/i });
await confirmBtn.click();

// Step 7: Verify success
await expect(page.locator('.v-snackbar')).toContainText('role changed');
```

### Switch Organization
```typescript
// Step 1: Click org selector trigger
const orgTrigger = page.locator('.org-dropdown-trigger');
await orgTrigger.click();

// Step 2: Wait for dropdown to open
await page.waitForTimeout(300);

// Step 3: Click organization
const orgItem = page.locator('.org-dropdown-item').filter({ hasText: /Testing2026!/ });
await orgItem.click();

// Step 4: Wait for page reload
await page.waitForLoadState('networkidle');

// Step 5: Verify org switched
await expect(page.locator('.org-dropdown-trigger')).toContainText('Testing2026!');
```

---

## 10. Test Credentials & URLs

```typescript
const testCredentials = {
  email: 'heidi@intnt.ai',
  password: 'testing2026!',
  organization: 'Testing2026!'
};

const urls = {
  login: 'https://dashboard.int3nt.info/login',
  organization: 'https://dashboard.int3nt.info/organization',
  dashboard: 'https://dashboard.int3nt.info'
};
```

---

## 11. Key Selectors Summary

| Element | Selector |
|---------|----------|
| Organization page container | `.organization-container` |
| Members table | `.organization-table` |
| Table row | `.v-data-table__tr` |
| Add Member button | `button:has-text("Add Member")` |
| Email input | `.v-text-field:nth(0) input` |
| Role select | `.v-select:nth(0)` |
| Confirm button | `.bg-btn-primary:has-text("Add\|Confirm\|Apply")` |
| Cancel button | `button:has-text("Cancel")` |
| Org switcher trigger | `.org-dropdown-trigger` |
| Org dropdown item | `.org-dropdown-item` |
| Success snackbar | `.v-snackbar:has-text("success\|updated")` |
| Error snackbar | `.v-snackbar:has-text("error\|failed")` |
| FormModal | `.v-overlay--active:has-text("Add Member\|Change Role")` |
| AlertModal | `.v-overlay--active:has-text("confirm\|alert\|deactivate")` |

---

**End of Reference Document**
