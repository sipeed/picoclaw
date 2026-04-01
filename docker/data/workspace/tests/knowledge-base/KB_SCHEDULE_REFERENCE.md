# Knowledge Base Schedule — Test Reference Document

## 0. Files Read

1. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseCard.vue`
2. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseDrawer.vue`
3. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseEmptyState.vue`
4. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseFilesModal.vue`
5. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseScheduleModal.vue`
6. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/knowledge-base/KnowledgeBaseStatusPill.vue`
7. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/pages/KnowledgeBasePage.vue`
8. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/stores/scheduleConfirm.store.ts`
9. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/stores/documentSearch.store.ts`

---

## 1. Knowledge Base List

### Navigation to Knowledge Base Page
- URL: `/knowledge-base`
- Sidebar link selector: `a:has-text("Knowledge Base")`
- Container: `.knowledge-base-container`
- Header: `h2` with text "Knowledge Base"

### Locate KB Bucket Card by Name
- Container: `.bucket-grid` (grid layout with 3 columns)
- Individual card: `.bucket-card` (Vuetify v-card with class `bucket-card`)
- Bucket name within card: `.bucket-name` (span element)
- To find "Picotest2" card: `.bucket-card` filtered by `.bucket-name` containing "Picotest2"

### Schedule Button on KB Card
- Selector: `.schedule-button` (Vuetify v-btn with text "Schedule")
- Located in `.action-buttons` section of the card
- Icon: `mdi-calendar-clock`
- Click to open Manage Schedule modal

---

## 2. Manage Schedule Modal

### Modal Selector
- Trigger: Click `.schedule-button` on a KB card
- Modal type: Vuetify `v-dialog` (standard modal, NOT custom drawer)
- Modal container: `.v-card` (rendered as dialog overlay)
- Modal title: "Manage Schedule"
- Modal subtitle: Document search name (e.g., "Picotest2")

### Verify Modal is Open
- Wait for modal to be visible: `page.locator('.v-dialog').filter({ hasText: /Manage Schedule/ }).waitFor({ state: 'visible', timeout: 10000 })`
- Or more specifically: `page.locator('.v-card-title').filter({ hasText: /Manage Schedule/ }).waitFor({ state: 'visible', timeout: 10000 })`

### Create Schedule Button
- Selector: `page.getByRole('button', { name: /Create Schedule/i })`
- Text: "Create Schedule"
- Visible when no schedule exists (initial state)
- Click opens the schedule form

---

## 3. Create/Edit Schedule Form

### Sync Type Selector (Radio Group)
- Component: `v-radio-group` (Vue component, renders as input[type="radio"] in DOM)
- Options: "Full Sync" and "Incremental Sync"
- **Correct Playwright selector for "Incremental Sync":**
  ```typescript
  page.getByRole('radio', { name: /Incremental Sync/i })
  ```
- Alternative (if role selector fails):
  ```typescript
  page.locator('input[type="radio"]').filter({ hasText: /Incremental Sync/ }).first()
  // or find label:
  page.locator('label').filter({ hasText: /Incremental Sync/ }).click()
  ```
- Source: `KnowledgeBaseScheduleModal.vue` line ~510: `<v-radio label="Full Sync" value="full" />` and `<v-radio label="Incremental Sync" value="incremental" />`

### Cron Expression Mode Toggle (SIMPLE vs ADVANCED)
- Component: `v-btn-toggle` (Vue component, renders as button elements in DOM)
- Two buttons: "Simple" and "Advanced"
- **Correct Playwright selector:**
  ```typescript
  // Find the toggle container and click the button
  page.locator('button').filter({ hasText: /Simple/i }).click()
  page.locator('button').filter({ hasText: /Advanced/i }).click()
  ```
- Or use button role:
  ```typescript
  page.getByRole('button', { name: /Simple/i }).click()
  page.getByRole('button', { name: /Advanced/i }).click()
  ```
- Default mode: "Simple"
- Source: `KnowledgeBaseScheduleModal.vue` line ~515-523: `<v-btn-toggle v-model="cronMode" ...>`

### SIMPLE Mode Fields

#### Frequency Dropdown
- Component: `v-select`
- Label: "Frequency"
- Options: "Hourly", "Daily", "Weekly", "Monthly"
- **Open dropdown:**
  ```typescript
  page.locator('.v-select').first().click()
  await page.waitForTimeout(500)
  ```
- **Select option (e.g., "Monthly"):**
  ```typescript
  page.locator('.v-overlay--active .v-list-item').filter({ hasText: /^Monthly$/ }).click()
  await page.waitForTimeout(300)
  ```

#### Select Day(s) Selector (for Monthly frequency)
- Component: `v-select` with `multiple` and `chips` attributes
- Label: "Select monthly days"
- Options: "Day 1", "Day 2", ..., "Day 31"
- **Open dropdown:**
  ```typescript
  page.locator('.v-select').nth(1).click()  // second v-select on the form
  await page.waitForTimeout(500)
  ```
- **Select "Day 1" (EXACT match to avoid matching "Day 10", "Day 11", etc.):**
  ```typescript
  page.locator('.v-overlay--active .v-list-item').filter({ hasText: /^Day 1$/ }).click()
  // or with exact: true
  page.getByRole('option', { name: 'Day 1', exact: true }).click()
  ```
- Source: `KnowledgeBaseScheduleModal.vue` line ~600-620: `v-select v-model="monthlyDays" :items="monthlyDayItems"`

#### At Time Input (for Monthly frequency)
- Component: `v-text-field` with `type="time"`
- Label: "Time"
- Input format: "HH:MM" (24-hour format)
- **Fill with "00:00":**
  ```typescript
  const timeInput = page.locator('input[type="time"]').last()  // last time input on form
  await timeInput.click()
  await timeInput.fill('00:00')
  await timeInput.press('Tab')  // trigger validation
  ```
- Source: `KnowledgeBaseScheduleModal.vue` line ~625-632: `v-text-field v-model="monthlyTime" type="time"`

#### Cron Expression Display/Preview (auto-updates)
- Component: `<div class="cron-preview">`
- Contains: `<code class="preview-value">{{ generatedCronExpression }}</code>`
- **Verify it's not empty:**
  ```typescript
  const cronPreview = page.locator('code.preview-value')
  const cronText = await cronPreview.textContent()
  expect(cronText).not.toBe('')
  expect(cronText).toMatch(/^\d+ \d+ \d+ \* \*$/)  // basic cron format
  ```
- Auto-updates when frequency, days, or time changes
- Source: `KnowledgeBaseScheduleModal.vue` line ~650-656: computed `generatedCronExpression` and line ~700-710: cron preview template

### Save Button
- Selector: `page.getByRole('button', { name: /Save/i })`
- Text: "Save"
- Located in form actions section
- Disabled until form is valid
- Click to save the schedule
- Source: `KnowledgeBaseScheduleModal.vue` line ~750-760: `v-btn ... @click="handleSave"`

---

## 4. Success Notification

### Notification Selector
- Component: Vuetify snackbar (`.v-snackbar`)
- Success message: "Schedule created successfully"
- **Verify notification:**
  ```typescript
  await expect(page.locator('.v-snackbar')).toContainText('Schedule created successfully')
  ```
- Or use snackbar store notification:
  ```typescript
  await page.locator('.v-snackbar').waitFor({ state: 'visible', timeout: 5000 })
  const text = await page.locator('.v-snackbar').textContent()
  expect(text).toContain('Schedule created successfully')
  ```

### Notification Duration
- Default: 3-5 seconds (auto-dismisses)
- Wait for it to appear: `timeout: 5000`
- Source: `KnowledgeBaseScheduleModal.vue` line ~430: `snackbarStore.show(t('knowledgeBasePage.scheduleModal.createSuccess'), 'green')`

---

## Key Implementation Notes

### Form Validation
- All form fields must be filled before Save button is enabled
- Cron expression is auto-generated in Simple mode
- In Advanced mode, user enters cron expression manually
- Timezone note: All times are converted to UTC for storage

### Modal Behavior
- Modal is a standard Vuetify `v-dialog`, NOT a custom drawer
- Modal closes after successful save
- Modal can be closed by clicking X button or Cancel button
- Form resets when modal is reopened

### Selectors Summary
- `.bucket-card` - KB bucket card
- `.bucket-name` - KB bucket name
- `.schedule-button` - Schedule button on card
- `.v-dialog` - Modal container
- `v-radio` → `input[type="radio"]` - Sync type radio buttons
- `v-btn-toggle` → `button` - Cron mode toggle
- `v-select` - Dropdowns (Frequency, Days)
- `input[type="time"]` - Time input
- `.cron-preview` - Cron expression preview
- `.v-snackbar` - Success notification
