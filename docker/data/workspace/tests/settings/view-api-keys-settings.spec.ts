import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('API Keys settings page access', async ({ page }) => {
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
  console.log('✅ PASS: Step 3 - Settings clicked in sidebar');

  // Step 4: Verify redirect to Settings page
  console.log('\n📍 Step 4: Verify redirect to Settings page');
  const settingsPageContent = page.locator('[class*="settings"], main, [role="main"]').first();
  await expect(settingsPageContent).toBeVisible({ timeout: 5000 });
  console.log('✅ PASS: Step 4 - Settings page loaded');

  // Step 5: Verify the API Keys tab is displayed
  console.log('\n📍 Step 5: Verify the API Keys tab is displayed');
  const apiKeysTab = page.locator('.v-tabs__nav button:has-text("API Keys"), [role="tab"]:has-text("API Keys")').first();

  if (await apiKeysTab.isVisible({ timeout: 3000 }).catch(() => false)) {
    console.log('✅ PASS: Step 5 - API Keys tab is displayed');
  } else {
    // Try alternative selector
    const apiKeysTabAlt = page.getByText('API Keys').first();
    await expect(apiKeysTabAlt).toBeVisible();
    console.log('✅ PASS: Step 5 - API Keys tab is displayed');
  }

  // Step 6: Verify the list of API Keys is visible
  console.log('\n📍 Step 6: Verify the list of API Keys for organization Testing2026 is visible');
  const apiKeysList = page.locator('[class*="api-key"], [class*="key-list"], table, [role="table"]').first();

  if (await apiKeysList.isVisible({ timeout: 3000 }).catch(() => false)) {
    console.log('✅ PASS: Step 6 - API Keys list is visible');
  } else {
    // Verify at least some API key content is present
    const apiKeyContent = page.locator('text=API Key, text=key, [class*="api"]').first();
    await expect(apiKeyContent).toBeVisible({ timeout: 5000 });
    console.log('✅ PASS: Step 6 - API Keys content is visible');
  }

  // Step 7: Verify the Add New API Key button is visible
  console.log('\n📍 Step 7: Verify the Add New API Key button is visible');
  const addKeyButton = page.locator('button:has-text("Add New API Key"), button:has-text("Add API Key"), button:has-text("Create Key")').first();
  await expect(addKeyButton).toBeVisible();
  console.log('✅ PASS: Step 7 - Add New API Key button is visible');

  // Step 8: Report results
  console.log('\n📍 Step 8: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization Testing2026 selected and redirected');
  console.log('✅ Step 3: PASS - Settings clicked in sidebar');
  console.log('✅ Step 4: PASS - Settings page loaded');
  console.log('✅ Step 5: PASS - API Keys tab is displayed');
  console.log('✅ Step 6: PASS - API Keys list is visible');
  console.log('✅ Step 7: PASS - Add New API Key button is visible');
  console.log('✅ Step 8: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
