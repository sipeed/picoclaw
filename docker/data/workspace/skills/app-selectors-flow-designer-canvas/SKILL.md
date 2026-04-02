---
name: app-selectors-flow-designer-canvas
description: DOM selectors and component map for the Flow Designer Canvas 206 page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

## Drag Connection — Critical Requirements

### 1. Zoom normalization (MANDATORY before EVERY drag)

Vue Flow canvas zoom state persists between operations. After modal interactions the canvas zoom level is unpredictable (may be at 20% minimum). At low zoom, handles are too small to hit reliably.

**Before EVERY drag connection, run this zoom normalization block:**

```typescript
// Zoom in to max first, then zoom out to ~100%
// (zoom-in first handles the case where canvas is already at minimum zoom)
await page.mouse.move(640, 360);
await page.keyboard.down('Control');
for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); } // zoom in to max (400%)
await page.keyboard.up('Control');
await page.waitForTimeout(200);
await page.keyboard.down('Control');
for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); } // zoom out to ~100%
await page.keyboard.up('Control');
await page.waitForTimeout(500);
// NOW read handle boundingBoxes and perform the drag
```

**Why zoom-in THEN zoom-out:** A zoom-out-only approach silently fails when the canvas is already at minimum zoom (20%) — more zoom-out does nothing, leaving handles too small. Zoom-in first guarantees a known maximum starting point, then zoom-out 10× reliably lands at ~100%.

**NEVER skip this block** even for the very first connection.

### 2. Re-read canvas transform after zoom changes (MANDATORY)

The `tf` variable read at test start becomes **stale** as soon as any zoom operation runs. Using a stale transform for node positioning silently moves nodes to wrong screen positions.

**Rule:** Any node positioned **after** the first zoom normalization block MUST re-read the transform immediately before that positioning:

```typescript
// Re-read immediately before positioning — do NOT reuse the original tf
const tfFresh = await page.locator('.vue-flow__transformationpane').evaluate(el => {
  const m = new DOMMatrix((el as HTMLElement).style.transform);
  return { scale: m.a, tx: m.e, ty: m.f };
});
const targetScreenX = 250 * tfFresh.scale + tfFresh.tx;
const targetScreenY = 200 * tfFresh.scale + tfFresh.ty;
```

Name each fresh read descriptively (`tfUU`, `tfReply2`, `tfOutput`, etc.) — never reuse a transform variable from a previous positioning step.

# Flow Designer Canvas 206 — Component Map

> Generated: 2026-03-30T00:13:12.862Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Designer Canvas 206

**URL:** `/flow-designer/206`

**Text Content (4):**

- [p] "Organization" → `.org-title`
- [text-secondary] "Testing"
- [p] "131%"
- [p] "Last saved 2 days ago"

### Flow Name Field (beside the home icon)

The flow name is displayed as a `<p class="text-secondary">` inside `.panel-container`. Clicking it replaces it with an `<input>` (no `type` attribute — do NOT use `input[type="text"]`).

**Rename flow (copy-paste ready):**

```typescript
// Click the flow name text to enter edit mode
await page.locator('.panel-container p.text-secondary').first().click();
await page.waitForTimeout(300);

// Wait for the input to appear, then fill it
const flowNameInput = page.locator('.panel-container input').first();
await flowNameInput.waitFor({ state: 'visible', timeout: 5000 });
await flowNameInput.fill('Your Flow Name');
await flowNameInput.press('Enter');
await page.waitForTimeout(300);
```

> **IMPORTANT:** Use `.panel-container input` — NOT `.panel-container input[type="text"]`. The input element has no `type` attribute and won't be matched by a type filter.

---

### Vue Flow Canvas

Container: `page.locator('.vue-flow')`

**Nodes (12):**

| data-id           | node id           | Type              | Label             | Icon                       | Selector                                            |
| ----------------- | ----------------- | ----------------- | ----------------- | -------------------------- | --------------------------------------------------- |
| START             | START             | Start             | START             | mdi-home                   | `page.locator('.node-container#START')`             |
| END               | END               | End               | END               | mdi-stop                   | `page.locator('.node-container#END')`               |
| UserUtterance     | UserUtterance     | UserUtterance     | UserUtterance     | mdi-account-voice          | `page.locator('.node-container#UserUtterance')`     |
| ReplyMessage      | ReplyMessage      | ReplyMessage      | ReplyMessage      | mdi-message-text-outline   | `page.locator('.node-container#ReplyMessage')`      |
| ReplyMessage\_    | ReplyMessage\_    | ReplyMessage      | ReplyMessage\_    | mdi-message-text-outline   | `page.locator('.node-container#ReplyMessage_')`     |
| CustomNode        | CustomNode        | CustomNode        | CustomNode        | mdi-vector-rectangle       | `page.locator('.node-container#CustomNode')`        |
| CustomTool        | CustomTool        | CustomTool        | CustomTool        | mdi-hammer-wrench          | `page.locator('.node-container#CustomTool')`        |
| RestAPINode       | RestAPINode       | RestAPINode       | RestAPINode       | mdi-api                    | `page.locator('.node-container#RestAPINode')`       |
| HumanAgentNode_9  | HumanAgentNode_9  | HumanAgentNode    | HumanAgentNode_9  | mdi-human-male-board       | `page.locator('.node-container#HumanAgentNode_9')`  |
| KnowledgeBaseNode | KnowledgeBaseNode | KnowledgeBaseNode | KnowledgeBaseNode | mdi-feature-search-outline | `page.locator('.node-container#KnowledgeBaseNode')` |
| Model             | Model             | Model             | Model             | mdi-watermark              | `page.locator('.node-container#Model')`             |
| SubGraph          | SubGraph          | SubGraph          | SubGraph          | mdi-graph-outline          | `page.locator('.node-container#SubGraph')`          |

To click a node (preferred): `await page.locator('.node-container#NODE_ID').click();`
Fallback: `await page.locator('[data-id="NODE_WRAPPER_ID"]').click();`

> **CRITICAL — Newly added nodes get auto-suffixed IDs.**
> The selectors in the table above (e.g. `#ReplyMessage`, `#UserUtterance`) only apply to nodes that already exist in this specific canvas.
> When you **add a new node** via the Add Nodes menu, Vue Flow assigns an auto-generated ID like `ReplyMessage_3`, `UserUtterance_4`, etc.
> **NEVER wait for `.node-container#ReplyMessage` after adding a node — it will never match.**
>
> To locate a newly added node, use `filter({ hasText })` and `.first()` or `.nth()`:
> ```typescript
> // First added node of that type:
> const newNode = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first();
> await newNode.waitFor({ state: 'visible', timeout: 10000 });
>
> // Second added node of that type:
> const secondNode = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).nth(1);
> ```
>
> ⚠️ **CRITICAL — hasText uses the node ID (camelCase), NOT the dropdown display name:**
> | Dropdown shows | hasText pattern to use |
> |---|---|
> | "Reply Message" | `{ hasText: /ReplyMessage/ }` — NO space |
> | "User Utterance" | `{ hasText: /UserUtterance/ }` — NO space |
>
> NEVER use `/Reply Message/` or `/User Utterance/` — these will NEVER match `.node-container`.
>
> After the user sets a custom Node ID (e.g. "input", "Output") and saves, you can then use the exact-ID selector:
> ```typescript
> page.locator('.node-container#input')   // after saving with Node ID = "input"
> page.locator('.node-container#Output')  // after saving with Node ID = "Output"
> ```

**Edges (5):**

| data-id                        | Source         | Target         | Description                               |
| ------------------------------ | -------------- | -------------- | ----------------------------------------- |
| e-START-ReplyMessage\_         | START          | ReplyMessage\_ | Edge from START to ReplyMessage\_         |
| e-UserUtterance-ReplyMessage   | UserUtterance  | ReplyMessage   | Edge from UserUtterance to ReplyMessage   |
| e-ReplyMessage-END             | ReplyMessage   | END            | Edge from ReplyMessage to END             |
| e-ReplyMessage\_-UserUtterance | ReplyMessage\_ | UserUtterance  | Edge from ReplyMessage\_ to UserUtterance |
| e-ReplyMessage\_-CustomNode    | ReplyMessage\_ | CustomNode     | Edge from ReplyMessage\_ to CustomNode    |

To click an edge: `await page.locator('[data-id="EDGE_ID"]').click();`

**Toolbar Panels:**

- **top left**: `icon-button` (.mdi), `mdi-unfold-more-vertical` (.code-editor-button), `mdi-shape-rectangle-plus` (.nodes-button)
- **top right**: `mdi-magnify` (.search-button), `mdi-history` (.history-button), `mdi-timer-sand-empty` (.history-button), `mdi-share-outline` (.history-button), `mdi-image-refresh` (.history-button), `mdi-bug` (.history-button), `mdi-content-save` (.history-button), `mdi-dots-vertical` (.action-button), `Publish` (button)
- **bottom left**: `icon-button` (.mdi), `icon-button` (.mdi), `mdi-undo` (.history-button), `mdi-redo` (.history-button)

### Modal selector strategy

Two modal types exist on this page:

**Type A — custom class** (node config modals opened by clicking a node):
Use `.modal-dialog` as the container — it is unique when open:
```typescript
await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 10000 });
```

**Type B — no custom class** (all other dialogs: save flow, session timer, confirmations, etc.):
These have no unique custom class. Use `.v-overlay--active` scoped to the dialog heading/title text:
```typescript
const dialog = page.locator('.v-overlay--active').filter({ hasText: /Dialog Title/ });
await dialog.waitFor({ state: 'visible', timeout: 10000 });
// Inputs inside this dialog:
const input = dialog.locator('.v-field__input');
// Buttons inside this dialog:
await dialog.locator('button').filter({ hasText: /^Save$/ }).click();
```

> ⚠️ NEVER use `page.locator('v-dialog')` — `v-dialog` is a Vue component name, not a CSS selector. It will never match anything in the DOM.
> For Type B modals, always scope ALL child selectors through the `dialog` locator variable — do not use `page.locator(...)` directly for inputs or buttons inside the dialog.

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

| Selector                         | Tag      | Classes                                                                                                                                              | Text                                                         |
| -------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| `#app`                           | `div`    | ``                                                                                                                                                   | HOrganizationTesting2026!DashboardFlow DesignerFlow TesterKn |
| `.topbar-intent`                 | `header` | `topbar-intent`                                                                                                                                      | H                                                            |
| `.logo-container`                | `div`    | `logo-container`                                                                                                                                     |                                                              |
| `.logo-wrapper`                  | `div`    | `logo-wrapper`                                                                                                                                       |                                                              |
| `.logo-intent`                   | `div`    | `logo-intent`                                                                                                                                        |                                                              |
| `.change-logo-btn`               | `button` | `change-logo-btn`                                                                                                                                    |                                                              |
| `.mdi`                           | `i`      | `mdi notranslate`                                                                                                                                    |                                                              |
| `#menu-activator`                | `div`    | `avatar-container`                                                                                                                                   | H                                                            |
| `.mdi`                           | `i`      | `mdi notranslate avatar-chevron`                                                                                                                     |                                                              |
| `.nav-drawer`                    | `nav`    | `nav-drawer`                                                                                                                                         | OrganizationTesting2026!DashboardFlow DesignerFlow TesterKno |
| `.org-title`                     | `p`      | `org-title`                                                                                                                                          | Organization                                                 |
| `.org-selector-wrapper`          | `div`    | `org-selector-wrapper`                                                                                                                               | Testing2026!                                                 |
| `.org-dropdown`                  | `div`    | `org-dropdown`                                                                                                                                       | Testing2026!                                                 |
| `.org-dropdown-trigger`          | `div`    | `org-dropdown-trigger`                                                                                                                               | Testing2026!                                                 |
| `.org-info`                      | `div`    | `org-info`                                                                                                                                           | Testing2026!                                                 |
| `.org-name`                      | `span`   | `org-name`                                                                                                                                           | Testing2026!                                                 |
| `.dropdown-arrow`                | `div`    | `dropdown-arrow`                                                                                                                                     |                                                              |
| `.[object`                       | `svg`    | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `.[object`                       | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `.mdi`                           | `i`      | `mdi notranslate arrow`                                                                                                                              |                                                              |
| `.menu-item-container`           | `div`    | `menu-item-container`                                                                                                                                | Dashboard                                                    |
| `.mdi`                           | `i`      | `mdi notranslate menu-item-icon`                                                                                                                     |                                                              |
| `.menu-item`                     | `span`   | `menu-item`                                                                                                                                          | Dashboard                                                    |
| `.main-layout-margin-left`       | `main`   | `main-layout-margin-left`                                                                                                                            | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.dnd-flow`                      | `div`    | `dnd-flow`                                                                                                                                           | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow`                      | `div`    | `vue-flow basic-flow`                                                                                                                                | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__viewport`            | `div`    | `vue-flow__viewport vue-flow__container`                                                                                                             | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__pane`                | `div`    | `vue-flow__pane vue-flow__container draggable`                                                                                                       | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.vue-flow__transformationpane`  | `div`    | `vue-flow__transformationpane vue-flow__container`                                                                                                   | ifelseSTARTENDUserUtteranceReplyMessageReplyMessage_CustomNo |
| `.[object`                       | `g`      | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `#e-START-ReplyMessage_`         | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `#e-UserUtterance-ReplyMessage`  | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `#e-ReplyMessage-END`            | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `#e-ReplyMessage_-UserUtterance` | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `.[object`                       | `rect`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `.[object`                       | `text`   | `[object SVGAnimatedString]`                                                                                                                         | if                                                           |
| `#e-ReplyMessage_-CustomNode`    | `path`   | `[object SVGAnimatedString]`                                                                                                                         |                                                              |
| `.vue-flow__edge-labels`         | `div`    | `vue-flow__edge-labels`                                                                                                                              |                                                              |
| `.vue-flow__nodes`               | `div`    | `vue-flow__nodes vue-flow__container`                                                                                                                | STARTENDUserUtteranceReplyMessageReplyMessage_CustomNodeCust |
| `.vue-flow__node`                | `div`    | `vue-flow__node vue-flow__node-Start nopan draggable selectable`                                                                                     | START                                                        |
| `#START`                         | `div`    | `node-container`                                                                                                                                     | START                                                        |
| `.terminal-node`                 | `div`    | `terminal-node`                                                                                                                                      | START                                                        |
| `.terminal-icon`                 | `div`    | `terminal-icon`                                                                                                                                      |                                                              |
| `.vue-flow__handle`              | `div`    | `vue-flow__handle vue-flow__handle-bottom vue-flow__handle-null nodrag nopan source connectable connectablestart connectableend connectionindicator` |                                                              |
| `.vue-flow__node`                | `div`    | `vue-flow__node vue-flow__node-End nopan draggable selectable`                                                                                       | END                                                          |
| `#END`                           | `div`    | `node-container`                                                                                                                                     | END                                                          |
| `.vue-flow__handle`              | `div`    | `vue-flow__handle vue-flow__handle-top vue-flow__handle-null nodrag nopan target connectable connectablestart connectableend connectionindicator`    |                                                              |
| `.vue-flow__node`                | `div`    | `vue-flow__node vue-flow__node-UserUtterance nopan draggable selectable`                                                                             | UserUtterance                                                |
| `#UserUtterance`                 | `div`    | `node-container`                                                                                                                                     | UserUtterance                                                |
| `.default-node`                  | `div`    | `default-node`                                                                                                                                       | UserUtterance                                                |

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

⚠️ CUSTOM COMPONENT — NOT a Vuetify v-select. Items use `.nodes-dropdown-item`, NOT `.v-list-item`.
Pick: `await page.locator('.nodes-dropdown-item').filter({ hasText: /ITEM_TEXT/ }).click()`
After picking, always close: `await page.keyboard.press('Escape'); await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {}); await page.waitForTimeout(300);`

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
- 3 dropdown(s): Node Version \*, Receiver Channel, Content Type
- Vue Flow: 12 node(s), 5 edge(s)
- headings: ReplyMessage
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Node Version \*"**: `Version 2.0.0`, `Version 1.0.0`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Receiver Channel"**: `Receiver Channel Reply`, `None`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Content Type"**: `Text Message`, `Media Message`, `Location Message`, `Button Message`, `List Picker Message`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

**Trigger:** Node: ReplyMessage\_
**Overlay:** `page.locator('.field-container')`
**Wait:** `await page.locator('.field-container').waitFor({ state: 'visible', timeout: 10000 })`
**Classes:** `field-container mb-4`

- Container: `.modal-dialog` (classes: `v-overlay v-overlay--active v-theme--mainTheme v-locale--is-ltr v-dialog modal-dialog v-overlay--scroll-blocked`)
- Title: "ReplyMessage\_"
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Message (`text`)
- Buttons: `Delete`, `Cancel`, `Save`
- 2 input(s): Node ID, Message
- 3 dropdown(s): Node Version \*, Receiver Channel, Content Type
- Vue Flow: 12 node(s), 5 edge(s)
- headings: ReplyMessage\_
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Node Version \*"**: `Version 2.0.0`, `Version 1.0.0`
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
- 3 dropdown(s): Is Tool, Method \*, Arguments
- Vue Flow: 12 node(s), 5 edge(s)
- headings: RestAPINode
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Is Tool"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Method \*"**: `get`, `post`, `put`, `patch`, `delete`
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
- 1 dropdown(s): Node Version \*
- Vue Flow: 12 node(s), 5 edge(s)
- headings: HumanAgentNode_9
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Node Version \*"**: `Version 2.0.0`, `Version 1.0.0`
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
  - The minimum \_score for matching documents. (`text`)
  - Number of chunks returned per matching document. (`text`)
  - Fields to return from the search response. (`text`)
  - Search Tool Description (`text`)
- Buttons: `Add Item`, `Add Item`, `Delete`, `Cancel`, `Save`
- Inputs:
  - Node ID (`text`)
  - Enter value (`text`)
  - Size (`number`)
  - The minimum \_score for matching documents. (`text`)
  - Number of chunks returned per matching document. (`text`)
  - Fields to return from the search response. (`text`)
  - Search Tool Description (`text`)
- Buttons: `Add Item`, `Add Item`, `Delete`, `Cancel`, `Save`
- 7 input(s): Node ID, Enter value, Size, The minimum \_score for matching documents., Number of chunks returned per matching document., Fields to return from the search response., Search Tool Description
- 3 dropdown(s): Is Tool, Source _, Document Search Cluster _
- Vue Flow: 12 node(s), 5 edge(s)
- headings: KnowledgeBaseNode
- buttons: Publish, Add Item, Add Item, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Is Tool"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source \*"**: `Document Search Options`, `Vector Search Options`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Document Search Cluster \*"**: `Picotest2`, `Picotest_3`
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
- 3 dropdown(s): Model Source _, Stream Output, Model _
- Vue Flow: 12 node(s), 5 edge(s)
- 4 Monaco editor(s)
- headings: Model
- buttons: Publish, Expand Editor, Add Item, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Model Source \*"**: `credentials`, `provided_models`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Stream Output"**: `True`, `False`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Model \*"**: `Azure Model Options`, `Bedrock Model Options`, `Vertex Model Options`
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
- Title: "Edge e-START-ReplyMessage\_"
- Buttons: `Delete`, `Cancel`, `Save`
- Buttons: `Delete`, `Cancel`, `Save`
- 3 dropdown(s): Type _, Source _, Target \*
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-START-ReplyMessage\_
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Type \*"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
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
- 3 dropdown(s): Type _, Source _, Target \*
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-UserUtterance-ReplyMessage
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Type \*"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
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
- 3 dropdown(s): Type _, Source _, Target \*
- Vue Flow: 12 node(s), 5 edge(s)
- headings: Edge e-ReplyMessage-END
- buttons: Publish, Delete, Cancel, Save
- custom: .enable-motion, .topbar-intent, .logo-container, .logo-wrapper, .logo-intent, .change-logo-btn, .mdi, #menu-activator, .mdi, .nav-drawer

**Dropdowns in modal:**

- **"Type \*"**: `Default`, `If`, `Else`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Source \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`
- **"Target \*"**: `START`, `END`, `UserUtterance`, `ReplyMessage`, `ReplyMessage_`, `CustomNode`, `CustomTool`, `RestAPINode`, `HumanAgentNode_9`, `KnowledgeBaseNode`, `Model`, `SubGraph`
  Open: `page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2).click()`
  Pick: `await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()`

---
