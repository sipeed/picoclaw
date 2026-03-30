# Flow Designer — Complete Reference for Playwright Tests

> Generated: 2026-03-30
> Based on comprehensive source code analysis of all Flow Designer components, pages, and stores

---

## 0. Files Read (Phase 1 — Source Code Discovery)

### Directory: src/components/flow-designer/
1. `FlowBoard.vue` — Main VueFlow canvas wrapper, node/edge click handlers, modal state management
2. `NodeConfigurationModal.vue` — Node config form with dynamic fields from schema, version selection, array field controller
3. `SaveFlowModal.vue` — Save Flow Version dialog (version name input, save button)
4. `EdgeConfigurationModal.vue` — Edge config form (type, source, target, expression for IF edges)
5. `DropzoneBackground.vue` — Drag-over visual feedback component (not directly tested)
6. `ErrorMessageModal.vue` — Flow configuration modal (not directly tested)
7. `FlowJsonEditor.vue` — Monaco editor for flow JSON (not directly tested)
8. `FlowRecorderChannelModal.vue` — Flow configuration modal (not directly tested)
9. `RetryOnImageFailureModal.vue` — Flow configuration modal (not directly tested)
10. `SessionTimerModal.vue` — Flow configuration modal (not directly tested)

### Directory: src/components/flow-designer/dropdown/
11. `NodesDropdown.vue` — Add Nodes menu (.nodes-dropdown-menu, .nodes-dropdown-item, custom component NOT Vuetify)
12. `HistoryDropdown.vue` — Flow version history panel (not directly tested)
13. `ActionMenuDropdown.vue` — Flow actions menu (Delete, Duplicate, Copy Link)
14. `SearchDropdown.vue` — Node search panel (not directly tested)
15. `AIModal.vue` — AI assistant modal (not directly tested)

### Directory: src/components/flow-designer/nodes/
16. `DefaultNode.vue` — Generic node component (uses Handle from @vue-flow/core)
17. `TerminalNode.vue` — START/END terminal nodes (uses Handle from @vue-flow/core)

### Pages
18. `src/pages/FlowDesigner.vue` — All Flows list page (Add New button, flow table)

### Stores
19. `src/stores/flow.store.ts` — Pinia store with node/edge manipulation, flow version management, metadata

---

## 1. Node Types

### Node IDs and Selectors

**Terminal Nodes (START / END):**
- START node: `.node-container#START` (type: 'Start')
- END node: `.node-container#END` (type: 'End')

**Dynamic Nodes (auto-suffixed IDs after creation):**
- When added via NodesDropdown: `ReplyMessage_3`, `UserUtterance_4`, etc.
- Selector after creation: `.node-container` with `.filter({ hasText: /NodeLabel/ })`
- After saving with custom Node ID (e.g., "input"): `.node-container#input`

**Available Node Types (from nodesMetadata):**
- `User Utterance` (dropdown display) → type: `UserUtterance` (camelCase, NO space)
- `Reply Message` (dropdown display) → type: `ReplyMessage` (camelCase, NO space)
- `Custom Node`, `Custom Tool`, `Rest API Node`, `Human Agent Node`, `Knowledge Base Node`, `Model`, `Sub Graph`

### Connection Handles

**CRITICAL: Handle Scope Rule**

Handles are siblings of `.node-container` inside `.vue-flow__node` wrapper. NEVER access handles directly via `.node-container`.

**Correct pattern:**
```typescript
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#SOURCE_ID') })
  .locator('.vue-flow__handle-bottom');

const targetHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#TARGET_ID') })
  .locator('.vue-flow__handle-top');
```

**Handle positions:**
- Source handle (outgoing): `.vue-flow__handle-bottom`
- Target handle (incoming): `.vue-flow__handle-top`

**For newly added nodes (not yet saved with custom ID):**
```typescript
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
  .first()
  .locator('.vue-flow__handle-bottom');
```

### How to Drag-Connect Nodes

**Step 1: Get source and target handle bounding boxes**
```typescript
const sourceBox = await sourceHandle.boundingBox();
const targetBox = await targetHandle.boundingBox();
if (!sourceBox || !targetBox) throw new Error('Handle not found');
```

**Step 2: Hover on source handle briefly**
```typescript
await page.mouse.move(sourceBox.x + sourceBox.width / 2, sourceBox.y + sourceBox.height / 2);
await page.waitForTimeout(200); // CRITICAL: Vue Flow needs hover state to detect drag
```

**Step 3: Drag with intermediate steps**
```typescript
await page.mouse.down();
await page.mouse.move(tgtBox.x + tgtBox.width / 2, tgtBox.y + tgtBox.height / 2, { steps: 20 });
await page.waitForTimeout(200); // CRITICAL: Dwell on target for mouseenter detection
await page.mouse.up();
await page.waitForTimeout(1000); // CRITICAL: 1000ms for Vue Flow to persist edge
```

**Step 4: Verify edge was created**
```typescript
// ONLY use data-id pattern — NEVER use .vue-flow__edge (ghost edge false positives)
await page.locator('[data-id^="e-SOURCE_ID-"]').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 2. Node Config Modals

### Modal Detection

**Container selector:** `.modal-dialog` (custom class on v-dialog)

**Wait for modal to open:**
```typescript
await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 10000 });
```

**Wait for loading to finish (if any):**
```typescript
await page.locator('.v-overlay--active').locator('text=/loading/i')
  .waitFor({ state: 'hidden', timeout: 10000 }).catch(() => {});
```

### Field Selectors Inside Modal

**Node ID field:**
- Selector: `.modal-dialog .field-container` → filter by label "Node ID" → `.v-field__input`
- ALWAYS use `.fill()` to change the value (no triple-click needed)
- ALWAYS press Tab after filling to trigger validation

**Message field (for Reply Message node):**
- Selector: `.modal-dialog .field-container` → filter by label "Message" → `.v-field__input`
- Pattern: `click()` → `fill()` → `press('Tab')`

**Dropdown fields (Node Version, Receiver Channel, Content Type):**
- These are Vuetify v-select components
- Open: `.v-select:visible` → `.nth(N)` → `.click()`
- Pick option: `.v-overlay--active .v-list-item` → `.filter({ hasText: /OPTION/ })` → `.click()`
- Example for Node Version (first dropdown):
  ```typescript
  await modal.locator('.v-select:visible').nth(0).click();
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Version 2.0.0/ }).click();
  ```

### Button Selectors Inside Modal

**Save button:** `.modal-dialog` → `button` → `.filter({ hasText: /^Save$/ })`
**Cancel button:** `.modal-dialog` → `button` → `.filter({ hasText: /^Cancel$/ })`
**Delete button:** `.modal-dialog` → `button` → `.filter({ hasText: /^Delete$/ })`

### Reply Message Node Config Fields

| Field | Type | Selector | Notes |
|-------|------|----------|-------|
| Node ID | text | `.modal-dialog .field-container` + label filter | Auto-populated, can be changed |
| Node Version | select | `.v-select:visible` nth(0) | Dropdown with versions |
| Receiver Channel | select | `.v-select:visible` nth(1) | Dropdown (Receiver Channel Reply, None) |
| Content Type | select | `.v-select:visible` nth(2) | Dropdown (Text Message, Media Message, etc.) |
| Message | text | `.modal-dialog .field-container` + label filter | Required field for message content |

### User Utterance Node Config Fields

| Field | Type | Selector | Notes |
|-------|------|----------|-------|
| Node ID | text | `.modal-dialog .field-container` + label filter | Auto-populated, can be changed |
| State Variable | text | `.modal-dialog .field-container` + label filter | Optional field |

---

## 3. Add Nodes Panel

### Opening the Panel

**Trigger:** `.nodes-button` (icon button with `mdi-shape-rectangle-plus`)
```typescript
await page.locator('.nodes-button').click();
await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
```

### Selecting a Node from the Menu

**Container:** `.nodes-dropdown-menu` (custom component, NOT Vuetify)
**Items:** `.nodes-dropdown-item` (custom component, NOT `.v-list-item`)

**Pick a node:**
```typescript
await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
```

### Closing the Menu

**CRITICAL:** Always close the dropdown before clicking on the canvas
```typescript
await page.keyboard.press('Escape');
await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
await page.waitForTimeout(300);
```

---

## 4. Flow Name Field

### Renaming the Flow

**Flow name location:** `.panel-container p.text-secondary` (displays current name)

**Edit mode:**
```typescript
// Click to enter edit mode
await page.locator('.panel-container p.text-secondary').first().click();
await page.waitForTimeout(300);

// Wait for input to appear
const flowNameInput = page.locator('.panel-container input').first();
await flowNameInput.waitFor({ state: 'visible', timeout: 5000 });

// Fill new name
await flowNameInput.fill('User Utterance');

// Press Enter to save
await flowNameInput.press('Enter');
await page.waitForTimeout(300);
```

**IMPORTANT:** The input has NO `type` attribute — do NOT use `input[type="text"]`

---

## 5. Save Flow (Disk Icon)

### Save Button

**Selector:** `.nodes-button` with icon `mdi-content-save` OR use icon selector
```typescript
await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
```

### Save Flow Version Modal

**Container:** Type B modal (no custom class) — scope to heading text
```typescript
const dialog = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
await dialog.waitFor({ state: 'visible', timeout: 10000 });
```

**Version Name Input:**
```typescript
const versionInput = dialog.locator('.v-field__input').first();
await versionInput.fill('UserUtteranceV1');
```

**Save Button in Modal:**
```typescript
await dialog.locator('button').filter({ hasText: /^Save$/ }).click();
```

### Success Toast Notification

**Selector:** `.v-snackbar`
```typescript
await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i);
```

---

## 6. Drag-Connect Strategy (Detailed)

### Pre-Connection Setup

**1. Move nodes to non-overlapping positions FIRST**

Nodes spawn at canvas center. Before connecting, move each to a unique position:

```typescript
// For 1st node (relative to START at y=0, height~40):
const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container#START') });
const prevBBox = await prevWrapper.boundingBox();
const newWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
  .first();
const newBBox = await newWrapper.boundingBox();
if (!prevBBox || !newBBox) throw new Error('Node not found');

const gap = 200; // 200px below START
const targetY = prevBBox.y + prevBBox.height + gap;
await page.mouse.move(newBBox.x + newBBox.width / 2, newBBox.y + newBBox.height / 2);
await page.mouse.down();
await page.mouse.move(newBBox.x + newBBox.width / 2, targetY, { steps: 10 });
await page.mouse.up();
await page.waitForTimeout(300);
```

**2. After moving, close any open modals and wait 500ms before connecting:**
```typescript
await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 10000 });
await page.waitForTimeout(500); // Canvas re-render time
```

### Full Drag Connection

```typescript
// Get handles
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#START') })
  .locator('.vue-flow__handle-bottom');
const targetHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
  .first()
  .locator('.vue-flow__handle-top');

// Get bounding boxes
const srcBox = await sourceHandle.boundingBox();
const tgtBox = await targetHandle.boundingBox();
if (!srcBox || !tgtBox) throw new Error('Handle not found');

// Hover on source
await page.mouse.move(srcBox.x + srcBox.width / 2, srcBox.y + srcBox.height / 2);
await page.waitForTimeout(200);

// Drag to target
await page.mouse.down();
await page.mouse.move(tgtBox.x + tgtBox.width / 2, tgtBox.y + tgtBox.height / 2, { steps: 20 });
await page.waitForTimeout(200);
await page.mouse.up();
await page.waitForTimeout(1000); // CRITICAL: 1000ms for persistence

// Verify edge exists
await page.locator('[data-id^="e-START-"]').waitFor({ state: 'visible', timeout: 5000 });
```

### Edge Verification ONLY Pattern

**FORBIDDEN (produces false positives with ghost edges):**
- `.vue-flow__edge` (any selector with this class)
- `.first()` or `.nth(N)` on `.vue-flow__edge`
- `toHaveCount()` on `.vue-flow__edge`

**CORRECT (only use this):**
```typescript
// Edge IDs follow: e-{sourceNodeId}-{targetNodeId}
await page.locator('[data-id^="e-START-"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-ReplyMessage_"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-input-"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-Output-"]').waitFor({ state: 'visible', timeout: 5000 });
```

---

## 7. Newly Added Nodes — ID Suffixing

**CRITICAL: Node IDs are auto-suffixed after creation**

When you add a node via NodesDropdown:
- Display name: "Reply Message" (with space)
- Type: `ReplyMessage` (camelCase, NO space)
- Auto-generated ID: `ReplyMessage_3`, `ReplyMessage_4`, etc.

**NEVER wait for `.node-container#ReplyMessage` after adding — it won't exist**

**Correct selector for newly added node:**
```typescript
// First Reply Message node added
const firstReplyMsg = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first();

// Second Reply Message node added
const secondReplyMsg = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).nth(1);
```

**After saving with custom Node ID (e.g., "input"):**
```typescript
// Now you can use the exact ID selector
const inputNode = page.locator('.node-container#input');
```

---

## 8. Node Positioning Strategy (MANDATORY)

**RULE: NEVER use fixed OFFSET_Y. All nodes spawn near canvas center.**

Fixed offsets cause overlap or off-screen positioning. Instead, position ALL nodes relative to their predecessor.

### Pattern for All Nodes (including 1st)

```typescript
// 1. Get previous node wrapper (START for 1st node, or previous added node)
const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container#PREV_ID') });
// OR for newly added: .filter({ has: page.locator('.node-container').filter({ hasText: /PrevLabel/ }) }).first()

// 2. Get previous node's bounding box
const prevBBox = await prevWrapper.boundingBox();

// 3. Get new node wrapper (use hasText with camelCase)
const newWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /NewNodeLabel/ }) })
  .nth(N); // N=0 for first of this type, N=1 for second, etc.

// 4. Get new node's bounding box
const newBBox = await newWrapper.boundingBox();

// 5. Validate both boxes exist
if (!prevBBox || !newBBox) throw new Error('Cannot position node — prev or new node not found');

// 6. Calculate target Y (200px below START, 150px below others)
const gap = 200; // or 150 for subsequent nodes
const targetY = prevBBox.y + prevBBox.height + gap;

// 7. Drag node to target position
await page.mouse.move(newBBox.x + newBBox.width / 2, newBBox.y + newBBox.height / 2);
await page.mouse.down();
await page.mouse.move(newBBox.x + newBBox.width / 2, targetY, { steps: 10 });
await page.mouse.up();
await page.waitForTimeout(300);
```

### Example Sequence

**Node 1 (Reply Message):**
- prev = START (id#START)
- gap = 200
- targetY = START.y + START.height + 200

**Node 2 (User Utterance / input):**
- prev = Reply Message (use hasText /ReplyMessage/ .first())
- gap = 150
- targetY = ReplyMessage.y + ReplyMessage.height + 150

**Node 3 (Reply Message / Output):**
- prev = User Utterance (id#input after saving)
- gap = 150
- targetY = input.y + input.height + 150

---

## 9. Test Timeout Configuration

**Multi-step flow canvas tests take longer than 30 seconds**

Set at the TOP of each test, BEFORE any steps:

```typescript
test('create flow with nodes', async ({ page }) => {
  test.setTimeout(120000); // ← First line, 2 minutes minimum
  // ... rest of test
});
```

---

## 10. Login & Navigation

**Login sequence (copy-paste from app-selectors/SKILL.md):**

```typescript
// Step 1 — Navigate
await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

// Step 2 — Fill credentials
await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
await page.getByRole('button', { name: /login/i }).click();

// Step 3 — Wait for org selection redirect
await page.waitForURL(/\?select_org/, { timeout: 20000 });

// Step 4 — Select organization
await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

**Navigate to Flow Designer:**
```typescript
await page.locator('a:has-text("Flow Designer")').click();
await page.waitForURL('**/flow-designer', { timeout: 10000 });
```

**Create new flow:**
```typescript
await page.locator('.m-auto').filter({ hasText: /Add New/ }).click();
// Wait for canvas page to load
await page.waitForURL('**/flow-designer/**', { timeout: 10000 });
```

---

## 11. Logging Format (MANDATORY)

**Every step MUST follow this exact format:**

```typescript
// BEFORE step action
console.log('📍 Step N: Description');

// ... perform action/assertion ...

// AFTER assertion succeeds (NOT before)
console.log('✅ PASS: Step N - Description');
```

**Summary block at end:**

```typescript
console.log('\n' + '='.repeat(70));
console.log('📊 TEST SUMMARY');
console.log('='.repeat(70));
console.log('✅ Step 1: PASS - Description');
console.log('✅ Step 2: PASS - Description');
// ... one line per step ...
console.log('='.repeat(70));
```

---

## 12. Key Selectors Quick Reference

| Component | Selector | Notes |
|-----------|----------|-------|
| Add Nodes button | `.nodes-button` | Icon: mdi-shape-rectangle-plus |
| Nodes dropdown menu | `.nodes-dropdown-menu` | Custom component |
| Nodes dropdown item | `.nodes-dropdown-item` | NOT .v-list-item |
| Save button | `.mdi-content-save` (parent button) | Icon selector |
| Node config modal | `.modal-dialog` | Custom class on v-dialog |
| Modal field container | `.modal-dialog .field-container` | Scope for label filtering |
| Modal input | `.v-field__input` | Inside .field-container |
| Modal dropdown | `.v-select:visible` | Inside modal |
| Modal dropdown option | `.v-overlay--active .v-list-item` | Teleported overlay |
| Flow name text | `.panel-container p.text-secondary` | Click to edit |
| Flow name input | `.panel-container input` | NO type attribute |
| Vue Flow canvas | `.vue-flow` | Main container |
| Node wrapper | `.vue-flow__node` | Parent of .node-container |
| Node container | `.node-container` | Contains node label |
| Node by ID | `.node-container#NODE_ID` | After saving with custom ID |
| Source handle | `.vue-flow__handle-bottom` | Inside .vue-flow__node |
| Target handle | `.vue-flow__handle-top` | Inside .vue-flow__node |
| Edge (ONLY pattern) | `[data-id^="e-SOURCE-"]` | Use ONLY this pattern |

---

## 13. Vuetify Patterns Used in Flow Designer

**v-dialog (modals):**
- Rendered as `.v-overlay--active` at end of body
- Custom class `.modal-dialog` added for node config modals
- Type B modals (no custom class) scoped by heading text

**v-select (dropdowns):**
- Rendered as `.v-overlay--active .v-list-item` at end of body
- Teleported, NOT inside parent component

**v-text-field (inputs):**
- Rendered as `.v-field__input` inside field container
- Vuetify wraps the actual `<input>` element

**v-btn (buttons):**
- Standard button element
- Use `getByRole('button', { name: /text/i })` or `.filter({ hasText: /text/ })`

---

## 14. Common Pitfalls & Solutions

| Issue | Wrong | Right |
|-------|-------|-------|
| Accessing handles | `.node-container.locator('.vue-flow__handle-bottom')` | `.vue-flow__node.filter(...).locator('.vue-flow__handle-bottom')` |
| Finding newly added nodes | `.node-container#ReplyMessage` (after adding) | `.node-container.filter({ hasText: /ReplyMessage/ }).first()` |
| Closing Add Nodes menu | No explicit close | `page.keyboard.press('Escape')` + waitFor hidden |
| Verifying edges | `.vue-flow__edge.first()` (ghost edge) | `[data-id^="e-SOURCE-"]` (real edge only) |
| Changing node ID | `.triple_click()` then type | `.fill('new value')` (fill clears automatically) |
| Flow name input selector | `input[type="text"]` | `.panel-container input` (NO type attribute) |
| Modal wait | No wait after open | Wait for `.v-overlay--active text=/loading/i` to disappear |
| Drag connection timing | 300ms wait after drag | 1000ms minimum for Vue Flow to persist |
| Node positioning | Fixed OFFSET_Y = -120 or 50 | Relative to previous node: targetY = prevBBox.y + prevBBox.height + gap |

---

## 15. Edge Cases & Special Handling

### Multiple Nodes of Same Type

When adding multiple "Reply Message" nodes:
- 1st: `.node-container.filter({ hasText: /ReplyMessage/ }).first()`
- 2nd: `.node-container.filter({ hasText: /ReplyMessage/ }).nth(1)`
- After saving with custom ID: `.node-container#CustomID`

### Node ID Changes During Edit

When you change a node's ID in the config modal:
- Old ID: `ReplyMessage_3`
- New ID (after save): `Output`
- Selector changes: `.node-container#Output`

### Edge Labels

- Default edge: no label (empty string)
- IF edge: label = "if"
- ELSE edge: label = "else"

### Dropdown Items in Modal

Vuetify dropdowns inside modals are teleported to body:
- WRONG: `modal.locator('.v-list-item')`
- RIGHT: `page.locator('.v-overlay--active .v-list-item')`

---

