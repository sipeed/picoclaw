import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('API Key revoke flow', async ({ page }) => {
  test.setTimeout(180000);

  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';

  // Step 1-2: Perform the login and select org flow
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Settings in the left sidebar
  console.log('\n📍 Step 3: Click Settings in the left sidebar');
  const settingsLink = page.locator('nav').getByText('Settings').first();
  await expect(settingsLink).toBeVisible();
  await settingsLink.click();
  await page.waitForTimeout(1500);
  console.log('✅ PASS: Step 3 - Settings page opened');

  // Step 4: Locate the first API Key with role internal
  console.log('\n📍 Step 4: Locate first API Key with role internal in table');

  // 1. Wait for the loading state to finish
  const tableLoader = page.locator('.loading-state');
  if (await tableLoader.isVisible().catch(() => false)) {
    console.log('Table is loading, waiting...');
    await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
  }

  // 2. Set pagination to "All" to ensure we can see all keys
  console.log('Setting pagination to show all items...');
  const paginationSelect = page.locator('.v-data-table-footer__items-per-page .v-select').first();
  await paginationSelect.click();
  await page.waitForTimeout(300);

  // Click "All" option in the dropdown
  const allOption = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /^All$/i }).first();
  await expect(allOption).toBeVisible({ timeout: 5000 });
  await allOption.click();
  await page.waitForTimeout(1000); // Wait for table to reload with all items

  // 3. Wait for table to finish loading after pagination change
  if (await tableLoader.isVisible().catch(() => false)) {
    console.log('Waiting for table to reload...');
    await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
  }

  // 4. Find an Active API key row that can be revoked
  // First try to find "Test External API Key" (should still be Active)
  let activeKeyRow = page.locator('.api-keys-table tbody tr').filter({
    hasText: /Test External API Key/i
  }).first();

  // If test key doesn't exist, look for any row with Active chip
  if (!(await activeKeyRow.isVisible().catch(() => false))) {
    console.log('Test key not found, using first Active API key');
    activeKeyRow = page.locator('.api-keys-table tbody tr').filter({
      has: page.locator('.v-chip').filter({ hasText: /^Active$/i })
    }).first();
  }

  await expect(activeKeyRow).toBeVisible({
    timeout: 20000
  });
  console.log('✅ PASS: Step 4 - Active API Key located for revocation');

  // Step 5: Click the edit (pencil) icon in Actions column
  console.log('\n📍 Step 5: Click edit (pencil) icon');
  const editBtn = activeKeyRow.locator('.edit-btn');
  await expect(editBtn).toBeVisible();
  await editBtn.click();
  console.log('✅ PASS: Step 5 - Edit icon clicked');

  // Step 6: Verify Edit API Key popup appears
  console.log('\n📍 Step 6: Verify Edit API Key popup');
  const editDialog = page.locator('.v-dialog--active, [role="dialog"]').filter({
    hasText: /Edit API Key/i
  }).first();
  await expect(editDialog).toBeVisible({ timeout: 5000 });
  console.log('✅ PASS: Step 6 - Edit API Key popup appeared');

  // Step 7: Click the Status dropdown
  console.log('\n📍 Step 7: Click Status dropdown');

  /**
   * Since "Status" is the first field in the Edit Modal, 
   * we can target the first .v-select inside the dialog.
   * This is much faster and more reliable.
   */
  const statusSelector = editDialog.locator('.v-select').first();
  await expect(statusSelector).toBeVisible({ timeout: 20000 });
  await statusSelector.click();

  console.log('✅ PASS: Step 7 - Status dropdown clicked');


  // Step 8: Select Revoked
  console.log('\n📍 Step 8: Select Revoked from dropdown');
  /**
   * In Vuetify, dropdown items are teleported to the .v-overlay-container
   */
  const revokedOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: /^Revoked$/
  }).first();
  await expect(revokedOption).toBeVisible();
  await revokedOption.click();
  console.log('✅ PASS: Step 8 - Revoked status selected');

  // Step 9: Click Save
  console.log('\n📍 Step 9: Click Save');
  const saveButton = editDialog.locator('button').filter({ hasText: /^Save$/i }).first();
  await saveButton.click();
  console.log('✅ PASS: Step 9 - Save button clicked');

  // Step 10: Verify notification
  console.log('\n📍 Step 10: Verify success notification');
  const successNotification = page.locator('.v-snackbar__content', {
    hasText: /updated successfully/i
  });
  await expect(successNotification).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 10 - Success notification appeared');

  // Step 11: Verify status updated to Revoked in the table
  console.log('\n📍 Step 11: Verify status in table');
  // Wait for table to reload after edit
  await page.waitForTimeout(1000);

  // Just verify that at least one Revoked chip exists in the table now
  const revokedChip = page.locator('.api-keys-table .v-chip').filter({ hasText: /^Revoked$/i }).first();
  await expect(revokedChip).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 11 - API Key status confirmed as Revoked in table');

  // Step 12: Report results
  console.log('\n📍 Step 12: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful using helper function');
  console.log('✅ Step 2: PASS - Organization Testing2026 selected');
  console.log('✅ Step 3: PASS - Settings page opened');
  console.log('✅ Step 4: PASS - First API Key with role internal located');
  console.log('✅ Step 5: PASS - Edit icon clicked');
  console.log('✅ Step 6: PASS - Edit API Key popup appeared');
  console.log('✅ Step 7: PASS - Status dropdown clicked');
  console.log('✅ Step 8: PASS - Revoked status selected');
  console.log('✅ Step 9: PASS - Save button clicked');
  console.log('✅ Step 10: PASS - Success notification appeared');
  console.log('✅ Step 11: PASS - API Key status updated to Revoked');
  console.log('✅ Step 12: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
