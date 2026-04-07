# Settings Page Reference Document

**Generated:** 2026-04-07 14:00  
**Status:** Complete

---

## 0. Files Read

✅ **All 7 files read successfully:**

1. `/src/pages/SettingsPage.vue` — Main settings page with API key management and tabs
2. `/src/components/FormModal.vue` — Form modal component with dynamic fields
3. `/src/components/AlertModal.vue` — Alert/confirmation modal component
4. `/src/components/common/TabsComponent.vue` — Tab navigation component
5. `/src/components/common/BreadcrumbComponent.vue` — Breadcrumb navigation
6. `/src/stores/apiKey.store.ts` — Pinia store for API key operations
7. `/src/stores/snackbar.store.ts` — Snackbar notification store

---

## 1. Settings Page Navigation

### Sidebar Selector to Navigate to Settings
```typescript
// From any page with sidebar, click Settings link
await page.locator('a:has-text("Settings")').click();

// Or navigate directly
await page.goto('https://dashboard.int3nt.info/settings');

// Verify URL
await page.waitForURL(/\/settings/);
```

### Page URL or Route Identifier
```typescript
// Expected URL after navigation
// URL: https://dashboard.int3nt.info/settings (or /settings route)

// Verify page loaded
await page.waitForURL(/\/settings/);
```

### How to Verify Settings Page is Loaded
```typescript
// Wait for breadcrumb
const breadcrumb = page.locator('nav.breadcrumb');
await breadcrumb.waitFor({ state: 'visible', timeout: 10000 });

// Wait for tabs
const tabs = page.locator('.tabs');
await tabs.waitFor({ state: 'visible', timeout: 10000 });

// Wait for section title
const title = page.locator('.section-title').first();
await expect(title).toContainText('API Keys');

// Wait for Add New API Key button
const addBtn = page.locator('.add-btn');
await addBtn.waitFor({ state: 'visible', timeout: 10000 });
```

---

## 2. API Key List

### Selector for the API Keys Table/List
```typescript
// API keys table
const apiKeysTable = page.locator('.api-keys-table');

// Table rows
const tableRows = page.locator('.api-keys-table tbody tr');

// Get specific row count
const rowCount = await page.locator('.api-keys-table tbody tr').count();
```

### How to Locate an API Key Row by Name
```typescript
// Find row by API key censored value (e.g., "sk-***1234")
const apiKeyRow = page.locator('.api-keys-table tbody tr')
  .filter({ hasText: /sk-\*\*\*/ });

// Find row by description
const apiKeyRow = page.locator('.api-keys-table tbody tr')
  .filter({ hasText: /my-api-key-description/ });

// Get specific cell value
const cellValue = apiKeyRow.locator('td').nth(2); // Description is 3rd column
```

### Selector for Key Status (Active/Revoked)
```typescript
// Status chip (shows "Active" or "Revoked")
const statusChip = page.locator('.v-chip').filter({ hasText: /Active|Revoked/ });

// Get status for specific key
const keyRow = page.locator('.api-keys-table tbody tr').first();
const status = keyRow.locator('.v-chip').filter({ hasText: /Active|Revoked/ });

// Check if active or revoked
const isActive = await status.filter({ hasText: /Active/ }).isVisible();
const isRevoked = await status.filter({ hasText: /Revoked/ }).isVisible();
```

### Selector for the Key Value (Masked or Visible)
```typescript
// Masked key value (e.g., "sk-***1234")
const maskedKey = page.locator('.api-keys-table tbody tr').first().locator('td').nth(0);

// Full key value in success modal
const fullKey = page.locator('.api-key-input').locator('input');

// Get key value
const keyValue = await fullKey.inputValue();
```

---

## 3. Create API Key

### Selector for the Create/Add API Key Button
```typescript
// Add New API Key button
const addBtn = page.locator('.add-btn');

// Or by text
const addBtn = page.locator('button').filter({ hasText: /Add New API Key|add new api key/i });
```

### Selector for the Modal That Opens
```typescript
// FormModal when creating API key
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });

// Or by v-dialog
const modal = page.locator('.v-dialog').first();
```

### Selector for the Key Name Input
```typescript
// Note: "Key Name" doesn't exist in SettingsPage
// Instead, there are: Role, Expires In Days, Description

// Role select field (first field in form)
const roleSelect = page.locator('.v-select').nth(0);

// Expires In Days input (second field)
const expiresInput = page.locator('.v-text-field').nth(1).locator('input');

// Description input (third field)
const descriptionInput = page.locator('.v-text-field').nth(2).locator('input');
```

### Selector for the Key Type Selector (Internal/External, if Applicable)
```typescript
// Role dropdown (internal/external)
const roleSelect = page.locator('.v-select').nth(0);

// Click to open dropdown
await roleSelect.click();
await page.waitForTimeout(300);

// Select role
const roleOption = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /Internal|External/ });
await roleOption.click();
```

### Selector for the Confirm/Create Button
```typescript
// Create API Key button in modal
const createBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /Create API Key|create api key/i });

// Or by class
const createBtn = page.locator('.bg-btn-primary').filter({ hasText: /Create/ });
```

### How the New Key Value is Displayed After Creation
```typescript
// After creation, a success modal appears with the full API key
// The key is displayed in a read-only text field
const keyField = page.locator('.api-key-input').locator('input');

// Get the key value
const apiKey = await keyField.inputValue();

// Key is initially masked with asterisks
// Can be revealed by clicking eye icon
const eyeBtn = page.locator('.eye-btn');
await eyeBtn.click();
```

### Selector for Copying the Key Value
```typescript
// Copy button in success modal
const copyBtn = page.locator('.copy-btn');

// Click to copy
await copyBtn.click();

// Success notification appears
await expect(page.locator('.v-snackbar')).toContainText('copied');
```

### Success Notification Selector and Text
```typescript
// Success snackbar after creating API key
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('API Key Created Successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 4. Edit API Key

### How to Open the Edit Modal (Button Selector on a Key Row)
```typescript
// Edit button (pencil icon) in each row
const editBtn = page.locator('.edit-btn').first();

// Or find edit button for specific row
const keyRow = page.locator('.api-keys-table tbody tr').first();
const editBtn = keyRow.locator('.edit-btn');

// Click to open edit modal
await editBtn.click();
```

### Selector for the Name Input in the Edit Modal
```typescript
// Note: Edit modal has different fields than Create modal
// Edit modal fields: Status (Active/Revoked), Description

// Status select in edit modal
const statusSelect = page.locator('.v-select').nth(0);

// Description input in edit modal
const descriptionInput = page.locator('.v-text-field').nth(0).locator('input');
```

### Selector for the Save Button
```typescript
// Save button in edit modal
const saveBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /Save|save/i });

// Or by class
const saveBtn = page.locator('.bg-btn-primary').filter({ hasText: /Save/ });
```

### Success Notification Selector and Text
```typescript
// Success snackbar after saving
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('API Key Updated Successfully');

// Wait for snackbar
await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 5. Revoke API Key

### Selector for the Revoke Button on a Key Row
```typescript
// Edit button (which opens modal to revoke)
const editBtn = page.locator('.api-keys-table tbody tr').first().locator('.edit-btn');

// After opening modal, change status to "Revoked"
const statusSelect = page.locator('.v-select').nth(0);
await statusSelect.click();
const revokeOption = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /Revoked/ });
await revokeOption.click();
```

### Selector for the Confirmation Modal/Dialog
```typescript
// Edit modal (used for both edit and revoke)
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key|edit api key/i });

// Or just check for modal
const modal = page.locator('.v-dialog').first();
```

### Selector for the Confirm Revoke Button
```typescript
// Save button in modal (after changing status to Revoked)
const saveBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /Save|save/i });
```

### How the Key Status Changes After Revoke (Visual Indicator)
```typescript
// After revoke, the status chip changes from "Active" to "Revoked"
const statusChip = page.locator('.api-keys-table tbody tr').first()
  .locator('.v-chip').filter({ hasText: /Revoked/ });

// Verify status changed
await expect(statusChip).toBeVisible();
```

### Success Notification Selector and Text
```typescript
// Success snackbar after revoke
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('API Key Updated Successfully');
```

---

## 6. Reactivate API Key

### Selector for the Reactivate Button on a Revoked Key Row
```typescript
// Edit button on revoked key row
const revokedKeyRow = page.locator('.api-keys-table tbody tr')
  .filter({ has: page.locator('.v-chip').filter({ hasText: /Revoked/ }) });
const editBtn = revokedKeyRow.locator('.edit-btn').first();

// Click to open edit modal
await editBtn.click();
```

### Selector for the Confirmation Modal/Dialog (if Any)
```typescript
// Edit modal
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });

// Change status to "Active"
const statusSelect = modal.locator('.v-select').nth(0);
await statusSelect.click();
const activeOption = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /Active/ });
await activeOption.click();
```

### How the Key Status Changes After Reactivation
```typescript
// After reactivation, status changes from "Revoked" to "Active"
const statusChip = page.locator('.api-keys-table tbody tr').first()
  .locator('.v-chip').filter({ hasText: /Active/ });

// Verify status changed
await expect(statusChip).toBeVisible();
```

### Success Notification Selector and Text
```typescript
// Success snackbar after reactivation
const snackbar = page.locator('.v-snackbar');

// Check for success message
await expect(snackbar).toContainText('API Key Updated Successfully');
```

---

## 7. Tabs (if Settings Has Multiple Tabs)

### Selector for Each Tab
```typescript
// Tabs container
const tabs = page.locator('.tabs');

// Individual tab by label
const apiKeysTab = page.locator('.tab').filter({ hasText: /API Keys/ });
const otherSettingsTab = page.locator('.tab').filter({ hasText: /Other Settings/ });

// Click tab
await apiKeysTab.click();
```

### How to Verify the Active Tab
```typescript
// Active tab has class "active"
const activeTab = page.locator('.tab.active');

// Verify which tab is active
const isApiKeysActive = await page.locator('.tab').filter({ hasText: /API Keys/ })
  .evaluate(el => el.classList.contains('active'));

// Check underline position
const underline = page.locator('.tab-underline');
await expect(underline).toBeVisible();
```

---

## 8. FormModal Pattern

### How FormModal Renders in the DOM (Overlay Selector)
```typescript
// FormModal uses v-dialog, renders as .v-overlay--active
const formModal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key|Edit API Key/ });

// Or by v-dialog
const modal = page.locator('.v-dialog').first();

// Wait for modal to be visible
await modal.waitFor({ state: 'visible', timeout: 10000 });
```

### Selector for Inputs Inside FormModal
```typescript
// Inside FormModal, inputs are GlobalFormField components
// They render as v-text-field or v-select

// Text input (e.g., Expires In Days)
const textInput = page.locator('.v-text-field').nth(0).locator('input');

// Select input (e.g., Role)
const selectInput = page.locator('.v-select').nth(0);

// Fill text input
await textInput.fill('30');

// Select dropdown option
await selectInput.click();
await page.waitForTimeout(300);
const option = page.locator('.v-overlay--active .v-list-item')
  .filter({ hasText: /Internal/ });
await option.click();
```

### Selector for Save/Confirm Button
```typescript
// Primary button (Create/Save)
const saveBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /Create API Key|Save|Edit/i });

// Or by class
const saveBtn = page.locator('.bg-btn-primary').filter({ hasText: /Create|Save|Edit/ });

// Click button
await saveBtn.click();
```

### Selector for Cancel/Close Button
```typescript
// Secondary button (Cancel)
const cancelBtn = page.locator('.v-overlay--active button')
  .filter({ hasText: /Cancel|cancel/i });

// Or close button (X)
const closeBtn = page.locator('.v-overlay--active .v-btn[icon="mdi-close"]');

// Click to close
await closeBtn.click();
```

---

## 9. Notifications

### Selector for Success Snackbar/Toast
```typescript
// Snackbar appears at bottom of screen
const snackbar = page.locator('.v-snackbar');

// Wait for snackbar to appear
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Check for success message
await expect(snackbar).toContainText('success|created|updated|copied');

// Common success messages:
// - "API Key Created Successfully"
// - "API Key Updated Successfully"
// - "API Key Copied"
```

### Selector for Error Snackbar/Toast
```typescript
// Error snackbar uses same selector
const errorSnackbar = page.locator('.v-snackbar');

// Check for error message
await expect(errorSnackbar).toContainText('error|failed|invalid');

// Common error messages:
// - "Failed to create API key"
// - "Failed to update API key"
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

## 10. Key Selectors Summary

| Element | Selector |
|---------|----------|
| Settings container | `.settings-container` |
| Add New API Key button | `.add-btn` |
| API Keys table | `.api-keys-table` |
| Table row | `.api-keys-table tbody tr` |
| Edit button | `.edit-btn` |
| Status chip | `.v-chip:has-text("Active\|Revoked")` |
| Tabs | `.tab` |
| Active tab | `.tab.active` |
| FormModal | `.v-overlay--active` (filtered by heading) |
| Role select | `.v-select:nth(0)` |
| Expires input | `.v-text-field:nth(1) input` |
| Description input | `.v-text-field:nth(2) input` |
| Save button | `.bg-btn-primary:has-text("Create\|Save")` |
| Cancel button | `button:has-text("Cancel")` |
| Copy button | `.copy-btn` |
| Success snackbar | `.v-snackbar:has-text("success\|created\|updated")` |
| Error snackbar | `.v-snackbar:has-text("error\|failed")` |

---

**End of Reference Document**
