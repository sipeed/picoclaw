---
name: app-selectors-flow-tester
description: DOM selectors and component map for the Flow Tester page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Flow Tester — Component Map

> Generated: 2026-03-27T09:45:14.390Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Tester
**URL:** `/flow-tester`

**Headings:**
- `h4` — "Enable SSE" (selector: `.stream-toggle-label`)


**Text Content (3):**
- [p] "Organization" → `.org-title`
- [v-card-title] "Select Conversation FlowTestingSelect VersionEdited else edgeEnable SSEOnOff" → `.chatbox-title`
- [v-card-text] "Additional bot response"

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Type here | `text` | `.message-field input` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Dropdowns / Selects (1):**
- **Select Conversation Flow** (select) — current: "Testing"
  - Selector: `.tester-select`
  - Open: `page.locator('.tester-select').click()`
  - Options: `Testing`, `Untitled`, `Untitled`, `Untitled`
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
- **Select Conversation FlowTestingSelect VersionEdited else edgeEnable SSEOnOff** → `.tester-container-card`
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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Select Conversation FlowTestingSelect VersionEdited else edg |
| `.tester-container` | `div` | `tester-container` | Select Conversation FlowTestingSelect VersionEdited else edg |
| `.tester-container-card` | `div` | `tester-container-card` | Select Conversation FlowTestingSelect VersionEdited else edg |
| `.chatbox-title` | `div` | `chatbox-title` | Select Conversation FlowTestingSelect VersionEdited else edg |
| `.tester-select` | `div` | `tester-select` | Testing |
| `.version-selector-container` | `div` | `version-selector-container` | Edited else edge |
| `.version-selector-button` | `button` | `version-selector-button` | Edited else edge |
| `.version-selector-text` | `span` | `version-selector-text` | Edited else edge |
| `.stream-toggle` | `div` | `stream-toggle` | Enable SSEOnOff |
| `.stream-toggle-label` | `h4` | `stream-toggle-label` | Enable SSE |
| `.stream-toggle-buttons` | `div` | `stream-toggle-buttons` | OnOff |
| `.chatbox` | `div` | `chatbox` | Additional bot response |
| `.bot-icon-trigger` | `div` | `bot-icon-trigger` |  |
| `.message-field` | `div` | `message-field` | Type here |
| `` | `label` | `` | Type here |
| `` | `input` | `` |  |

---

## Playwright Usage — Verified Patterns (from source code)

### Version Selection
The app AUTO-SELECTS the first version when a flow finishes loading (Vue watcher).
DO NOT click the version dropdown and try to pick an item — during loading, skeleton `.version-item` elements
have no click handler and clicking them silently does nothing.

CORRECT — wait for auto-selection:
```typescript
await expect(page.locator('.version-selector-text'))
  .not.toContainText('Select Version', { timeout: 15000 });
```

Only open the dropdown manually if you specifically need to pick a non-default version:
```typescript
await page.locator('.version-selector-button').click();
await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
// Wait for real items (not skeleton) by waiting for .version-date to appear
await page.locator('.version-dropdown-menu .version-date').first().waitFor({ state: 'visible', timeout: 10000 });
await page.locator('.version-dropdown-menu .version-item').first().click();
```

### Bot Message Rendering — isOutput determines which element
Bot messages render in one of two ways depending on the `isOutput` flag from the API:

| isOutput | Rendered as | Selector |
|----------|-------------|----------|
| `true`   | `.message-card` with `.message-text` inside (visible in chat) | `.chatbox .message-text` |
| `false`  | Collapsed `v-expansion-panel` labelled "Additional bot response" | `.chatbox .v-expansion-panel` |

The **final output** of a flow (last Reply Message in the chain) has `isOutput: true` → `.message-text`.
**Intermediate** messages (e.g. prompts/instructions mid-flow) have `isOutput: false` → expansion panel.

CORRECT — verify final bot output:
```typescript
await expect(page.locator('.chatbox .message-text').last())
  .toContainText('expected text', { timeout: 15000 });
```

CORRECT — verify an intermediate bot message (expansion panel):
```typescript
await expect(page.locator('.chatbox .v-expansion-panel')).toBeVisible({ timeout: 15000 });
```

### User Messages
```typescript
await expect(page.locator('.chatbox .message-card-user .message-text').last())
  .toContainText('expected text', { timeout: 5000 });
```

---

#### Discovered Modals / Dialogs

**Trigger:** `page.locator('button:has-text("Edited else edge")').click()`
**Overlay:** `page.locator('.version-dropdown-menu')`
**Wait:** `await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `version-dropdown-menu`

- 1 input(s): Type here
- 2 dropdown(s): Select Conversation Flow, ?
- toggle(s): Enable SSE [**On** | Off]
- 1 card(s)
- headings: Select Version, Enable SSE
- buttons: Edited else edge, On, Off
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Dropdowns in modal:**
- **"Select Conversation Flow"**: `Testing`, `Untitled`, `Untitled`, `Untitled`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

#### Explored Expansion Panels

**Panel "Additional bot response" (0):**
- 1 input(s): Type here
- 1 dropdown(s): Select Conversation Flow
- toggle(s): Enable SSE [**On** | Off]
- 1 card(s)
- headings: Enable SSE
- buttons: Edited else edge, On, Off
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

---

