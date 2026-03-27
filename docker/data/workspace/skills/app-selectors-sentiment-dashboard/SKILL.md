---
name: app-selectors-sentiment-dashboard
description: DOM selectors and component map for the Sentiment Dashboard page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Sentiment Dashboard — Component Map

> Generated: 2026-03-27T09:41:54.793Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Sentiment Dashboard
**URL:** `/sentiment`

**Headings:**
- `h3` — "Top 10 Intents"
- `h3` — "Accuracy"
- `h3` — "Sentiments"

**Toolbar Buttons:** `Select bots: 0 selected`
**Toolbar Buttons:** `Show column`

**Text Content (1):**
- [p] "Organization" → `.org-title`

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Datepicker input | `text` | `[data-testid="dp-input"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "10"
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (3):**
- `page.locator('.bot-data-download')`
  classes: `v-btn v-btn--elevated v-theme--mainTheme v-btn--density-comfortable v-btn--size-large v-btn--variant-elevated bot-data-download`
- `page.locator('.menu-dropdown')` [disabled]
  classes: `v-btn v-btn--disabled v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text menu-dropdown`
- `page.locator('.table-dropdown')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text table-dropdown`

**Icon Buttons (5):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:**
- Rows: 1 (paginated)
- Sample: No data available

**Table 2:**
- Rows: 1
- Sample: No data available

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Pagination:** 0 pages

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

**Trigger:** `page.locator('button:has-text("Show column")').click()`
**Overlay:** `page.locator('.column-search')`
**Wait:** `await page.locator('.column-search').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-input v-input--horizontal v-input--center-affix v-input--density-compact v-theme--mainTheme v-locale--is-ltr v-text-field column-search`

- Inputs:
  - Search column (`text`)
- Buttons: `Apply`
- 2 input(s): Datepicker input, Search column
- 1 dropdown(s): ?
- 2 table(s)
- headings: Top 10 Intents, Accuracy, Sentiments
- buttons: Download CSV, Select bots: 0 selected, Show column, Apply
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"Dropdown 1"**: `10`, `25`, `50`, `100`, `All`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---

