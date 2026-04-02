---
name: app-selectors-flow-designer
description: DOM selectors and component map for the Flow Designer page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Flow Designer — Component Map

> Generated: 2026-03-27T09:37:20.035Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Designer
**URL:** `/flow-designer`

**Headings:**
- `h2` — "All Flows"


**Text Content (1):**
- [p] "Organization" → `.org-title`

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Search flow name | `text` | `.search-input input` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "10"
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Buttons (1):**
- `page.locator('.m-auto')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto px-12 text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`

**Icon Buttons (14):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- mdi-pencil (`mdi-pencil`) → `.action-btn`
- mdi-content-copy (`mdi-content-copy`) → `.action-btn`
- mdi-delete (`mdi-delete`) → `.action-btn`
- mdi-pencil (`mdi-pencil`) → `.action-btn`
- mdi-content-copy (`mdi-content-copy`) → `.action-btn`
- mdi-delete (`mdi-delete`) → `.action-btn`
- mdi-pencil (`mdi-pencil`) → `.action-btn`
- mdi-content-copy (`mdi-content-copy`) → `.action-btn`
- mdi-delete (`mdi-delete`) → `.action-btn`
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:** `.flow-data-table`
- Columns: `Flow Name` | `Last Update` | `Actions`
- Rows: 3 (paginated)
- Sample: Untitled | 27 Mar 2026, 9:33 AM | 

**Table 2:**
- Columns: `Flow Name` | `Last Update` | `Actions`
- Rows: 3
- Sample: Untitled | 27 Mar 2026, 9:33 AM | 

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')` ★
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Pagination:** 0 pages

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
| `.flow-table-container` | `div` | `flow-table-container` | Flow NameLast UpdateActionsUntitled27 Mar 2026, 9:33 AMUntit |
| `.table-controls` | `div` | `table-controls` |  |
| `.search-section` | `div` | `search-section` |  |
| `.search-input` | `div` | `search-input` |  |
| `` | `input` | `` |  |
| `.flow-data-table` | `div` | `flow-data-table` | Flow NameLast UpdateActionsUntitled27 Mar 2026, 9:33 AMUntit |
| `.action-icons` | `div` | `action-icons` |  |
| `.action-btn` | `button` | `action-btn` |  |
| `[data-testid="v-pagination-root"]` | `nav` | `` |  |
| `[data-testid="v-pagination-first"]` | `li` | `` |  |

---

