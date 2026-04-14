import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Deactivate organization member flow and disabled organization access', async ({ page }) => {
  test.setTimeout(180000); // 3 minutes timeout
  const adminEmail = 'heidi@intnt.ai';
  const adminPassword = 'testing2026!';
  const memberEmail = 'heidi+1@intnt.ai';
  const memberPassword = 'testing2026!!';
  const organizationName = 'Testing2026!';

  // Step 1: Login and select organization
  await loginAndSelectOrg(page, adminEmail, adminPassword, organizationName);

  // Step 2: Click Organization in left sidebar
  console.log('\n📍 Step 2: Click Organization in left sidebar');
  const organizationLink = page.locator('nav').locator('a').filter({ hasText: /^Organization$/i });
  await expect(organizationLink).toBeVisible();
  await organizationLink.click();
  console.log('✅ PASS: Step 2 - Organization settings page opened');

  // Step 3: Locate member row
  console.log(`\n📍 Step 3: Locate member row with email ${memberEmail}`);
  // 1. Wait for table to finish loading
  const tableLoader = page.locator('.v-data-table-server .v-progress-linear');
  if (await tableLoader.isVisible().catch(() => false)) {
    await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
  }

  // 2. Find row by email
  const memberRow = page.locator('tr').filter({ hasText: memberEmail });
  await expect(memberRow).toBeVisible({
    timeout: 10000
  });
  console.log('✅ PASS: Step 3 - Member row located');

  // Step 4: Click three-dot actions icon
  console.log('\n📍 Step 4: Click three-dot actions icon (⋮)');
  // Using the specific Vuetify icon class
  const actionsMenu = memberRow.locator('.mdi-dots-vertical');
  await actionsMenu.click();
  console.log('✅ PASS: Step 4 - Actions menu clicked');

  // Step 5: Click Deactivate in the teleported list
  console.log('\n📍 Step 5: Click Deactivate');
  const deactivateOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: /^Deactivate$/i
  }).first();
  await expect(deactivateOption).toBeVisible();
  await deactivateOption.click();
  console.log('✅ PASS: Step 5 - Deactivate option clicked');

  // Step 6: Handle Confirmation Dialog
  console.log('\n📍 Step 6: Confirm Deactivation');
  const confirmDialog = page.getByRole('dialog').filter({ hasText: /Are you sure/i });
  await expect(confirmDialog).toBeVisible({ timeout: 5000 });

  // Click the "Yes, deactivate" button as defined in pages.json
  const confirmBtn = confirmDialog.getByRole('button', { name: /Yes, deactivate/i });
  await confirmBtn.click();
  console.log('✅ PASS: Step 6 - Deactivation confirmed in modal');

  // Step 7: Verify status changes to Disabled (Not Deactivated)
  console.log('\n📍 Step 7: Verify member status changes to Disabled');
  // Re-fetch the row to ensure we check the fresh state
  const updatedRow = page.locator('tr').filter({ hasText: memberEmail });
  await expect(updatedRow.locator('td', { hasText: /^Disabled$/i })).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 7 - Member status changed to "Disabled"');

  // Step 8: Logout
  console.log('\n📍 Step 8: Logout');
  const profileMenu = page.locator('#menu-activator');
  await profileMenu.click();
  const logoutBtn = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /Logout/i }).first();
  await logoutBtn.click();
  await page.waitForURL('**/login', { timeout: 15000 });
  console.log('✅ PASS: Step 8 - Logged out successfully');

  // Step 9: Login as deactivated member
  console.log(`\n📍 Step 9: Login as ${memberEmail}`);
  const loginForm = page.locator('.login-card');
  await loginForm.locator('input').nth(0).fill(memberEmail);
  await loginForm.locator('input').nth(1).fill(memberPassword);
  await page.locator('button[type="submit"]').click();

  await page.waitForURL('**/?select_org', { timeout: 15000 });
  console.log('✅ PASS: Step 9 - Logged in and on selection page');

  // Step 10: Locate Disabled Organization card
  console.log('\n📍 Step 10: Locate Disabled Organization card');
  // Wait for org list loader
  const orgLoader = page.locator('.loading-container');
  if (await orgLoader.isVisible().catch(() => false)) {
    await expect(orgLoader).not.toBeVisible({ timeout: 10000 });
  }

  // Find the SPECIFIC organization card that shows "disabled"
  const disabledOrgCard = page.locator('.organization-card').filter({
    hasText: organizationName
  }).filter({ hasText: /disabled/i });

  await expect(disabledOrgCard).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 10 - Found organization with disabled tag');

  // Step 11: Verify access denied snackbar
  console.log('\n📍 Step 11: Verify access denied snackbar');
  await disabledOrgCard.click();

  const errorSnackbar = page.locator('.v-snackbar__content', {
    hasText: /Active organization user not found/i
  });
  await expect(errorSnackbar).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 11 - Error notification appeared correctly');

  // Step 12: Report results
  console.log('\n📍 Step 12: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login and select organization');
  console.log('✅ Step 2: PASS - Organization settings page opened');
  console.log('✅ Step 3: PASS - Member row located');
  console.log('✅ Step 4: PASS - Actions menu clicked');
  console.log('✅ Step 5: PASS - Deactivate option clicked');
  console.log('✅ Step 6: PASS - Confirmation dialog confirmed');
  console.log('✅ Step 7: PASS - Member status changed to Disabled');
  console.log('✅ Step 8: PASS - Logged out successfully');
  console.log('✅ Step 9: PASS - Login successful with deactivated member');
  console.log('✅ Step 10: PASS - Disabled organization located');
  console.log('✅ Step 11: PASS - Error notification appeared');
  console.log('✅ Step 12: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
