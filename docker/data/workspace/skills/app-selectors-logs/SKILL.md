---
name: app-selectors-logs
description: DOM selectors and component map for the Logs page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Logs — Component Map

> Generated: 2026-03-26T07:08:23.669Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Logs
**URL:** `/logs`

**Headings:**
- `h2` — "Conversation Logs"

**Toolbar Buttons:** `Show column`

**Text Content (3):**
- [p] "Organization" → `.org-title`
- [p] "No node events found"
- [text-caption] "Try adjusting your date range"

**Buttons (3):**
- `page.locator('button:has-text("Load Data")')`
- `page.locator('.table-dropdown')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text table-dropdown`
- `page.locator('.export-button')` [disabled]
  classes: `v-btn v-btn--disabled v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-flat export-button`

**⚠️ Dynamic Buttons:**
- "Datetime Start: Feb 26, 2026" → pattern: `Datetime Start: {DATE}`
  Use: `page.locator('button:visible').nth(N)`
- "Datetime End: Mar 26, 2026" → pattern: `Datetime End: {DATE}`
  Use: `page.locator('button:visible').nth(N)`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Table 1:** `.custom-table`
- Columns: `Event Timestamp` | `Node Type` | `Node ID Name` | `Conversation ID` | `Flow ID` | `Sequence ID` | `Model Name` | `Latency (ms)` | `Input Tokens` | `Output Tokens` | `Total Tokens` | `Start Timestamp` | `End Timestamp` | `Error Type` | `Input Message`
- Rows: 1
- Sample: No node events foundTry adjusting your date range

**Table 2:**
- Columns: `Event Timestamp` | `Node Type` | `Node ID Name` | `Conversation ID` | `Flow ID` | `Sequence ID` | `Model Name` | `Latency (ms)` | `Input Tokens` | `Output Tokens` | `Total Tokens` | `Start Timestamp` | `End Timestamp` | `Error Type` | `Input Message`
- Rows: 1
- Sample: No node events foundTry adjusting your date range

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')` ★
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (37):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | HOrganizationTesting2026!DashboardFlow DesignerFlow TesterKn |
| `.topbar-intent` | `header` | `topbar-intent` | H |
| `.logo-container` | `div` | `logo-container` |  |
| `.logo-wrapper` | `div` | `logo-wrapper` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.change-logo-btn` | `button` | `change-logo-btn` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#menu-activator` | `div` | `avatar-container` | H |
| `.mdi` | `i` | `mdi notranslate avatar-chevron` |  |
| `.nav-drawer` | `nav` | `nav-drawer` | OrganizationTesting2026!DashboardFlow DesignerFlow TesterKno |
| `.org-title` | `p` | `org-title` | Organization |
| `.org-selector-wrapper` | `div` | `org-selector-wrapper` | Testing2026! |
| `.org-dropdown` | `div` | `org-dropdown` | Testing2026! |
| `.org-dropdown-trigger` | `div` | `org-dropdown-trigger` | Testing2026! |
| `.org-info` | `div` | `org-info` | Testing2026! |
| `.org-name` | `span` | `org-name` | Testing2026! |
| `.dropdown-arrow` | `div` | `dropdown-arrow` |  |
| `.[object` | `svg` | `[object SVGAnimatedString]` |  |
| `.[object` | `path` | `[object SVGAnimatedString]` |  |
| `.mdi` | `i` | `mdi notranslate arrow` |  |
| `.menu-item-container` | `div` | `menu-item-container` | Dashboard |
| `.mdi` | `i` | `mdi notranslate menu-item-icon` |  |
| `.menu-item` | `span` | `menu-item` | Dashboard |
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Conversation LogsDatetime Start: Feb 26, 2026Datetime End: M |
| `.logs-container` | `div` | `logs-container` | Conversation LogsDatetime Start: Feb 26, 2026Datetime End: M |
| `.filter-controls` | `div` | `filter-controls` | Datetime Start: Feb 26, 2026Datetime End: Mar 26, 2026 Load  |
| `.dropdown-button` | `button` | `dropdown-button` | Datetime Start: Feb 26, 2026 |
| `.dropdown-text` | `span` | `dropdown-text` | Datetime Start: Feb 26, 2026 |
| `.dropdown-value` | `span` | `dropdown-value` | Feb 26, 2026 |
| `.table-container` | `div` | `table-container` | Show columnEvent Timestamp Node Type Node ID Name Conversati |
| `.agent-data-table` | `div` | `agent-data-table` | Show columnEvent Timestamp Node Type Node ID Name Conversati |
| `.table-dropdown` | `button` | `table-dropdown` | Show column |
| `.custom-table` | `div` | `custom-table` | Event Timestamp Node Type Node ID Name Conversation ID Flow  |
| `.table-header-cell` | `div` | `table-header-cell sortable active` | Event Timestamp |
| `.table-header-cell` | `div` | `table-header-cell` | Node Type |
| `.no-data` | `div` | `no-data` | No node events foundTry adjusting your date range |
| `.export-button` | `button` | `export-button` | Export to CSV |

#### Discovered Modals / Dialogs

**Trigger:** `page.locator('button:has-text("Show column")').click()`
**Overlay:** `page.locator('.column-search')`
**Wait:** `await page.locator('.column-search').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-input v-input--horizontal v-input--center-affix v-input--density-compact v-theme--mainTheme v-locale--is-ltr v-text-field column-search`

- Inputs:
  - Search column (`text`)
- Buttons: `Apply`
- 1 input(s): Search column
- 30 checkbox(es)
- 2 table(s)
- headings: Conversation Logs
- buttons: Load Data, Show column, Export to CSV, Apply
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

---

