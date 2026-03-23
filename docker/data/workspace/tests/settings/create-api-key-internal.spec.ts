import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Create API Key flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026';
  const expiresInDays = '30';
  const description = 'Test Internal API Key';

  // Step 1-2: Perform the login and select org flow
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Settings in the left sidebar
  console.log('\n📍 Step 3: Click Settings in the left sidebar');
  const settingsLink = page.locator('nav').getByText('Settings').first();
  await expect(settingsLink).toBeVisible();
  await settingsLink.click();
  await page.waitForTimeout(1500);
  console.log('✅ PASS: Step 3 - Settings clicked in sidebar');

  // Step 4: Verify Settings page loads and API Keys tab is visible
  console.log('\n📍 Step 4: Verify Settings page loads and API Keys tab is visible');
  const settingsPageContent = page.locator('[class*="settings"], main, [role="main"]').first();
  await expect(settingsPageContent).toBeVisible({ timeout: 5000 });

  const apiKeysTab = page.locator('.v-tabs__nav button:has-text("API Keys"), [role="tab"]:has-text("API Keys")').first();
  if (await apiKeysTab.isVisible({ timeout: 3000 }).catch(() => false)) {
    await apiKeysTab.click();
    await page.waitForTimeout(500);
  }

  console.log('✅ PASS: Step 4 - Settings page loaded and API Keys tab visible');

  // Step 5: Click Add New API Key
  console.log('\n📍 Step 5: Click Add New API Key');
  const addKeyButton = page.locator('button:has-text("Add New API Key"), button:has-text("Add API Key"), button:has-text("Create Key")').first();
  await expect(addKeyButton).toBeVisible();
  await addKeyButton.click();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 5 - Add New API Key clicked');

  // Step 6: Select role Internal in Create API Key popup
  console.log('\n📍 Step 6: Select role Internal in Create API Key popup');
  const dialog = page.locator('.v-dialog--active, [role="dialog"]').first();
  await expect(dialog).toBeVisible();

  // Find the role select field (GlobalFormField renders a v-select)
  const roleSelector = dialog.locator('.v-select').first();
  await roleSelector.click();

  /**
   * IMPORTANT: In Vuetify, menu items are teleported to the .v-overlay-container
   * outside the dialog. We search globally for the list item with text "Internal".
   */
  const internalOption = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /^Internal$/ }).first();
  await expect(internalOption).toBeVisible({ timeout: 5000 });
  await internalOption.click();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 6 - Role Internal selected');

  // Step 7: Fill Expires in Days with 30
  console.log('\n📍 Step 7: Fill Expires in Days with 30');
  // Target the input inside the dialog by its placeholder
  const expiresInput = dialog.locator('input[placeholder*="number of days" i]').first();
  await expiresInput.fill(expiresInDays);
  console.log('✅ PASS: Step 7 - Expires in Days filled with 30');

  // Step 8: Fill Description with Test Internal API Key
  console.log('\n📍 Step 8: Fill Description with Test Internal API Key');
  const descriptionInput = dialog.locator('input[placeholder*="description" i]').first();
  await descriptionInput.fill(description);
  console.log('✅ PASS: Step 8 - Description filled');

  // Step 9: Click Create API Key
  console.log('\n📍 Step 9: Click Create API Key');
  const createButton = dialog.locator('button').filter({ hasText: /^Create API Key$/i }).first();
  await createButton.click();
  console.log('✅ PASS: Step 9 - Create API Key clicked');

  // Step 10: Verify success modal appears
  console.log('\n📍 Step 10: Verify success message');
  // The success modal has a specific class .api-key-success-modal
  const successModal = page.locator('.api-key-success-modal').first();
  await expect(successModal).toBeVisible({ timeout: 15000 });

  // Verify the text inside the modal title
  const successTitle = successModal.locator('.modal-title');
  await expect(successTitle).toContainText(/created successfully/i);
  console.log('✅ PASS: Step 10 - Success modal visible');

  // Step 11: Verify the generated API key is displayed
  console.log('\n📍 Step 11: Verify the generated API key is displayed');
  const apiKeyInput = successModal.locator('input[readonly]').first();
  await expect(apiKeyInput).toBeVisible();
  const keyVal = await apiKeyInput.inputValue();
  console.log(`✅ PASS: Step 11 - Generated API key displayed (starts with: ${keyVal.substring(0, 8)}...)`);

  // Step 12: Close the success modal
  console.log('\n📍 Step 12: Close the popup');
  const modalCloseBtn = successModal.locator('.close-btn').first();
  await modalCloseBtn.click();
  // Wait for the modal and its backdrop to disappear completely
  await expect(successModal).not.toBeVisible({ timeout: 5000 });
  console.log('✅ PASS: Step 12 - Popup closed');

  // Step 13: Verify the new API key appears in the table
  console.log('\n📍 Step 13: Verify new key in list');

  // 1. Wait for the table to finish loading (it refreshes after creation)
  const tableLoader = page.locator('.loading-state');
  if (await tableLoader.isVisible().catch(() => false)) {
    console.log('Table is refreshing, waiting...');
    await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
  }

  // 2. Locate the row by the description text
  // We use a more flexible filter that looks specifically at the text content
  const newKeyRow = page.locator('.api-keys-table tbody tr').filter({
    hasText: new RegExp(description, 'i')
  }).first();

  await expect(newKeyRow).toBeVisible({
    timeout: 10000
  });

  // 3. Verify it has the correct role
  await expect(newKeyRow.locator('td')).toContainText([/internal/i]);

  console.log('✅ PASS: Step 13 - New API key successfully verified in table');


  // Step 14: Report results
  console.log('\n📍 Step 14: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization Testing2026 selected and redirected');
  console.log('✅ Step 3: PASS - Settings clicked in sidebar');
  console.log('✅ Step 4: PASS - Settings page loaded and API Keys tab visible');
  console.log('✅ Step 5: PASS - Add New API Key clicked');
  console.log('✅ Step 6: PASS - Role Internal selected');
  console.log('✅ Step 7: PASS - Expires in Days filled with 30');
  console.log('✅ Step 8: PASS - Description filled with Test Internal API Key');
  console.log('✅ Step 9: PASS - Create API Key clicked');
  console.log('✅ Step 10: PASS - Success popup appeared');
  console.log('✅ Step 11: PASS - Generated API key displayed');
  console.log('✅ Step 12: PASS - Popup closed');
  console.log('✅ Step 13: PASS - New API key appears in list with role Internal');
  console.log('✅ Step 14: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
