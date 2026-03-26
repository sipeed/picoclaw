---
name: app-selectors-knowledge-base
description: DOM selectors and component map for the Knowledge Base page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Knowledge Base — Component Map

> Generated: 2026-03-26T12:26:24.472Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Knowledge Base
**URL:** `/knowledge-base`

**Headings:**
- `h2` — "Knowledge Base"


**Text Content (1):**
- [p] "Organization" → `.org-title`

**Buttons (7):**
- `page.locator('.create-button')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-flat create-button`
- `page.locator('.schedule-button')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text schedule-button`
- `page.locator('.sync-button')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text sync-button`
- `page.locator('.schedule-button')` *(dup)*
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text schedule-button`
- `page.locator('.sync-button')` *(dup)*
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text sync-button`
- `page.locator('.schedule-button')` *(dup)*
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text schedule-button`
- `page.locator('.sync-button')` *(dup)*
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text sync-button`

**Icon Buttons (4):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- mdi-dots-vertical (`mdi-dots-vertical`)
- mdi-dots-vertical (`mdi-dots-vertical`)
- mdi-dots-vertical (`mdi-dots-vertical`)

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')` ★
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (42):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Knowledge BaseCreate Knowledge Base BucketPicotest1ReadySour |
| `.knowledge-base-container` | `div` | `knowledge-base-container` | Knowledge BaseCreate Knowledge Base BucketPicotest1ReadySour |
| `.knowledge-base-header` | `div` | `knowledge-base-header` | Knowledge BaseCreate Knowledge Base Bucket |
| `.create-button` | `button` | `create-button` | Create Knowledge Base Bucket |
| `.bucket-grid` | `div` | `bucket-grid` | Picotest1ReadySource TypeGoogle Cloud StorageKnowledge files |
| `.bucket-card` | `div` | `bucket-card` | Picotest1ReadySource TypeGoogle Cloud StorageKnowledge files |
| `.bucket-header` | `div` | `bucket-header` | Picotest1Ready |
| `.bucket-title` | `div` | `bucket-title` | Picotest1Ready |
| `.bucket-name` | `span` | `bucket-name` | Picotest1 |
| `.knowledge-base-status-pill` | `span` | `knowledge-base-status-pill knowledge-base-status-pill--ready` | Ready |
| `.bucket-info` | `div` | `bucket-info` | Source TypeGoogle Cloud StorageKnowledge filesScheduleSync |
| `.source-type` | `div` | `source-type` | Source TypeGoogle Cloud Storage |
| `.label` | `div` | `label` | Source Type |
| `.value` | `div` | `value` | Google Cloud Storage |
| `.files-count` | `div` | `files-count is-clickable` | Knowledge files |
| `.action-buttons` | `div` | `action-buttons` | ScheduleSync |
| `.schedule-button` | `button` | `schedule-button` | Schedule |
| `.sync-button` | `button` | `sync-button` | Sync |
| `.files-count` | `div` | `files-count` | Web pages (0) |

#### Discovered Modals / Dialogs

**Trigger:** `page.locator('button:has-text("Create Knowledge Base Bucket")').click()`
**Overlay:** `page.locator('.custom-drawer-overlay')`
**Wait:** `await page.locator('.custom-drawer-overlay').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `custom-drawer-overlay`

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
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Selected dropdown = None" inside modal:**
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
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Selected dropdown = None":**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Continue (dropdown = None)" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 2: Search Engine Configuration"
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Back`, `Submit`
- 2 input(s): https://file/ioO2IQEG6cEqO3eZJVZAz2V?node, 2309230_3ANM
- 1 dropdown(s): Search Engine
- headings: Knowledge Base, Create KB Bucket, Step 2: Search Engine Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Back, Submit
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Continue (dropdown = None)":**
- **"Search Engine"**: `Elasticsearch`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Selected dropdown = gemini-flash-2.5" inside modal:**
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
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Selected dropdown = gemini-flash-2.5":**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Continue (dropdown = gemini-flash-2.5)" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 2: Search Engine Configuration"
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Back`, `Submit`
- 2 input(s): https://file/ioO2IQEG6cEqO3eZJVZAz2V?node, 2309230_3ANM
- 1 dropdown(s): Search Engine
- headings: Knowledge Base, Create KB Bucket, Step 2: Search Engine Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Back, Submit
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Continue (dropdown = gemini-flash-2.5)":**
- **"Search Engine"**: `Elasticsearch`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Selected dropdown = Google Cloud Storage" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - Enter the name of GCS Bucket (Optional) (`text`)
  - Limit access to a specific folder/path prefix (`text`)
- Buttons: `Test Connection`, `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - Enter the name of GCS Bucket (Optional) (`text`)
  - Limit access to a specific folder/path prefix (`text`)
- Buttons: `Test Connection`, `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - Enter the name of GCS Bucket (Optional) (`text`)
  - Limit access to a specific folder/path prefix (`text`)
- Buttons: `Test Connection`, `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 3 input(s): Enter knowledge group name, Enter the name of GCS Bucket (Optional), Limit access to a specific folder/path prefix
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Google Cloud Storage Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Test Connection, Continue
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Selected dropdown = Google Cloud Storage":**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Continue (dropdown = Google Cloud Storage)" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 2: Search Engine Configuration"
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Back`, `Submit`
- 2 input(s): https://file/ioO2IQEG6cEqO3eZJVZAz2V?node, 2309230_3ANM
- 1 dropdown(s): Search Engine
- headings: Knowledge Base, Create KB Bucket, Step 2: Search Engine Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Back, Submit
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns after "Continue (dropdown = Google Cloud Storage)":**
- **"Search Engine"**: `Elasticsearch`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Selected dropdown = Azure Blob Storage" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - Website URL (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - Website URL (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - Website URL (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 2 input(s): Enter knowledge group name, Website URL
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns after "Selected dropdown = Azure Blob Storage":**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Continue (dropdown = Azure Blob Storage)" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 2: Search Engine Configuration"
- Inputs:
  - https://file/ioO2IQEG6cEqO3eZJVZAz2V?node (`text`)
  - 2309230_3ANM (`text`)
- Buttons: `Back`, `Submit`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Back`, `Submit`
- 2 input(s): https://file/ioO2IQEG6cEqO3eZJVZAz2V?node, 2309230_3ANM
- 1 dropdown(s): Search Engine
- headings: Knowledge Base, Create KB Bucket, Step 2: Search Engine Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Back, Submit
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns after "Continue (dropdown = Azure Blob Storage)":**
- **"Search Engine"**: `Elasticsearch`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Selected dropdown = Website Crawler" inside modal:**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 2 input(s): Enter knowledge group name, https://example.com
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns after "Selected dropdown = Website Crawler":**
- **"LLM transformer model to parse documents (optional)"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source Type *"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Expansion Panels after "Selected dropdown = Website Crawler":**
- **"Web Crawler Parameters"** (index 0)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Web Crawler Parameters/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 3 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours)
- 2 textarea(s)
- 2 dropdown(s): LLM transformer model to parse documents (optional), Source Type *
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Authentication"** (index 1)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Authentication/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 6 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123...
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"HTML Processing"** (index 2)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /HTML Processing/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 7 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated)
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Crawl Rules"** (index 3)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Crawl Rules/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 7 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated)
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Extraction Rules"** (index 4)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Extraction Rules/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 7 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated)
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Crawl Depth and Limits"** (index 5)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Crawl Depth and Limits/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 10 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Purge and Extraction Settings"** (index 6)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Purge and Extraction Settings/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 10 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"Field Size Limits"** (index 7)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /Field Size Limits/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 19 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

- **"URL Limits"** (index 8)
  Open: `page.locator('.v-expansion-panel-title').filter({ hasText: /URL Limits/ }).click()`
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer


#### Explored Expansion Panels

**Panel "Web Crawler Parameters" (0):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Authentication" (1):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "HTML Processing" (2):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Crawl Rules" (3):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Extraction Rules" (4):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Crawl Depth and Limits" (5):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Purge and Extraction Settings" (6):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "Field Size Limits" (7):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Panel "URL Limits" (8):**
- Container: `.custom-drawer-overlay` (classes: `custom-drawer-overlay`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.custom-drawer` (classes: `custom-drawer`)
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-header` (classes: `drawer-header`)
- Container: `.drawer-content` (classes: `drawer-content`)
- Title: "Step 1: Source Settings"
- Inputs:
  - Enter knowledge group name (`text`)
  - https://example.com (`text`)
  - 0 /6    (every 6 hours) (`text`)
  - username (`text`)
  - password (`password`)
  - Bearer token123... (`text`)
  - script, style, nav (comma-separated) (`text`)
  - 1 (`number`)
  - 1000 (`number`)
  - 1 (`number`)
  - 1000 (`number`)
  - 5242880 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10 (`number`)
  - 10 (`number`)
  - 512 (`number`)
  - 512 (`number`)
  - 10485760 (`number`)
  - 900 (`number`)
  - 2048 (`number`)
  - 16 (`number`)
  - 10 (`number`)
  - 10 (`number`)
- Buttons: `Continue`
- Container: `.drawer-actions` (classes: `drawer-actions`)
- Buttons: `Continue`
- 24 input(s): Enter knowledge group name, https://example.com, 0 /6    (every 6 hours), username, password, Bearer token123..., script, style, nav (comma-separated), 1, 1000, 1, 1000, 5242880, 512, 512, 10, 10, 512, 512, 10485760, 900, 2048, 16, 10, 10
- 2 textarea(s)
- 3 dropdown(s): LLM transformer model to parse documents (optional), Source Type *, Authentication Type
- 4 checkbox(es)
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings, Website Crawler Configuration
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

---

