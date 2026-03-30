---
name: app-selectors
description: Global login and organization selection flow for dashboard.int3nt.info. Use this when writing Playwright tests that require authentication.
---

# Global Login & Org Selection

> Generated: 2026-03-30T00:13:12.890Z

## Test Credentials

| Field | Value |
|-------|-------|
| Email | `heidi@intnt.ai` |
| Password | `testing2026!` |
| Organization | `Testing2026!` |

## Login Flow (copy-paste ready)

> ⚠️ **COPY THIS ENTIRE BLOCK VERBATIM. Do NOT rewrite, simplify, or invent alternative selectors.**
> - URL MUST be `https://dashboard.int3nt.info/login` (with `/login`) — NOT the root URL
> - Email selector MUST be `.v-text-field().nth(0).locator('input')` — NEVER fill `.v-text-field` directly (it is a div, not an input)
> - Password selector MUST be `.v-text-field().nth(1).locator('input')` — NEVER `input[type="password"]`
> - Do NOT add `.click()` before `.fill()` on the outer `.v-text-field` — fill the inner `input` directly
> - Login button MUST use `getByRole('button', { name: /login/i })` — the button text is **"Login"** (ONE word, no space). NEVER use `filter({ hasText: /Log In/ })` or any other selector.
> - After clicking login, you MUST wait for `?select_org` redirect BEFORE selecting org — NEVER skip straight to dashboard
> - DO NOT replace `waitForURL(/\?select_org/)` with `waitForURL('**/dashboard')` — the org selection step is MANDATORY

```typescript
// Step 1 — Navigate
await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

// Step 2 — Fill credentials (use EXACTLY these selectors — .v-text-field is a div, the input is INSIDE it)
await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
await page.getByRole('button', { name: /login/i }).click();

// Step 3 — Wait for redirect to org selection
await page.waitForURL(/\?select_org/, { timeout: 20000 });

// Step 4 — Wait for loader to disappear, then select organization
const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
if (await loader.first().isVisible().catch(() => false)) {
  await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
}
await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

## Key Selectors

| Element | Selector |
|---------|----------|
| Email input | `.v-text-field:nth(0) input` |
| Password input | `.v-text-field:nth(1) input` |
| Login button | `getByRole('button', { name: /login/i })` |
| Org card | `.organization-card` |
| Org name in card | `.organization-name` |
| Org dropdown trigger | `.org-dropdown-trigger` |
| Org dropdown item | `.org-dropdown-item` |

**Known orgs:** "Testing2026!", "Testing"

---

## Playwright Patterns for This App

> This app uses **Vue.js 3 + Vuetify 3 + Vue Flow**. The following patterns MUST be followed to avoid common failures.

### 0. Valid Playwright API methods — never invent methods

Only use methods that exist in the Playwright `Locator` API. Common hallucinated methods that DO NOT EXIST:

| ❌ Wrong (invented) | ✅ Correct Playwright API |
|---|---|
| `.triple_click()` | DO NOT USE — method does not exist |
| `.triple_click?.()` | DO NOT USE — optional chaining `?.` silently returns undefined; the call does nothing |
| `.doubleClick()` | `.dblclick()` |
| `.selectText()` | `.press('Control+a')` or `.press('Meta+a')` |
| `.clearText()` | `.clear()` or `.fill('')` |
| `.typeText('x')` | `.pressSequentially('x')` |
| `.filter({ hasAttribute: ... })` | not supported — use `page.locator('[attr="val"]')` |

**To change an input field value** (including Node ID fields): `fill()` automatically clears before typing. No triple-click needed:
```typescript
await input.click();
await input.fill('new value');  // clears existing value and types new one
await input.press('Tab');       // trigger Vue validation
```

---

### 1. Nodes dropdown — always close before touching the canvas

After clicking `.nodes-dropdown-item` to add a node, the dropdown stays open and intercepts all subsequent clicks. Always dismiss it first:

```typescript
await page.locator('.nodes-dropdown-item').filter({ hasText: /Node Name/ }).click();
await page.keyboard.press('Escape');
await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
await page.waitForTimeout(300);
```

### 2. Waiting for a node config modal

`.field-container` matches multiple elements (every field in the form). Use `.modal-dialog` instead — it is unique when a modal is open:

```typescript
await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 10000 });
```

### 3. Vuetify form validation — enabling the Save button

The Save button is `disabled` until Vue's validation fires. Auto-populated fields do not count as "touched". Always use **click → fill → Tab** to trigger validation:

```typescript
await input.click();
await input.fill('value');
await input.press('Tab');
```

### 4. Targeting a specific labeled input in a Vuetify modal

`input[type="text"]` also matches Vuetify combobox/select inputs (role="combobox"). Use the label to scope precisely:

```typescript
page.locator('.modal-dialog .field-container')
  .filter({ has: page.locator('label', { hasText: /^Label Name/ }) })
  .locator('.v-field__input')
```

### 5. Moving a newly added node to avoid overlap

Nodes spawn at the same position on the canvas and stack on top of each other. Always move each node to a unique position immediately after adding it, before clicking or connecting it.

Drag the `.vue-flow__node` wrapper (parent of `.node-container`) using `page.mouse`.

**`.nth(N)` placement rule:** put `.nth(N)` on the OUTER `.vue-flow__node` result — NEVER inside the `has` filter:
```typescript
// WRONG — .nth inside has doesn't scope to the parent wrapper:
.filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).nth(1) }).first()
// CORRECT — put .nth on the outer vue-flow__node result:
.filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) }).nth(1)
```

---

**⚠️ MANDATORY POSITIONING STRATEGY — use for EVERY added node (including the 1st):**

> NEVER use a fixed OFFSET_Y. All nodes spawn near the same canvas center.
> Fixed offsets cause overlap OR push nodes off-screen above the viewport.
> Instead, position ALL nodes relative to their predecessor in the flow chain.

- **1st added node** → position relative to **START** (200px below START's bottom edge)
- **2nd added node** → position relative to 1st node (150px below)
- **3rd added node** → position relative to 2nd node (150px below)

```typescript
// Use this same pattern for ALL nodes — only change the prevWrapper and newWrapper locators:

// For 1st node: prev = START
const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container#START') });
// For 2nd node (auto-suffixed ID): prev = 1st node by hasText
// const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) }).first();
// For 3rd node (saved with custom ID): prev = 2nd node by ID
// const prevWrapper = page.locator('.vue-flow__node').filter({ has: page.locator('.node-container#input') });

const prevBBox = await prevWrapper.boundingBox();

// Find the new node — use hasText with camelCase (ReplyMessage, UserUtterance — NO spaces):
const newWrapper = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /NewNodeLabel/ }) })
  .nth(N); // N=0 (.first()) for first of this type, N=1 for second of same type
const newBBox = await newWrapper.boundingBox();

if (!prevBBox || !newBBox) throw new Error('Cannot position node — prev or new node not found');

const gap = 200; // 200 for 1st node after START, 150 for all subsequent
const targetY = prevBBox.y + prevBBox.height + gap;
await page.mouse.move(newBBox.x + newBBox.width / 2, newBBox.y + newBBox.height / 2);
await page.mouse.down();
await page.mouse.move(newBBox.x + newBBox.width / 2, targetY, { steps: 10 });
await page.mouse.up();
await page.waitForTimeout(300);
```

❌ **WRONG:**
```typescript
OFFSET_Y = -120  // fixed offset from spawn — node may go off-screen if spawn is near top
OFFSET_Y = 50 for node 2, OFFSET_Y = 50 for node 3  // same center → same Y → overlap
page.mouse.move(nodeBox.x + OFFSET_X, nodeBox.y + OFFSET_Y)  // top-left drift
```

### 6. Clicking canvas nodes — use evaluate even after moving

Vue Flow adds new nodes near the center of the canvas, so they stack on top of each other. `.click()` fails when another node intercepts at that screen position. Use `evaluate` to fire the click directly on the DOM element:

```typescript
await nodeLocator.evaluate(el => (el as HTMLElement).click());
```

### 7. Vue Flow drag connections — NEVER use dragTo()

> Vue Flow requires intermediate `mousemove` events to register a connection. `dragTo()` does not generate these events reliably. Also: **move nodes to non-overlapping positions BEFORE connecting** — handles on stacked nodes cannot be targeted precisely.

`dragTo()` does not fire the intermediate `mousemove` events that Vue Flow needs to register a connection. Use `page.mouse` with `steps`.

**CRITICAL: `.vue-flow__handle` elements may be siblings of `.node-container`, not children.**
Always scope handle lookups via the `.vue-flow__node` wrapper — never via `.node-container` directly:

```typescript
// ❌ WRONG — handle may not be inside .node-container:
const sourceHandle = page.locator('.node-container#SOURCE_ID').locator('.vue-flow__handle-bottom');

// ✅ CORRECT — always go through .vue-flow__node wrapper:
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#SOURCE_ID') })
  .locator('.vue-flow__handle-bottom');

// ✅ CORRECT — for auto-suffixed nodes (locate by text):
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
  .first()
  .locator('.vue-flow__handle-bottom');
```

**Full drag connection pattern:**

```typescript
const sourceHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#SOURCE_ID') })
  .locator('.vue-flow__handle-bottom');
const targetHandle = page.locator('.vue-flow__node')
  .filter({ has: page.locator('.node-container#TARGET_ID') })
  .locator('.vue-flow__handle-top');

const sourceBox = await sourceHandle.boundingBox();
const targetBox = await targetHandle.boundingBox();
if (!sourceBox || !targetBox) throw new Error('Handle not found — check node IDs and visibility');

// CRITICAL: hover on source handle briefly before mousedown — Vue Flow needs hover state to start drag detection
await page.mouse.move(sourceBox.x + sourceBox.width / 2, sourceBox.y + sourceBox.height / 2);
await page.waitForTimeout(200); // pause on handle
await page.mouse.down();
await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + targetBox.height / 2, { steps: 20 }); // 20 steps minimum
await page.waitForTimeout(200); // CRITICAL: dwell on target handle — Vue Flow needs mouseenter to register target
await page.mouse.up();
await page.waitForTimeout(1000);
```

**NEVER use a silent if-guard:**
```typescript
// ❌ WRONG — silently skips the drag if a handle is not found:
if (sourceBox && targetBox) { /* drag */ }

// ✅ CORRECT — throw explicitly so the failure is visible:
if (!sourceBox || !targetBox) throw new Error('Handle not found');
```

> **Why steps?** Vue Flow listens to `mousemove` during drag to draw the connection line and detect the target handle. Without intermediate steps, it never registers the connection even if mouseup lands on the correct handle.

### 8. Verifying edges after a drag connection

**ONLY use `[data-id^="e-{source}-"]` for edge verification. NEVER use `.vue-flow__edge` CSS class.**

**Why:** Vue Flow creates a **ghost edge** element with class `.vue-flow__edge` during ANY drag operation (for visual feedback). This ghost element causes `.first()` and `.nth(N)` to produce **false positives** — they match the ghost, not a real persistent edge. Only **real, persisted edges** have a `data-id` attribute.

**After each drag, wait 1000ms (not 300ms) then verify with `data-id`:**

```typescript
await page.mouse.up();
await page.waitForTimeout(1000); // give Vue Flow time to persist the edge

// Edge IDs follow pattern: e-{sourceNodeId}-{targetNodeId}
await page.locator('[data-id^="e-START-"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-ReplyMessage_"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-input-"]').waitFor({ state: 'visible', timeout: 5000 });
await page.locator('[data-id^="e-Output-"]').waitFor({ state: 'visible', timeout: 5000 });
```

❌ **ALL of these cause false positives with ghost edges — NEVER USE:**
```typescript
await page.locator('.vue-flow__edge').first().waitFor(...)   // matches ghost edge
await page.locator('.vue-flow__edge').nth(N).waitFor(...)    // matches ghost edge
await expect(page.locator('.vue-flow__edge')).toHaveCount(N) // counts ghost edges
let count = await page.locator('.vue-flow__edge').count()    // counts ghost edges
```
