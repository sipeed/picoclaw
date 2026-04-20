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
  
  // Wait for redirect to org selection
  await page.waitForURL(/\?select_org/, { timeout: 60000 });
  
  // Wait for loader to disappear
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 30000 });
  }
  
  console.log('✅ PASS: Step 1 - Login successful, redirected to org selection');

  // ============================================================================
  // STEP 2: Select Organization
  // ============================================================================
  console.log('📍 Step 2: Select organization "Testing"');

  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 20000 });
  await page.locator('.organization-card').filter({ has: page.locator(':text-is("Testing2026!")') }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 30000 });

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
  
  await page.locator('.tester-select').waitFor({ state: 'visible', timeout: 20000 });
  await page.locator('.message-field input').waitFor({ state: 'visible', timeout: 20000 });
  
  console.log('✅ PASS: Step 5 - Flow Tester page loaded successfully');

  // ============================================================================
  // STEP 6: Open flow dropdown and select "Model Node with Parser"
  // ============================================================================
  console.log('📍 Step 6: Open flow dropdown and select "Model Node with Parser"');

  await page.locator('.tester-select').click();
  await page.locator('.v-overlay--active').waitFor({ state: 'visible', timeout: 5000 });
  // If multiple flows with same name exist, select the last one (oldest)
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /model node with parser/i }).last().click();
  await page.waitForTimeout(500);

  console.log('✅ PASS: Step 6 - "Model Node with Parser" flow selected');

  // ============================================================================
  // STEP 7: Open version dropdown and select "Model Node with Parser" version
  // ============================================================================
  console.log('📍 Step 7: Open version dropdown and select "Model Node with Parser" version');
  
  // Click version selector button
  await page.locator('.version-selector-button').click();
  
  // Wait for dropdown to be visible
  await page.locator('.version-dropdown-menu').waitFor({ state: 'visible', timeout: 5000 });
  
  // Wait for real items to load (not skeleton) by waiting for .version-date
  await page.locator('.version-dropdown-menu .version-date').first().waitFor({ state: 'visible', timeout: 40000 });
  
  // Click the version item by name
  await page.locator('.version-dropdown-menu .version-item')
    .filter({ hasText: /model node with parser/i })
    .click();
  
  // Verify selection completed
  await expect(page.locator('.version-selector-text'))
    .not.toContainText('Select Version', { timeout: 40000 });
  
  console.log('✅ PASS: Step 7 - Version "Model Node with Parser" selected');

  // ============================================================================
  // STEP 8: Click message input field
  // ============================================================================
  console.log('📍 Step 8: Click message input field at the bottom');
  
  await page.locator('.message-field input').click();
  
  console.log('✅ PASS: Step 8 - Message input field focused');

  // ============================================================================
  // STEP 9: Type "hello" message
  // ============================================================================
  console.log('📍 Step 9: Type "hello" message');
  
  await page.locator('.message-field input').fill('hello');
  
  console.log('✅ PASS: Step 9 - "hello" message typed');

  // ============================================================================
  // STEP 10: Click send button (arrow icon)
  // ============================================================================
  console.log('📍 Step 10: Click send button (arrow icon)');
  
  await page.locator('.message-field .v-field__append-inner').click();
  await page.waitForTimeout(300);
  
  console.log('✅ PASS: Step 10 - Send button clicked');

  // ============================================================================
  // STEP 11: Wait for typing indicator to appear and then disappear
  // ============================================================================
  console.log('📍 Step 11: Wait for bot to finish responding (typing indicator disappears)');
  
  await page.locator('.typing-indicator').waitFor({ state: 'hidden', timeout: 40000 });
  
  console.log('✅ PASS: Step 11 - Typing indicator disappeared, bot finished responding');

  // ============================================================================
  // STEP 12: Verify first bot response appears
  // ============================================================================
  console.log('📍 Step 12: Verify first bot response contains "Below is the Model node result"');
  
  await expect(page.locator('.chatbox .message-card .message-text').first())
    .toContainText('Below is the Model node result', { timeout: 30000 });
  
  console.log('✅ PASS: Step 12 - First bot response appeared with expected text');

  // ============================================================================
  // STEP 13: Verify second bot response contains parsed JSON output
  // ============================================================================
  console.log('📍 Step 13: Verify second bot response contains parsed JSON (city, condition, temperature)');
  
  await expect(page.locator('.chatbox .message-card .message-text').last())
    .toContainText('city:', { timeout: 30000 });
  
  await expect(page.locator('.chatbox .message-card .message-text').last())
    .toContainText('condition:', { timeout: 5000 });
  
  await expect(page.locator('.chatbox .message-card .message-text').last())
    .toContainText('temperature:', { timeout: 5000 });
  
  console.log('✅ PASS: Step 13 - Second bot response contains parsed JSON output');

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
  console.log('✅ Step 7: PASS - Version "Model Node with Parser" selected');
  console.log('✅ Step 8: PASS - Message input field focused');
  console.log('✅ Step 9: PASS - "hello" message typed');
  console.log('✅ Step 10: PASS - Send button clicked');
  console.log('✅ Step 11: PASS - Typing indicator disappeared, bot finished responding');
  console.log('✅ Step 12: PASS - First bot response appeared with expected text');
  console.log('✅ Step 13: PASS - Second bot response contains parsed JSON output');
  console.log('='.repeat(70));
});
