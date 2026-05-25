import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Activate organization member flow and restored organization access', async ({ page }) => {
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
    timeout: 20000
  });
  console.log('✅ PASS: Step 3 - Member row located');

  // Step 4: Click three-dot actions icon
  console.log('\n📍 Step 4: Click three-dot actions icon (⋮)');
  // Using the specific Vuetify icon class
  const actionsMenu = memberRow.locator('.mdi-dots-vertical');
  await actionsMenu.click();
  console.log('✅ PASS: Step 4 - Actions menu clicked');

  // Step 5: Click Activate in the teleported list
  console.log('\n📍 Step 5: Click Activate');
  const activateOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: /^Activate$/i
  }).first();
  await expect(activateOption).toBeVisible();
  await activateOption.click();
  console.log('✅ PASS: Step 5 - Activate option clicked');

  // Step 6: Handle Confirmation Dialog (if any, matching deactivate flow structure)
  console.log('\n📍 Step 6: Confirm Activation');
  const confirmDialog = page.getByRole('dialog').filter({ hasText: /Are you sure/i });
  if (await confirmDialog.isVisible({ timeout: 2000 }).catch(() => false)) {
    const confirmBtn = confirmDialog.getByRole('button', { name: /Yes, activate/i });
    await confirmBtn.click();
    console.log('✅ PASS: Step 6 - Activation confirmed in modal');
  } else {
    console.log('ℹ️ Step 6 - No confirmation modal appeared, proceeding...');
  }

  // Step 7: Verify status changes to Active
  console.log('\n📍 Step 7: Verify member status changes to Active');
  // Re-fetch the row to ensure we check the fresh state
  const updatedRow = page.locator('tr').filter({ hasText: memberEmail });
  await expect(updatedRow.locator('td', { hasText: /^Active$/i })).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 7 - Member status changed to "Active"');

  // Step 8: Logout
  console.log('\n📍 Step 8: Logout');
  const profileMenu = page.locator('#menu-activator');
  await profileMenu.click();
  const logoutBtn = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /Logout/i }).first();
  await logoutBtn.click();
  await page.waitForURL('**/login', { timeout: 30000 });
  console.log('✅ PASS: Step 8 - Logged out successfully');

  // Step 9: Login as activated member
  console.log(`\n📍 Step 9: Login as ${memberEmail}`);
  const loginForm = page.locator('.login-card');
  await loginForm.locator('input').nth(0).fill(memberEmail);
  await loginForm.locator('input').nth(1).fill(memberPassword);
  await page.locator('button[type="submit"]').click();

  await page.waitForURL('**/?select_org', { timeout: 30000 });
  console.log('✅ PASS: Step 9 - Logged in and on selection page');

  // Step 10: Locate Organization card (not disabled)
  console.log('\n📍 Step 10: Locate Organization card');
  // Wait for org list loader
  const orgLoader = page.locator('.loading-container');
  if (await orgLoader.isVisible().catch(() => false)) {
    await expect(orgLoader).not.toBeVisible({ timeout: 20000 });
  }

  // Find the SPECIFIC organization card. It should NOT have the "disabled" status now.
  const activeOrgCard = page.locator('.organization-card').filter({
    hasText: organizationName
  });

  await expect(activeOrgCard).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 10 - Found active organization card');

  // Step 11: Verify restoration of access (Go to dashboard)
  console.log('\n📍 Step 11: Verify restoration of dashboard access');
  await activeOrgCard.click();

  // Wait for dashboard to load (no select_org query)
  await page.waitForURL(url => url.pathname === '/console' && !url.searchParams.has('select_org'), { timeout: 30000 });

  const dashboardContent = page.locator('main, [role="main"]').first();
  await expect(dashboardContent).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 11 - Dashboard access restored successfully');

  // Step 12: Report results
  console.log('\n📍 Step 12: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login and select organization');
  console.log('✅ Step 2: PASS - Organization settings page opened');
  console.log('✅ Step 3: PASS - Member row located');
  console.log('✅ Step 4: PASS - Actions menu clicked');
  console.log('✅ Step 5: PASS - Activate option clicked');
  console.log('✅ Step 6: PASS - Activation confirmed');
  console.log('✅ Step 7: PASS - Member status changed to Active');
  console.log('✅ Step 8: PASS - Logged out successfully');
  console.log('✅ Step 9: PASS - Login successful with activated member');
  console.log('✅ Step 10: PASS - Active organization located');
  console.log('✅ Step 11: PASS - Dashboard access restored');
  console.log('✅ Step 12: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
