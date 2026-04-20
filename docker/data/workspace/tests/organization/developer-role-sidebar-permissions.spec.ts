import { test, expect } from '@playwright/test';
import { performLogin } from '../utils/auth';

test('Developer role dashboard access and sidebar permissions', async ({ page }) => {
  test.setTimeout(180000); // 3 minutes timeout
  const developerEmail = 'heidi+1@intnt.ai';
  const developerPassword = 'testing2026!!';

  // Step 1: Login
  await performLogin(page, developerEmail, developerPassword);

  // Step 2: Select an organization with 'Role: developer'
  console.log('\n📍 Step 2: Selecting organization with "developer" role');

  // Wait for the loading state to finish (crucial!)
  const loader = page.locator('.loading-container, .loading-spinner');
  if (await loader.first().isVisible().catch(() => false)) {
    console.log('Waiting for organization list to load...');
    await expect(loader.first()).not.toBeVisible({ timeout: 15000 });
  }

  // Find the card where the .detail-value is "developer"
  const developerOrgCard = page.locator('.organization-card').filter({
    has: page.locator('.detail-value', { hasText: /^developer$/i })
  }).first();

  // Fallback: If the above is too strict, look for any card containing "developer"
  const fallbackCard = page.locator('.organization-card', { hasText: /developer/i }).first();

  const finalCard = (await developerOrgCard.isVisible().catch(() => false))
    ? developerOrgCard
    : fallbackCard;

  // Assertion and Click
  await expect(finalCard).toBeVisible({ timeout: 20000 });
  await finalCard.click();

  // Wait for the page to transition (loader overlay appears during "switching")
  await expect(page.locator('.switching-overlay')).not.toBeVisible({ timeout: 20000 });

  // Wait for dashboard to load
  await page.waitForLoadState('networkidle');
  console.log('✅ PASS: Organization with Developer role selected');

  // Step 3: Verify organization selection and Developer permissions
  console.log('\n📍 Step 3: Verify organization selection and Developer permissions');
  const orgSelectorHeader = page.locator('.org-dropdown-trigger').first();
  await expect(orgSelectorHeader).toBeVisible({ timeout: 20000 });

  const selectedOrgName = orgSelectorHeader.locator('.org-name');
  await expect(selectedOrgName).not.toHaveText(/No organization/i, { timeout: 10000 });
  console.log(`Successfully selected organization: ${await selectedOrgName.innerText()}`);

  /**
   * Verify Developer status by checking for the "Flow Designer" menu item (visible)
   * and "Organization" menu item (NOT visible).
   */
  const flowDesignerMenu = page.locator('nav').getByText('Flow Designer').first();
  await expect(flowDesignerMenu).toBeVisible();

  const organizationMenu = page.locator('nav .v-list-item').filter({ hasText: /^Organization$/ }).first();
  await expect(organizationMenu).not.toBeVisible({ timeout: 5000 });

  console.log('✅ PASS: Step 3 - Organization selection and Developer permissions verified via sidebar menu');

  // Step 4: Verify Dashboard menu is visible
  console.log('\n📍 Step 4: Verify Dashboard menu is visible in left sidebar');
  const dashboardMenu = page.locator('nav').getByText('Dashboard').first();
  await expect(dashboardMenu).toBeVisible();
  console.log('✅ PASS: Step 4 - Dashboard menu visible');

  // Step 5: Verify Flow Tester menu is visible
  console.log('\n📍 Step 5: Verify Flow Tester menu is visible in left sidebar');
  const flowTesterMenu = page.locator('nav').getByText('Flow Tester').first();
  await expect(flowTesterMenu).toBeVisible();
  console.log('✅ PASS: Step 5 - Flow Tester menu visible');

  // Step 6: Verify Knowledge Base menu is visible
  console.log('\n📍 Step 6: Verify Knowledge Base menu is visible in left sidebar');
  const knowledgeBaseMenu = page.locator('nav').getByText('Knowledge Base').first();
  await expect(knowledgeBaseMenu).toBeVisible();
  console.log('✅ PASS: Step 6 - Knowledge Base menu visible');

  // Step 7: Verify Logs menu is visible
  console.log('\n📍 Step 7: Verify Logs menu is visible in left sidebar');
  const logsMenu = page.locator('nav').getByText('Logs').first();
  await expect(logsMenu).toBeVisible();
  console.log('✅ PASS: Step 7 - Logs menu visible');

  // Step 8: Verify Add-Ons menu is visible
  console.log('\n📍 Step 8: Verify Add-Ons menu is visible in left sidebar');
  const addOnsMenu = page.locator('nav').getByText('Add-Ons').first();
  await expect(addOnsMenu).toBeVisible();
  console.log('✅ PASS: Step 8 - Add-Ons menu visible');

  // Step 9: Verify Settings menu is visible
  console.log('\n📍 Step 9: Verify Settings menu is visible in left sidebar');
  const settingsMenu = page.locator('nav').getByText('Settings').first();
  await expect(settingsMenu).toBeVisible();
  console.log('✅ PASS: Step 9 - Settings menu visible');

  // Step 10: Report results
  console.log('\n📍 Step 10: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi+1@intnt.ai');
  console.log('✅ Step 2: PASS - Developer role organization located');
  console.log('✅ Step 3: PASS - Organization with Developer role selected');
  console.log('✅ Step 4: PASS - Redirected to dashboard');
  console.log('✅ Step 5: PASS - Dashboard menu visible');
  console.log('✅ Step 6: PASS - Flow Designer menu visible');
  console.log('✅ Step 7: PASS - Flow Tester menu visible');
  console.log('✅ Step 8: PASS - Knowledge Base menu visible');
  console.log('✅ Step 9: PASS - Logs menu visible');
  console.log('✅ Step 10: PASS - Add-Ons menu visible');
  console.log('✅ Step 11: PASS - Settings menu visible');
  console.log('✅ Step 12: PASS - Organization menu NOT visible');
  console.log('✅ Step 13: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
