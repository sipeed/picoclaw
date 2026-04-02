# Flow Designer Test Reference Document

## 0. Files Read (Phase 1)

**Total files read: 12**

1. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/pages/FlowDesigner.vue
2. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/pages/FlowCanvas.vue
3. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-designer/FlowBoard.vue
4. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-designer/NodeConfigurationModal.vue
5. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-designer/SaveFlowModal.vue
6. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-designer/EdgeConfigurationModal.vue
7. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-designer/dropdown/NodesDropdown.vue
8. /home/picoclaw/.picoclaw/workspace/context/dashboard/src/stores/flow.store.ts
9. /skills/playwright/SKILL.md
10. /skills/app-selectors/SKILL.md
11. /skills/app-selectors-flow-designer/SKILL.md
12. /skills/app-selectors-flow-designer-canvas/SKILL.md

---

## 1. Node Types

### START Node
- **Type:** Start (Terminal)
- **Container Selector:** `.node-container#START`
- **Vue Flow Wrapper:** `.vue-flow__node` (parent of `.node-container#START`)
- **Handle Selector:** `.vue-flow__node` filter `hasText: /START/` → `.vue-flow__handle-bottom`
- **Position:** Default (250, 0)
- **Icon:** mdi-home
- **Drag Connection Strategy:** Use handle bottom for source connections

### END Node
- **Type:** End (Terminal)
- **Container Selector:** `.node-container#END`
- **Vue Flow Wrapper:** `.vue-flow__node` (parent of `.node-container#END`)
- **Handle Selector:** `.vue-flow__node` filter `hasText: /END/` → `.vue-flow__handle-top`
- **Position:** Default (250, 400)
- **Icon:** mdi-stop
- **Drag Connection Strategy:** Use handle top for target connections

### Knowledge Base Node
- **Type:** KnowledgeBaseNode
- **Auto-suffixed ID:** KnowledgeBaseNode_N (where N is auto-generated index)
- **Container Selector (after adding):** `.node-container` filter `hasText: /KnowledgeBaseNode/`
- **Vue Flow Wrapper:** `.vue-flow__node` filter `has: .node-container` filter `hasText: /KnowledgeBaseNode/`
- **Handle Selectors:**
  - Source (bottom): `.vue-flow__handle-bottom`
  - Target (top): `.vue-flow__handle-top`
- **Icon:** mdi-feature-search-outline
- **Drag Connection Strategy:** Same as all nodes - use page.mouse with intermediate steps

### Reply Message Node
- **Type:** ReplyMessage
- **Auto-suffixed ID:** ReplyMessage_N (where N is auto-generated index)
- **Container Selector (after adding):** `.node-container` filter `hasText: /ReplyMessage/`
- **Vue Flow Wrapper:** `.vue-flow__node` filter `has: .node-container` filter `hasText: /ReplyMessage/`
- **Handle Selectors:**
  - Source (bottom): `.vue-flow__handle-bottom`
  - Target (top): `.vue-flow__handle-top`
- **Icon:** mdi-message-text-outline
- **Drag Connection Strategy:** Same as all nodes - use page.mouse with intermediate steps

---

## 2. Node Config Modals

### Knowledge Base Node Modal

**How it opens:**
```typescript
// Click the node after it's added to canvas
const nodeLocator = page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }).first();
await nodeLocator.evaluate(el => (el as HTMLElement).click());
```

**Modal Selector:** `.modal-dialog` (unique when open)

**Wait for modal:**
```typescript
await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 10000 });
```

**Fields and Selectors:**

| Field | Type | Selector | Notes |
|-------|------|----------|-------|
| Node ID | text | `.modal-dialog` → `.field-container` filter `has label: /Node ID/` → `.v-field__input` | Custom ID to save |
| Is Tool | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(0) | Options: True, False |
| Source | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(1) | Options: Document Search Options, Vector Search Options |
| Document Search Cluster | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(2) | Options: testing kb, Picotest2, etc. |
| Search Fields | array | `.modal-dialog` → `button` filter `hasText: /Add Item/`.nth(0) | Click "Add Item" to add entries |
| Size | number | `.modal-dialog` → `.field-container` filter `has label: /Size/` → `.v-field__input` | Default: 5 |
| Min Score | text | `.modal-dialog` → `.field-container` filter `has label: /minimum _score/i` → `.v-field__input` | Minimum score for matching |
| Inner Hits Size | text | `.modal-dialog` → `.field-container` filter `has label: /Number of chunks/i` → `.v-field__input` | Chunks per document |
| Response Fields | array | `.modal-dialog` → `button` filter `hasText: /Add Item/`.nth(1) | Click "Add Item" to add entries |
| Arguments | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(3) | Options: Knowledge Search Arguments, None |
| Query | text | `.modal-dialog` → `.field-container` filter `has label: /Query/` → `.v-field__input` | Search query |

**Dropdown Behavior:**
- Click `.v-select` or `.v-autocomplete` to open
- Wait for `.v-overlay--active` to appear
- Click `.v-list-item` filter `hasText: /OPTION_TEXT/`
- Overlay closes automatically

**Buttons:**
- Delete: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Delete/ })`
- Cancel: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Cancel/ })`
- Save: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Save/ })`

### Reply Message Node Modal

**How it opens:**
```typescript
// Click the node after it's added to canvas
const nodeLocator = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first();
await nodeLocator.evaluate(el => (el as HTMLElement).click());
```

**Modal Selector:** `.modal-dialog` (unique when open)

**Wait for modal:**
```typescript
await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 10000 });
```

**Fields and Selectors:**

| Field | Type | Selector | Notes |
|-------|------|----------|-------|
| Node ID | text | `.modal-dialog` → `.field-container` filter `has label: /Node ID/` → `.v-field__input` | Custom ID to save |
| Node Version | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(0) | Auto-populated |
| Receiver Channel | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(1) | Auto-populated |
| Content Type | select | `.modal-dialog` → `.v-select:visible,.v-autocomplete:visible,.v-combobox:visible`.nth(2) | Auto-populated |
| Message | text | `.modal-dialog` → `.field-container` filter `has label: /Message/` → `.v-field__input` | Template syntax supported |

**Buttons:**
- Delete: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Delete/ })`
- Cancel: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Cancel/ })`
- Save: `page.locator('.modal-dialog').locator('button').filter({ hasText: /Save/ })`

---

## 3. Add Nodes Panel

**Open Selector:**
```typescript
await page.locator('.nodes-button').click();
```

**Wait Condition:**
```typescript
await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 10000 });
```

**Node Selection Method:**
```typescript
await page.locator('.nodes-dropdown-item').filter({ hasText: /Knowledge Base Node/ }).click();
```

**Close Dropdown (MANDATORY after selecting):**
```typescript
await page.keyboard.press('Escape');
await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
await page.waitForTimeout(300);
```

**Available Nodes in Dropdown:**
- User Utterance
- Custom Node
- Custom Tool
- Rest API Node
- Human Agent Node
- Knowledge Base Node ← for this test
- Model
- Reply Message ← for this test
- Sub Graph

---

## 4. Flow Name Field

**Location:** `.panel-container` (top-left of canvas, beside home icon)

**Selector (text mode):**
```typescript
const flowNameText = page.locator('.panel-container p.text-secondary').first();
```

**Interaction Pattern:**
```typescript
// Click to enter edit mode
await page.locator('.panel-container p.text-secondary').first().click();
await page.waitForTimeout(300);

// Wait for input to appear
const flowNameInput = page.locator('.panel-container input').first();
await flowNameInput.waitFor({ state: 'visible', timeout: 5000 });

// Fill with new name
await flowNameInput.fill('Knowledge Base');

// Press Enter to save
await flowNameInput.press('Enter');
await page.waitForTimeout(300);
```

**Important:** Input has NO `type` attribute — do NOT use `input[type="text"]`

---

## 5. Save Flow

**Save Button Selector:**
```typescript
const saveBtn = page.locator('.history-button').filter({ has: page.locator('mdi-content-save') });
// OR
const saveBtn = page.locator('button').filter({ has: page.locator('.mdi-content-save') });
```

**Save Flow Version Modal:**
```typescript
// Modal opens after clicking Save button
const modal = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
await modal.waitFor({ state: 'visible', timeout: 10000 });
```

**Version Input Selector:**
```typescript
const versionInput = modal.locator('.v-text-field').locator('input').first();
```

**Version Input Interaction:**
```typescript
await versionInput.fill('KnowledgebaseV1');
```

**Save Button in Modal:**
```typescript
await modal.locator('button').filter({ hasText: /^Save$/ }).click();
```

**Success Toast:**
```typescript
await expect(page.locator('.v-snackbar')).toContainText('success', { timeout: 10000 });
```

---

## 6. Drag-Connect Strategy (CRITICAL)

**Exact steps using Vue Flow handles:**

```typescript
// Get source node wrapper
const sourceWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#START') });

// Get target node wrapper
const targetWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#Kb') });

// Get source handle (bottom)
const sourceHandle = sourceWrapper.locator('.vue-flow__handle-bottom');

// Get target handle (top)
const targetHandle = targetWrapper.locator('.vue-flow__handle-top');

// Get bounding boxes
const sourceBox = await sourceHandle.boundingBox();
const targetBox = await targetHandle.boundingBox();

if (!sourceBox || !targetBox) throw new Error('Handle not found');

// CRITICAL: Hover on source handle briefly before mousedown
await page.mouse.move(sourceBox.x + sourceBox.width / 2, sourceBox.y + sourceBox.height / 2);
await page.waitForTimeout(200); // pause on handle

// Start drag
await page.mouse.down();

// Move to target with intermediate steps (minimum 20 steps)
await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + targetBox.height / 2, { steps: 20 });

// CRITICAL: Dwell on target handle (Vue Flow needs mouseenter to register)
await page.waitForTimeout(200);

// Release
await page.mouse.up();

// Wait for edge to persist
await page.waitForTimeout(1000);
```

**Edge Verification (MANDATORY):**
```typescript
// Count edges BEFORE drag
const edgesBefore = await page.locator('.vue-flow__edge[data-id]').count();

// ... perform drag ...

// Wait 1000ms for persistence
await page.waitForTimeout(1000);

// Count edges AFTER drag
const edgesAfter = await page.locator('.vue-flow__edge[data-id]').count();

// Verify edge was created
if (edgesAfter <= edgesBefore) {
  throw new Error(`Edge NOT created — before: ${edgesBefore}, after: ${edgesAfter}`);
}
```

**NEVER use:**
- `.vue-flow__edge.first()`
- `.vue-flow__edge.nth(N)`
- `.vue-flow__edge[data-id^="e-START-"]` (matches ghost edges)
- `.toHaveCount(N)` on `.vue-flow__edge` (counts ghost edges)

**ALWAYS use:**
- `[data-id^="e-{source}-"]` for real edge verification
- `[data-id]` to count real edges only

---

## 7. Node Positioning Strategy (MANDATORY)

**Canvas Transform:**
```typescript
// Get canvas transformation
const pane = page.locator('.vue-flow__transformationpane');
const tf = await pane.evaluate((el) => {
  const transform = el.style.transform;
  const match = transform.match(/translate\(([-\d.]+)px, ([-\d.]+)px\) scale\(([-\d.]+)\)/);
  if (!match) return { tx: 0, ty: 0, scale: 1 };
  return { tx: parseFloat(match[1]), ty: parseFloat(match[2]), scale: parseFloat(match[3]) };
});
```

**Position Node (relative to predecessor):**
```typescript
// For 1st node after START:
const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container#START') });
const prevBBox = await prevWrapper.boundingBox();

const newWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }) })
  .first();
const newBBox = await newWrapper.boundingBox();

if (!prevBBox || !newBBox) throw new Error('Cannot position node');

const gap = 200; // 200px below START's bottom edge
const targetY = prevBBox.y + prevBBox.height + gap;

await page.mouse.move(newBBox.x + newBBox.width / 2, newBBox.y + newBBox.height / 2);
await page.mouse.down();
await page.mouse.move(newBBox.x + newBBox.width / 2, targetY, { steps: 10 });
await page.mouse.up();
await page.waitForTimeout(300);
```

---

## Summary

- **Node containers:** `.node-container#{ID}` or `.node-container` filter `hasText: /TYPE/`
- **Vue Flow wrappers:** Always `.vue-flow__node` filter `has: .node-container#{ID}`
- **Handles:** Always scope via wrapper, use `.vue-flow__handle-bottom` or `.vue-flow__handle-top`
- **Modals:** `.modal-dialog` for node config, `.v-overlay--active` filter `hasText: /Title/` for other dialogs
- **Dropdowns:** Click `.v-select`, wait for `.v-overlay--active`, click `.v-list-item`
- **Drag connections:** Use `page.mouse` with 20+ steps, hover before mousedown, dwell on target
- **Edge verification:** Count `[data-id]` attributes only, never use CSS class selectors
- **Positioning:** Relative to predecessor with 200px gap for 1st node, 150px for subsequent nodes

---
