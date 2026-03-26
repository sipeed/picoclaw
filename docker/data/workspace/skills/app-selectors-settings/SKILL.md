---
name: app-selectors-settings
description: DOM selectors and component map for the Settings page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Settings — Component Map

> Generated: 2026-03-26T12:28:42.194Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Settings
**URL:** `/settings`

**Headings:**
- `h2` — "API Keys" (selector: `.section-title`)


**Text Content (3):**
- [p] "Organization" → `.org-title`
- [p] "These are your API Keys to access the flow designer using our available APIs. You can create new API keys, copy an API key, or revoke an API key." → `.section-description`
- [p] "For more information, see here" → `.section-info`

**Buttons (3):**
- `page.locator('.dropdown-button')`
  classes: `dropdown-button`
- `page.locator('.dropdown-button')`
  classes: `dropdown-button`
- `page.locator('.add-btn')`
  classes: `v-btn v-btn--elevated v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-elevated text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary add-btn`

**Icon Buttons (12):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`
- mdi-pencil (`mdi-pencil`) → `.edit-btn`

**Links (2):**
- [Settings](/settings) → `page.locator('a:has-text("Settings")')`
- [here](#) → `page.locator('a:has-text("here")')`

**Table 1:** `.api-keys-table`
- Columns: `API Key ID` | `Role` | `Description` | `Status` | `Creation Date` | `Expired Date` | `Actions`
- Rows: 11
- Sample: e8d40*****1d3f3 | document_search | Auto-generated for document search: Picotestwebcrawler2 | Revoked | 24 Mar 2026, 4:00 AM | 24 Mar 2027, 4:00 AM | 

**Table 2:**
- Columns: `API Key ID` | `Role` | `Description` | `Status` | `Creation Date` | `Expired Date` | `Actions`
- Rows: 11
- Sample: e8d40*****1d3f3 | document_search | Auto-generated for document search: Picotestwebcrawler2 | Revoked | 24 Mar 2026, 4:00 AM | 24 Mar 2027, 4:00 AM | 

**Chips:** `Revoked`, `Active`, `Revoked`, `Revoked`, `Revoked`, `Revoked`, `Revoked`, `Active`, `Revoked`, `Active`, `Revoked`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')` ★
- `page.locator('a:has-text("Organization")')`

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

**Trigger:** `page.locator('button:has-text("Add new API Key")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Title: "Create New API Key"
- Inputs:
  - Enter number of days (`text`)
  - Enter description (`text`)
- Buttons: `Cancel`, `Create API Key`
- 2 input(s): Enter number of days, Enter description
- 1 dropdown(s): Role *
- 2 table(s)
- headings: API Keys, Create New API Key
- buttons: Role:All, Status:All, Add new API Key, Cancel, Create API Key
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"Role *"**: `Internal`, `External`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**After clicking "Create API Key" inside modal:**
- Title: "Create New API Key"
- Inputs:
  - Enter number of days (`text`)
  - Enter description (`text`)
- Buttons: `Cancel`, `Create API Key`
- 2 input(s): Enter number of days, Enter description
- 1 dropdown(s): Role *
- 2 table(s)
- headings: API Keys, Create New API Key
- buttons: Role:All, Status:All, Add new API Key, Cancel, Create API Key
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns after "Create API Key":**
- **"Role *"**: `Internal`, `External`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---

