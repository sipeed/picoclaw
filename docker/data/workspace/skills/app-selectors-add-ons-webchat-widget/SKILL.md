---
name: app-selectors-add-ons-webchat-widget
description: DOM selectors and component map for the Add-Ons Webchat Widget page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Add-Ons Webchat Widget — Component Map

> Generated: 2026-03-27T06:44:19.611Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Add-Ons Webchat Widget
**URL:** `/add-ons/webchat-widget`

**Headings:**
- `h1` — "Webchat Widget" (selector: `.title`)
- `h2` — "Overview"
- `h3` — "Connected Items"


**Text Content (5):**
- [p] "Organization" → `.org-title`
- [p] "Add a customizable webchat widget to your website for seamless customer interactions" → `.description`
- [p] "Author: IntentAI" → `.author`
- [p] "The Webchat Widget allows you to embed a fully customizable chat interface on your website. Engage with your customers in real-time, provide instant support, and enhance user experience." → `.overview-text`
- [p] "No connected items found"

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add-OnsWebchat WidgetInstallWebchat WidgetAdd a customizable |
| `.breadcrumb` | `nav` | `breadcrumb` | Add-OnsWebchat Widget |
| `.router-link-active` | `a` | `router-link-active breadcrumb-item` | Add-Ons |
| `.mdi` | `i` | `mdi notranslate breadcrumb-separator` |  |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Webchat Widget |
| `.main-content` | `div` | `main-content` | InstallWebchat WidgetAdd a customizable webchat widget to yo |
| `.sidebar` | `div` | `sidebar` | InstallWebchat WidgetAdd a customizable webchat widget to yo |
| `.logo` | `div` | `logo` |  |
| `.widget-logo` | `img` | `widget-logo` |  |
| `.install-btn` | `button` | `install-btn` | Install |
| `.info` | `div` | `info` | Webchat WidgetAdd a customizable webchat widget to your webs |
| `.header` | `div` | `header` | Webchat Widget |
| `.name` | `span` | `name` | Webchat Widget |
| `.description` | `p` | `description` | Add a customizable webchat widget to your website for seamle |
| `.author` | `p` | `author` | Author: IntentAI |
| `.content-area` | `div` | `content-area` | Webchat WidgetOverviewConfigureReviewsOverviewThe Webchat Wi |
| `.title` | `h1` | `title` | Webchat Widget |
| `.tabs` | `div` | `tabs` | OverviewConfigureReviews |
| `.tab-header` | `div` | `tab-header` | OverviewConfigureReviews |
| `.tab` | `div` | `tab active` | Overview |
| `.tab` | `div` | `tab` | Configure |
| `.tab-content` | `div` | `tab-content` | OverviewThe Webchat Widget allows you to embed a fully custo |
| `.overview-text` | `p` | `overview-text` | The Webchat Widget allows you to embed a fully customizable  |
| `.table-empty-state` | `div` | `table-empty-state` | No connected items found |

---

