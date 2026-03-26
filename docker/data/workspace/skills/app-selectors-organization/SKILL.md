---
name: app-selectors-organization
description: DOM selectors and component map for the Organization page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Organization — Component Map

> Generated: 2026-03-26T12:27:57.684Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Organization
**URL:** `/organization`

**Headings:**
- `h2` — "Organization Team"
- `h2` — "Bot Icons"


**Text Content (6):**
- [p] "Organization" → `.org-title`
- [text-secondary] "Team"
- [p] "View, add and manage your organizations' members here"
- [p] "Manage icons that can be used for your bots"
- [p] "(max 5MB, only PNG, JPG, and JPEG are supported, best to use square images)" → `.bot-icons-requirements`
- [p] "No icons uploaded yet"

**Dropdowns / Selects (1):**
- **Dropdown 1** (select) — current: "20"
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `10`, `25`, `50`, `100`, `All`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**File Inputs:**
- File 1 (image/png, image/jpeg, image/jpg)

**Buttons (2):**
- `page.locator('button:has-text("Add a Member")')`
- `page.locator('button:has-text("Upload Icon")')`

**Icon Buttons (5):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- First page (`mdi-page-first`)
- Previous page (`mdi-chevron-left`)
- Next page (`mdi-chevron-right`)
- Last page (`mdi-page-last`)

**Table 1:** `.organization-table`
- Columns: `Member's name` | `Role` | `Email` | `Invite on` | `Status` | `Actions`
- Rows: 8 (paginated)
- Sample:  | agent | heidi+1@intnt.ai | 17 Mar 2026 | Active | 

**Table 2:**
- Columns: `Member's name` | `Role` | `Email` | `Invite on` | `Status` | `Actions`
- Rows: 8
- Sample:  | agent | heidi+1@intnt.ai | 17 Mar 2026 | Active | 

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')` ★

**Pagination:** 0 pages

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

**Trigger:** `page.locator('button:has-text("Add a Member")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Title: "Add a Member"
- Inputs:
  - Email (`text`)
- Buttons: `Cancel`, `Add`
- 1 input(s): Email
- 2 dropdown(s): ?, Role *
- 2 table(s)
- headings: Organization Team, Bot Icons, Add a Member
- buttons: Add a Member, Upload Icon, Cancel, Add
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"Role *"**: `admin`, `developer`, `agent`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Add" inside modal:**
- Title: "Add a Member"
- Inputs:
  - Email (`text`)
- Buttons: `Cancel`, `Add`
- 1 input(s): Email
- 2 dropdown(s): ?, Role *
- 2 table(s)
- headings: Organization Team, Bot Icons, Add a Member
- buttons: Add a Member, Upload Icon, Cancel, Add
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Add":**
- **"Role *"**: `admin`, `developer`, `agent`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---

