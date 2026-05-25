import { test, expect } from '@playwright/test';

test('Model Node without Parser Flow - User sends "hello" and receives narcissistic reply', async ({ page }) => {
  test.setTimeout(180000); // ← first line, 90 seconds for model response

  // ========== STEP 1: Navigate to login page ==========
  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.locator('.login-card').waitFor({ state: 'visible', timeout: 20000 });
  console.log('✅ PASS: Step 1 - Login page loaded');

  // ========== STEP 2: Fill credentials and login ==========
  console.log('📍 Step 2: Fill credentials and login');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 2 - Credentials submitted');

  // ========== STEP 3: Wait for org selection or dashboard redirect ==========
  console.log('📍 Step 3: Wait for org selection or dashboard redirect');
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 60000 });
  console.log('✅ PASS: Step 3 - Redirected from login');

  // ========== STEP 4: Select organization ==========
  console.log('📍 Step 4: Select organization');
  if (page.url().includes('select_org')) {
    const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
    if (await loader.first().isVisible().catch(() => false)) {
      await loader.first().waitFor({ state: 'hidden', timeout: 30000 });
    }
    const orgCards = page.locator('.organization-card');
    await orgCards.first().waitFor({ state: 'visible', timeout: 20000 });
    await orgCards.filter({ has: page.locator(':text-is("Testing2026!")') }).first().click();
    await page.waitForURL(url => !url.href.includes('select_org'), { timeout: 30000 });
  }
  console.log('✅ PASS: Step 4 - Organization selected');

  // ========== STEP 5: Navigate to Flow Tester ==========
  console.log('📍 Step 5: Click Flow Tester from sidebar');
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/flow-tester/, { timeout: 30000 });
  await page.locator('.tester-container-card').waitFor({ state: 'visible', timeout: 20000 });
  console.log('✅ PASS: Step 5 - Flow Tester page loaded');

  // ========== STEP 6: Select flow "Node without Parser" ==========
  console.log('📍 Step 6: Select flow "Node without Parser"');
  const flowSelect = page.locator('.selector-bar--welcome .selector-pill-select').first();
  await flowSelect.waitFor({ state: 'visible', timeout: 60000 });
  await flowSelect.click();
  await page.locator('.v-overlay--active').waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /node without parser/i }).last().click();
  await expect(flowSelect).toContainText(/without parser/i, { timeout: 15000 });
  console.log('✅ PASS: Step 6 - Flow selected');

  // ========== STEP 7: Wait for version selector to stabilize ==========
  console.log('📍 Step 7: Wait for version selector to stabilize');
  await page.waitForTimeout(1000); // Allow flow to load
  console.log('✅ PASS: Step 7 - Version selector stabilized');

  // ========== STEP 8: Select version "node without parser" ==========
  console.log('📍 Step 8: Select latest version (first item)');
  const versionText = page.locator('.version-selector-text');
  const versionButton = page.locator('.version-selector-button');
  await versionButton.click();
  const versionMenu = page.locator('.version-dropdown-menu');
  await versionMenu.waitFor({ state: 'visible', timeout: 15000 });
  await versionMenu.locator('.version-date').first().waitFor({ state: 'visible', timeout: 60000 });
  await versionMenu.locator('.version-item').first().click();
  await versionMenu.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => {});
  await expect(versionText).not.toContainText('Select Version', { timeout: 60000 });
  console.log('✅ PASS: Step 8 - Version selected');

  // ========== STEP 9: Click message input field ==========
  console.log('📍 Step 9: Click message input field');
  const messageInput = page.getByRole('textbox', { name: /Type Your Message Here/i });
  await messageInput.waitFor({ state: 'visible', timeout: 60000 });
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
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 40000 });
  console.log('✅ PASS: Step 12 - Bot finished responding');

  // ========== STEP 13: Verify first bot message appears ==========
  console.log('📍 Step 13: Verify first bot message appears');
  const firstBotMessage = page.locator('.bot-bubble-content').first();
  await firstBotMessage.waitFor({ state: 'visible', timeout: 90000 });
  const firstBotText = (await firstBotMessage.textContent()) ?? '';
  const normalizedFirstBotText = firstBotText.toLowerCase();
  const isModelNodeText = normalizedFirstBotText.includes('below is the model node result');
  const isCustomNodeText = normalizedFirstBotText.includes('below is the custom node result');
  expect(isModelNodeText || isCustomNodeText).toBeTruthy();
  console.log('✅ PASS: Step 13 - First bot message verified');

  // ========== STEP 14: Verify second bot message (narcissistic reply) is not empty ==========
  console.log('📍 Step 14: Verify second bot message (narcissistic reply) is not empty');
  const lastBotMessage = page.locator('.bot-bubble-content').last();
  await lastBotMessage.waitFor({ state: 'visible', timeout: 20000 });
  const lastBotText = await lastBotMessage.textContent();
  expect(lastBotText?.trim().length).toBeGreaterThan(0);
  console.log('✅ PASS: Step 14 - Second bot message is not empty');

  // ========== STEP 15: Verify user message appears in chat ==========
  console.log('📍 Step 15: Verify user message "hello" appears in chat');
  const lastUserMessage = page.locator('.user-bubble-content').last();
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
