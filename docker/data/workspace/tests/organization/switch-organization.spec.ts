import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Organization switching flow', async ({ page }) => {
  test.setTimeout(180000); // 3 minutes timeout
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const firstOrganization = 'Testing';
  const secondOrganization = 'Testing2026!';

  // Step 1 & 2: Login and select first organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, firstOrganization);

  // Step 3: Verify the user is logged in under organization Testing
  console.log('\n📍 Step 3: Verify user is logged in under organization Testing');
  const orgTrigger = page.locator('.org-dropdown-trigger');
  await expect(orgTrigger).toBeVisible({ timeout: 20000 });
  await expect(orgTrigger).toContainText(firstOrganization);
  console.log('✅ PASS: Step 3 - User verified logged in under organization Testing');

  // Step 4: Click the org switcher trigger to open the dropdown
  console.log('\n📍 Step 4: Click the organization selector on the left sidebar');
  await orgTrigger.click();
  console.log('✅ PASS: Step 4 - Organization selector clicked on left sidebar');

  // Step 5: Verify both organizations are visible in the dropdown
  console.log('\n📍 Step 5: Verify all available organizations are visible');
  const firstOrgInList = page.locator('.org-dropdown-item').filter({ hasText: firstOrganization }).first();
  const secondOrgInList = page.locator('.org-dropdown-item').filter({ hasText: secondOrganization }).first();

  await expect(firstOrgInList).toBeVisible({ timeout: 20000 });
  await expect(secondOrgInList).toBeVisible({ timeout: 20000 });
  console.log(`✅ PASS: Step 5 - Organizations visible: ${firstOrganization}, ${secondOrganization}`);

  // Step 6: Click the second organization to switch
  console.log(`\n📍 Step 6: Switching to organization: ${secondOrganization}`);
  await secondOrgInList.click();

  await page.waitForLoadState('networkidle');
  console.log(`✅ PASS: Step 6 - Organization ${secondOrganization} selected`);

  // Step 7: Verify active organization switched to Testing2026!
  console.log('\n📍 Step 7: Verify active organization is Testing2026!');
  await expect(page.locator('.org-dropdown-trigger')).toContainText(secondOrganization, { timeout: 30000 });
  await expect(page).not.toHaveURL(/login|select_org/);
  console.log('✅ PASS: Step 7 - Active organization is Testing2026!');

  // Step 8: Report results
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization Testing selected');
  console.log('✅ Step 3: PASS - User verified logged in under organization Testing');
  console.log('✅ Step 4: PASS - Organization selector clicked on sidebar');
  console.log('✅ Step 5: PASS - All available organizations visible');
  console.log('✅ Step 6: PASS - Organization Testing2026! selected');
  console.log('✅ Step 7: PASS - Active organization is Testing2026!');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
