import { test, expect } from '@playwright/test';

test('Knowledge Base Flow — Send "Hello" and verify bot response', async ({ page }) => {
  test.setTimeout(180000); // 2 minutes — KB retrieval can be slow

  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  console.log('✅ PASS: Step 1 - Navigated to login page');

  console.log('📍 Step 2: Fill credentials and click Login');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 2 - Credentials filled and Login clicked');

  console.log('📍 Step 3: Wait for redirect to org selection or dashboard');
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 60000 });
  console.log('✅ PASS: Step 3 - Redirected from login');

  console.log('📍 Step 4: Handle org selection if needed');
  if (page.url().includes('select_org')) {
    const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
    if (await loader.first().isVisible().catch(() => false)) {
      await loader.first().waitFor({ state: 'hidden', timeout: 30000 });
    }
    const orgCards = page.locator('.organization-card');
    await orgCards.first().waitFor({ state: 'visible', timeout: 20000 });
    await orgCards.filter({ has: page.locator(':text-is("Testing")') }).first().click();
    await page.waitForURL(url => !url.href.includes('select_org'), { timeout: 30000 });
  }
  console.log('✅ PASS: Step 4 - Org selection handled');

  console.log('📍 Step 5: Navigate to Flow Tester from sidebar');
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/\/flow-tester/, { timeout: 30000 });
  console.log('✅ PASS: Step 5 - Navigated to Flow Tester');

  console.log('📍 Step 6: Verify Flow Tester page is displayed');
  await page.locator('.tester-container-card').waitFor({ state: 'visible', timeout: 20000 });
  console.log('✅ PASS: Step 6 - Flow Tester page displayed');

  console.log('📍 Step 7: Open flow dropdown and select "Knowledge Base"');
  const flowSelect = page.locator('.selector-bar--welcome .selector-pill-select').first();
  await flowSelect.waitFor({ state: 'visible', timeout: 60000 });
  await flowSelect.click();
  const flowListbox = page.locator('[role="listbox"]').filter({ has: page.locator('[role="option"]') }).last();
  await flowListbox.waitFor({ state: 'visible', timeout: 15000 });
  await flowListbox.locator('[role="option"]').filter({ hasText: /knowledge base/i }).last().click();
  console.log('✅ PASS: Step 7 - Knowledge Base flow selected');

  console.log('📍 Step 8: Wait for flow to load');
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 8 - Flow loaded');

  console.log('📍 Step 9: Click message input field');
  const messageInput = page.getByRole('textbox', { name: /Type Your Message Here/i });
  await messageInput.click();
  console.log('✅ PASS: Step 9 - Message input field clicked');

  console.log('📍 Step 10: Type "Hello" in message field');
  await messageInput.fill('Hello');
  console.log('✅ PASS: Step 10 - Message "Hello" typed');

  console.log('📍 Step 11: Press Enter to send message');
  await messageInput.press('Enter');
  console.log('✅ PASS: Step 11 - Message sent via Enter key');

  console.log('📍 Step 12: Wait for typing indicator to appear and disappear');
  const typingIndicator = page.locator('.typing-indicator');
  await typingIndicator.waitFor({ state: 'visible', timeout: 40000 }).catch(() => {});
  await typingIndicator.waitFor({ state: 'hidden', timeout: 120000 });
  console.log('✅ PASS: Step 12 - Bot response streaming completed');

  console.log('📍 Step 13: Verify user message "Hello" appears in chat');
  const userMsg = page.locator('.user-bubble-content, .chatbox .message-card-user .message-text').last();
  await expect(userMsg).toContainText('Hello', { timeout: 20000 });
  console.log('✅ PASS: Step 13 - User message "Hello" verified');

  console.log('📍 Step 14: Verify bot response appears and is not empty');
  const botOutputBubble = page.locator('.bot-bubble-content').last();
  let botText: string | null = null;
  try {
    await botOutputBubble.waitFor({ state: 'visible', timeout: 90000 });
    botText = await botOutputBubble.textContent();
  } catch {
    const detailsButton = page.getByRole('button', { name: /Additional bot details/i }).last();
    await detailsButton.waitFor({ state: 'visible', timeout: 90000 });
    await detailsButton.click();
    const detailsText = page.locator('.v-expansion-panel-text').last();
    await detailsText.waitFor({ state: 'visible', timeout: 15000 });
    botText = await detailsText.textContent();
  }
  expect(botText?.trim().length).toBeGreaterThan(0);
  console.log('✅ PASS: Step 14 - Bot response verified (non-empty)');

  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Navigated to login page');
  console.log('✅ Step 2: PASS - Credentials filled and Login clicked');
  console.log('✅ Step 3: PASS - Redirected from login');
  console.log('✅ Step 4: PASS - Org selection handled');
  console.log('✅ Step 5: PASS - Navigated to Flow Tester');
  console.log('✅ Step 6: PASS - Flow Tester page displayed');
  console.log('✅ Step 7: PASS - Knowledge Base flow selected');
  console.log('✅ Step 8: PASS - Flow loaded');
  console.log('✅ Step 9: PASS - Message input field clicked');
  console.log('✅ Step 10: PASS - Message "Hello" typed');
  console.log('✅ Step 11: PASS - Message sent via Enter key');
  console.log('✅ Step 12: PASS - Bot response streaming completed');
  console.log('✅ Step 13: PASS - User message "Hello" verified');
  console.log('✅ Step 14: PASS - Bot response verified (non-empty)');
  console.log('='.repeat(70));
});
