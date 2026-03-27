---
name: app-selectors-dashboard
description: DOM selectors and component map for the Dashboard page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Dashboard — Component Map

> Generated: 2026-03-27T06:39:30.905Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Dashboard
**URL:** `/`

**Headings:**
- `h2` — "Dashboard"
- `h2` — "Total Conversations" (selector: `.metric-title`)
- `h1` — "0" (selector: `.metric-value`)
- `h2` — "Executions" (selector: `.metric-title`)
- `h2` — "Error Rate" (selector: `.metric-title`)
- `h1` — "0.00%" (selector: `.metric-value`)
- `h2` — "Latency" (selector: `.metric-title`)
- `h1` — "–" (selector: `.metric-value`)
- `h2` — "Total Tokens" (selector: `.metric-title`)
- `h2` — "Knowledge Base Buckets" (selector: `.metric-title`)
- `h1` — "2" (selector: `.metric-value`)


**Text Content (1):**
- [p] "Organization" → `.org-title`

**Buttons (1):**
- `page.locator('.date-range-button')`
  classes: `date-range-button`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')` ★
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | DashboardMar 20, 2026 - Mar 27, 2026Total Conversations0Exec |
| `.dashboard-container` | `div` | `dashboard-container` | DashboardMar 20, 2026 - Mar 27, 2026Total Conversations0Exec |
| `.dashboard-header` | `div` | `dashboard-header` | Dashboard |
| `.date-range-button` | `button` | `date-range-button` | Mar 20, 2026 - Mar 27, 2026 |
| `.date-range-text` | `span` | `date-range-text` | Mar 20, 2026 - Mar 27, 2026 |
| `.date-range-value` | `span` | `date-range-value` | Mar 20, 2026 - Mar 27, 2026 |
| `.metrics-section` | `div` | `metrics-section` | Total Conversations0Executions0Error Rate0.00%Latency–Total  |
| `.metric-card` | `div` | `metric-card` | Total Conversations0 |
| `.metric-title` | `h2` | `metric-title` | Total Conversations |
| `.metric-value-container` | `div` | `metric-value-container` | 0 |
| `.metric-value` | `h1` | `metric-value` | 0 |
| `.chart-section` | `div` | `chart-section` |  |
| `.executions-chart-container` | `div` | `executions-chart-container` |  |

---

