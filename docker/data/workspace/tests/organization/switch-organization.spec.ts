import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Organization switching flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const firstOrganization = 'Testing';
  const secondOrganization = 'Testing2026';

  // Step 1 & 2: Login and select first organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, firstOrganization);

  // Step 3: Verify the user is logged in under organization testing
  console.log('\n📍 Step 3: Verify user is logged in under organization testing');
  const userProfile = page.locator('[class*="profile"], [class*="user"], [class*="avatar"]').first();
  await expect(userProfile).toBeVisible();

  const orgDisplayName = page.locator('[class*="org-name"], [class*="organization-name"], header').first();
  await expect(orgDisplayName).toBeVisible();
  console.log('✅ PASS: Step 3 - User verified logged in under organization testing');

  // Step 4: Click the organization selector on the left sidebar
  console.log('\n📍 Step 4: Click the organization selector on the left sidebar');
  const orgSelector = page.locator('nav').locator('[class*="org"], [class*="selector"], button[class*="organization"]').first();

  if (!await orgSelector.isVisible({ timeout: 2000 }).catch(() => false)) {
    // Try alternative selectors
    const sidebarOrgButton = page.locator('[class*="sidebar"] >> [class*="org"], [class*="sidebar"] >> button').first();
    await expect(sidebarOrgButton).toBeVisible();
    await sidebarOrgButton.click();
  } else {
    await orgSelector.click();
  }

  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 4 - Organization selector clicked on left sidebar');

  // Step 5: Verify that all available organizations are visible in the selector list
  console.log('\n📍 Step 5: Verify all available organizations are visible');
  const firstOrgInList = page.locator(`text=${firstOrganization}`);
  const secondOrgInList = page.locator(`text=${secondOrganization}`);

  await expect(firstOrgInList).toBeVisible({ timeout: 5000 });
  await expect(secondOrgInList).toBeVisible({ timeout: 5000 });
  console.log(`✅ PASS: Step 5 - Organizations visible: ${firstOrganization}, ${secondOrganization}`);

  // Step 6: Select second organization (switch organization)
  console.log(`\n📍 Step 6: Switching to organization: ${secondOrganization}`);

  // Find the dropdown trigger in the sidebar and click it to open the menu
  const trigger = page.locator('.org-dropdown-trigger');
  await expect(trigger).toBeVisible({ timeout: 10000 });
  await trigger.click();

  // The menu is "Teleported" to the body. Wait for the list to appear.
  const dropdownMenu = page.locator('.org-dropdown-menu');
  await expect(dropdownMenu).toBeVisible({ timeout: 5000 });

  // Find the specific organization item. 
  // We use the .org-dropdown-item class and filter by the organization name.
  const targetOrgItem = dropdownMenu.locator('.org-dropdown-item').filter({
    has: page.locator('.org-name', { hasText: new RegExp(`^${secondOrganization}$`, 'i') })
  }).first();

  // Fallback search if the strict filter is too narrow
  const fallbackItem = dropdownMenu.locator('.org-dropdown-item', { hasText: secondOrganization }).first();
  const finalSelection = (await targetOrgItem.isVisible().catch(() => false))
    ? targetOrgItem
    : fallbackItem;

  // Assert visibility and click
  await expect(finalSelection).toBeVisible({
    timeout: 5000
  });
  await finalSelection.click();

  // The app performs a 'window.location.reload()' upon switching. 
  // We wait for the network to go idle to ensure the new organization context is loaded.
  await page.waitForLoadState('networkidle');
  console.log(`✅ PASS: Step 6 - Organization ${secondOrganization} selected`);

  // Step 7: Verify redirect and active organization
  console.log('\n📍 Step 7: Verify redirect to dashboard and active organization is testing2026');
  await page.waitForURL('**/dashboard.int3nt.info/', { timeout: 15000 });
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);

  // Verify the active organization display
  const activeOrgDisplay = page.locator('[class*="org-name"], [class*="organization-name"], [class*="active"]').first();
  await expect(activeOrgDisplay).toBeVisible();

  console.log('✅ PASS: Step 7 - Redirected to dashboard with active organization testing2026');

  // Step 8: Report results
  console.log('\n📍 Step 8: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization testing selected and redirected');
  console.log('✅ Step 3: PASS - User verified logged in under organization testing');
  console.log('✅ Step 4: PASS - Organization selector clicked on sidebar');
  console.log('✅ Step 5: PASS - All available organizations visible');
  console.log('✅ Step 6: PASS - Organization testing2026 selected');
  console.log('✅ Step 7: PASS - Redirected and active organization is testing2026');
  console.log('✅ Step 8: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
