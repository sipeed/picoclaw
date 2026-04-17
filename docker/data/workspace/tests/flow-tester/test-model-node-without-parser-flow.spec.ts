import { test, expect } from '@playwright/test';

test('Model Node without Parser Flow - User sends "hello" and receives narcissistic reply', async ({ page }) => {
  test.setTimeout(90000); // ← first line, 90 seconds for model response

  // ========== STEP 1: Navigate to login page ==========
  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.locator('.login-card').waitFor({ state: 'visible', timeout: 10000 });
  console.log('✅ PASS: Step 1 - Login page loaded');

  // ========== STEP 2: Fill credentials and login ==========
  console.log('📍 Step 2: Fill credentials and login');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 2 - Credentials submitted');

  // ========== STEP 3: Wait for org selection redirect ==========
  console.log('📍 Step 3: Wait for org selection redirect');
  await page.waitForURL(/\?select_org/, { timeout: 20000 });
  console.log('✅ PASS: Step 3 - Redirected to org selection');

  // ========== STEP 4: Select organization ==========
  console.log('📍 Step 4: Select organization');
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
  }
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ has: page.locator(':text-is("Testing")') }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
  console.log('✅ PASS: Step 4 - Organization selected');

  // ========== STEP 5: Navigate to Flow Tester ==========
  console.log('📍 Step 5: Click Flow Tester from sidebar');
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/flow-tester/, { timeout: 15000 });
  await page.locator('.tester-container-card').waitFor({ state: 'visible', timeout: 10000 });
  console.log('✅ PASS: Step 5 - Flow Tester page loaded');

  // ========== STEP 6: Select flow "Node without Parser" ==========
  console.log('📍 Step 6: Select flow "Node without Parser"');
  await page.locator('.tester-select').click();
  await page.locator('.v-overlay--active').waitFor({ state: 'visible', timeout: 5000 });
  // If multiple flows with same name exist, select the last one (oldest)
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /node without parser/i }).last().click();
  console.log('✅ PASS: Step 6 - Flow selected');

  // ========== STEP 7: Wait for version selector to stabilize ==========
  console.log('📍 Step 7: Wait for version selector to stabilize');
  await page.waitForTimeout(1000); // Allow flow to load
  console.log('✅ PASS: Step 7 - Version selector stabilized');

  // ========== STEP 8: Open version dropdown and select version ==========
  console.log('📍 Step 8: Open version dropdown and select version "node without parser"');
  await page.locator('.version-selector-button').click();
  await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
  // Wait for real items (not skeleton) — .version-date only appears on real items
  await page.locator('.version-dropdown-menu .version-date').first()
    .waitFor({ state: 'visible', timeout: 20000 });
  // Now click the version by name
  await page.locator('.version-dropdown-menu .version-item')
    .filter({ hasText: /node without parser/i })
    .click();
  // Verify version was selected
  await expect(page.locator('.version-selector-text'))
    .not.toContainText('Select Version', { timeout: 20000 });
  console.log('✅ PASS: Step 8 - Version selected');

  // ========== STEP 9: Click message input field ==========
  console.log('📍 Step 9: Click message input field');
  const messageInput = page.locator('.message-field input');
  await messageInput.waitFor({ state: 'visible', timeout: 5000 });
  await messageInput.click();
  console.log('✅ PASS: Step 9 - Message input field focused');

  // ========== STEP 10: Type "hello" ==========
  console.log('📍 Step 10: Type "hello"');
  await messageInput.fill('hello');
  console.log('✅ PASS: Step 10 - Message typed');

  // ========== STEP 11: Send message (press Enter) ==========
  console.log('📍 Step 11: Send message via Enter key');
  await messageInput.press('Enter');
  console.log('✅ PASS: Step 11 - Message sent');

  // ========== STEP 12: Wait for typing indicator to disappear ==========
  console.log('📍 Step 12: Wait for bot to finish responding (typing indicator disappears)');
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 20000 });
  console.log('✅ PASS: Step 12 - Bot finished responding');

  // ========== STEP 13: Verify first bot message appears ==========
  console.log('📍 Step 13: Verify first bot message appears');
  const firstBotMessage = page.locator('.chatbox .message-card .message-text').first();
  await expect(firstBotMessage)
    .toContainText('Below is the Model node result. it should return your input narcisticly', { timeout: 20000 });
  console.log('✅ PASS: Step 13 - First bot message verified');

  // ========== STEP 14: Verify second bot message (narcissistic reply) is not empty ==========
  console.log('📍 Step 14: Verify second bot message (narcissistic reply) is not empty');
  const lastBotMessage = page.locator('.chatbox .message-card .message-text').last();
  await lastBotMessage.waitFor({ state: 'visible', timeout: 20000 });
  const lastBotText = await lastBotMessage.textContent();
  expect(lastBotText?.trim().length).toBeGreaterThan(0);
  console.log('✅ PASS: Step 14 - Second bot message is not empty');

  // ========== STEP 15: Verify user message appears in chat ==========
  console.log('📍 Step 15: Verify user message "hello" appears in chat');
  const lastUserMessage = page.locator('.chatbox .message-card-user .message-text').last();
  await expect(lastUserMessage)
    .toContainText('hello', { timeout: 5000 });
  console.log('✅ PASS: Step 15 - User message verified');

  // ========== TEST SUMMARY ==========
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login page loaded');
  console.log('✅ Step 2: PASS - Credentials submitted');
  console.log('✅ Step 3: PASS - Redirected to org selection');
  console.log('✅ Step 4: PASS - Organization selected');
  console.log('✅ Step 5: PASS - Flow Tester page loaded');
  console.log('✅ Step 6: PASS - Flow selected');
  console.log('✅ Step 7: PASS - Version selector stabilized');
  console.log('✅ Step 8: PASS - Version selected');
  console.log('✅ Step 9: PASS - Message input field focused');
  console.log('✅ Step 10: PASS - Message typed');
  console.log('✅ Step 11: PASS - Message sent');
  console.log('✅ Step 12: PASS - Bot finished responding');
  console.log('✅ Step 13: PASS - First bot message verified');
  console.log('✅ Step 14: PASS - Second bot message is not empty');
  console.log('✅ Step 15: PASS - User message verified');
  console.log('='.repeat(70));
});
