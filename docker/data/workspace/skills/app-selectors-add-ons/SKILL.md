---
name: app-selectors-add-ons
description: DOM selectors and component map for the Add-Ons page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Add-Ons — Component Map

> Generated: 2026-03-27T09:44:24.211Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Add-Ons
**URL:** `/add-ons`

**Headings:**
- `h2` — "Add-Ons"
- `h4` — "Bird Messaging Channel" (selector: `.addon-name`)
- `h4` — "Telegram Bot" (selector: `.addon-name`)
- `h4` — "Twilio Messaging Service" (selector: `.addon-name`)
- `h4` — "Chatwoot CRM" (selector: `.addon-name`)
- `h4` — "AWS SES Email Service" (selector: `.addon-name`)
- `h4` — "Microsoft Graph Email Service" (selector: `.addon-name`)
- `h4` — "Webchat Widget" (selector: `.addon-name`)


**Text Content (10):**
- [p] "Organization" → `.org-title`
- [p] "Make the most of your work by adding the channels and models that will save up time and boost your workflow."
- [p] "IntentAI" → `.addon-author`
- [p] "Send messages through Bird." → `.addon-description`
- [p] "Send messages through Telegram." → `.addon-description`
- [p] "Send messages through Twilio." → `.addon-description`
- [p] "Manage customer interactions with Chatwoot." → `.addon-description`
- [p] "Send emails through AWS SES." → `.addon-description`
- [p] "Send and receive email via Microsoft Graph (Outlook)." → `.addon-description`
- [p] "Add a customizable webchat widget to your website for seamless customer interactions" → `.addon-description`

**Input Fields (1):**

| # | Label | Type | Selector |
|---|-------|------|----------|
| 1 | Search | `text` | `.search-input input` |

**Input selector rule:** Use `input[placeholder="..."]` or `.nth(N)` on scoped container inputs. Do NOT use `.filter({ hasText })` on a `div` to match placeholder text — placeholders are attributes, not visible text content.

**Buttons (9):**
- `page.locator('.dropdown-button')`
  classes: `dropdown-button`
- `page.locator('.dropdown-button')`
  classes: `dropdown-button`
- `page.locator('.install-button')`
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`
- `page.locator('.install-button')` *(dup)*
  classes: `install-button`

**Icon Buttons (1):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')` ★
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (52):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | Add-OnsAdd-OnsMake the most of your work by adding the chann |
| `.breadcrumb` | `nav` | `breadcrumb` | Add-Ons |
| `.breadcrumb-item` | `span` | `breadcrumb-item` | Add-Ons |
| `.search-input` | `div` | `search-input` |  |
| `` | `input` | `` |  |
| `.dropdown-button` | `button` | `dropdown-button` | Sort:Default |
| `.dropdown-text` | `span` | `dropdown-text` | Sort:Default |
| `.dropdown-key` | `span` | `dropdown-key` | Sort: |
| `.dropdown-value` | `span` | `dropdown-value` | Default |
| `.tabs` | `div` | `tabs` | AllChannelsWidgets |
| `.tab-header` | `div` | `tab-header` | AllChannelsWidgets |
| `.tab` | `div` | `tab active` | All |
| `.tab` | `div` | `tab` | Channels |
| `.addons-content` | `div` | `addons-content` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.section` | `div` | `section` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.addons-grid` | `div` | `addons-grid` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.addon-card` | `div` | `addon-card` | Bird Messaging ChannelIntentAISend messages through Bird.Cha |
| `.card-header` | `div` | `card-header` | Bird Messaging ChannelIntentAI |
| `.addon-info` | `div` | `addon-info` | Bird Messaging ChannelIntentAI |
| `.addon-icon` | `div` | `addon-icon` |  |
| `.addon-icon` | `img` | `addon-icon` |  |
| `.addon-details` | `div` | `addon-details` | Bird Messaging ChannelIntentAI |
| `.addon-name` | `h4` | `addon-name` | Bird Messaging Channel |
| `.addon-author` | `p` | `addon-author` | IntentAI |
| `.addon-description` | `p` | `addon-description` | Send messages through Bird. |
| `.card-footer` | `div` | `card-footer` | ChannelInstall |
| `.category-tag` | `span` | `category-tag channel` | Channel |

#### Discovered Modals / Dialogs

**Trigger:** `page.locator('button:has-text("Sort:Default")').click()`
**Overlay:** `page.locator('.dropdown-menu')`
**Wait:** `await page.locator('.dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-list v-theme--mainTheme v-list--density-default v-list--one-line dropdown-menu`

- Container: `.dropdown-menu` (classes: `v-list v-theme--mainTheme v-list--density-default v-list--one-line dropdown-menu`)
- 1 input(s): Search
- headings: Add-Ons, Bird Messaging Channel, Telegram Bot, Twilio Messaging Service, Chatwoot CRM, AWS SES Email Service, Microsoft Graph Email Service, Webchat Widget
- buttons: Sort:Default, Category:All, Install, Install, Install, Install, Install, Install, Install
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

---

