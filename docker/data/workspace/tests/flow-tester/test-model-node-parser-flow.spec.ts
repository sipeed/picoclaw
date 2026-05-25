import { test, expect } from '@playwright/test';

test('Flow Tester - Model Node with Parser Flow Test', async ({ page }) => {
  test.setTimeout(180000);

  // ============================================================================
  // STEP 1: Login
  // ============================================================================
  console.log('📍 Step 1: Login to dashboard');
  
  await page.goto('/login', { waitUntil: 'networkidle' });
  
  // Fill credentials
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 60000 });
  console.log('✅ PASS: Step 1 - Login successful, redirected from login');

  // ============================================================================
  // STEP 2: Select Organization
  // ============================================================================
  console.log('📍 Step 2: Select organization "Testing2026!"');
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

  console.log('✅ PASS: Step 2 - Organization selected, redirected to dashboard');

  // ============================================================================
  // STEP 3: Verify redirect to dashboard
  // ============================================================================
  console.log('📍 Step 3: Verify redirect to dashboard');
  
  await expect(page).toHaveURL(/https:\/\/dashboard\.int3nt\.info\/?$/);
  
  console.log('✅ PASS: Step 3 - Redirected to dashboard');

  // ============================================================================
  // STEP 4: Click Flow Tester on sidebar
  // ============================================================================
  console.log('📍 Step 4: Click Flow Tester on left sidebar');
  
  await page.locator('a:has-text("Flow Tester")').click();
  await page.waitForURL(/flow-tester/, { timeout: 30000 });
  
  console.log('✅ PASS: Step 4 - Navigated to Flow Tester page');

  // ============================================================================
  // STEP 5: Verify Flow Tester page is displayed
  // ============================================================================
  console.log('📍 Step 5: Verify Flow Tester page is displayed');
  
  await page.locator('.tester-container-card').waitFor({ state: 'visible', timeout: 20000 });
  const messageInput = page.getByRole('textbox', { name: /Type Your Message Here/i });
  await messageInput.waitFor({ state: 'visible', timeout: 20000 });
  
  console.log('✅ PASS: Step 5 - Flow Tester page loaded successfully');

  // ============================================================================
  // STEP 6: Open flow dropdown and select "Model Node with Parser"
  // ============================================================================
  console.log('📍 Step 6: Open flow dropdown and select "Model Node with Parser"');

  const flowSelect = page.locator('.selector-bar--welcome .selector-pill-select').first();
  await flowSelect.waitFor({ state: 'visible', timeout: 60000 });
  await flowSelect.click();
  const flowListbox = page.locator('[role="listbox"]')
    .filter({ has: page.locator('[role="option"]') })
    .last();
  await flowListbox.waitFor({ state: 'visible', timeout: 15000 });
  await flowListbox.locator('[role="option"]').filter({ hasText: /model node with parser/i }).last().click();
  await page.waitForTimeout(500);

  console.log('✅ PASS: Step 6 - "Model Node with Parser" flow selected');

  // ============================================================================
  // STEP 7: Select version "Model Node with Parser"
  // ============================================================================
  console.log('📍 Step 7: Select version "Model Node with Parser"');
  const versionText = page.locator('.version-selector-text');
  const versionButton = page.locator('.version-selector-button');
  await versionButton.click();
  const versionMenu = page.locator('.version-dropdown-menu');
  await versionMenu.waitFor({ state: 'visible', timeout: 15000 });
  await versionMenu.locator('.version-date').first().waitFor({ state: 'visible', timeout: 60000 });
  await versionMenu.locator('.version-item').filter({ hasText: /model node with parser/i }).first().click();
  await versionMenu.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => {});
  await expect(versionText).toContainText(/model node with parser/i, { timeout: 60000 });
  console.log('✅ PASS: Step 7 - Version selected');

  // ============================================================================
  // STEP 8: Send message
  // ============================================================================
  console.log('📍 Step 8: Send "hello" message');
  const botBubbles = page.locator('.bot-bubble-content');
  const initialBotCount = await botBubbles.count();
  await messageInput.fill('hello');
  await messageInput.press('Enter');
  await expect(page.locator('.user-bubble-content').last()).toContainText('hello', { timeout: 20000 });
  console.log('✅ PASS: Step 8 - Message sent');

  // ============================================================================
  // STEP 9: Wait for bot to finish responding
  // ============================================================================
  console.log('📍 Step 9: Wait for bot to finish responding');
  const typingIndicator = page.locator('.typing-indicator');
  await typingIndicator.waitFor({ state: 'visible', timeout: 40000 }).catch(() => {});
  await typingIndicator.waitFor({ state: 'hidden', timeout: 90000 });
  console.log('✅ PASS: Step 9 - Bot finished responding');

  // ============================================================================
  // STEP 10: Verify bot response contains Tokyo weather details
  // ============================================================================
  console.log('📍 Step 10: Verify bot response contains Tokyo weather details');
  await expect(botBubbles.nth(initialBotCount))
    .toContainText(/Below is the Model node result/i, { timeout: 90000 });

  const firstBotText = (await botBubbles.nth(initialBotCount).textContent()) ?? '';
  expect(firstBotText.toLowerCase()).toContain('tokyo');

  const lastBotText = (await botBubbles.last().textContent()) ?? '';
  expect(lastBotText).toMatch(/city/i);
  expect(lastBotText).toMatch(/condition/i);
  expect(lastBotText).toMatch(/temperature/i);
  console.log('✅ PASS: Step 10 - Bot response contains Tokyo weather details');

  // ============================================================================
  // TEST SUMMARY
  // ============================================================================
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful, redirected to org selection');
  console.log('✅ Step 2: PASS - Organization selected, redirected to dashboard');
  console.log('✅ Step 3: PASS - Redirected to dashboard');
  console.log('✅ Step 4: PASS - Navigated to Flow Tester page');
  console.log('✅ Step 5: PASS - Flow Tester page loaded successfully');
  console.log('✅ Step 6: PASS - "Model Node with Parser" flow selected');
  console.log('✅ Step 7: PASS - Version selected');
  console.log('✅ Step 8: PASS - Message sent');
  console.log('✅ Step 9: PASS - Bot finished responding');
  console.log('✅ Step 10: PASS - Bot response contains Tokyo weather details');
  console.log('='.repeat(70));
});
