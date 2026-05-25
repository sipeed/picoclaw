import { test, expect } from '@playwright/test';

test('Flow Tester - User Utterance Flow Test', async ({ page }) => {
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
  // STEP 5: Locate Select Conversation Flow dropdown
  // ============================================================================
  console.log('📍 Step 5: Locate the Select Conversation Flow dropdown');
  
  const flowSelect = page.locator('.selector-bar--welcome .selector-pill-select').first();
  await flowSelect.waitFor({ state: 'visible', timeout: 60000 });
  
  console.log('✅ PASS: Step 5 - Select Conversation Flow dropdown is visible');

  // ============================================================================
  // STEP 6: Click and open dropdown
  // ============================================================================
  console.log('📍 Step 6: Click Select Conversation Flow dropdown');
  
  await flowSelect.click();
  await page.locator('.v-overlay--active').waitFor({ state: 'visible', timeout: 15000 });
  
  console.log('✅ PASS: Step 6 - Dropdown opened');

  // ============================================================================
  // STEP 7: Select "User Utterance" flow
  // ============================================================================
  console.log('📍 Step 7: Select "User Utterance" flow from dropdown');
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /user utterance/i }).last().click();
  await page.waitForTimeout(500);

  console.log('✅ PASS: Step 7 - "User Utterance" flow selected');

  // ============================================================================
  // STEP 8: Verify selected flow is "User Utterance"
  // ============================================================================
  console.log('📍 Step 8: Verify selected flow is "User Utterance"');
  await expect(flowSelect).toContainText(/user utterance/i, { timeout: 15000 });
  console.log('✅ PASS: Step 8 - Selected flow verified');

  // ============================================================================
  // STEP 9: Wait for flow to load
  // ============================================================================
  console.log('📍 Step 9: Select version "UserUtteranceV1"');
  const versionText = page.locator('.version-selector-text');
  const versionButton = page.locator('.version-selector-button');

  for (let attempt = 1; attempt <= 3; attempt++) {
    await versionButton.click();
    const versionMenu = page.locator('.version-dropdown-menu');
    await versionMenu.waitFor({ state: 'visible', timeout: 15000 });
    await versionMenu.locator('.version-date').first().waitFor({ state: 'visible', timeout: 60000 });

    await versionMenu.locator('.version-item').filter({ hasText: /UserUtteranceV1/i }).first().click();
    await versionMenu.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => {});

    const currentVersion = ((await versionText.textContent()) ?? '').trim();
    if (/UserUtteranceV1/i.test(currentVersion)) break;

    if (attempt === 3) {
      throw new Error(`Expected version to be UserUtteranceV1, got "${currentVersion}"`);
    }
  }

  console.log('✅ PASS: Step 9 - Version "UserUtteranceV1" selected');

  // ============================================================================
  // STEP 10: Send "HELLO" message
  // ============================================================================
  console.log('📍 Step 10: Send "HELLO" message to the bot');
  
  const messageInput = page.getByRole('textbox', { name: /Type Your Message Here/i });
  await messageInput.fill('HELLO');
  await messageInput.press('Enter');
  
  console.log('✅ PASS: Step 10 - "HELLO" message sent');

  // ============================================================================
  // STEP 11: Wait for typing indicator to disappear (bot finished responding)
  // ============================================================================
  console.log('📍 Step 11: Wait for typing indicator to disappear (bot finished responding)');
  await page.locator('.typing-indicator').waitFor({ state: 'visible', timeout: 40000 }).catch(() => {});
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 90000 });
  console.log('✅ PASS: Step 11 - Typing indicator disappeared, bot finished');

  // ============================================================================
  // STEP 12: Verify bot responds with expected message
  // ============================================================================
  console.log('📍 Step 12: Verify bot responds with "Write something below to test the user utterance node:"');
  const firstBotBubble = page.locator('.bot-bubble-content').first();
  const promptBubble = page.locator('.bot-bubble-content')
    .filter({ hasText: /Write something below to test the user utterance node:/i })
    .first();
  try {
    await expect(firstBotBubble)
      .toContainText('Write something below to test the user utterance node:', { timeout: 90000 });
  } catch {
    await expect(promptBubble).toBeVisible({ timeout: 90000 });
  }
  console.log('✅ PASS: Step 12 - Bot responded with expected message');

  // ============================================================================
  // STEP 13: Send "I want to talk" message
  // ============================================================================
  console.log('📍 Step 13: Send "I want to talk" message to the bot');
  
  await messageInput.fill('I want to talk');
  await messageInput.press('Enter');
  
  console.log('✅ PASS: Step 13 - "I want to talk" message sent');

  // ============================================================================
  // STEP 14: Wait for bot to finish responding
  // ============================================================================
  console.log('📍 Step 14: Wait for bot to finish responding');
  await page.locator('.typing-indicator').waitFor({ state: 'visible', timeout: 40000 }).catch(() => {});
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 90000 });
  console.log('✅ PASS: Step 14 - Bot finished responding');

  // ============================================================================
  // STEP 15: Verify bot responds with the same message
  // ============================================================================
  console.log('📍 Step 15: Verify bot responds with "I want to talk"');
  
  await expect(page.locator('.bot-bubble-content').last())
    .toContainText('I want to talk', { timeout: 30000 });
  
  console.log('✅ PASS: Step 15 - Bot responded with "I want to talk"');

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
  console.log('✅ Step 5: PASS - Select Conversation Flow dropdown is visible');
  console.log('✅ Step 6: PASS - Dropdown opened');
  console.log('✅ Step 7: PASS - "User Utterance" flow selected');
  console.log('✅ Step 8: PASS - Selected flow verified');
  console.log('✅ Step 9: PASS - Version "UserUtteranceV1" auto-loaded');
  console.log('✅ Step 10: PASS - "HELLO" message sent');
  console.log('✅ Step 11: PASS - Typing indicator disappeared, bot finished');
  console.log('✅ Step 12: PASS - Bot responded with expected message');
  console.log('✅ Step 13: PASS - "I want to talk" message sent');
  console.log('✅ Step 14: PASS - Bot finished responding');
  console.log('✅ Step 15: PASS - Bot responded with "I want to talk"');
  console.log('='.repeat(70));
});
