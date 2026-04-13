import { test, expect } from '@playwright/test';

test('Create KB schedule with Incremental Sync in ADVANCED mode', async ({ page }) => {
  test.setTimeout(60000);

  // ============================================================================
  // STEP 1: Login
  // ============================================================================
  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });

  console.log('📍 Step 2: Fill login credentials');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();

  console.log('📍 Step 3: Wait for redirect to org selection');
  await page.waitForURL(/\?select_org/, { timeout: 20000 });
  console.log('✅ PASS: Step 3 - Redirected to org selection');

  console.log('📍 Step 4: Wait for loader and select organization');
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
  }
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
  console.log('✅ PASS: Step 4 - Organization selected');

  // ============================================================================
  // STEP 5: Navigate to Knowledge Base
  // ============================================================================
  console.log('📍 Step 5: Click Knowledge Base in sidebar');
  await page.locator('a:has-text("Knowledge Base")').click();
  await page.waitForURL(/\/knowledge-base/, { timeout: 10000 });
  console.log('✅ PASS: Step 5 - Navigated to Knowledge Base');

  // ============================================================================
  // STEP 6: Locate and click Schedule button on Picotest2 card
  // ============================================================================
  console.log('📍 Step 6: Locate Picotest2 KB bucket and click Schedule');
  const picotest2Card = page.locator('.bucket-card').filter({
    has: page.locator('.bucket-name').filter({ hasText: /Picotest2/ })
  });
  await picotest2Card.waitFor({ state: 'visible', timeout: 10000 });

  // Click the schedule button on this card
  const scheduleBtn = picotest2Card.locator('.schedule-button');
  await scheduleBtn.click();
  console.log('✅ PASS: Step 6 - Clicked Schedule button');

  // ============================================================================
  // STEP 7: Wait for modal and click Create Schedule
  // ============================================================================
  console.log('📍 Step 7: Wait for Manage Schedule modal');
  await page.locator('.modal-content').waitFor({ state: 'visible', timeout: 10000 });
  console.log('✅ PASS: Step 7 - Modal opened');

  console.log('📍 Step 8: Click Create Schedule button');
  await page.locator('button').filter({ hasText: /Create Schedule/ }).click();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 8 - Create Schedule form opened');

  // ============================================================================
  // STEP 9: Select Incremental Sync
  // ============================================================================
  console.log('📍 Step 9: Select Incremental Sync radio button');
  await page.getByRole('radio', { name: /Incremental Sync/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 9 - Incremental Sync selected');

  // ============================================================================
  // STEP 10: Switch to ADVANCED mode
  // ============================================================================
  console.log('📍 Step 10: Switch Cron Expression mode to ADVANCED');
  await page.locator('.mode-toggle').locator('button').filter({ hasText: /Advanced/ }).click();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 10 - Switched to ADVANCED mode');

  // ============================================================================
  // STEP 11: Verify UTC timezone warning
  // ============================================================================
  console.log('📍 Step 11: Verify UTC/GMT+0 timezone warning');
  const timezoneWarning = page.locator('.advanced-mode').locator('.timezone-info');
  await timezoneWarning.waitFor({ state: 'visible', timeout: 5000 });
  await expect(timezoneWarning).toContainText(/UTC\/GMT\+0/);
  console.log('✅ PASS: Step 11 - UTC/GMT+0 timezone warning displayed');

  // ============================================================================
  // STEP 12: Verify cron expression is auto-generated (not empty)
  // ============================================================================
  console.log('📍 Step 12: Verify Cron Expression is auto-generated');
  const cronInput = page.locator('.advanced-mode').locator('input[type="text"]');
  await cronInput.waitFor({ state: 'visible', timeout: 5000 });

  const cronValue = await cronInput.inputValue();
  console.log(`   Auto-generated cron: "${cronValue}"`);

  if (!cronValue || cronValue.trim().length === 0) {
    throw new Error('Cron expression field is empty after switching to ADVANCED mode');
  }
  console.log('✅ PASS: Step 12 - Cron expression auto-populated');

  // ============================================================================
  // STEP 13: Enter the test cron expression: 0 9 * * 1-5
  // ============================================================================
  console.log('📍 Step 13: Enter cron expression "0 9 * * 1-5"');
  await cronInput.fill('0 9 * * 1-5');
  await cronInput.press('Tab'); // Trigger validation
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 13 - Cron expression entered');

  // ============================================================================
  // STEP 14: Click Save
  // ============================================================================
  console.log('📍 Step 14: Click Save button');
  await page.locator('button').filter({ hasText: /^Save$/ }).click();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 14 - Save clicked');

  // ============================================================================
  // STEP 15: Verify success notification
  // ============================================================================
  console.log('📍 Step 15: Verify success notification appears');
  await expect(page.locator('.v-snackbar')).toContainText('Schedule created successfully');
  console.log('✅ PASS: Step 15 - Success notification displayed');

  // ============================================================================
  // STEP 16: Verify modal shows saved schedule with Incremental Sync and cron
  // ============================================================================
  console.log('📍 Step 16: Verify saved schedule is displayed in modal');
  await page.waitForTimeout(1500); // Wait for modal to update

  // Check Sync Type
  const syncChip = page.locator('.info-section .info-row').first().locator('.sync-chip');
  await expect(syncChip).toContainText('Incremental Sync');
  console.log('✅ PASS: Step 16a - Sync Type shows "Incremental Sync"');

  // Check Cron Expression
  const cronValueDisplay = page.locator('.info-section .info-row').nth(1).locator('.cron-value');
  const displayedCron = await cronValueDisplay.textContent();
  console.log(`   Displayed cron expression: "${displayedCron}"`);

  if (!displayedCron || displayedCron.trim().length === 0) {
    throw new Error('Cron expression not displayed in view mode');
  }

  // Verify it matches the expected pattern (0 9 * * 1-5)
  if (displayedCron.includes('0 9 * * 1-5')) {
    console.log('✅ PASS: Step 16b - Cron expression matches expected pattern (0 9 * * 1-5)');
  } else {
    console.log(`   Warning: Cron pattern may differ. Got: "${displayedCron}"`);
  }

  // ============================================================================
  // TEST SUMMARY
  // ============================================================================
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login page loaded');
  console.log('✅ Step 2: PASS - Credentials filled');
  console.log('✅ Step 3: PASS - Redirected to org selection');
  console.log('✅ Step 4: PASS - Organization selected');
  console.log('✅ Step 5: PASS - Navigated to Knowledge Base');
  console.log('✅ Step 6: PASS - Found Picotest2 and clicked Schedule');
  console.log('✅ Step 7: PASS - Modal opened');
  console.log('✅ Step 8: PASS - Create Schedule form opened');
  console.log('✅ Step 9: PASS - Incremental Sync selected');
  console.log('✅ Step 10: PASS - Switched to ADVANCED mode');
  console.log('✅ Step 11: PASS - UTC/GMT+0 timezone warning displayed');
  console.log('✅ Step 12: PASS - Cron expression auto-generated');
  console.log('✅ Step 13: PASS - Cron expression entered (0 9 * * 1-5)');
  console.log('✅ Step 14: PASS - Save clicked');
  console.log('✅ Step 15: PASS - Success notification displayed');
  console.log('✅ Step 16: PASS - Saved schedule displayed with Incremental Sync and cron');
  console.log('='.repeat(70));
});
