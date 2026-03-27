---
name: app-selectors-flow-designer-canvas
description: DOM selectors and component map for the Flow Designer Canvas page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Flow Designer Canvas — Component Map

> Generated: 2026-03-27T09:40:14.788Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Designer Canvas
**URL:** `/flow-designer/206`


**Text Content (4):**
- [p] "Organization" → `.org-title`
- [text-secondary] "Testing"
- [p] "131%"
- [p] "Last saved 1 hour ago"

### Vue Flow Canvas

Container: `page.locator('.vue-flow')`

**Nodes (12):**

| data-id | Type | Label | Icon | Selector |
|---------|------|-------|------|----------|
| START | Start | START | mdi-home | `page.locator('[data-id="START"]')` |
| END | End | END | mdi-stop | `page.locator('[data-id="END"]')` |
| UserUtterance | UserUtterance | UserUtterance | mdi-account-voice | `page.locator('[data-id="UserUtterance"]')` |
| ReplyMessage | ReplyMessage | ReplyMessage | mdi-message-text-outline | `page.locator('[data-id="ReplyMessage"]')` |
| ReplyMessage_ | ReplyMessage | ReplyMessage_ | mdi-message-text-outline | `page.locator('[data-id="ReplyMessage_"]')` |
| CustomNode | CustomNode | CustomNode | mdi-vector-rectangle | `page.locator('[data-id="CustomNode"]')` |
| CustomTool | CustomTool | CustomTool | mdi-hammer-wrench | `page.locator('[data-id="CustomTool"]')` |
| RestAPINode | RestAPINode | RestAPINode | mdi-api | `page.locator('[data-id="RestAPINode"]')` |
| HumanAgentNode_9 | HumanAgentNode | HumanAgentNode_9 | mdi-human-male-board | `page.locator('[data-id="HumanAgentNode_9"]')` |
| KnowledgeBaseNode | KnowledgeBaseNode | KnowledgeBaseNode | mdi-feature-search-outline | `page.locator('[data-id="KnowledgeBaseNode"]')` |
| Model | Model | Model | mdi-watermark | `page.locator('[data-id="Model"]')` |
| SubGraph | SubGraph | SubGraph | mdi-graph-outline | `page.locator('[data-id="SubGraph"]')` |

To click a node: `await page.locator('[data-id="NODE_ID"]').click();`

**Edges (5):**

| data-id | Source | Target | Description |
|---------|--------|--------|-------------|
| e-START-ReplyMessage_ | START | ReplyMessage_ | Edge from START to ReplyMessage_ |
| e-UserUtterance-ReplyMessage | UserUtterance | ReplyMessage | Edge from UserUtterance to ReplyMessage |
| e-ReplyMessage-END | ReplyMessage | END | Edge from ReplyMessage to END |
| e-ReplyMessage_-UserUtterance | ReplyMessage_ | UserUtterance | Edge from ReplyMessage_ to UserUtterance |
| e-ReplyMessage_-CustomNode | ReplyMessage_ | CustomNode | Edge from ReplyMessage_ to CustomNode |

To click an edge: `await page.locator('[data-id="EDGE_ID"]').click();`

**Toolbar Panels:**

- **top left**: `icon-button` (.mdi), `mdi-unfold-more-vertical` (.code-editor-button), `mdi-shape-rectangle-plus` (.nodes-button)
- **top right**: `mdi-magnify` (.search-button), `mdi-history` (.history-button), `mdi-timer-sand-empty` (.history-button), `mdi-share-outline` (.history-button), `mdi-image-refresh` (.history-button), `mdi-bug` (.history-button), `mdi-content-save` (.history-button), `mdi-dots-vertical` (.action-button), `Publish` (button)
- **bottom left**: `icon-button` (.mdi), `icon-button` (.mdi), `mdi-undo` (.history-button), `mdi-redo` (.history-button)

**Buttons (1):**
- `page.locator('button:has-text("Publish")')`

**Icon Buttons (13):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- mdi-unfold-more-vertical (`mdi-unfold-more-vertical`) → `.code-editor-button`
- mdi-shape-rectangle-plus (`mdi-shape-rectangle-plus`) → `.nodes-button`
- mdi-magnify (`mdi-magnify`) → `.search-button`
- mdi-history (`mdi-history`) → `.history-button`
- mdi-timer-sand-empty (`mdi-timer-sand-empty`) → `.history-button`
- mdi-share-outline (`mdi-share-outline`) → `.history-button`
- mdi-image-refresh (`mdi-image-refresh`) → `.history-button`
- mdi-bug (`mdi-bug`) → `.history-button`
- mdi-content-save (`mdi-content-save`) → `.history-button`
- mdi-dots-vertical (`mdi-dots-vertical`) → `.action-button`
- mdi-undo (`mdi-undo`) → `.history-button`
- mdi-redo (`mdi-redo`) → `.history-button`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')` ★
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (82):**

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
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.dnd-flow` | `div` | `dnd-flow` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow` | `div` | `vue-flow basic-flow` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__viewport` | `div` | `vue-flow__viewport vue-flow__container` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__pane` | `div` | `vue-flow__pane vue-flow__container draggable` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__transformationpane` | `div` | `vue-flow__transformationpane vue-flow__container` | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.[object` | `g` | `[object SVGAnimatedString]` |  |
| `#e-START-ReplyMessage_` | `path` | `[object SVGAnimatedString]` |  |
| `#e-UserUtterance-ReplyMessage` | `path` | `[object SVGAnimatedString]` |  |
| `#e-ReplyMessage-END` | `path` | `[object SVGAnimatedString]` |  |
| `#e-ReplyMessage_-UserUtterance` | `path` | `[object SVGAnimatedString]` |  |
| `.[object` | `rect` | `[object SVGAnimatedString]` |  |
| `.[object` | `text` | `[object SVGAnimatedString]` | if |
| `#e-ReplyMessage_-CustomNode` | `path` | `[object SVGAnimatedString]` |  |
| `.vue-flow__edge-labels` | `div` | `vue-flow__edge-labels` |  |
| `.vue-flow__nodes` | `div` | `vue-flow__nodes vue-flow__container` | STARTENDUserUtteranceReplyMessageReplyMessage_CustomNodeCust |
| `.vue-flow__node` | `div` | `vue-flow__node vue-flow__node-Start nopan draggable selectable` | START |
| `#START` | `div` | `node-container` | START |
| `.terminal-node` | `div` | `terminal-node` | START |
| `.terminal-icon` | `div` | `terminal-icon` |  |
| `.vue-flow__handle` | `div` | `vue-flow__handle vue-flow__handle-bottom vue-flow__handle-null nodrag nopan source connectable connectablestart connectableend connectionindicator` |  |
| `.vue-flow__node` | `div` | `vue-flow__node vue-flow__node-End nopan draggable selectable` | END |
| `#END` | `div` | `node-container` | END |
| `.vue-flow__handle` | `div` | `vue-flow__handle vue-flow__handle-top vue-flow__handle-null nodrag nopan target connectable connectablestart connectableend connectionindicator` |  |
| `.vue-flow__node` | `div` | `vue-flow__node vue-flow__node-UserUtterance nopan draggable selectable` | UserUtterance |
| `#UserUtterance` | `div` | `node-container` | UserUtterance |
| `.default-node` | `div` | `default-node` | UserUtterance |

#### Discovered Modals / Dialogs

**Trigger:** Panel: mdi
**Overlay:** `page.locator('.dropzone')`
**Wait:** `await page.locator('.dropzone').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `dropzone`

- Buttons: `Upload Logo`
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Change Logo
- buttons: Publish, Upload Logo
- custom: .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer, .org-title

**Trigger:** Panel: mdi-unfold-more-vertical
**Overlay:** `page.locator('.lines-content')`
**Wait:** `await page.locator('.lines-content').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `lines-content monaco-editor-background`

- 1 textarea(s)
- Vue Flow: 12 node(s), 5 edge(s)
- 2 Monaco editor(s)
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Panel: mdi-shape-rectangle-plus
**Overlay:** `page.locator('.nodes-dropdown-menu')`
**Wait:** `await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `nodes-dropdown-menu`
**Menu items:** `User Utterance`, `Custom Node`, `Custom Tool`, `Rest API Node`, `Human Agent Node`, `Knowledge Base Node`, `Model`, `Reply Message`, `Sub Graph`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /ITEM_TEXT/ }).click()`

- Container: `.nodes-dropdown-menu` (classes: `nodes-dropdown-menu`)
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Add Nodes
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Panel: mdi-magnify
**Overlay:** `page.locator('.search-dropdown-menu')`
**Wait:** `await page.locator('.search-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `search-dropdown-menu`

- Container: `.search-dropdown-menu` (classes: `search-dropdown-menu`)
- Inputs:
  - Search (`text`)
- 1 input(s): Search
- Vue Flow: 12 node(s), 5 edge(s)
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Panel: mdi-history
**Overlay:** `page.locator('.history-dropdown-menu')`
**Wait:** `await page.locator('.history-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `history-dropdown-menu`

- Container: `.history-dropdown-menu` (classes: `history-dropdown-menu`)
- 1 dropdown(s): ?
- Vue Flow: 12 node(s), 5 edge(s)
- headings: History
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Dropdown 1"**: `All`, `Autosave`, `Manual Save`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Panel: mdi-timer-sand-empty
**Overlay:** `page.locator('.history-dropdown-menu')`
**Wait:** `await page.locator('.history-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `history-dropdown-menu`

- Container: `.history-dropdown-menu` (classes: `history-dropdown-menu`)
- 1 dropdown(s): ?
- Vue Flow: 12 node(s), 5 edge(s)
- headings: History
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Dropdown 1"**: `All`, `Autosave`, `Manual Save`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Panel: mdi-share-outline
**Overlay:** `page.locator('.history-dropdown-menu')`
**Wait:** `await page.locator('.history-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `history-dropdown-menu`

- Container: `.history-dropdown-menu` (classes: `history-dropdown-menu`)
- 1 dropdown(s): ?
- Vue Flow: 12 node(s), 5 edge(s)
- headings: History
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Dropdown 1"**: `All`, `Autosave`, `Manual Save`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Panel: mdi-image-refresh
**Overlay:** `page.locator('.history-dropdown-menu')`
**Wait:** `await page.locator('.history-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `history-dropdown-menu`

- Container: `.history-dropdown-menu` (classes: `history-dropdown-menu`)
- 1 dropdown(s): ?
- Vue Flow: 12 node(s), 5 edge(s)
- headings: History
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Dropdown 1"**: `All`, `Autosave`, `Manual Save`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Panel: mdi-bug
**Overlay:** `page.locator('.history-dropdown-menu')`
**Wait:** `await page.locator('.history-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `history-dropdown-menu`

- Container: `.history-dropdown-menu` (classes: `history-dropdown-menu`)
- 1 dropdown(s): ?
- Vue Flow: 12 node(s), 5 edge(s)
- headings: History
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Dropdown 1"**: `All`, `Autosave`, `Manual Save`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Panel: mdi-dots-vertical
**Overlay:** `page.locator('.menu-action-dropdown-menu')`
**Wait:** `await page.locator('.menu-action-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `menu-action-dropdown-menu`
**Menu items:** `Delete`, `Duplicate`, `Copy Link`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /ITEM_TEXT/ }).click()`

- Container: `.menu-action-dropdown-menu` (classes: `menu-action-dropdown-menu`)
- Vue Flow: 12 node(s), 5 edge(s)
- buttons: Publish
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Panel: mdi
**Overlay:** `page.locator('.dropzone')`
**Wait:** `await page.locator('.dropzone').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `dropzone`

- Buttons: `Upload Logo`
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Change Logo
- buttons: Publish, Upload Logo
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Panel: mdi
**Overlay:** `page.locator('.dropzone')`
**Wait:** `await page.locator('.dropzone').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `dropzone`

- Buttons: `Upload Logo`
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Change Logo
- buttons: Publish, Upload Logo
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Node: UserUtterance
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "UserUtterance"
- Inputs:
  - Node ID (`text`)
  - State Variable (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - State Variable (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 2 input(s): Node ID, State Variable
- Vue Flow: 12 node(s), 5 edge(s)
- headings: UserUtterance
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Node: ReplyMessage
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "ReplyMessage"
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 2 input(s): Node ID, Message
- 3 dropdown(s): Node Version *, Receiver Channel, Content Type
- Vue Flow: 12 node(s), 5 edge(s)
- headings: ReplyMessage
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Node Version *"**: `Version 2.0.0`, `Version 1.0.0`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Receiver Channel"**: `Receiver Channel Reply`, `None`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Content Type"**: `Text Message`, `Media Message`, `Location Message`, `Button Message`, `List Picker Message`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: ReplyMessage_
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "ReplyMessage_"
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 2 input(s): Node ID, Message
- 3 dropdown(s): Node Version *, Receiver Channel, Content Type
- Vue Flow: 12 node(s), 5 edge(s)
- headings: ReplyMessage_
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Node Version *"**: `Version 2.0.0`, `Version 1.0.0`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Receiver Channel"**: `Receiver Channel Reply`, `None`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Content Type"**: `Text Message`, `Media Message`, `Location Message`, `Button Message`, `List Picker Message`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: CustomNode
**Overlay:** `page.locator('.lines-content')`
**Wait:** `await page.locator('.lines-content').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `lines-content monaco-editor-background`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "CustomNode"
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 1 input(s): Node ID
- 1 textarea(s)
- Vue Flow: 12 node(s), 5 edge(s)
- 2 Monaco editor(s)
- headings: CustomNode
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** Node: CustomTool
**Overlay:** `page.locator('.lines-content')`
**Wait:** `await page.locator('.lines-content').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `lines-content monaco-editor-background`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "CustomTool"
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 1 input(s): Node ID
- 1 textarea(s)
- 1 dropdown(s): Is Tool
- Vue Flow: 12 node(s), 5 edge(s)
- 2 Monaco editor(s)
- headings: CustomTool
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Is Tool"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: RestAPINode
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "RestAPINode"
- Inputs:
  - Node ID (`text`)
  - Base Url (`text`)
  - Tool Description (`text`)
  - Timeout (`number`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Base Url (`text`)
  - Tool Description (`text`)
  - Timeout (`number`)
- Buttons: `Delete`, `Cancel`, `Save`
- 4 input(s): Node ID, Base Url, Tool Description, Timeout
- 4 textarea(s)
- 3 dropdown(s): Is Tool, Method *, Arguments
- Vue Flow: 12 node(s), 5 edge(s)
- headings: RestAPINode
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Is Tool"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Method *"**: `get`, `post`, `put`, `patch`, `delete`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Arguments"**: `API Request Arguments`, `None`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: HumanAgentNode_9
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "HumanAgentNode_9"
- Inputs:
  - Node ID (`text`)
  - Channel Id (`number`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Channel Id (`number`)
- Buttons: `Delete`, `Cancel`, `Save`
- 2 input(s): Node ID, Channel Id
- 1 dropdown(s): Node Version *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: HumanAgentNode_9
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Node Version *"**: `Version 2.0.0`, `Version 1.0.0`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: KnowledgeBaseNode
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "KnowledgeBaseNode"
- Inputs:
  - Node ID (`text`)
  - Enter value (`text`)
  - Size (`number`)
  - The minimum _score for matching documents. (`text`)
  - Number of chunks returned per matching document. (`text`)
  - Fields to return from the search response. (`text`)
  - Search Tool Description (`text`)
- Buttons: `Add Item`, `Add Item`, `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Enter value (`text`)
  - Size (`number`)
  - The minimum _score for matching documents. (`text`)
  - Number of chunks returned per matching document. (`text`)
  - Fields to return from the search response. (`text`)
  - Search Tool Description (`text`)
- Buttons: `Add Item`, `Add Item`, `Delete`, `Cancel`, `Save`
- 7 input(s): Node ID, Enter value, Size, The minimum _score for matching documents., Number of chunks returned per matching document., Fields to return from the search response., Search Tool Description
- 3 dropdown(s): Is Tool, Source *, Document Search Cluster *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: KnowledgeBaseNode
- buttons: Publish, Add Item, Add Item, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Is Tool"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `Document Search Options`, `Vector Search Options`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Document Search Cluster *"**: `Picotest2`, `Picotest_3`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: Model
**Overlay:** `page.locator('.lines-content')`
**Wait:** `await page.locator('.lines-content').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `lines-content monaco-editor-background`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Model"
- Inputs:
  - Node ID (`text`)
  - Temperature (`number`)
  - Openai Api Key (`text`)
  - Azure Endpoint (`text`)
  - Deployment Name (`text`)
  - Openai Api Version (`text`)
- Buttons: `Expand Editor`, `Add Item`, `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Temperature (`number`)
  - Openai Api Key (`text`)
  - Azure Endpoint (`text`)
  - Deployment Name (`text`)
  - Openai Api Version (`text`)
- Buttons: `Expand Editor`, `Add Item`, `Delete`, `Cancel`, `Save`
- 6 input(s): Node ID, Temperature, Openai Api Key, Azure Endpoint, Deployment Name, Openai Api Version
- 3 textarea(s)
- 3 dropdown(s): Model Source *, Stream Output, Model *
- Vue Flow: 12 node(s), 5 edge(s)
- 4 Monaco editor(s)
- headings: Model
- buttons: Publish, Expand Editor, Add Item, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Model Source *"**: `credentials`, `provided_models`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Stream Output"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Model *"**: `Azure Model Options`, `Bedrock Model Options`, `Vertex Model Options`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: SubGraph
**Overlay:** `page.locator('.lines-content')`
**Wait:** `await page.locator('.lines-content').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `lines-content monaco-editor-background`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "SubGraph"
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 1 input(s): Node ID
- 3 textarea(s)
- Vue Flow: 12 node(s), 5 edge(s)
- 6 Monaco editor(s)
- headings: SubGraph
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Trigger:** `page.locator('button:has-text("Edge: Edge from START to ReplyMessage_")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Edge e-START-ReplyMessage_"
- Buttons: `Delete`, `Cancel`, `Save`
- Buttons: `Delete`, `Cancel`, `Save`
- 3 dropdown(s): Type *, Source *, Target *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-START-ReplyMessage_
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Type *"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** `page.locator('button:has-text("Edge: Edge from UserUtterance to ReplyMessage")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Edge e-UserUtterance-ReplyMessage"
- Buttons: `Delete`, `Cancel`, `Save`
- Buttons: `Delete`, `Cancel`, `Save`
- 3 dropdown(s): Type *, Source *, Target *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-UserUtterance-ReplyMessage
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Type *"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** `page.locator('button:has-text("Edge: Edge from ReplyMessage to END")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Edge e-ReplyMessage-END"
- Buttons: `Delete`, `Cancel`, `Save`
- Buttons: `Delete`, `Cancel`, `Save`
- 3 dropdown(s): Type *, Source *, Target *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-ReplyMessage-END
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Type *"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** `page.locator('button:has-text("Edge: Edge from ReplyMessage_ to UserUtterance")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Edge e-ReplyMessage_-UserUtterance"
- Inputs:
  - Expression (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Expression (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 1 input(s): Expression
- 3 dropdown(s): Type *, Source *, Target *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-ReplyMessage_-UserUtterance
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Type *"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** `page.locator('button:has-text("Edge: Edge from ReplyMessage_ to CustomNode")').click()`
**Overlay:** `page.locator('.close-btn')`
**Wait:** `await page.locator('.close-btn').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `v-btn v-btn--icon v-theme--mainTheme v-btn--density-default v-btn--size-small v-btn--variant-flat close-btn`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "Edge e-ReplyMessage_-CustomNode"
- Buttons: `Delete`, `Cancel`, `Save`
- Buttons: `Delete`, `Cancel`, `Save`
- 3 dropdown(s): Type *, Source *, Target *
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-ReplyMessage_-CustomNode
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**
- **"Type *"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target *"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---

