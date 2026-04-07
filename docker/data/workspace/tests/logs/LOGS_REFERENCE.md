# Logs Page Reference Document

**Generated:** 2026-04-07 12:30  
**Status:** Complete

---

## 0. Files Read

✅ **All 4 files read successfully:**

1. `/src/pages/LogsPage.vue` — Logs page with date filters, data table, and export functionality
2. `/src/components/DataTable.vue` — Generic data table component with column visibility
3. `/src/stores/flow.store.ts` — Flow store with `getNodeEvent()` and `exportNodeEvent()` methods
4. `/src/stores/snackbar.store.ts` — Snackbar store for notifications

---

## 1. Logs Page Navigation

### Sidebar Selector to Navigate to Logs
```typescript
// From any page with sidebar, click Logs link
await page.locator('a:has-text("Logs")').click();

// Alternative: By role
await page.getByRole('link', { name: /Logs/i }).click();
```

### Page URL / Route Identifier
```typescript
// Expected URL after navigation
// URL: https://dashboard.int3nt.info/logs (or /logs route)

// Verify page loaded
await page.waitForURL(/\/logs/);
```

### How to Verify Logs Page is Loaded
```typescript
// Wait for page title
const title = page.locator('h2');
await expect(title).toContainText('Logs');

// Wait for date filter controls
const filterControls = page.locator('.filter-controls');
await filterControls.waitFor({ state: 'visible', timeout: 10000 });

// Wait for table container
const tableContainer = page.locator('.table-container');
await tableContainer.waitFor({ state: 'visible', timeout: 10000 });

// Wait for data table to load
const dataTable = page.locator('.agent-data-table');
await dataTable.waitFor({ state: 'visible', timeout: 10000 });
```

---

## 2. Logs Table

### Selector for Logs Table/List Container
```typescript
// Main table wrapper
const table = page.locator('.agent-data-table');

// Or v-data-table
const dataTable = page.locator('.v-data-table');

// Table container (includes toolbar and pagination)
const tableContainer = page.locator('.table-container');
```

### Selector for a Log Row
```typescript
// All rows in the table
const rows = page.locator('.v-data-table__tr');

// First row
const firstRow = page.locator('.v-data-table__tr').first();

// Row by content (e.g., containing specific event timestamp)
const row = page.locator('.v-data-table__tr').filter({ hasText: /2026-04-07/ });

// Get cell value from row
const rowCell = page.locator('.v-data-table__tr').first().locator('.v-data-table__td');
```

### How to Identify Columns
**Available columns in LogsPage.vue:**

| Column Key | Column Title (i18n) | Data Type | Notes |
|---|---|---|---|
| `event_timestamp` | `logsPage.tableHeaders.event_timestamp` | datetime | Sortable, formatted to locale string |
| `node_type` | `logsPage.tableHeaders.node_type` | string | Node type (e.g., "ReplyMessage", "CustomNode") |
| `node_id_name` | `logsPage.tableHeaders.node_id_name` | string | Node ID or name |
| `conversation_id` | `logsPage.tableHeaders.conversation_id` | string | Conversation identifier |
| `conversation_flow_id` | `logsPage.tableHeaders.conversation_flow_id` | string | Flow ID |
| `sequence_id` | `logsPage.tableHeaders.sequence_id` | string | Sequence identifier |
| `model_name` | `logsPage.tableHeaders.model_name` | string | Model name (if applicable) |
| `latency_ms` | `logsPage.tableHeaders.latency_ms` | number | Latency in milliseconds, right-aligned |
| `input_tokens` | `logsPage.tableHeaders.input_tokens` | number | Input token count, right-aligned |
| `output_tokens` | `logsPage.tableHeaders.output_tokens` | number | Output token count, right-aligned |
| `total_tokens` | `logsPage.tableHeaders.total_tokens` | number | Total token count, right-aligned |
| `start_timestamp` | `logsPage.tableHeaders.start_timestamp` | datetime | Start time, formatted to locale string |
| `end_timestamp` | `logsPage.tableHeaders.end_timestamp` | datetime | End time, formatted to locale string |
| `error_type` | `logsPage.tableHeaders.error_type` | string | Error type or "No Error", center-aligned |
| `input_message` | `logsPage.tableHeaders.input_message` | string | Input message (truncated to 100 chars) |
| `output_message` | `logsPage.tableHeaders.output_message` | string | Output message (truncated to 100 chars) |

### Selector for Filtering/Searching Logs
```typescript
// Date range filter controls
const filterControls = page.locator('.filter-controls');

// From Date button
const fromDateButton = page.locator('.dropdown-button').first();

// To Date button
const toDateButton = page.locator('.dropdown-button').nth(1);

// Load Data button
const loadDataButton = page.locator('button').filter({ hasText: /Load Data/ });

// Column search input (inside Show Column menu)
const columnSearch = page.locator('.column-search input');
```

### Selector for Date Range Picker
```typescript
// From Date picker (v-date-picker)
const fromDatePicker = page.locator('v-date-picker').first();

// To Date picker
const toDatePicker = page.locator('v-date-picker').nth(1);

// Date picker opens in a v-card via v-menu
// After clicking the dropdown button, the date picker appears:
const datePicker = page.locator('.v-card').locator('v-date-picker');

// Select a date (example: click on a date in the calendar)
await page.locator('button').filter({ hasText: /^15$/ }).click(); // Click day 15
```

---

## 3. Log Row Interaction

### How to Open a Log Detail
```typescript
// Click a row to open/expand it
// Note: LogsPage.vue does NOT have expand buttons — rows are clickable
const row = page.locator('.v-data-table__tr').first();
await row.click();

// Alternative: Click a specific cell
const cell = page.locator('.v-data-table__td').first();
await cell.click();
```

### Selector for Expanded Log Detail Panel/Modal
```typescript
// LogsPage.vue does NOT have a dedicated detail modal/panel
// Instead, it displays all data in the table with columns that can be toggled

// Column visibility menu (Show Column dropdown)
const columnMenu = page.locator('.v-menu');

// If a detail modal were to be implemented, it would likely be:
const detailModal = page.locator('.v-overlay--active').filter({ hasText: /Log Detail/ });
```

### Selector for Closing the Detail Panel
```typescript
// If a modal exists, close via the close button
const closeBtn = page.locator('button[icon="mdi-close"]');
await closeBtn.click();

// Or press Escape
await page.keyboard.press('Escape');
```

---

## 4. Download Logs

### Selector for the Download/Export Button
```typescript
// Export button at bottom of page
const exportButton = page.locator('.export-button');

// Or by text
const exportBtn = page.locator('button').filter({ hasText: /Export CSV|Download/ });

// Or by icon
const exportBtnByIcon = page.locator('button').filter({ has: page.locator('i.mdi-download') });
```

### File Format
```typescript
// File format: CSV
// Filename pattern: node-events-export-{timestamp}.csv
// Example: node-events-export-2026-04-07T12-30-45.csv

// The timestamp is generated as:
// const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, '-');
// Result: 2026-04-07T12-30-45
```

### Correct Playwright Pattern for File Download
```typescript
// COPY THIS PATTERN VERBATIM
const [download] = await Promise.all([
  page.waitForEvent('download'),
  page.locator('.export-button').click(),
]);
const filePath = await download.path();
expect(filePath).toBeTruthy();

// Verify file format
const fileName = download.suggestedFilename();
expect(fileName).toMatch(/^node-events-export-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}\.csv$/);

// Read file content (optional)
const fs = require('fs');
const content = fs.readFileSync(filePath, 'utf-8');
expect(content).toContain('event_timestamp'); // CSV header
```

### Filters That Must Be Applied Before Downloading
```typescript
// Date range is MANDATORY before export
// The export button is disabled until dates are set:
const exportButton = page.locator('.export-button');
// :disabled attribute is set if !canLoadData (no date range)

// Required steps before export:
// 1. Set from date
await page.locator('.dropdown-button').first().click();
await page.locator('button').filter({ hasText: /^15$/ }).click(); // Select day

// 2. Set to date
await page.locator('.dropdown-button').nth(1).click();
await page.locator('button').filter({ hasText: /^20$/ }).click(); // Select day

// 3. Load data (optional but recommended before export)
await page.locator('button').filter({ hasText: /Load Data/ }).click();
await page.waitForTimeout(1000);

// 4. Export
const [download] = await Promise.all([
  page.waitForEvent('download'),
  page.locator('.export-button').click(),
]);
```

---

## 5. DataTable Component

### How DataTable Renders in the DOM
```typescript
// DataTable is a generic component used in multiple pages
// In LogsPage, it wraps Vuetify's v-data-table

// Structure:
// .agent-data-table (main wrapper)
//   ├── v-toolbar (column visibility menu)
//   ├── v-data-table (Vuetify data table)
//   │   ├── thead (headers)
//   │   └── tbody (rows)
//   └── .pagination-container (pagination controls)

const table = page.locator('.agent-data-table');
const toolbar = table.locator('.v-toolbar');
const dataTable = table.locator('.v-data-table');
const pagination = table.locator('.pagination-container');
```

### Selector for Table Rows
```typescript
// All data rows
const allRows = page.locator('.v-data-table__tr');

// Get row count
const rowCount = await page.locator('.v-data-table__tr').count();

// Get specific row
const secondRow = page.locator('.v-data-table__tr').nth(1);

// Get row cells
const cells = page.locator('.v-data-table__tr').first().locator('.v-data-table__td');
const firstCell = cells.first();
```

### Selector for Table Headers
```typescript
// All headers
const headers = page.locator('.v-data-table-header th');

// Header by column name
const eventTimestampHeader = page.locator('.v-data-table-header th')
  .filter({ hasText: /event_timestamp|Event Timestamp/ });

// Header cell text
const headerText = await page.locator('.v-data-table-header th').first().textContent();

// Sortable headers (clickable)
const sortableHeader = page.locator('.table-header-cell.sortable');
```

### Selector for Pagination Controls
```typescript
// Pagination container
const paginationContainer = page.locator('.pagination-container');

// Rows per page selector
const itemsPerPageSelect = page.locator('.pagination-select');

// Pagination info text (e.g., "1-5 of 100")
const paginationInfo = page.locator('.pagination-info');

// Previous button
const prevButton = page.locator('.pagination-buttons .v-btn').nth(0);

// Next button
const nextButton = page.locator('.pagination-buttons .v-btn').nth(1);

// Click next page
await page.locator('.pagination-buttons .v-btn').nth(1).click();
```

---

## 6. Notifications

### Selector for Success Snackbar/Toast
```typescript
// Snackbar appears at bottom of screen
const snackbar = page.locator('.v-snackbar');

// Wait for snackbar to appear
await snackbar.waitFor({ state: 'visible', timeout: 5000 });

// Check for success message
await expect(snackbar).toContainText('Export complete');

// Snackbar with specific severity (green = success)
const successSnackbar = page.locator('.v-snackbar').filter({ hasText: /success|complete|downloaded/ });
```

### Typical Wait Timeout
```typescript
// Snackbars appear quickly: 3-5 seconds
const timeout = 5000; // 5 seconds

// Snackbar auto-dismisses after 3 seconds (default timeout in snackbar.store.ts)
await snackbar.waitFor({ state: 'hidden', timeout: 10000 });

// Wait for notification after export
await page.waitForTimeout(500); // Brief wait for snackbar to render
await expect(page.locator('.v-snackbar')).toContainText('Export', { timeout: 5000 });
```

---

## 7. Common Interactions

### Load Data with Date Range
```typescript
// Step 1: Set from date
await page.locator('.dropdown-button').first().click();
await page.waitForTimeout(300);
const fromDatePicker = page.locator('.v-card').locator('v-date-picker').first();
await fromDatePicker.waitFor({ state: 'visible' });
// Click a date (e.g., 1st of month)
await page.locator('button').filter({ hasText: /^1$/ }).click();

// Step 2: Set to date
await page.locator('.dropdown-button').nth(1).click();
await page.waitForTimeout(300);
const toDatePicker = page.locator('.v-card').locator('v-date-picker').nth(1);
await toDatePicker.waitFor({ state: 'visible' });
// Click a date (e.g., 10th of month)
await page.locator('button').filter({ hasText: /^10$/ }).click();

// Step 3: Load data
await page.locator('button').filter({ hasText: /Load Data/ }).click();
await page.waitForTimeout(1000); // Wait for table to populate

// Verify table has rows
await expect(page.locator('.v-data-table__tr')).toHaveCount(1); // At least 1 row (excluding header)
```

### Toggle Column Visibility
```typescript
// Open column menu
await page.locator('.table-dropdown').click();
await page.waitForTimeout(300);

// Search for column
const columnSearch = page.locator('.column-search input');
await columnSearch.fill('latency');

// Uncheck column
const checkbox = page.locator('v-checkbox').filter({ hasText: /latency_ms/ });
await checkbox.click();

// Apply changes
await page.locator('button').filter({ hasText: /Apply/ }).click();
```

### Sort by Column
```typescript
// Click sortable header
const header = page.locator('.table-header-cell.sortable').filter({ hasText: /event_timestamp/ });
await header.click();

// Verify sort direction indicator appears
const sortIcon = header.locator('.v-icon');
await expect(sortIcon).toBeVisible();

// Click again to reverse sort
await header.click();
```

---

## 8. Flow Store Integration

### `getNodeEvent()` method (used by LogsPage)
```typescript
// From flow.store.ts
const data = await flowStore.getNodeEvent({
  startDate: '2026-04-01T00:00:00.000Z',
  endDate: '2026-04-07T23:59:59.999Z',
  orderBy: 'event_timestamp',
  direction: 'desc',
  limit: 5,
  offset: 0,
  includeTotal: true
});

// Returns: PaginatedResponse<NodeEvent>
// {
//   items: [...],
//   total: 100,
//   offset: 0,
//   limit: 5,
//   pages: 20
// }
```

### `exportNodeEvent()` method (used by export button)
```typescript
// From flow.store.ts
const blob = await flowStore.exportNodeEvent({
  startDate: '2026-04-01T00:00:00.000Z',
  endDate: '2026-04-07T23:59:59.999Z',
  orderBy: 'event_timestamp',
  direction: 'desc'
});

// Returns: Blob (CSV file)
// flowStore.isExporting is set to true during export
```

---

## 9. Snackbar Store Integration

### Success Notification
```typescript
// From snackbar.store.ts
snackbarStore.show(
  'Export complete',
  'green',  // severity
  3000      // timeout (ms)
);

// Renders as .v-snackbar with green background
```

### Loading Notification
```typescript
// For long-running operations
snackbarStore.showLoading('Exporting data...');

// Later, hide it
snackbarStore.hide();
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
  logs: 'https://dashboard.int3nt.info/logs',
  dashboard: 'https://dashboard.int3nt.info'
};
```

---

**End of Reference Document**
