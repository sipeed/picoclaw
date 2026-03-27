---
name: app-selectors-add-ons-twilio-install
description: DOM selectors and component map for the Add-Ons Twilio Install page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Add-Ons Twilio Install — Component Map

> Generated: 2026-03-27T06:44:43.615Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Add-Ons Twilio Install
**URL:** `/add-ons/twilio/install`

**Headings:**
- `h2` — "Twilio Messaging Service Installation"


**Text Content (2):**
- [p] "Organization" → `.org-title`
- [p] "Send messages through Twilio."

**Input Fields (5):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Enter channel name | `text` | `input[placeholder="Enter channel name"]` |
| 2 | Enter Account SID... | `text` | `input[placeholder="Enter Account SID..."]` |
| 3 | Enter Auth Token... | `password` | `input[placeholder="Enter Auth Token..."]` |
| 4 | Enter Messaging Service SID... | `text` | `input[placeholder="Enter Messaging Service SID..."]` |
| 5 | Enter Phone Number... | `text` | `input[placeholder="Enter Phone Number..."]` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (2):**
- **Service ID** (select)
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `Untitled`, `Untitled`, `Untitled`, `Untitled`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`
- **Token Streaming Mode (optional)** (select) — current: "Disabled (no streaming)"
  - Selector: `.v-select:nth(1)`
  - Open: `page.locator('.v-select:nth(1)').click()`
  - Options: `High (sentence chunks)`, `Low (newline chunks)`, `Disabled (no streaming)`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Buttons (1):**
- `page.locator('button:has-text("Save")')`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Links (2):**
- [Add-Ons](/add-ons) → `page.locator('a:has-text("Add-Ons")')`
- [Twilio Messaging Service](/add-ons/twilio) → `page.locator('a:has-text("Twilio Messaging Service")')`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')` ★
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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add-OnsTwilio Messaging ServiceInstallTwilio Messaging Servi |
| `.breadcrumb` | `nav` | `breadcrumb` | Add-OnsTwilio Messaging ServiceInstall |
| `.router-link-active` | `a` | `router-link-active breadcrumb-item` | Add-Ons |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `a` | `breadcrumb-item` | Twilio Messaging Service |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Install |
| `.form-section-label` | `label` | `form-section-label` | Channel Name |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `` | `input` | `` |  |
| `` | `div` | `` |  |
| `.optional-indicator` | `span` | `optional-indicator` | (optional) |
| `` | `div` | `` |  |

---

