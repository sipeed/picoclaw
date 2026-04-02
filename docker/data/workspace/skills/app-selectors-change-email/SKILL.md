---
name: app-selectors-change-email
description: DOM selectors and component map for the Change Email page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Change Email — Component Map

> Generated: 2026-03-27T09:43:44.482Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Change Email
**URL:** `/change-email`

**Headings:**
- `h2` — "Change Email"


**Text Content (2):**
- [p] "Organization" → `.org-title`
- [p] "Change your current email" → `.subtitle`

**Input Fields (2):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | New Email | `text` | `input[placeholder="New Email"]` |
| 2 | Confirm New Email | `text` | `input[placeholder="Confirm New Email"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Buttons (1):**
- `page.locator('.confirm-btn')` [disabled]
  classes: `v-btn v-btn--block v-btn--disabled v-theme--mainTheme v-btn--density-default v-btn--size-default v-btn--variant-elevated confirm-btn`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Links (1):**
- [Profile Settings](/profile) → `page.locator('a:has-text("Profile Settings")')`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Profile SettingsChange EmailChange EmailChange your current  |
| `.profile-container` | `div` | `profile-container` | Profile SettingsChange EmailChange EmailChange your current  |
| `.profile-header` | `div` | `profile-header` | Profile SettingsChange EmailChange EmailChange your current  |
| `.breadcrumb` | `nav` | `breadcrumb` | Profile SettingsChange Email |
| `.breadcrumb-item` | `a` | `breadcrumb-item` | Profile Settings |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Change Email |
| `.subtitle` | `p` | `subtitle` | Change your current email |
| `.email-form` | `div` | `email-form` | New Email * Confirm New Email * Confirm |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.confirm-btn` | `button` | `confirm-btn` | Confirm |

---

