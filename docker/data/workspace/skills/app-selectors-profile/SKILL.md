---
name: app-selectors-profile
description: DOM selectors and component map for the Profile page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Profile — Component Map

> Generated: 2026-03-26T12:28:18.020Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Profile
**URL:** `/profile`

**Headings:**
- `h2` — "Profile Settings"
- `text-h3` — "H"


**Text Content (2):**
- [p] "Organization" → `.org-title`
- [p] "Upload your photo and details here"

**Input Fields (2):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | First Name | `text` | `.custom-drawer input[type="text"]` |
| 2 | Last Name | `text` | `.custom-drawer input[type="text"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**File Inputs:**
- H (image/*)

**Buttons (5):**
- `page.locator('.upload-btn')`
  classes: `v-btn v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-outlined upload-btn`
- `page.locator('.delete-btn')` [disabled]
  classes: `v-btn v-btn--disabled v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-text delete-btn`
- `page.locator('.change-email-link')`
  classes: `change-email-link`
- `page.locator('.change-password-link')`
  classes: `change-password-link`
- `page.locator('.m-auto')` [disabled]
  classes: `v-btn v-btn--disabled v-theme--mainTheme v-btn--density-default v-btn--size-large v-btn--variant-flat m-auto mt-8 px-8 w-fit text-capitalize font-weight-bold bg-btn-primary text-btn-text-primary`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

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
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.change-email-link` | `button` | `change-email-link` | Change Email |
| `.change-password-link` | `button` | `change-password-link` | Change Password |
| `.m-auto` | `button` | `m-auto` | Save Changes |

---

