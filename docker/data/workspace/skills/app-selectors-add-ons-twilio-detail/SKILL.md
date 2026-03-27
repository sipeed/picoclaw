---
name: app-selectors-add-ons-twilio-detail
description: DOM selectors and component map for the Add-Ons Twilio Detail page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Add-Ons Twilio Detail — Component Map

> Generated: 2026-03-27T06:44:35.546Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Add-Ons Twilio Detail
**URL:** `/add-ons/twilio`

**Headings:**
- `h1` — "Twilio Messaging Service" (selector: `.title`)
- `h2` — "Overview"
- `h3` — "Connected Flows"


**Text Content (5):**
- [p] "Organization" → `.org-title`
- [p] "Send messages through Twilio." → `.description`
- [p] "Author: IntentAI" → `.author`
- [p] "An overview of the Twilio Messaging Service." → `.overview-text`
- [p] "No connected flows found"

**Buttons (1):**
- `page.locator('.install-btn')`
  classes: `install-btn`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Links (1):**
- [Add-Ons](/add-ons) → `page.locator('a:has-text("Add-Ons")')`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')` ★
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (47):**

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
| `.breadcrumb` | `nav` | `breadcrumb` | Add-OnsTwilio Messaging Service |
| `.router-link-active` | `a` | `router-link-active breadcrumb-item` | Add-Ons |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Twilio Messaging Service |
| `.main-content` | `div` | `main-content` | InstallTwilio Messaging ServiceSend messages through Twilio. |
| `.sidebar` | `div` | `sidebar` | InstallTwilio Messaging ServiceSend messages through Twilio. |
| `.logo` | `div` | `logo` |  |
| `.channel-logo` | `img` | `channel-logo` |  |
| `.install-btn` | `button` | `install-btn` | Install |
| `.info` | `div` | `info` | Twilio Messaging ServiceSend messages through Twilio.Author: |
| `.header` | `div` | `header` | Twilio Messaging Service |
| `.name` | `span` | `name` | Twilio Messaging Service |
| `.description` | `p` | `description` | Send messages through Twilio. |
| `.author` | `p` | `author` | Author: IntentAI |
| `.content-area` | `div` | `content-area` | Twilio Messaging ServiceOverviewConfigureReviewsOverviewAn o |
| `.title` | `h1` | `title` | Twilio Messaging Service |
| `.tabs` | `div` | `tabs` | OverviewConfigureReviews |
| `.tab-header` | `div` | `tab-header` | OverviewConfigureReviews |
| `.tab` | `div` | `tab active` | Overview |
| `.tab` | `div` | `tab` | Configure |
| `.tab-content` | `div` | `tab-content` | OverviewAn overview of the Twilio Messaging Service.Connecte |
| `.overview-text` | `p` | `overview-text` | An overview of the Twilio Messaging Service. |
| `.table-empty-state` | `div` | `table-empty-state` | No connected flows found |

---

