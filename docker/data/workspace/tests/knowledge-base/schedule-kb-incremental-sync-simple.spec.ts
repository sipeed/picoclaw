import { test, expect } from '@playwright/test';

test('Schedule KB with Incremental Sync in Simple Mode', async ({ page }) => {
  test.setTimeout(60000);

  // ============================================================================
  // STEP 1: LOGIN
  // ============================================================================
  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });

  console.log('📍 Step 2: Fill email and password');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  await page.getByRole('button', { name: /login/i }).click();

  console.log('📍 Step 3: Wait for organization selection redirect');
  await page.waitForURL(/\?select_org/, { timeout: 20000 });

  console.log('📍 Step 4: Select organization "Testing2026!"');
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
  }
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });

  console.log('✅ PASS: Step 1-4 - Logged in and selected organization');

  // ============================================================================
  // STEP 5: NAVIGATE TO KNOWLEDGE BASE
  // ============================================================================
  console.log('📍 Step 5: Navigate to Knowledge Base');
  await page.locator('a:has-text("Knowledge Base")').click();
  await page.waitForURL(/\/knowledge-base/, { timeout: 10000 });
  await page.locator('.knowledge-base-container').waitFor({ state: 'visible', timeout: 10000 });

  console.log('✅ PASS: Step 5 - Navigated to Knowledge Base page');

  // ============================================================================
  // STEP 6: LOCATE PICOTEST2 BUCKET AND CLICK SCHEDULE BUTTON
  // ============================================================================
  console.log('📍 Step 6: Locate "Picotest2" KB bucket');
  const picotest2Card = page.locator('.bucket-card').filter({
    has: page.locator('.bucket-name').filter({ hasText: /^Picotest2$/ })
  });
  await picotest2Card.waitFor({ state: 'visible', timeout: 10000 });

  console.log('📍 Step 7: Click Schedule button on Picotest2 card');
  const scheduleButton = picotest2Card.locator('.schedule-button');
  await scheduleButton.click();

  console.log('✅ PASS: Step 6-7 - Located Picotest2 and clicked Schedule button');

  // ============================================================================
  // STEP 8: WAIT FOR MANAGE SCHEDULE MODAL TO APPEAR
  // ============================================================================
  console.log('📍 Step 8: Wait for Manage Schedule modal to appear');
  const modal = page.locator('.v-card-title').filter({ hasText: /Manage Schedule/ });
  await modal.waitFor({ state: 'visible', timeout: 10000 });

  console.log('✅ PASS: Step 8 - Manage Schedule modal appeared');

  // ============================================================================
  // STEP 9: CLICK CREATE SCHEDULE BUTTON
  // ============================================================================
  console.log('📍 Step 9: Click "Create Schedule" button');
  await page.getByRole('button', { name: /Create Schedule/i }).click();
  await page.waitForTimeout(500);

  console.log('✅ PASS: Step 9 - Clicked Create Schedule button');

  // ============================================================================
  // STEP 10: SELECT INCREMENTAL SYNC
  // ============================================================================
  console.log('📍 Step 10: Select "Incremental Sync" radio button');
  const incrementalSyncRadio = page.getByRole('radio', { name: /Incremental Sync/i });
  await incrementalSyncRadio.click();
  await page.waitForTimeout(300);

  // Verify it's selected
  const isChecked = await incrementalSyncRadio.isChecked();
  expect(isChecked).toBe(true);

  console.log('✅ PASS: Step 10 - Selected Incremental Sync');

  // ============================================================================
  // STEP 11: VERIFY SIMPLE MODE IS ACTIVE (OR SWITCH IF NEEDED)
  // ============================================================================
  console.log('📍 Step 11: Verify Cron Expression mode is SIMPLE');
  const simpleButton = page.getByRole('button', { name: /Simple/i });
  const simpleButtonClasses = await simpleButton.getAttribute('class');

  // Check if Simple button is already active/selected
  if (!simpleButtonClasses?.includes('v-btn--active')) {
    console.log('   → Simple mode not active, switching to Simple mode');
    await simpleButton.click();
    await page.waitForTimeout(500);
  }

  console.log('✅ PASS: Step 11 - Verified SIMPLE mode is active');

  // ============================================================================
  // STEP 12: SELECT FREQUENCY = MONTHLY
  // ============================================================================
  console.log('📍 Step 12: Open Frequency dropdown and select "Monthly"');
  const frequencySelect = page.locator('.v-select').first();
  await frequencySelect.click();
  await page.waitForTimeout(500);

  const monthlyOption = page.locator('.v-overlay--active .v-list-item').filter({
    hasText: /^Monthly$/
  });
  await monthlyOption.click();
  await page.waitForTimeout(300);

  console.log('✅ PASS: Step 12 - Selected "Monthly" frequency');

  // ============================================================================
  // STEP 13: SELECT DAY 1 FROM MONTHLY DAYS SELECTOR
  // ============================================================================
  console.log('📍 Step 13: Open monthly days selector and select "Day 1"');
  const monthlyDaysSelect = page.locator('.v-select').nth(1);
  await monthlyDaysSelect.click();
  await page.waitForTimeout(500);

  const day1Option = page.locator('.v-overlay--active .v-list-item').filter({
    hasText: /^Day 1$/
  });
  await day1Option.click();
  await page.waitForTimeout(300);

  console.log('✅ PASS: Step 13 - Selected "Day 1"');

  // ============================================================================
  // STEP 14: FILL AT TIME INPUT WITH 00:00
  // ============================================================================
  console.log('📍 Step 14: Fill "At time" input with "00:00"');
  const timeInput = page.locator('input[type="time"]').last();
  await timeInput.click();
  await timeInput.fill('00:00');
  await timeInput.press('Tab');
  await page.waitForTimeout(300);

  console.log('✅ PASS: Step 14 - Filled time input with "00:00"');

  // ============================================================================
  // STEP 15: VERIFY CRON EXPRESSION IS GENERATED
  // ============================================================================
  console.log('📍 Step 15: Verify Cron Expression is auto-generated and not empty');
  const cronPreview = page.locator('code.preview-value');
  await cronPreview.waitFor({ state: 'visible', timeout: 5000 });

  const cronText = await cronPreview.textContent();
  expect(cronText).not.toBe('');
  expect(cronText).toBeTruthy();
  console.log(`   → Generated Cron Expression: ${cronText}`);

  console.log('✅ PASS: Step 15 - Cron Expression auto-generated');

  // ============================================================================
  // STEP 16: CLICK SAVE BUTTON
  // ============================================================================
  console.log('📍 Step 16: Click "Save" button');
  const saveButton = page.getByRole('button', { name: /Save/i });
  await saveButton.click();
  await page.waitForTimeout(1000);

  console.log('✅ PASS: Step 16 - Clicked Save button');

  // ============================================================================
  // STEP 17: VERIFY SUCCESS NOTIFICATION
  // ============================================================================
  console.log('📍 Step 17: Verify success notification appears');
  const snackbar = page.locator('.v-snackbar');
  await snackbar.waitFor({ state: 'visible', timeout: 5000 });

  const snackbarText = await snackbar.textContent();
  expect(snackbarText).toContain('Schedule created successfully');

  console.log('✅ PASS: Step 17 - Success notification displayed');

  // ============================================================================
  // TEST SUMMARY
  // ============================================================================
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1-4: PASS - Logged in and selected organization');
  console.log('✅ Step 5: PASS - Navigated to Knowledge Base page');
  console.log('✅ Step 6-7: PASS - Located Picotest2 and clicked Schedule button');
  console.log('✅ Step 8: PASS - Manage Schedule modal appeared');
  console.log('✅ Step 9: PASS - Clicked Create Schedule button');
  console.log('✅ Step 10: PASS - Selected Incremental Sync');
  console.log('✅ Step 11: PASS - Verified SIMPLE mode is active');
  console.log('✅ Step 12: PASS - Selected Monthly frequency');
  console.log('✅ Step 13: PASS - Selected Day 1');
  console.log('✅ Step 14: PASS - Filled time input with 00:00');
  console.log('✅ Step 15: PASS - Cron Expression auto-generated');
  console.log('✅ Step 16: PASS - Clicked Save button');
  console.log('✅ Step 17: PASS - Success notification displayed');
  console.log('='.repeat(70));
});
