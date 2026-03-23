import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Change member role from Admin to Developer flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const memberEmail = 'heidi+1@intnt.ai';
  const organizationName = 'Testing2026';
  const newRole = 'developer';

  // Step 1 & 2: Login and select organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Organization in the left sidebar
  console.log('\n📍 Step 3: Click Organization in the left sidebar');
  const organizationLink = page.locator('nav').locator('a:has-text("Organization"), button:has-text("Organization"), [class*="sidebar"] >> text=Organization').first();
  await expect(organizationLink).toBeVisible();
  await organizationLink.click();
  await page.waitForTimeout(1500);
  console.log('✅ PASS: Step 3 - Organization settings page opened');

  // Step 4: Locate the row with email
  console.log(`\n📍 Step 4: Locate row with email ${memberEmail}`);
  // We locate the TR specifically to ensure we are in the right container
  const memberRow = page.locator('tr').filter({ hasText: memberEmail });
  await expect(memberRow).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 4 - Member row found');

  // Step 5: Click the three-dot actions menu (⋮)
  console.log('\n📍 Step 5: Click three-dot actions menu (⋮)');
  // The icon class is mdi-dots-vertical as per the vue file
  const actionsMenu = memberRow.locator('.mdi-dots-vertical').first();
  await expect(actionsMenu).toBeVisible();
  await actionsMenu.click();
  console.log('✅ PASS: Step 5 - Actions menu clicked');

  // Step 6: Click Change Role in the teleported menu
  console.log('\n📍 Step 6: Click Change Role');
  // Vuetify teleports the list to .v-overlay-container
  const changeRoleOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: /Change Role/i
  }).first();
  await expect(changeRoleOption).toBeVisible();
  await changeRoleOption.click();
  console.log('✅ PASS: Step 6 - Change Role option selected');

  // Step 7: Handle Change Role Dialog
  console.log('\n📍 Step 7: Open Change Role popup');
  const roleDialog = page.getByRole('dialog').filter({ hasText: /Change Role/i });
  await expect(roleDialog).toBeVisible();

  // Click the role dropdown
  const roleDropdown = roleDialog.locator('.v-select').first();
  await roleDropdown.click();

  console.log('✅ PASS: Step 7 - Select Role dropdown clicked');

  // Step 8: Select developer role
  console.log(`\n📍 Step 8: Select ${newRole} role`);
  const developerOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: new RegExp(`^${newRole}$`, 'i')
  }).first();
  await expect(developerOption).toBeVisible();
  await developerOption.click();
  console.log(`✅ PASS: Step 8 - ${newRole} role selected`);

  // Step 9: Click Apply (Note: It is "Apply" in pages.json, not "Save")
  console.log('\n📍 Step 9: Click Apply button');
  const applyButton = roleDialog.getByRole('button', { name: /Apply/i });
  await applyButton.click();
  console.log('✅ PASS: Step 9 - Apply clicked');

  // Step 10: Handle Confirmation Dialog
  console.log('\n📍 Step 10: Confirm change');
  const confirmDialog = page.getByRole('dialog').filter({ hasText: /Are you sure/i });
  await expect(confirmDialog).toBeVisible({ timeout: 5000 });

  const confirmButton = confirmDialog.getByRole('button', { name: /Yes, change/i });
  await confirmButton.click();
  console.log('✅ PASS: Step 10 - Confirmed in second modal');

  // Step 11: Verify notification
  console.log('\n📍 Step 11: Verify success notification');
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /Role changed successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 11 - Success notification verified');

  // Step 12: Verify update in table
  console.log(`\n📍 Step 12: Verify ${memberEmail} is now a ${newRole}`);
  // Find the row again and check the role cell
  const updatedRow = page.locator('tr').filter({ hasText: memberEmail });
  await expect(updatedRow.locator('td').filter({ hasText: new RegExp(newRole, 'i') })).toBeVisible();
  console.log(`✅ PASS: Step 12 - Role change verified in team list`);

  // Step 13: Report results
  console.log('\n📍 Step 13: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization testing2026 selected and redirected');
  console.log('✅ Step 3: PASS - Organization settings page opened');
  console.log('✅ Step 4: PASS - Member row with heidi+1@intnt.ai located');
  console.log('✅ Step 5: PASS - Three-dot actions menu clicked');
  console.log('✅ Step 6: PASS - Change Role option clicked');
  console.log('✅ Step 7: PASS - Select Role dropdown clicked');
  console.log('✅ Step 8: PASS - Developer role selected');
  console.log('✅ Step 9: PASS - Save button clicked');
  console.log('✅ Step 10: PASS - Confirmation dialog verified');
  console.log('✅ Step 11: PASS - Success notification verified');
  console.log('✅ Step 12: PASS - Member role updated to Developer');
  console.log('✅ Step 13: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
