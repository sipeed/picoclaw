---
name: app-selectors-protected
description: >
  Comprehensive DOM selectors and component map for protected pages on dashboard.int3nt.info.
  Use this skill when writing or fixing Playwright tests. Contains verified
  selectors for every page.
---

# dashboard.int3nt.info — Protected Pages Component Map

> Generated: 2026-03-26T01:24:58.975Z
> Never use auto-generated IDs (`input-v-N`) — they change on re-render.
> Never match dynamic-text buttons by exact text — use patterns or position.

## Protected Pages (Login + Org Required)

### Dashboard
**URL:** `/`

**Headings:**
- `h2` — "Dashboard"
- `h2` — "Total Conversations"
- `h1` — "0"
- `h2` — "Executions"
- `h2` — "Error Rate"
- `h1` — "0.00%"
- `h2` — "Latency"
- `h1` — "–"
- `h2` — "Total Tokens"
- `h2` — "Knowledge Base Buckets"
- `h1` — "1"


**Text Content (1):**
- [p] "Organization"

**Buttons (1):**
- `page.locator('button:has-text("Mar 19, 2026 - Mar 26, 2026")')`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')` ★
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

**Custom Elements & IDs (36):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | DashboardMar 19, 2026 - Mar 26, 2026Total Conversations0Exec |
| `.dashboard-container` | `div` | `dashboard-container` | DashboardMar 19, 2026 - Mar 26, 2026Total Conversations0Exec |
| `.dashboard-header` | `div` | `dashboard-header` | Dashboard |
| `.date-range-button` | `button` | `date-range-button` | Mar 19, 2026 - Mar 26, 2026 |
| `.date-range-text` | `span` | `date-range-text` | Mar 19, 2026 - Mar 26, 2026 |
| `.date-range-value` | `span` | `date-range-value` | Mar 19, 2026 - Mar 26, 2026 |
| `.metrics-section` | `div` | `metrics-section` | Total Conversations0Executions0Error Rate0.00%Latency–Total  |
| `.metric-card` | `div` | `metric-card` | Total Conversations0 |
| `.metric-title` | `h2` | `metric-title` | Total Conversations |
| `.metric-value-container` | `div` | `metric-value-container` | 0 |
| `.metric-value` | `h1` | `metric-value` | 0 |
| `.chart-section` | `div` | `chart-section` |  |
| `.executions-chart-container` | `div` | `executions-chart-container` |  |

---

### Manage Chatbot
**URL:** `/manage-chatbot`

**Headings:**
- `h3` — "Chatbot"

**Dropdowns / Selects (1):**
- **Select** (select)
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Options: `No data available`
  - Pick option: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (5):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | ChatbotSelectSelectDashboardFlow DesignerFlow TesterKnowledg |
| `#input-v-4-label` | `label` | `` | Select |
| `#input-v-4` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#input-v-4-messages` | `div` | `` |  |

---

### Config Test
**URL:** `/config-test`

**Headings:**
- `h1` — "Please select a bot"


**Text Content (1):**
- [p] "Organization"

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

**Custom Elements & IDs (25):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Please select a bot |
| `.select-bot-title` | `h1` | `select-bot-title` | Please select a bot |

---

### Flow Designer
**URL:** `/flow-designer`

**Headings:**
- `h2` — "All Flows"


**Text Content (1):**
- [p] "Organization"

**Input Fields (1):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Search flow name | `text` | `page.locator('.v-field__input').nth(0)` |

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "10"
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick option: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (1):**
- `page.locator('button:has-text("Add New")')`

**Icon Buttons (23):**
- mdi-pencil-outline (`mdi-pencil-outline`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- mdi-pencil (`mdi-pencil`)
- mdi-content-copy (`mdi-content-copy`)
- mdi-delete (`mdi-delete`)
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:**
- Columns: `Flow Name` | `Last Update` | `Actions`
- Rows: 6 (paginated)
- Sample row: Untitled | 26 Mar 2026, 8:17 AM | 

**Table 2:**
- Columns: `Flow Name` | `Last Update` | `Actions`
- Rows: 6
- Sample row: Untitled | 26 Mar 2026, 8:17 AM | 

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')` ★
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Pagination:** 0 page(s)

**Avatars:** 1 visible

**Custom Elements & IDs (40):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | All FlowsAdd NewActiveArchivedFlow NameLast UpdateActionsUnt |
| `.header-container` | `div` | `header-container` | All FlowsAdd New |
| `.m-auto` | `button` | `m-auto` | Add New |
| `.tabs` | `div` | `tabs` | ActiveArchived |
| `.tab-header` | `div` | `tab-header` | ActiveArchived |
| `.tab` | `div` | `tab active` | Active |
| `.tab` | `div` | `tab` | Archived |
| `.flow-table-container` | `div` | `flow-table-container` | Flow NameLast UpdateActionsUntitled26 Mar 2026, 8:17 AMUntit |
| `.table-controls` | `div` | `table-controls` |  |
| `.search-section` | `div` | `search-section` |  |
| `.search-input` | `div` | `search-input` |  |
| `#input-v-7` | `input` | `` |  |
| `.flow-data-table` | `div` | `flow-data-table` | Flow NameLast UpdateActionsUntitled26 Mar 2026, 8:17 AMUntit |
| `.action-icons` | `div` | `action-icons` |  |
| `.action-btn` | `button` | `action-btn` |  |
| `[data-testid="v-pagination-root"]` | `nav` | `` |  |
| `[data-testid="v-pagination-first"]` | `li` | `` |  |

---

### Knowledge Management
**URL:** `/knowledge-management`


**Text Content (2):**
- [p] "Organization"
- [v-card-text] "Add and manage your knowledge base files to customise your bot."

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Cards (1):**
- **Card 1**
  Text: "Add and manage your knowledge base files to customise your bot."

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

**Custom Elements & IDs (26):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add and manage your knowledge base files to customise your b |
| `.main-container` | `div` | `main-container` | Add and manage your knowledge base files to customise your b |
| `.page-title` | `div` | `page-title` |  |

---

### Knowledge Base
**URL:** `/knowledge-base`

**Headings:**
- `h2` — "Knowledge Base"


**Text Content (1):**
- [p] "Organization"

**Buttons (3):**
- `page.locator('button:has-text("Create Knowledge Base Bucket")')`
- `page.locator('button:has-text("Schedule")')`
- `page.locator('button:has-text("Sync")')`

**Icon Buttons (2):**
- mdi-pencil-outline (`mdi-pencil-outline`)
- mdi-dots-vertical (`mdi-dots-vertical`)

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')` ★
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

**Custom Elements & IDs (41):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Knowledge BaseCreate Knowledge Base BucketPicotest_3ReadySou |
| `.knowledge-base-container` | `div` | `knowledge-base-container` | Knowledge BaseCreate Knowledge Base BucketPicotest_3ReadySou |
| `.knowledge-base-header` | `div` | `knowledge-base-header` | Knowledge BaseCreate Knowledge Base Bucket |
| `.create-button` | `button` | `create-button` | Create Knowledge Base Bucket |
| `.bucket-grid` | `div` | `bucket-grid` | Picotest_3ReadySource TypeGoogle Cloud StorageKnowledge file |
| `.bucket-card` | `div` | `bucket-card` | Picotest_3ReadySource TypeGoogle Cloud StorageKnowledge file |
| `.bucket-header` | `div` | `bucket-header` | Picotest_3Ready |
| `.bucket-title` | `div` | `bucket-title` | Picotest_3Ready |
| `.bucket-name` | `span` | `bucket-name` | Picotest_3 |
| `.knowledge-base-status-pill` | `span` | `knowledge-base-status-pill knowledge-base-status-pill--ready` | Ready |
| `.bucket-info` | `div` | `bucket-info` | Source TypeGoogle Cloud StorageKnowledge filesScheduleSync |
| `.source-type` | `div` | `source-type` | Source TypeGoogle Cloud Storage |
| `.label` | `div` | `label` | Source Type |
| `.value` | `div` | `value` | Google Cloud Storage |
| `.files-count` | `div` | `files-count is-clickable` | Knowledge files |
| `.action-buttons` | `div` | `action-buttons` | ScheduleSync |
| `.schedule-button` | `button` | `schedule-button` | Schedule |
| `.sync-button` | `button` | `sync-button` | Sync |

#### Discovered Modals / Dialogs

**Triggered by:** `page.locator('button:has-text("Create Knowledge Base Bucket")').click()`
**Overlay selector:** `page.locator('.custom-drawer-overlay')`
**Wait for overlay:** `await page.locator('.custom-drawer-overlay').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `custom-drawer-overlay`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 1 input(s): Enter knowledge group name
- 2 dropdown(s): unlabeled, unlabeled
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Continue
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

- **Dropdown: "Dropdown 1"** — 2 option(s): `None`, `gemini-flash-2.5`
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`
- **Dropdown: "Dropdown 2"** — 3 option(s): `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(1).click()`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

---

### Sentiment Dashboard
**URL:** `/sentiment`

**Headings:**
- `h3` — "Top 10 Intents"
- `h3` — "Accuracy"
- `h3` — "Sentiments"

**Toolbar Buttons:** `Select bots: 0 selected`
**Toolbar Buttons:** `Show column`

**Text Content (1):**
- [p] "Organization"

**Input Fields (1):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Datepicker input | `text` | `page.locator('.v-field__input').nth(0)` |

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "10"
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick option: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (3):**
- `page.locator('button:has-text("Download CSV")')`
- `page.locator('button:has-text("Select bots: 0 selected")')` [disabled]
- `page.locator('button:has-text("Show column")')`

**Icon Buttons (5):**
- mdi-pencil-outline (`mdi-pencil-outline`)
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:**
- Rows: 1 (paginated)
- Sample row: No data available

**Table 2:**
- Rows: 1
- Sample row: No data available

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Pagination:** 0 page(s)

**Avatars:** 1 visible

**Custom Elements & IDs (51):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Dashboard DATEDownload CSV Select bots: 0 selectedTop 10 Int |
| `.data-container` | `div` | `data-container` | Dashboard DATEDownload CSV Select bots: 0 selectedTop 10 Int |
| `.data-display-container` | `div` | `data-display-container` | Dashboard DATEDownload CSV |
| `.data-display-row` | `div` | `data-display-row` | Dashboard DATEDownload CSV |
| `.menu-row-height` | `div` | `menu-row-height` | Dashboard |
| `.dashboard-title` | `span` | `dashboard-title` | Dashboard |
| `.mdi` | `i` | `mdi notranslate dashboard-title-icon` |  |
| `.date-title` | `span` | `date-title` | DATE |
| `.date-picker-container` | `div` | `date-picker-container` |  |
| `.dp__main` | `div` | `dp__main dp__theme_light` |  |
| `.dp__input_wrap` | `div` | `dp__input_wrap` |  |
| `[data-testid="dp-input"]` | `input` | `dp__pointer dp__input_readonly dp__input dp__input_icon_pad dp__input_reg` |  |
| `.dp--clear-btn` | `button` | `dp--clear-btn` |  |
| `.bot-data-download` | `button` | `bot-data-download` | Download CSV |
| `.bot-selector-menu-container` | `div` | `bot-selector-menu-container` | Select bots: 0 selected |
| `.bot-selector-menu` | `header` | `bot-selector-menu` | Select bots: 0 selected |
| `.menu-dropdown` | `button` | `menu-dropdown` | Select bots: 0 selected |
| `.bot-selector-val` | `span` | `bot-selector-val` | 0 selected |
| `.doughnut-card` | `div` | `doughnut-card` | Top 10 Intents |
| `.plot-container` | `div` | `plot-container` |  |
| `.legend-container` | `div` | `legend-container` |  |
| `.conv-count-title` | `div` | `conv-count-title` | Comparison |
| `.chart-container` | `div` | `chart-container` |  |
| `.conv-table-title` | `div` | `conv-table-title` | Bot conversation |
| `.agent-data-table` | `div` | `agent-data-table` | Show column No data availableItems per page:100-0 of 0 |
| `.table-dropdown` | `button` | `table-dropdown` | Show column |
| `[data-testid="v-pagination-root"]` | `nav` | `` |  |

#### Discovered Modals / Dialogs

**Triggered by:** `page.locator('button:has-text("Show column")').click()`
**Overlay selector:** `page.locator('#v-menu-v-8')`
**Wait for overlay:** `await page.locator('#v-menu-v-8').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `v-overlay v-overlay--absolute v-overlay--active v-theme--mainTheme v-locale--is-ltr v-menu`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.v-list` (classes: `v-list v-theme--mainTheme v-list--density-default v-list--one-line`)
- Inputs:
  - Search column (`text`)
- Buttons: `Apply`
- 2 input(s): Datepicker input, Search column
- 1 dropdown(s): unlabeled
- 2 table(s)
- headings: Top 10 Intents, Accuracy, Sentiments
- buttons: Download CSV, Select bots: 0 selected, Show column, Apply
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

- **Dropdown: "Dropdown 1"** — 5 option(s): `10`, `25`, `50`, `100`, `All`
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

---

### Organization
**URL:** `/organization`

**Headings:**
- `h2` — "Organization Team"
- `h2` — "Bot Icons"


**Text Content (6):**
- [p] "Organization"
- [text-secondary] "Team"
- [p] "View, add and manage your organizations' members here"
- [p] "Manage icons that can be used for your bots"
- [p] "(max 5MB, only PNG, JPG, and JPEG are supported, best to use square images)"
- [p] "No icons uploaded yet"

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "20"
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick option: `page.locator('.v-list-item:has-text("OPTION")').click()`

**File Inputs (1):**
- File 1 (accept: image/png, image/jpeg, image/jpg)

**Buttons (2):**
- `page.locator('button:has-text("Add a Member")')`
- `page.locator('button:has-text("Upload Icon")')`

**Icon Buttons (5):**
- mdi-pencil-outline (`mdi-pencil-outline`)
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:**
- Columns: `Member's name` | `Role` | `Email` | `Invite on` | `Status` | `Actions`
- Rows: 8 (paginated)
- Sample row:  | agent | heidi+1@intnt.ai | 17 Mar 2026 | Active | 

**Table 2:**
- Columns: `Member's name` | `Role` | `Email` | `Invite on` | `Status` | `Actions`
- Rows: 8
- Sample row:  | agent | heidi+1@intnt.ai | 17 Mar 2026 | Active | 

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')` ★

**Pagination:** 0 page(s)

**Avatars:** 1 visible

**Custom Elements & IDs (36):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | OrganizationTeamAdd a MemberOrganization TeamView, add and m |
| `.organization-container` | `div` | `organization-container` | OrganizationTeamAdd a MemberOrganization TeamView, add and m |
| `.organization-header` | `div` | `organization-header` | OrganizationTeamAdd a Member |
| `.organization-breadcrumb` | `div` | `organization-breadcrumb` | OrganizationTeam |
| `.organization-actions` | `div` | `organization-actions` | Add a Member |
| `.organization-table` | `div` | `organization-table` | Member's nameRoleEmailInvite onStatusActionsagentheidi+1@int |
| `.bold-text` | `span` | `bold-text` | agent |
| `.status-active` | `span` | `status-active` | Active |
| `[data-testid="v-pagination-root"]` | `nav` | `` |  |
| `[data-testid="v-pagination-first"]` | `li` | `` |  |
| `.bot-icons-header` | `div` | `bot-icons-header` | Bot IconsManage icons that can be used for your bots(max 5MB |
| `.bot-icons-requirements` | `p` | `bot-icons-requirements` | (max 5MB, only PNG, JPG, and JPEG are supported, best to use |
| `.bot-icons-actions` | `div` | `bot-icons-actions` | Upload Icon |

#### Discovered Modals / Dialogs

**Triggered by:** `page.locator('button:has-text("Add a Member")').click()`
**Overlay selector:** `page.locator('.align-center')`
**Wait for overlay:** `await page.locator('.align-center').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog v-overlay--scroll-blocked`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.align-center` (classes: `v-row align-center justify-center`)
- Title: "Add a Member"
- Inputs:
  - Email (`text`)
- Buttons: `Cancel`, `Add`
- 1 input(s): Email
- 2 dropdown(s): unlabeled, unlabeled
- 2 table(s)
- headings: Organization Team, Bot Icons, Add a Member
- buttons: Add a Member, Upload Icon, Cancel, Add
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

- **Dropdown: "Dropdown 2"** — 3 option(s): `admin`, `developer`, `agent`
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(1).click()`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

---

### Profile
**URL:** `/profile`

**Headings:**
- `h2` — "Profile Settings"
- `text-h3` — "H"


**Text Content (2):**
- [p] "Organization"
- [p] "Upload your photo and details here"

**Input Fields (2):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | First Name | `text` | `page.locator('.v-field__input').nth(0)` |
| 2 | Last Name | `text` | `page.locator('.v-field__input').nth(1)` |

**File Inputs (1):**
- H (accept: image/*)

**Buttons (5):**
- `page.locator('button:has-text("Upload New Picture")')`
- `page.locator('button:has-text("Delete")')` [disabled]
- `page.locator('button:has-text("Change Email")')`
- `page.locator('button:has-text("Change Password")')`
- `page.locator('button:has-text("Save Changes")')` [disabled]

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 2 visible

**Custom Elements & IDs (43):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Profile SettingsProfile SettingsUpload your photo and detail |
| `.profile-container` | `div` | `profile-container` | Profile SettingsProfile SettingsUpload your photo and detail |
| `.profile-header` | `div` | `profile-header` | Profile SettingsProfile SettingsUpload your photo and detail |
| `.breadcrumb` | `nav` | `breadcrumb` | Profile Settings |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Profile Settings |
| `.profile-picture-section` | `div` | `profile-picture-section` | HUpload New PictureDelete |
| `.profile-picture` | `div` | `profile-picture` | H |
| `.profile-picture-actions` | `div` | `profile-picture-actions` | Upload New PictureDelete |
| `.upload-btn` | `button` | `upload-btn` | Upload New Picture |
| `.delete-btn` | `button` | `delete-btn` | Delete |
| `.form-fields` | `div` | `form-fields` | First Name * Last Name * |
| `.name-fields` | `div` | `name-fields` | First Name * Last Name * |
| `.name-field` | `div` | `name-field` | First Name * |
| `#input-v-9` | `input` | `` |  |
| `#input-v-9-messages` | `div` | `` |  |
| `#input-v-12` | `input` | `` |  |
| `#input-v-12-messages` | `div` | `` |  |
| `.change-email-link` | `button` | `change-email-link` | Change Email |
| `.change-password-link` | `button` | `change-password-link` | Change Password |
| `.m-auto` | `button` | `m-auto` | Save Changes |

---

### Change Email
**URL:** `/change-email`

**Headings:**
- `h2` — "Change Email"


**Text Content (2):**
- [p] "Organization"
- [p] "Change your current email"

**Input Fields (2):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | New Email | `text` | `page.locator('.v-field__input').nth(0)` |
| 2 | Confirm New Email | `text` | `page.locator('.v-field__input').nth(1)` |

**Buttons (1):**
- `page.locator('button:has-text("Confirm")')` [disabled]

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Links (1):**
- [Profile Settings](/profile) — `page.locator('a:has-text("Profile Settings")')`

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Profile SettingsChange EmailChange EmailChange your current  |
| `.profile-container` | `div` | `profile-container` | Profile SettingsChange EmailChange EmailChange your current  |
| `.profile-header` | `div` | `profile-header` | Profile SettingsChange EmailChange EmailChange your current  |
| `.breadcrumb` | `nav` | `breadcrumb` | Profile SettingsChange Email |
| `.breadcrumb-item` | `a` | `breadcrumb-item` | Profile Settings |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Change Email |
| `.subtitle` | `p` | `subtitle` | Change your current email |
| `.email-form` | `div` | `email-form` | New Email * Confirm New Email * Confirm |
| `#input-v-6` | `input` | `` |  |
| `#input-v-6-messages` | `div` | `` |  |
| `#input-v-9` | `input` | `` |  |
| `#input-v-9-messages` | `div` | `` |  |
| `.confirm-btn` | `button` | `confirm-btn` | Confirm |

---

### Change Password
**URL:** `/change-password`

**Headings:**
- `h2` — "Change Password"


**Text Content (2):**
- [p] "Organization"
- [p] "Change your current password"

**Input Fields (2):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | New Password | `password` | `page.locator('.v-field__input').nth(0)` |
| 2 | Confirm New Password | `password` | `page.locator('.v-field__input').nth(1)` |

**Buttons (1):**
- `page.locator('button:has-text("Confirm")')` [disabled]

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Links (1):**
- [Profile Settings](/profile) — `page.locator('a:has-text("Profile Settings")')`

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Profile SettingsChange PasswordChange PasswordChange your cu |
| `.profile-container` | `div` | `profile-container` | Profile SettingsChange PasswordChange PasswordChange your cu |
| `.profile-header` | `div` | `profile-header` | Profile SettingsChange PasswordChange PasswordChange your cu |
| `.breadcrumb` | `nav` | `breadcrumb` | Profile SettingsChange Password |
| `.breadcrumb-item` | `a` | `breadcrumb-item` | Profile Settings |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Change Password |
| `.subtitle` | `p` | `subtitle` | Change your current password |
| `.password-form` | `div` | `password-form` | New Password * Confirm New Password * Confirm |
| `#input-v-6` | `input` | `` |  |
| `#input-v-6-messages` | `div` | `` |  |
| `#input-v-9` | `input` | `` |  |
| `#input-v-9-messages` | `div` | `` |  |
| `.confirm-btn` | `button` | `confirm-btn` | Confirm |

---

### Settings
**URL:** `/settings`

**Headings:**
- `h2` — "API Keys"


**Text Content (3):**
- [p] "Organization"
- [p] "These are your API Keys to access the flow designer using our available APIs. You can create new API keys, copy an API key, or revoke an API key."
- [p] "For more information, see here"

**Buttons (3):**
- `page.locator('button:has-text("Role:All")')`
- `page.locator('button:has-text("Status:All")')`
- `page.locator('button:has-text("Add new API Key")')`

**Icon Buttons (8):**
- mdi-pencil-outline (`mdi-pencil-outline`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)
- mdi-pencil (`mdi-pencil`)

**Links (2):**
- [Settings](/settings) — `page.locator('a:has-text("Settings")')`
- [here](#) — `page.locator('a:has-text("here")')`

**Table 1:**
- Columns: `API Key ID` | `Role` | `Description` | `Status` | `Creation Date` | `Expired Date` | `Actions`
- Rows: 7
- Sample row: e8d40*****1d3f3 | document_search | Auto-generated for document search: Picotestwebcrawler2 | Revoked | 24 Mar 2026, 11:00 AM | 24 Mar 2027, 11:00 AM | 

**Table 2:**
- Columns: `API Key ID` | `Role` | `Description` | `Status` | `Creation Date` | `Expired Date` | `Actions`
- Rows: 7
- Sample row: e8d40*****1d3f3 | document_search | Auto-generated for document search: Picotestwebcrawler2 | Revoked | 24 Mar 2026, 11:00 AM | 24 Mar 2027, 11:00 AM | 

**Chips (7):** `Revoked`, `Active`, `Revoked`, `Revoked`, `Revoked`, `Revoked`, `Revoked`

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')` ★
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

**Tooltips (7):**
- "Auto-generated for document search: Picotestwebcrawler2 " (on `td`)
- "Auto-generated for document search: Picotest_3 " (on `td`)
- "Auto-generated for document search: Picotest3 " (on `td`)
- "Auto-generated for document search: Picotest2 " (on `td`)
- "Auto-generated for document search: Picotest1 " (on `td`)
- "Auto-generated for document search: Picotest4 " (on `td`)
- "Auto-generated for document search: Picotest2 " (on `td`)

**Custom Elements & IDs (49):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | SettingsAPI KeyAPI KeysOther SettingsAPI KeysThese are your  |
| `.settings-container` | `div` | `settings-container` | SettingsAPI KeyAPI KeysOther SettingsAPI KeysThese are your  |
| `.settings-header` | `div` | `settings-header` | SettingsAPI Key |
| `.breadcrumb` | `nav` | `breadcrumb` | SettingsAPI Key |
| `.router-link-active` | `a` | `router-link-active router-link-exact-active breadcrumb-item` | Settings |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | API Key |
| `.settings-tabs` | `div` | `settings-tabs` | API KeysOther Settings |
| `.tabs` | `div` | `tabs` | API KeysOther Settings |
| `.tab-header` | `div` | `tab-header` | API KeysOther Settings |
| `.tab` | `div` | `tab active` | API Keys |
| `.tab` | `div` | `tab` | Other Settings |
| `.api-keys-content` | `div` | `api-keys-content` | API KeysThese are your API Keys to access the flow designer  |
| `.section-title` | `h2` | `section-title` | API Keys |
| `.section-description` | `p` | `section-description` | These are your API Keys to access the flow designer using ou |
| `.section-info` | `p` | `section-info` | For more information, see here |
| `.info-link` | `a` | `info-link` | here |
| `.actions-row` | `div` | `actions-row` | Role:AllStatus:AllAdd new API Key |
| `.dropdown-button` | `button` | `dropdown-button` | Role:All |
| `.dropdown-text` | `span` | `dropdown-text` | Role:All |
| `.dropdown-key` | `span` | `dropdown-key` | Role: |
| `.dropdown-value` | `span` | `dropdown-value` | All |
| `.add-btn` | `button` | `add-btn` | Add new API Key |
| `.api-keys-table` | `div` | `api-keys-table` | API Key IDRoleDescriptionStatusCreation DateExpired DateActi |
| `.description-column` | `td` | `description-column` | Auto-generated for document search: Picotestwebcrawler2 |
| `.edit-btn` | `button` | `edit-btn` |  |

#### Discovered Modals / Dialogs

**Triggered by:** `page.locator('button:has-text("Add new API Key")').click()`
**Overlay selector:** `page.locator('.align-center')`
**Wait for overlay:** `await page.locator('.align-center').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog v-overlay--scroll-blocked`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.align-center` (classes: `v-row align-center justify-center`)
- Title: "Create New API Key"
- Inputs:
  - Enter number of days (`text`)
  - Enter description (`text`)
- Buttons: `Cancel`, `Create API Key`
- 2 input(s): Enter number of days, Enter description
- 1 dropdown(s): unlabeled
- 2 table(s)
- chips: Revoked, Active, Revoked, Revoked, Revoked, Revoked, Revoked
- headings: API Keys, Create New API Key
- buttons: Role:All, Status:All, Add new API Key, Cancel, Create API Key
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

- **Dropdown: "Dropdown 1"** — 2 option(s): `Internal`, `External`
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

---

### Logs
**URL:** `/logs`

**Headings:**
- `h2` — "Conversation Logs"

**Toolbar Buttons:** `Show column`

**Text Content (3):**
- [p] "Organization"
- [p] "No node events found"
- [text-caption] "Try adjusting your date range"

**Buttons (3):**
- `page.locator('button:has-text("Load Data")')`
- `page.locator('button:has-text("Show column")')`
- `page.locator('button:has-text("Export to CSV")')` [disabled]

**⚠️ Dynamic Buttons (DO NOT match by text):**
- "Datetime Start: Feb 26, 2026" → pattern: `Datetime Start: {DATE}`
  Use position: `page.locator('button:visible').nth(N)`
- "Datetime End: Mar 26, 2026" → pattern: `Datetime End: {DATE}`
  Use position: `page.locator('button:visible').nth(N)`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Table 1:**
- Columns: `Event Timestamp` | `Node Type` | `Node ID Name` | `Conversation ID` | `Flow ID` | `Sequence ID` | `Model Name` | `Latency (ms)` | `Input Tokens` | `Output Tokens` | `Total Tokens` | `Start Timestamp` | `End Timestamp` | `Error Type` | `Input Message`
- Rows: 1
- Sample row: No node events foundTry adjusting your date range

**Table 2:**
- Columns: `Event Timestamp` | `Node Type` | `Node ID Name` | `Conversation ID` | `Flow ID` | `Sequence ID` | `Model Name` | `Latency (ms)` | `Input Tokens` | `Output Tokens` | `Total Tokens` | `Start Timestamp` | `End Timestamp` | `Error Type` | `Input Message`
- Rows: 1
- Sample row: No node events foundTry adjusting your date range

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')` ★
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Avatars:** 1 visible

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

**Triggered by:** `page.locator('button:has-text("Show column")').click()`
**Overlay selector:** `page.locator('#v-menu-v-8')`
**Wait for overlay:** `await page.locator('#v-menu-v-8').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `v-overlay v-overlay--absolute v-overlay--active v-theme--mainTheme v-locale--is-ltr v-menu`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.v-list` (classes: `v-list v-theme--mainTheme v-list--density-default v-list--one-line`)
- Inputs:
  - Search column (`text`)
- Buttons: `Apply`
- 1 input(s): Search column
- 30 checkbox(es): Event Timestamp, Event Timestamp, Node Type, Node Type, Node ID Name, Node ID Name, Conversation ID, Conversation ID, Flow ID, Flow ID, Sequence ID, Sequence ID, Model Name, Model Name, Latency (ms), Latency (ms), Input Tokens, Input Tokens, Output Tokens, Output Tokens, Total Tokens, Total Tokens, Start Timestamp, Start Timestamp, End Timestamp, End Timestamp, Error Type, Error Type, Input Message, Input Message
- 2 table(s)
- headings: Conversation Logs
- buttons: Load Data, Show column, Export to CSV, Apply
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

---

### Add-Ons
**URL:** `/add-ons`

**Headings:**
- `h2` — "Add-Ons"
- `h4` — "Bird Messaging Channel"
- `h4` — "Telegram Bot"
- `h4` — "Twilio Messaging Service"
- `h4` — "Chatwoot CRM"
- `h4` — "AWS SES Email Service"
- `h4` — "Microsoft Graph Email Service"
- `h4` — "Webchat Widget"


**Text Content (10):**
- [p] "Organization"
- [p] "Make the most of your work by adding the channels and models that will save up time and boost your workflow."
- [p] "IntentAI"
- [p] "Send messages through Bird."
- [p] "Send messages through Telegram."
- [p] "Send messages through Twilio."
- [p] "Manage customer interactions with Chatwoot."
- [p] "Send emails through AWS SES."
- [p] "Send and receive email via Microsoft Graph (Outlook)."
- [p] "Add a customizable webchat widget to your website for seamless customer interactions"

**Input Fields (1):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Search | `text` | `page.locator('.v-field__input').nth(0)` |

**Buttons (9):**
- `page.locator('button:has-text("Sort:Default")')`
- `page.locator('button:has-text("Category:All")')`
- `page.locator('button:has-text("Install")')`
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*
- `page.locator('button:has-text("Install")').nth(N)` *(duplicate text — use .nth())*

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')` ★
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Images:** "Bird Messaging Channel logo", "Telegram Bot logo", "Twilio Messaging Service logo", "Chatwoot CRM logo", "AWS SES Email Service logo", "Microsoft Graph Email Service logo", "Webchat Widget logo"

**Avatars:** 1 visible

**Custom Elements & IDs (52):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add-OnsAdd-OnsMake the most of your work by adding the chann |
| `.breadcrumb` | `nav` | `breadcrumb` | Add-Ons |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Add-Ons |
| `.search-input` | `div` | `search-input` |  |
| `#input-v-6` | `input` | `` |  |
| `.dropdown-button` | `button` | `dropdown-button` | Sort:Default |
| `.dropdown-text` | `span` | `dropdown-text` | Sort:Default |
| `.dropdown-key` | `span` | `dropdown-key` | Sort: |
| `.dropdown-value` | `span` | `dropdown-value` | Default |
| `.tabs` | `div` | `tabs` | AllChannelsWidgets |
| `.tab-header` | `div` | `tab-header` | AllChannelsWidgets |
| `.tab` | `div` | `tab active` | All |
| `.tab` | `div` | `tab` | Channels |
| `.addons-content` | `div` | `addons-content` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.section` | `div` | `section` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.addons-grid` | `div` | `addons-grid` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.addon-card` | `div` | `addon-card` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.card-header` | `div` | `card-header` | Bird Messaging ChannelIntentAI |
| `.addon-info` | `div` | `addon-info` | Bird Messaging ChannelIntentAI |
| `.addon-icon` | `div` | `addon-icon` |  |
| `.addon-icon` | `img` | `addon-icon` |  |
| `.addon-details` | `div` | `addon-details` | Bird Messaging ChannelIntentAI |
| `.addon-name` | `h4` | `addon-name` | Bird Messaging Channel |
| `.addon-author` | `p` | `addon-author` | IntentAI |
| `.addon-description` | `p` | `addon-description` | Send messages through Bird. |
| `.card-footer` | `div` | `card-footer` | ChannelInstall |
| `.category-tag` | `span` | `category-tag channel` | Channel |

#### Discovered Modals / Dialogs

**Triggered by:** `page.locator('button:has-text("Sort:Default")').click()`
**Overlay selector:** `page.locator('.dropdown-menu')`
**Wait for overlay:** `await page.locator('.dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**CSS classes:** `v-overlay v-overlay--absolute v-overlay--active v-theme--mainTheme v-locale--is-ltr v-menu`

- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- Container: `.dropdown-menu` (classes: `v-list v-theme--mainTheme v-list--density-default v-list--one-line dropdown-menu`)
- 1 input(s): Search
- headings: Add-Ons, Bird Messaging Channel, Telegram Bot, Twilio Messaging Service, Chatwoot CRM, AWS SES Email Service, Microsoft Graph Email Service, Webchat Widget
- buttons: Sort:Default, Category:All, Install, Install, Install, Install, Install, Install, Install
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

---

### Flow Tester
**URL:** `/flow-tester`

**Headings:**
- `h4` — "Enable SSE"


**Text Content (3):**
- [p] "Organization"
- [v-card-title] "Select Conversation FlowUntitledSelect VersionInitial versionEnable SSEOnOff"
- [v-card-text] "Additional bot response"

**Input Fields (1):**

| # | Label | Type | Playwright Selector |
|---|-------|------|---------------------|
| 1 | Type here | `text` | `page.locator('.v-field__input').nth(0)` |

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "Untitled"
  - Open: `page.locator('.v-select, .v-autocomplete, .v-combobox').nth(0).click()`
  - Options: `Untitled`, `Untitled`, `Untitled`, `Untitled`, `Untitled`, `Untitled`, `Untitled`
  - Pick option: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (4):**
- `page.locator('button:has-text("Initial version")')`
- `page.locator('button:has-text("On")')`
- `page.locator('button:has-text("Off")')`
- `page.locator('button:has-text("Additional bot response")')`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`)

**Cards (1):**
- **Select Conversation FlowUntitledSelect VersionInitial versionEnable SSEOnOff**
  Text: "Additional bot response"

**Sidebar Navigation (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')` ★
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Expansion Panels (1):**
- **Additional bot response** — closed
  `page.locator('.v-expansion-panel-title').nth(0).click()`

**Avatars:** 1 visible

**Custom Elements & IDs (39):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Select Conversation FlowUntitledSelect VersionInitial versio |
| `.tester-container` | `div` | `tester-container` | Select Conversation FlowUntitledSelect VersionInitial versio |
| `.tester-container-card` | `div` | `tester-container-card` | Select Conversation FlowUntitledSelect VersionInitial versio |
| `.chatbox-title` | `div` | `chatbox-title` | Select Conversation FlowUntitledSelect VersionInitial versio |
| `.tester-select` | `div` | `tester-select` | Untitled |
| `.version-selector-container` | `div` | `version-selector-container` | Initial version |
| `.version-selector-button` | `button` | `version-selector-button` | Initial version |
| `.version-selector-text` | `span` | `version-selector-text` | Initial version |
| `.stream-toggle` | `div` | `stream-toggle` | Enable SSEOnOff |
| `.stream-toggle-label` | `h4` | `stream-toggle-label` | Enable SSE |
| `.stream-toggle-buttons` | `div` | `stream-toggle-buttons` | OnOff |
| `.chatbox` | `div` | `chatbox` | Additional bot response |
| `.bot-icon-trigger` | `div` | `bot-icon-trigger` |  |
| `.message-field` | `div` | `message-field` | Type here |
| `#input-v-16-label` | `label` | `` | Type here |
| `#input-v-16` | `input` | `` |  |

#### Explored Expansion Panels

**Panel: "Additional bot response"** (index 0):
- Container: `.v-navigation-drawer__prepend` (classes: `v-navigation-drawer__prepend`)
- Container: `.v-navigation-drawer__content` (classes: `v-navigation-drawer__content`)
- 1 input(s): Type here
- 1 dropdown(s): unlabeled
- 1 card(s): Select Conversation FlowUntitledSelect VersionInitial versionEnable SSEOnOff
- headings: Enable SSE
- buttons: Initial version, On, Off, Additional bot response
- custom elements: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title, .org-selector-wrapper, .org-dropdown, .org-dropdown-trigger, .org-info, .org-name

---

