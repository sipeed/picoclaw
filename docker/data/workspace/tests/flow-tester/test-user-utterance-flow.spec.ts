import { test, expect } from '@playwright/test';

test('Flow Tester - User Utterance Flow Test', async ({ page }) => {
  test.setTimeout(90000);

  // ============================================================================
  // STEP 1: Login
  // ============================================================================
  console.log('📍 Step 1: Login to dashboard');
  
  await page.goto('/login', { waitUntil: 'networkidle' });
  
  // Fill credentials
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();
  
  // Wait for redirect to org selection
  await page.waitForURL(/\?select_org/, { timeout: 20000 });
  
  // Wait for loader to disappear
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
  }
  
  console.log('✅ PASS: Step 1 - Login successful, redirected to org selection');

  // ============================================================================
  // STEP 2: Select Organization
  // ============================================================================
  console.log('📍 Step 2: Select organization "Testing"');

  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ has: page.locator(':text-is("Testing2026!")') }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });

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
  await page.waitForURL(/flow-tester/, { timeout: 15000 });
  
  console.log('✅ PASS: Step 4 - Navigated to Flow Tester page');

  // ============================================================================
  // STEP 5: Locate Select Conversation Flow dropdown
  // ============================================================================
  console.log('📍 Step 5: Locate the Select Conversation Flow dropdown');
  
  await page.locator('.tester-select').waitFor({ state: 'visible', timeout: 10000 });
  
  console.log('✅ PASS: Step 5 - Select Conversation Flow dropdown is visible');

  // ============================================================================
  // STEP 6: Click and open dropdown
  // ============================================================================
  console.log('📍 Step 6: Click Select Conversation Flow dropdown');
  
  await page.locator('.tester-select').click();
  await page.waitForTimeout(300);
  
  console.log('✅ PASS: Step 6 - Dropdown opened');

  // ============================================================================
  // STEP 7: Select "User Utterance" flow
  // ============================================================================
  console.log('📍 Step 7: Select "User Utterance" flow from dropdown');

  // If multiple flows with same name exist, select the last one (oldest)
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /User Utterance/ }).last().click();
  await page.waitForTimeout(500);

  console.log('✅ PASS: Step 7 - "User Utterance" flow selected');

  // ============================================================================
  // STEP 8: Open version dropdown and select "UserUtteranceV1"
  // ============================================================================
  console.log('📍 Step 8: Open version dropdown and select "UserUtteranceV1"');
  
  // Click version selector button
  await page.locator('.version-selector-button').click();
  
  // Wait for dropdown to be visible
  await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
  
  // Wait for real items to load (not skeleton) by waiting for .version-date
  await page.locator('.version-dropdown-menu .version-date').first().waitFor({ state: 'visible', timeout: 10000 });
  
  // Click the version item by name
  await page.locator('.version-dropdown-menu .version-item')
    .filter({ hasText: /UserUtteranceV1/ })
    .click();
  
  // Verify selection completed
  await expect(page.locator('.version-selector-text'))
    .not.toContainText('Select Version', { timeout: 10000 });
  
  console.log('✅ PASS: Step 8 - Version "UserUtteranceV1" selected');

  // ============================================================================
  // STEP 9: Send "HELLO" message
  // ============================================================================
  console.log('📍 Step 9: Send "HELLO" message to the bot');
  
  await page.locator('.message-field input').fill('HELLO');
  await page.locator('.message-field input').press('Enter');
  
  console.log('✅ PASS: Step 9 - "HELLO" message sent');

  // ============================================================================
  // STEP 10: Verify bot responds with expected message
  // ============================================================================
  console.log('📍 Step 10: Verify bot responds with "Write something below to test the user utterance node:"');
  
  await expect(page.locator('.chatbox .message-text').last())
    .toContainText('Write something below to test the user utterance node:', { timeout: 15000 });
  
  console.log('✅ PASS: Step 10 - Bot responded with expected message');

  // ============================================================================
  // STEP 11: Wait for typing indicator to disappear
  // ============================================================================
  console.log('📍 Step 11: Wait for typing indicator to disappear (bot finished responding)');
  
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 20000 });
  
  console.log('✅ PASS: Step 11 - Typing indicator disappeared, bot finished');

  // ============================================================================
  // STEP 12: Send "I want to talk" message
  // ============================================================================
  console.log('📍 Step 12: Send "I want to talk" message to the bot');
  
  await page.locator('.message-field input').fill('I want to talk');
  await page.locator('.message-field input').press('Enter');
  
  console.log('✅ PASS: Step 12 - "I want to talk" message sent');

  // ============================================================================
  // STEP 13: Verify bot responds with the same message
  // ============================================================================
  console.log('📍 Step 13: Verify bot responds with "I want to talk"');
  
  await expect(page.locator('.chatbox .message-text').last())
    .toContainText('I want to talk', { timeout: 15000 });
  
  console.log('✅ PASS: Step 13 - Bot responded with "I want to talk"');

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
  console.log('✅ Step 8: PASS - Version "UserUtteranceV1" selected');
  console.log('✅ Step 9: PASS - "HELLO" message sent');
  console.log('✅ Step 10: PASS - Bot responded with expected message');
  console.log('✅ Step 11: PASS - Typing indicator disappeared, bot finished');
  console.log('✅ Step 12: PASS - "I want to talk" message sent');
  console.log('✅ Step 13: PASS - Bot responded with "I want to talk"');
  console.log('='.repeat(70));
});
