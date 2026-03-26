---
name: app-selectors-knowledge-base
description: DOM selectors and component map for the Knowledge Base page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Knowledge Base — Component Map

> Generated: 2026-03-26T07:05:57.372Z
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
- 2 dropdown(s): ?, ?
- headings: Knowledge Base, Create KB Bucket, Step 1: Source Settings
- buttons: Create Knowledge Base Bucket, Schedule, Sync, Schedule, Sync, Schedule, Sync, Continue
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"Dropdown 1"**: `None`, `gemini-flash-2.5`
  Open: `page.locator('.v-select').filter({ hasText: /Dropdown 1/ }).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Dropdown 2"**: `Google Cloud Storage`, `Azure Blob Storage`, `Website Crawler`
  Open: `page.locator('.v-select').filter({ hasText: /Dropdown 2/ }).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---

