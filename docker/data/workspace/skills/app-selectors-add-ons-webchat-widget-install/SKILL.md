---
name: app-selectors-add-ons-webchat-widget-install
description: DOM selectors and component map for the Add-Ons Webchat Widget Install page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Add-Ons Webchat Widget Install — Component Map

> Generated: 2026-03-27T06:44:29.646Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Add-Ons Webchat Widget Install
**URL:** `/add-ons/webchat-widget/install`

**Headings:**
- `h2` — "Webchat Widget Installation"


**Text Content (3):**
- [p] "Organization" → `.org-title`
- [p] "Add a customizable webchat widget to your website for seamless customer interactions"
- [p] "Enter the allowed origins for the widget" → `.form-section-description`

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Enter origin | `text` | `input[placeholder="Enter origin"]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (1):**
- **Service ID** (select)
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `Untitled`, `Untitled`, `Untitled`, `Untitled`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (2):**
- `page.locator('button:has-text("Add Origin")')`
- `page.locator('button:has-text("Save")')`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Links (2):**
- [Add-Ons](/add-ons) → `page.locator('a:has-text("Add-Ons")')`
- [Webchat Widget](/add-ons/webchat-widget) → `page.locator('a:has-text("Webchat Widget")')`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')` ★
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (34):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add-OnsWebchat WidgetInstall Webchat Widget InstallationAdd  |
| `.breadcrumb` | `nav` | `breadcrumb` | Add-OnsWebchat WidgetInstall |
| `.router-link-active` | `a` | `router-link-active breadcrumb-item` | Add-Ons |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `a` | `breadcrumb-item` | Webchat Widget |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Install |
| `.form-section-label` | `label` | `form-section-label` | Allowed Origins |
| `.form-section-description` | `p` | `form-section-description` | Enter the allowed origins for the widget |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |

---

