---
name: app-selectors-flow-tester
description: DOM selectors and component map for the Flow Tester page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Flow Tester — Component Map

> Generated: 2026-03-27T06:44:54.834Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Tester
**URL:** `/flow-tester`

**Headings:**
- `h4` — "Enable SSE" (selector: `.stream-toggle-label`)


**Text Content (3):**
- [p] "Organization" → `.org-title`
- [v-card-title] "Select Conversation FlowUntitledSelect Versionv1.0Enable SSEOnOff" → `.chatbox-title`
- [v-card-text] "Additional bot response"

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Type here | `text` | `.message-field input` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (1):**
- **Select Conversation Flow** (select) — current: "Untitled"
  - Selector: `.tester-select`
  - Open: `page.locator('.tester-select').click()`
  - Options: `Untitled`, `Untitled`, `Untitled`, `Untitled`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Button Toggles (`.v-btn-toggle`):**
- **Enable SSE:** On | Off — active: "On"
  Selector: `page.locator('.stream-toggle-buttons')`
  To select an option: `page.locator('.stream-toggle-buttons').getByText('OPTION_TEXT').click()`

**Buttons (3):**
- `page.locator('.version-selector-button')`
  classes: `version-selector-button`
- `page.locator('button:has-text("On")')`
- `page.locator('button:has-text("Off")')`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Cards (1):**
- **Select Conversation FlowUntitledSelect Versionv1.0Enable SSEOnOff** → `.tester-container-card`
  "Additional bot response"

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')` ★
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Expansion Panels:**
- **Additional bot response** (closed) → `page.locator('.v-expansion-panel-title:nth(0)').click()`

**Expansion Panel Selector Rule:** Use `.v-expansion-panel-title` (or `.crawler-sections .v-expansion-panel-title`) with text filter. Do NOT use `button:has-text(...)` for these section headers.

**Custom Elements & IDs (39):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Select Conversation FlowUntitledSelect Versionv1.0Enable SSE |
| `.tester-container` | `div` | `tester-container` | Select Conversation FlowUntitledSelect Versionv1.0Enable SSE |
| `.tester-container-card` | `div` | `tester-container-card` | Select Conversation FlowUntitledSelect Versionv1.0Enable SSE |
| `.chatbox-title` | `div` | `chatbox-title` | Select Conversation FlowUntitledSelect Versionv1.0Enable SSE |
| `.tester-select` | `div` | `tester-select` | Untitled |
| `.version-selector-container` | `div` | `version-selector-container` | v1.0 |
| `.version-selector-button` | `button` | `version-selector-button` | v1.0 |
| `.version-selector-text` | `span` | `version-selector-text` | v1.0 |
| `.stream-toggle` | `div` | `stream-toggle` | Enable SSEOnOff |
| `.stream-toggle-label` | `h4` | `stream-toggle-label` | Enable SSE |
| `.stream-toggle-buttons` | `div` | `stream-toggle-buttons` | OnOff |
| `.chatbox` | `div` | `chatbox` | Additional bot response |
| `.bot-icon-trigger` | `div` | `bot-icon-trigger` |  |
| `.message-field` | `div` | `message-field` | Type here |
| `` | `label` | `` | Type here |
| `` | `input` | `` |  |

#### Explored Expansion Panels

**Panel "Additional bot response" (0):**
- 1 input(s): Type here
- 1 dropdown(s): Select Conversation Flow
- toggle(s): Enable SSE [**On** | Off]
- 1 card(s)
- headings: Enable SSE
- buttons: v1.0, On, Off
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

---

