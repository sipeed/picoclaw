import { test, expect } from '@playwright/test';

test('Knowledge Base Flow — Send "Hello" and verify bot response', async ({ page }) => {
  test.setTimeout(120000); // 2 minutes — KB retrieval can be slow

  console.log('📍 Step 1: Navigate to login page');
  await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });
  console.log('✅ PASS: Step 1 - Navigated to login page');

  console.log('📍 Step 2: Fill credentials and click Login');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 2 - Credentials filled and Login clicked');

  console.log('📍 Step 3: Wait for redirect to org selection or dashboard');
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 20000 });
  console.log('✅ PASS: Step 3 - Redirected from login');

  console.log('📍 Step 4: Handle org selection if needed');
  if (page.url().includes('select_org')) {
    const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
    if (await loader.first().isVisible().catch(() => false)) {
      await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
    }
    await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
    await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
    await page.waitForURL(url => !url.href.includes('select_org'), { timeout: 15000 });
  }
  console.log('✅ PASS: Step 4 - Org selection handled');

  console.log('📍 Step 5: Navigate to Flow Tester from sidebar');
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/\/flow-tester/, { timeout: 15000 });
  console.log('✅ PASS: Step 5 - Navigated to Flow Tester');

  console.log('📍 Step 6: Verify Flow Tester page is displayed');
  await page.locator('.tester-container-card').waitFor({ state: 'visible', timeout: 10000 });
  console.log('✅ PASS: Step 6 - Flow Tester page displayed');

  console.log('📍 Step 7: Open flow dropdown and select "Knowledge Base"');
  await page.locator('.tester-select').click();
  await page.locator('.v-overlay--active').waitFor({ state: 'visible', timeout: 5000 });
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /knowledge base/i }).click();
  console.log('✅ PASS: Step 7 - Knowledge Base flow selected');

  console.log('📍 Step 8: Wait for flow to load and select latest version');
  // Wait for version selector to be ready
  await page.locator('.version-selector-text').waitFor({ state: 'visible', timeout: 10000 });
  
  // Open version dropdown
  await page.locator('.version-selector-button').click();
  await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
  
  // Wait for real items to load (not skeleton)
  await page.locator('.version-dropdown-menu .version-date').first()
    .waitFor({ state: 'visible', timeout: 20000 });
  
  // Click first (latest) version
  await page.locator('.version-dropdown-menu .version-item').first().click();
  
  // Verify selection completed
  await expect(page.locator('.version-selector-text'))
    .not.toContainText('Select Version', { timeout: 20000 });
  console.log('✅ PASS: Step 8 - Latest version selected');

  console.log('📍 Step 9: Click message input field');
  await page.locator('.message-field input').click();
  console.log('✅ PASS: Step 9 - Message input field clicked');

  console.log('📍 Step 10: Type "Hello" in message field');
  await page.locator('.message-field input').fill('Hello');
  console.log('✅ PASS: Step 10 - Message "Hello" typed');

  console.log('📍 Step 11: Press Enter to send message');
  await page.locator('.message-field input').press('Enter');
  console.log('✅ PASS: Step 11 - Message sent via Enter key');

  console.log('📍 Step 12: Wait for typing indicator to appear and disappear');
  // Wait for typing indicator to appear (confirms message was sent)
  await page.locator('.typing-indicator').waitFor({ state: 'visible', timeout: 10000 });
  // Wait for typing indicator to disappear (confirms bot response is ready)
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 60000 });
  console.log('✅ PASS: Step 12 - Bot response streaming completed');

  console.log('📍 Step 13: Verify user message "Hello" appears in chat');
  const userMsg = page.locator('.chatbox .message-card-user .message-text').last();
  await userMsg.waitFor({ state: 'visible', timeout: 10000 });
  await expect(userMsg).toContainText('Hello', { timeout: 5000 });
  console.log('✅ PASS: Step 13 - User message "Hello" verified');

  console.log('📍 Step 14: Verify bot response appears and is not empty');
  const botMsg = page.locator('.chatbox .message-card .message-text').last();
  await botMsg.waitFor({ state: 'visible', timeout: 60000 });
  const botText = await botMsg.textContent();
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
  console.log('✅ Step 8: PASS - Latest version selected');
  console.log('✅ Step 9: PASS - Message input field clicked');
  console.log('✅ Step 10: PASS - Message "Hello" typed');
  console.log('✅ Step 11: PASS - Message sent via Enter key');
  console.log('✅ Step 12: PASS - Bot response streaming completed');
  console.log('✅ Step 13: PASS - User message "Hello" verified');
  console.log('✅ Step 14: PASS - Bot response verified (non-empty)');
  console.log('='.repeat(70));
});
