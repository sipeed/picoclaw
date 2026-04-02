# Flow Tester Reference Document

## 0. Files Read

1. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/components/flow-tester/BotIcon.vue`
2. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/pages/FlowTester.vue`
3. `/home/picoclaw/.picoclaw/workspace/context/dashboard/src/stores/flowTester.store.ts`

---

## 1. Flow Selection

### Selector for "Select Conversation Flow" dropdown
- **Selector:** `.tester-select`
- **How to open:** `await page.locator('.tester-select').click();`
- **How to pick a flow by name:** 
  ```typescript
  await page.locator('.tester-select').click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /USER_FLOW_NAME/ }).click();
  ```

### Selector for "Select Version" button
- **Selector:** `.version-selector-button`
- **Selector for version dropdown:** `.version-dropdown-menu`
- **How to open the version dropdown and pick a version:**
  ```typescript
  // CRITICAL: Wait for real items to load (not skeleton) before clicking
  await page.locator('.version-selector-button').click();
  await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
  // Wait for .version-date to appear (skeleton items don't have this)
  await page.locator('.version-dropdown-menu .version-date').first().waitFor({ state: 'visible', timeout: 10000 });
  // Now click the version item by name
  await page.locator('.version-dropdown-menu .version-item')
    .filter({ hasText: /VERSION_NAME/ })
    .click();
  // Verify selection completed
  await expect(page.locator('.version-selector-text'))
    .not.toContainText('Select Version', { timeout: 10000 });
  ```

---

## 2. Chat Interface

### Message input field
- **Selector:** `.message-field input`
- **How to type and send:**
  ```typescript
  await page.locator('.message-field input').fill('MESSAGE_TEXT');
  await page.locator('.message-field input').press('Enter');
  ```

### Bot messages in chat
- **Selector for bot message card:** `.chatbox .message-card` (only when `isOutput: true`)
- **Selector for message text:** `.chatbox .message-text`
- **How to verify a specific bot message appears:**
  ```typescript
  // For the LAST (most recent) bot message:
  await expect(page.locator('.chatbox .message-text').last())
    .toContainText('EXPECTED_BOT_TEXT', { timeout: 15000 });
  ```

### User messages in chat
- **Selector for user message card:** `.chatbox .message-card-user`
- **Selector for user message text:** `.chatbox .message-card-user .message-text`
- **How to verify a specific user message appears:**
  ```typescript
  // For the LAST (most recent) user message:
  await expect(page.locator('.chatbox .message-card-user .message-text').last())
    .toContainText('EXPECTED_USER_TEXT', { timeout: 5000 });
  ```

### Typing indicator (bot is responding)
- **Selector:** `.typing-indicator`
- **How to wait for bot to finish responding:**
  ```typescript
  // Wait for typing indicator to disappear (confirms SSE stream is closed)
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 20000 });
  ```

---

## 3. Initial Bot Message

From the source code (`flowTester.store.ts` line 23):
```typescript
messages: [{ text: 'Hi there! Test your flow here!', sender: UserType.AI }]
```

The store initializes with a default greeting message. This message appears in `.message-text` as a bot message.

**However, when a flow is selected and tested:**
- The flow may or may not send an automatic first message depending on its configuration
- Follow the Steps section exactly — it will tell you whether to wait for an initial bot message or send a user message first
- For the "User Utterance" flow in this test, the bot sends an automatic message after the user sends "HELLO"

---

## 4. Strict Mode Rules for Chat Messages

The chatbox accumulates messages as the conversation progresses. After the first message exchange, there will be multiple `.message-text` and `.message-card-user` elements.

**MANDATORY — use `.first()` or `.last()` on multi-element locators:**
- Latest bot message: `page.locator('.chatbox .message-text').last()`
- First bot message: `page.locator('.chatbox .message-text').first()` (only if no user message yet)
- First user message: `page.locator('.chatbox .message-card-user').first()`
- Latest user message: `page.locator('.chatbox .message-card-user').last()`

**FORBIDDEN — will cause strict mode violation when chat has 2+ messages:**
- `page.locator('.chatbox .message-text')` (without `.first()` or `.last()`)
- `page.locator('.chatbox .message-card-user')` (without `.first()` or `.last()`)

**Always use `toContainText()` NOT `toHaveText()`** to handle whitespace variations.

---

## 5. SSE Stream and Loading Guard

From the source code (`flowTester.store.ts`):
- `loadingMessage` is a state flag that guards the message send
- If `loadingMessage === true`, `sendMessageToStore()` silently rejects the input
- SSE streaming means `loadingMessage` can stay true for several seconds after the bot response appears in the DOM
- **CRITICAL:** Always wait for the typing indicator to disappear before sending a subsequent message

**Pattern:**
```typescript
// After sending first message and bot responds, wait for typing indicator
await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 20000 });
// NOW safe to send the next message
await page.locator('.message-field input').fill('NEXT_MESSAGE');
await page.locator('.message-field input').press('Enter');
```

---
