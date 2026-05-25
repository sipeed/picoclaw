import { test, expect } from '@playwright/test';

test('Custom Node Flow - Send message and verify bot responses', async ({ page }) => {
  test.setTimeout(180000); // ← first line, before any steps

  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  console.log('✅ PASS: Step 1 - Login page loaded');

  console.log('📍 Step 2: Fill email and password');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  console.log('✅ PASS: Step 2 - Credentials filled');

  console.log('📍 Step 3: Click login button');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 3 - Login button clicked');

  console.log('📍 Step 4: Wait for organization selection or dashboard redirect');
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 60000 });
  if (page.url().includes('select_org')) {
    console.log('📍 Step 4a: Organization selection page detected - selecting Testing2026! org');
    const orgCards = page.locator('.organization-card');
    await orgCards.first().waitFor({ state: 'visible', timeout: 20000 });
    await orgCards.filter({ has: page.locator(':text-is("Testing2026!")') }).first().click();
    await page.waitForURL(url => !url.href.includes('select_org'), { timeout: 30000 });
  }
  console.log('✅ PASS: Step 4 - Post-login navigation complete');

  console.log('📍 Step 5: Navigate to Flow Tester');
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/flow-tester/, { timeout: 30000 });
  console.log('✅ PASS: Step 5 - Flow Tester page loaded');

  console.log('📍 Step 6: Select Custom Node flow from dropdown');
  const flowSelect = page.locator('.selector-bar--welcome .selector-pill-select').first();
  await flowSelect.waitFor({ state: 'visible', timeout: 60000 });
  await flowSelect.click();
  const flowListbox = page.locator('[role="listbox"]')
    .filter({ has: page.locator('[role="option"]') })
    .last();
  await flowListbox.waitFor({ state: 'visible', timeout: 15000 });
  await flowListbox.locator('[role="option"]').filter({ hasText: /custom node/i }).last().click();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 6 - Custom Node flow selected');

  console.log('📍 Step 7: Wait for flow to load');
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 7 - Flow loaded');

  console.log('📍 Step 8: Click message input field');
  const messageInput = page.getByRole('textbox', { name: /Type Your Message Here/i });
  await messageInput.click();
  console.log('✅ PASS: Step 8 - Message input field focused');

  console.log('📍 Step 9: Type message "Hello"');
  await messageInput.fill('Hello');
  console.log('✅ PASS: Step 9 - Message "Hello" typed');

  console.log('📍 Step 10: Send message by pressing Enter');
  await messageInput.press('Enter');
  console.log('✅ PASS: Step 10 - Message sent');

  console.log('📍 Step 11: Wait for typing indicator to disappear');
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 40000 });
  console.log('✅ PASS: Step 11 - Bot finished responding');

  console.log('📍 Step 12: Verify user message "Hello" appears in chat');
  const lastUserMessage = page.locator('.user-bubble-content, .chatbox .message-card-user .message-text').last();
  await expect(lastUserMessage).toContainText('Hello', { timeout: 20000 });
  console.log('✅ PASS: Step 12 - User message "Hello" verified');

  console.log('📍 Step 13: Verify first bot response appears');
  await expect(page.locator('.bot-bubble-content').first())
    .toContainText('Below is the Custom Node Result', { timeout: 20000 });
  console.log('✅ PASS: Step 13 - First bot response verified');

  console.log('📍 Step 14: Verify second bot response contains user input with appended text');
  await expect(page.locator('.bot-bubble-content').last())
    .toContainText('is what you inputted', { timeout: 20000 });
  console.log('✅ PASS: Step 14 - Second bot response verified');

  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login page loaded');
  console.log('✅ Step 2: PASS - Credentials filled');
  console.log('✅ Step 3: PASS - Login button clicked');
  console.log('✅ Step 4: PASS - Post-login navigation complete');
  console.log('✅ Step 5: PASS - Flow Tester page loaded');
  console.log('✅ Step 6: PASS - Custom Node flow selected');
  console.log('✅ Step 7: PASS - Flow loaded');
  console.log('✅ Step 8: PASS - Message input field focused');
  console.log('✅ Step 9: PASS - Message "Hello" typed');
  console.log('✅ Step 10: PASS - Message sent');
  console.log('✅ Step 11: PASS - Bot finished responding');
  console.log('✅ Step 12: PASS - User message "Hello" verified');
  console.log('✅ Step 13: PASS - First bot response verified');
  console.log('✅ Step 14: PASS - Second bot response verified');
  console.log('='.repeat(70));
});
