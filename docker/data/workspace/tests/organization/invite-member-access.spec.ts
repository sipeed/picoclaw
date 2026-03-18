import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Invite member to organization flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const invitedEmail = 'heidi+111@intnt.ai';
  const invitedPassword = 'testing2026!!';
  const organizationName = 'Testing2026';
  const adminRole = 'admin';

  // Step 1 & 2: Login and select organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Organization in the left sidebar
  console.log('\n📍 Step 3: Click Organization in the left sidebar');
  const organizationLink = page.locator('nav').locator('a:has-text("Organization")').first();
  await expect(organizationLink).toBeVisible();
  await organizationLink.click();
  console.log('✅ PASS: Step 3 - Organization settings page opened');

  // Step 4: Click Add A Member
  console.log('\n📍 Step 4: Click Add A Member');
  const addMemberButton = page.getByRole('button', { name: /Add a Member/i }).first();
  await expect(addMemberButton).toBeVisible();
  await addMemberButton.click();
  console.log('✅ PASS: Step 4 - Add A Member modal opened');

  // Step 5: Enter email
  console.log(`\n📍 Step 5: Enter email ${invitedEmail}`);
  /**
   * Use getByRole('dialog') to find the modal, then find the input.
   * Scoping prevents interference with background elements.
   */
  const addMemberModal = page.getByRole('dialog').filter({ hasText: /Add a Member/i });
  await expect(addMemberModal).toBeVisible({ timeout: 10000 });

  const memberEmailInput = addMemberModal.locator('input').first();
  await expect(memberEmailInput).toBeVisible();
  await memberEmailInput.fill(invitedEmail);
  console.log(`✅ PASS: Step 5 - Email entered: ${invitedEmail}`);

  // Step 6: Select role admin
  console.log(`\n📍 Step 6: Select role ${adminRole}`);
  const roleSelector = addMemberModal.locator('.v-select').first();
  await roleSelector.click();

  // Vuetify teleports menus to .v-overlay-container
  const adminOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: new RegExp(`^${adminRole}$`, 'i')
  }).first();
  await expect(adminOption).toBeVisible({ timeout: 5000 });
  await adminOption.click();
  console.log(`✅ PASS: Step 6 - Role ${adminRole} selected`);

  // Step 7: Click Add
  console.log('\n📍 Step 7: Click Add');
  const addButton = addMemberModal.getByRole('button', { name: /^Add$/i }).first();
  await expect(addButton).toBeVisible();
  await addButton.click();
  console.log('✅ PASS: Step 7 - Add button clicked');

  // Step 8: Verify notification
  console.log('\n📍 Step 8: Verify notification "Member invite sent successfully"');
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /invite sent successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 8 - Success notification verified');

  // Step 9: Verify email in list
  console.log(`\n📍 Step 9: Verify ${invitedEmail} in Organization Team list`);
  const memberInList = page.locator('.organization-table td', { hasText: invitedEmail });
  await expect(memberInList).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 9 - Invited email verified in team list');

  // Step 10: Logout
  console.log('\n📍 Step 10: Logout');
  const profileMenu = page.locator('#menu-activator');
  await profileMenu.click();
  const logoutBtn = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /Logout/i }).first();
  await logoutBtn.click();
  await page.waitForURL('**/login', { timeout: 15000 });
  console.log('✅ PASS: Step 10 - Logged out');

  // Step 11-13: Login with invited user and verify org
  console.log(`\n📍 Step 11: Login with ${invitedEmail}`);
  const loginEmailInput = page.locator('.v-form').locator('input').nth(0);
  const loginPassInput = page.locator('.v-form').locator('input').nth(1);
  await loginEmailInput.fill(invitedEmail);
  await loginPassInput.fill(invitedPassword);
  await page.locator('button[type="submit"]').click();

  await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 15000 });
  console.log('✅ PASS: Step 11/12 - Login successful and redirected to selection');

  // Verify org visibility (Step 13)
  console.log(`\n📍 Step 13: Verify ${organizationName} visible in selector`);
  // Wait for loader to disappear if it's there
  if (await page.locator('.loading-container').isVisible().catch(() => false)) {
    await expect(page.locator('.loading-container')).not.toBeVisible({ timeout: 15000 });
  }
  const orgCard = page.locator('.organization-card').filter({ hasText: organizationName });
  await expect(orgCard).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 13 - Organization visible');

  // Step 14/15: Select and land on dashboard
  console.log('\n📍 Step 14: Select organization');
  await orgCard.click();
  await page.waitForURL(url => url.pathname === '/' && !url.searchParams.has('select_org'), { timeout: 15000 });
  console.log('✅ PASS: Step 14/15 - Redirection and selection successful');


  // Step 16: Report results
  console.log('\n📍 Step 16: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization testing2026 selected and redirected');
  console.log('✅ Step 3: PASS - Organization settings page opened');
  console.log('✅ Step 4: PASS - Add A Member button clicked');
  console.log('✅ Step 5: PASS - Email entered: heidi+111@intnt.ai');
  console.log('✅ Step 6: PASS - Role admin selected');
  console.log('✅ Step 7: PASS - Add button clicked');
  console.log('✅ Step 8: PASS - Success notification verified');
  console.log('✅ Step 9: PASS - Invited email verified in team list');
  console.log('✅ Step 10: PASS - Logged out successfully');
  console.log('✅ Step 11: PASS - Login successful with heidi+111@intnt.ai');
  console.log('✅ Step 12: PASS - Redirected to organization selection page');
  console.log('✅ Step 13: PASS - Organization testing2026 visible');
  console.log('✅ Step 14: PASS - Organization testing2026 selected');
  console.log('✅ Step 15: PASS - Redirected to dashboard');
  console.log('✅ Step 16: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
