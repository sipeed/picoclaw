# Settings Page — Playwright Reference

> Generated: 2026-04-17
> Source: `/home/picoclaw/.picoclaw/workspace/context/dashboard/src`
> App: dashboard.int3nt.info (Vue 3 + Vuetify 3)

---

## 0. Files Read

All seven Phase 1 files were read in full before producing this document:

1. `src/pages/SettingsPage.vue`
2. `src/components/FormModal.vue`
3. `src/components/AlertModal.vue`
4. `src/components/common/TabsComponent.vue`
5. `src/components/common/BreadcrumbComponent.vue`
6. `src/stores/apiKey.store.ts`
7. `src/stores/snackbar.store.ts`

Additional files read for completeness:

8. `src/components/GlobalSnackBar.vue`
9. `src/locales/en/pages.json` (settingsPage keys)
10. `skills/app-selectors-settings/SKILL.md` (live DOM snapshot)

---

## 1. Settings Page Navigation

### Sidebar selector
```typescript
// The sidebar is a <nav class="nav-drawer">
// The Settings link is an <a> tag inside it
await page.locator('a:has-text("Settings")').click();
```

### Page URL
```
/settings
```

### Verify the Settings page is loaded
```typescript
await page.waitForURL(/\/settings/, { timeout: 15000 });

// Wait for the page container
await page.locator('.settings-container').waitFor({ state: 'visible', timeout: 10000 });

// Confirm the section heading is visible
await expect(page.locator('.section-title')).toContainText('API Keys');
```

### Full navigation snippet (after login + org selection)
```typescript
await page.locator('a:has-text("Settings")').click();
await page.waitForURL(/\/settings/, { timeout: 15000 });
await page.locator('.settings-container').waitFor({ state: 'visible', timeout: 10000 });
```

---

## 2. API Key List

### Table selector
```typescript
// The v-data-table-server renders with class .api-keys-table
page.locator('.api-keys-table')
```

### Locate a row by description text
```typescript
// Rows are standard <tr> elements inside the Vuetify data table
// Scope by a known text value in the Description column
page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' })
```

### Locate a row by censored key value (API Key ID column)
```typescript
// The api_key_censored field renders as plain text in the first column
page.locator('.api-keys-table tr').filter({ hasText: 'e8d40*****1d3f3' })
```

### Key status chip
```typescript
// Each row has a v-chip for status. Colors map to states:
//   'success' (green outline) → Active
//   'error'  (red outline)   → Revoked
//   'warning' (yellow outline) → Expired

// Get the chip in a specific row:
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
const statusChip = row.locator('.v-chip');
await expect(statusChip).toContainText('Active');   // or 'Revoked' / 'Expired'

// Check chip color class:
await expect(statusChip).toHaveClass(/v-chip--variant-outlined/);
// For revoked: chip has color="error"
// For active:  chip has color="success"
// For expired: chip has color="warning"
```

### Key value (censored display — api_key_censored)
```typescript
// The censored key value is in the first data column (API Key ID)
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
const keyCell = row.locator('td').nth(0);
await expect(keyCell).toContainText('*****'); // censored format: e.g. e8d40*****1d3f3
```

---

## 3. Create API Key

### "Add new API Key" button
```typescript
// Button class: .add-btn (orange background)
// Button text: "Add new API Key"
await page.locator('.add-btn').click();
// OR by role:
await page.getByRole('button', { name: /Add new API Key/i }).click();
```

### Wait for the Create modal to open
```typescript
// FormModal renders a v-dialog → .v-overlay--active containing a v-card
// The modal has NO custom class on the card itself — use Type B modal pattern
// Scope by the visible title text "Create New API Key"
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });

// Alternative: wait for the close button (confirmed present in DOM snapshot)
await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 });
```

### Role selector (select field — index 0 inside modal)
```typescript
// The Role field is a v-select (GlobalFormField type="select")
// It is the FIRST visible dropdown inside the modal overlay
// Label: "Role *"   Options: Internal | External

// Open the role dropdown:
await page.locator('.v-overlay--active .v-select').nth(0).click();
await page.waitForTimeout(500);

// Pick an option from the teleported list:
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Internal/ }).click();
await page.waitForTimeout(300);

// NOTE: From the live DOM snapshot, the Role dropdown inside the modal is:
// page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()
// (nth(0) is the items-per-page select in the table; nth(1) is the Role field in the modal)
await page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click();
await page.waitForTimeout(500);
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Internal/ }).click();
await page.waitForTimeout(300);
```

### "Expires in Days" input
```typescript
// Placeholder: "Enter number of days"   Field type: text   Key: expires_in_days
// Scoped inside the modal overlay
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });
const expiresInput = modal.locator('input[placeholder="Enter number of days"]');
await expiresInput.click();
await expiresInput.fill('365');
await expiresInput.press('Tab');
```

### Description input (optional)
```typescript
// Placeholder: "Enter description"   Field type: text   Key: description
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });
const descriptionInput = modal.locator('input[placeholder="Enter description"]');
await descriptionInput.click();
await descriptionInput.fill('My test key');
await descriptionInput.press('Tab');
```

### Confirm / Create button
```typescript
// Button text: "Create API Key"
// Located inside the modal v-card
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });
await modal.locator('button').filter({ hasText: /^Create API Key$/ }).click();
```

### How the new key value is displayed after creation
After a successful create:
1. The FormModal closes (`showAddApiKeyModal = false`).
2. A **separate success modal** opens — a plain `v-dialog` with `v-card.api-key-success-modal`.
3. The key value is shown in a `v-text-field` (class `.api-key-input`) as masked asterisks by default.
4. Clicking the eye button (`.eye-btn`) reveals the plain-text key.

```typescript
// Wait for the success modal card
const successModal = page.locator('.api-key-success-modal');
await successModal.waitFor({ state: 'visible', timeout: 10000 });

// The key value field (masked by default)
const keyField = successModal.locator('.api-key-input input');
await expect(keyField).toBeVisible();

// Reveal the key by clicking the eye button
await successModal.locator('.eye-btn').click();
// Now keyField shows the real key value
```

### Copy the key value
```typescript
// "Copy" button inside the success modal actions area
await successModal.locator('.copy-btn').click();
// OR by text:
await successModal.locator('button').filter({ hasText: /^Copy$/ }).click();
```

### Success snackbar after creation
```typescript
// Text: "API Key created successfully"  (i18n: settingsPage.apiKeyCreatedSuccess)
// Color: green
await expect(page.locator('.v-snackbar')).toContainText('API Key created successfully');
```

---

## 4. Edit API Key

### Open the edit modal (pencil icon button on a row)
```typescript
// Each row has a v-btn icon="mdi-pencil" with class .edit-btn
// To click the edit button for a specific row, filter by row content first:
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
await row.locator('.edit-btn').click();

// OR by row index (0-based, skipping the header row):
await page.locator('.edit-btn').nth(0).click();
```

### Wait for the Edit modal
```typescript
// Title: "Edit API Key"
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });
```

### Status selector inside edit modal (revoked/active toggle)
```typescript
// The Status field is a v-select with options: Active | Revoked
// Key: "revoked"   Values: 'false' (Active) | 'true' (Revoked)
// It is the first v-select inside the edit modal overlay

const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
await modal.locator('.v-select').nth(0).click();
await page.waitForTimeout(500);
// Pick from teleported overlay:
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Active/ }).click();
await page.waitForTimeout(300);
```

### Description input in edit modal
```typescript
// Placeholder: "Enter description"   Key: description
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
const descInput = modal.locator('input[placeholder="Enter description"]');
await descInput.click();
await descInput.fill('Updated description');
await descInput.press('Tab');
```

### Save button
```typescript
// Button text: "Save"
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
await modal.locator('button').filter({ hasText: /^Save$/ }).click();
```

### Success snackbar after edit
```typescript
// Text: "API Key updated successfully"  (i18n: settingsPage.apiKeyUpdatedSuccess)
await expect(page.locator('.v-snackbar')).toContainText('API Key updated successfully');
```

---

## 5. Revoke API Key

> **There is no dedicated "Revoke" button.** Revocation is done through the **Edit modal** by
> changing the Status dropdown from "Active" to "Revoked".

### Steps to revoke
```typescript
// 1. Open the edit modal for the target row
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
await row.locator('.edit-btn').click();

// 2. Wait for edit modal
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });

// 3. Change Status to "Revoked"
await modal.locator('.v-select').nth(0).click();
await page.waitForTimeout(500);
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /^Revoked$/ }).click();
await page.waitForTimeout(300);

// 4. Save
await modal.locator('button').filter({ hasText: /^Save$/ }).click();
```

### How the key status changes after revoke
```typescript
// The row's v-chip switches to color="error" (red outline) and text "Revoked"
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
await expect(row.locator('.v-chip')).toContainText('Revoked');
```

### Success snackbar after revoke
```typescript
// Text: "API Key updated successfully"  (same as edit — store calls updateApiKey)
await expect(page.locator('.v-snackbar')).toContainText('API Key updated successfully');
```

---

## 6. Reactivate API Key

> **There is no dedicated "Reactivate" button.** Reactivation is done through the **Edit modal**
> by changing the Status dropdown from "Revoked" to "Active".

### Steps to reactivate
```typescript
// 1. Open the edit modal for the revoked key row
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
await row.locator('.edit-btn').click();

// 2. Wait for edit modal
const modal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });

// 3. Change Status to "Active"
await modal.locator('.v-select').nth(0).click();
await page.waitForTimeout(500);
await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /^Active$/ }).click();
await page.waitForTimeout(300);

// 4. Save
await modal.locator('button').filter({ hasText: /^Save$/ }).click();
```

### How the key status changes after reactivation
```typescript
// The row's v-chip switches to color="success" (green outline) and text "Active"
const row = page.locator('.api-keys-table tr').filter({ hasText: 'my-key-description' });
await expect(row.locator('.v-chip')).toContainText('Active');
```

### Success snackbar after reactivation
```typescript
// Text: "API Key updated successfully"
await expect(page.locator('.v-snackbar')).toContainText('API Key updated successfully');
```

---

## 7. Tabs

The Settings page has two tabs rendered by `TabsComponent`. The active tab is tracked by
`activeTab` ref (default: `'api-keys'`).

### Tab selectors
```typescript
// Tabs are <div class="tab"> elements inside <div class="tab-header">
// The active tab has class "tab active"

// "API Keys" tab (id: 'api-keys')
const apiKeysTab = page.locator('.tab-header .tab').filter({ hasText: /^API Keys$/ });

// "Other Settings" tab (id: 'other-settings')
const otherSettingsTab = page.locator('.tab-header .tab').filter({ hasText: /^Other Settings$/ });

// Click a tab:
await apiKeysTab.click();
await otherSettingsTab.click();
```

### Verify the active tab
```typescript
// The active tab has class "tab active"
await expect(page.locator('.tab-header .tab').filter({ hasText: /^API Keys$/ }))
  .toHaveClass(/active/);

// Verify the correct content panel is shown:
// API Keys tab → .api-keys-content is visible
await expect(page.locator('.api-keys-content')).toBeVisible();

// Other Settings tab → .other-settings-content is visible
await expect(page.locator('.other-settings-content')).toBeVisible();
```

### Orange underline indicator
```typescript
// The active tab has an animated underline: <div class="tab-underline">
// It is positioned absolutely using inline styles (width + left).
// Do NOT rely on its position for assertions — use the "active" class instead.
```

---

## 8. FormModal Pattern

`FormModal.vue` is used for both the **Create API Key** and **Edit API Key** modals.
It renders a `v-dialog` (no custom class on the card) inside a `.v-overlay--active` overlay.

### Overlay selector (Type B — no custom class)
```typescript
// NEVER use .v-dialog, [role="dialog"], or .v-dialog--active
// Always scope to .v-overlay--active filtered by the modal's heading text

// Create modal:
const createModal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });

// Edit modal:
const editModal = page.locator('.v-overlay--active').filter({ hasText: /Edit API Key/ });
```

### Wait for modal to be visible
```typescript
await createModal.waitFor({ state: 'visible', timeout: 10000 });
// Alternative: wait for the close button (always present when modal is open)
await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 });
```

### Inputs inside FormModal
```typescript
// FormModal fields are rendered by <GlobalFormField> which wraps Vuetify inputs.
// Use placeholder text to target text inputs (most reliable):
const modal = page.locator('.v-overlay--active').filter({ hasText: /Create New API Key/ });

// Text input by placeholder:
modal.locator('input[placeholder="Enter number of days"]')
modal.locator('input[placeholder="Enter description"]')

// Select / dropdown — use .v-select by index inside the modal:
modal.locator('.v-select').nth(0)   // first select in modal (Role in Create; Status in Edit)

// Always click → fill → Tab to trigger Vue validation:
await input.click();
await input.fill('value');
await input.press('Tab');
```

### Save / Confirm button
```typescript
// Create modal: "Create API Key"
await modal.locator('button').filter({ hasText: /^Create API Key$/ }).click();

// Edit modal: "Save"
await modal.locator('button').filter({ hasText: /^Save$/ }).click();
```

### Cancel / Close button
```typescript
// Cancel button (secondary, grey background):
await modal.locator('button').filter({ hasText: /^Cancel$/ }).click();

// X close button (top-right, class .close-btn):
await modal.locator('.close-btn').click();
// Note: .close-btn is a v-btn with icon="mdi-close" positioned absolutely top-right of the card
```

### Disabled state
```typescript
// The primary button is disabled while the store loading flag is true:
//   Create: apiKeyStore.loading.creating
//   Edit:   apiKeyStore.loading.updating
// Wait for it to be enabled before clicking:
await expect(modal.locator('button').filter({ hasText: /^Create API Key$/ }))
  .not.toBeDisabled({ timeout: 10000 });
```

---

## 9. Notifications

All notifications are rendered by `GlobalSnackBar.vue` using Vuetify's `v-snackbar`.
The snackbar is positioned `bottom right`. Default timeout: **3000 ms**.

### Success snackbar
```typescript
// Selector: .v-snackbar (Vuetify renders this in the DOM when active)
await expect(page.locator('.v-snackbar')).toBeVisible({ timeout: 5000 });
await expect(page.locator('.v-snackbar')).toContainText('API Key created successfully');
```

### Error snackbar
```typescript
// Errors from the store use severity 'red' but still render in .v-snackbar
await expect(page.locator('.v-snackbar')).toBeVisible({ timeout: 5000 });
// No standard error text is shown to the user from SettingsPage — errors are console.error only.
// If a backend error occurs, the snackbar may not appear; check for absence of success message.
```

### Exact snackbar text strings (from i18n en/pages.json)

| Action | Snackbar text |
|--------|--------------|
| Create API Key | `"API Key created successfully"` |
| Copy key to clipboard | `"API Key copied to clipboard"` |
| Edit / Update API Key | `"API Key updated successfully"` |

### Recommended wait pattern
```typescript
// Wait for snackbar to appear, then assert text:
const snackbar = page.locator('.v-snackbar');
await snackbar.waitFor({ state: 'visible', timeout: 5000 });
await expect(snackbar).toContainText('API Key created successfully');

// If you need to wait for the snackbar to disappear before the next step:
await snackbar.waitFor({ state: 'hidden', timeout: 8000 });
```

### Typical timeout guidance
```typescript
// Snackbar appears: within 1–2 s of action completing
// Snackbar auto-hides: after 3000 ms (default timeout in snackbar.store.ts)
// Use waitFor timeout of 5000 ms for appearance assertions
// Use waitFor timeout of 8000 ms for disappearance assertions
```

---

## Appendix — Quick Selector Reference

| Element | Selector |
|---------|----------|
| Settings page container | `.settings-container` |
| Sidebar → Settings link | `a:has-text("Settings")` |
| Breadcrumb nav | `.breadcrumb` |
| Tab bar | `.tab-header` |
| "API Keys" tab | `.tab-header .tab` filtered `hasText: /^API Keys$/` |
| "Other Settings" tab | `.tab-header .tab` filtered `hasText: /^Other Settings$/` |
| Active tab | `.tab-header .tab.active` |
| API Keys table | `.api-keys-table` |
| Table row (by description) | `.api-keys-table tr` filtered `hasText: 'description'` |
| Status chip in row | `row.locator('.v-chip')` |
| Edit (pencil) button in row | `row.locator('.edit-btn')` |
| Role filter dropdown button | `.dropdown-button` `.nth(0)` |
| Status filter dropdown button | `.dropdown-button` `.nth(1)` |
| Add new API Key button | `.add-btn` |
| Create modal overlay | `.v-overlay--active` filtered `hasText: /Create New API Key/` |
| Edit modal overlay | `.v-overlay--active` filtered `hasText: /Edit API Key/` |
| Modal close (X) button | `.close-btn` (inside modal) |
| Modal Cancel button | `modal.locator('button').filter({ hasText: /^Cancel$/ })` |
| Modal Create button | `modal.locator('button').filter({ hasText: /^Create API Key$/ })` |
| Modal Save button | `modal.locator('button').filter({ hasText: /^Save$/ })` |
| Role select in Create modal | `.v-select:visible` etc `.nth(1)` (0 is table pagination) |
| Expires in Days input | `input[placeholder="Enter number of days"]` |
| Description input | `input[placeholder="Enter description"]` |
| Success modal card | `.api-key-success-modal` |
| Key value field (success modal) | `.api-key-success-modal .api-key-input input` |
| Eye (show/hide) button | `.api-key-success-modal .eye-btn` |
| Copy button (success modal) | `.api-key-success-modal .copy-btn` |
| Snackbar | `.v-snackbar` |
