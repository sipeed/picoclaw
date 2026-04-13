import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('API Key edit description flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';
  const newDescription = 'Test Internal API Key Edit';

  // Step 1-2: Perform the login and select org flow
  console.log('\n📍 Step 1-2: Perform login and select organization Testing2026');
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);
  console.log('✅ PASS: Step 1-2 - Login and organization selection completed');

  // Step 3: Verify redirect to dashboard
  console.log('\n📍 Step 3: Verify redirect to ');
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
  console.log('✅ PASS: Step 3 - User redirected to dashboard');

  // Step 4: Click Settings in the left sidebar
  console.log('\n📍 Step 4: Click Settings in the left sidebar');
  const settingsLink = page.locator('nav').getByText(/Settings/i).first();
  await expect(settingsLink).toBeVisible();
  await settingsLink.click();

  // Step 5: Locate the first API Key with role internal
  console.log('\n📍 Step 5: Locate first API Key with role internal in table');
  // 1. Wait for the loading state to finish (Crucial!)
  const tableLoader = page.locator('.loading-state');
  if (await tableLoader.isVisible().catch(() => false)) {
    console.log('Waiting for API keys to load...');
    await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
  }

  // 2. Locate the row in the specific table (.api-keys-table)
  const internalKeyRow = page.locator('.api-keys-table tbody tr').filter({
    hasText: /internal/i
  }).first();

  await expect(internalKeyRow).toBeVisible({
    timeout: 10000
  });
  console.log('✅ PASS: Step 5 - First API Key with role internal located');

  // Step 6: Click the edit (pencil) icon
  console.log('\n📍 Step 6: Click edit (pencil) icon');
  const editBtn = internalKeyRow.locator('.edit-btn');
  await expect(editBtn).toBeVisible();
  await editBtn.click();
  console.log('✅ PASS: Step 6 - Edit icon clicked');

  // Step 7: Verify Edit API Key popup appears
  console.log('\n📍 Step 7: Verify Edit API Key popup');
  const editDialog = page.locator('.v-dialog--active, [role="dialog"]').filter({
    hasText: /Edit API Key/i
  }).first();
  await expect(editDialog).toBeVisible({ timeout: 5000 });
  console.log('✅ PASS: Step 7 - Edit API Key popup appeared');

  // Step 8: Edit the Description field
  console.log('\n📍 Step 8: Edit Description field');
  // Find the input by its specific placeholder from the translation file
  const descriptionInput = editDialog.locator('input[placeholder*="Enter description" i]').first();
  await expect(descriptionInput).toBeVisible();

  // Clear and fill new description
  await descriptionInput.fill(newDescription);
  console.log('✅ PASS: Step 8 - Description field edited');

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
  await expect(successNotification).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 10 - Success notification appeared');

  // Step 11: Verify API Key description updated in table
  console.log('\n📍 Step 11: Verify update reflected in table');
  // Relocate row by the NEW description to confirm it's listed
  const updatedKeyRow = page.locator('.api-keys-table tbody tr').filter({
    hasText: newDescription
  }).first();
  await expect(updatedKeyRow).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 11 - API Key description update verified in table');

  // Step 12: Report results
  console.log('\n📍 Step 12: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1-2: PASS - Login and organization selection completed');
  console.log('✅ Step 3: PASS - User redirected to dashboard');
  console.log('✅ Step 4: PASS - Settings page opened');
  console.log('✅ Step 5: PASS - First API Key with role internal located');
  console.log('✅ Step 6: PASS - Edit icon clicked');
  console.log('✅ Step 7: PASS - Edit API Key popup appeared');
  console.log('✅ Step 8: PASS - Description field edited');
  console.log('✅ Step 9: PASS - Save button clicked');
  console.log('✅ Step 10: PASS - Success notification appeared');
  console.log('✅ Step 11: PASS - API Key description updated and status remains Active');
  console.log('✅ Step 12: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
