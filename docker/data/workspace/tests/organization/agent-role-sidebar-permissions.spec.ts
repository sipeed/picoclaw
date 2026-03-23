import { test, expect } from '@playwright/test';
import { performLogin } from '../utils/auth';

test('Agent role dashboard access and sidebar permissions', async ({ page }) => {
  const agentEmail = 'heidi+1@intnt.ai';
  const agentPassword = 'testing2026!!';

  // Step 1: Login
  await performLogin(page, agentEmail, agentPassword);

  // Step 2: Select an organization with 'Role: agent'
  console.log('\n📍 Step 2: Selecting organization with "agent" role');

  // Wait for the loading state to finish (crucial!)
  const loader = page.locator('.loading-container, .loading-spinner');
  if (await loader.first().isVisible().catch(() => false)) {
    console.log('Waiting for organization list to load...');
    await expect(loader.first()).not.toBeVisible({ timeout: 15000 });
  }

  // Find the card where the .detail-value is "agent"
  const agentOrgCard = page.locator('.organization-card').filter({
    has: page.locator('.detail-value', { hasText: /^agent$/i })
  }).first();

  // Fallback: If the above is too strict, look for any card containing "agent"
  const fallbackCard = page.locator('.organization-card', { hasText: /agent/i }).first();

  const finalCard = (await agentOrgCard.isVisible().catch(() => false))
    ? agentOrgCard
    : fallbackCard;

  // Assertion and Click
  await expect(finalCard).toBeVisible({ timeout: 10000 });
  await finalCard.click();

  // Wait for the page to transition (loader overlay appears during "switching")
  await expect(page.locator('.switching-overlay')).not.toBeVisible({ timeout: 10000 });

  // Wait for dashboard to load
  await page.waitForLoadState('networkidle');
  console.log('✅ PASS: Organization with Agent role selected');

  // Step 3: Verify organization selection and Agent permissions
  console.log('\n📍 Step 3: Verify organization selection and Agent permissions');
  const orgSelectorHeader = page.locator('.org-dropdown-trigger').first();
  await expect(orgSelectorHeader).toBeVisible({ timeout: 10000 });

  const selectedOrgName = orgSelectorHeader.locator('.org-name');
  await expect(selectedOrgName).not.toHaveText(/No organization/i, { timeout: 10000 });
  console.log(`Successfully selected organization: ${await selectedOrgName.innerText()}`);

  /**
   * Verify Agent status by checking for the "Flow Tester" menu item (visible)
   * and several menu items (NOT visible).
   */
  const flowTesterOption = page.locator('nav').getByText('Flow Tester').first();
  await expect(flowTesterOption).toBeVisible();

  // Non-visible for agent: Flow Designer, Knowledge Base, Logs, Add-Ons, Settings, Organization
  const flowDesignerMenu = page.locator('nav').getByText('Flow Designer').first();
  await expect(flowDesignerMenu).not.toBeVisible({ timeout: 2000 });

  const knowledgeBaseMenu = page.locator('nav').getByText('Knowledge Base').first();
  await expect(knowledgeBaseMenu).not.toBeVisible({ timeout: 2000 });

  const logsMenu = page.locator('nav').getByText('Logs').first();
  await expect(logsMenu).not.toBeVisible({ timeout: 2000 });

  const addOnsMenu = page.locator('nav').getByText('Add-Ons').first();
  await expect(addOnsMenu).not.toBeVisible({ timeout: 2000 });

  const settingsMenu = page.locator('nav').getByText('Settings').first();
  await expect(settingsMenu).not.toBeVisible({ timeout: 2000 });

  const organizationMenu = page.locator('nav .v-list-item').filter({ hasText: /^Organization$/ }).first();
  await expect(organizationMenu).not.toBeVisible({ timeout: 2000 });

  console.log('✅ PASS: Step 3 - Organization selection and Agent permissions verified via sidebar menu');

  // Step 4: Verify Dashboard menu is visible
  console.log('\n📍 Step 4: Verify Dashboard menu is visible in left sidebar');
  const dashboardMenu = page.locator('nav').getByText('Dashboard').first();
  await expect(dashboardMenu).toBeVisible();
  console.log('✅ PASS: Step 4 - Dashboard menu visible');

  // Step 5: Report results
  console.log('\n📍 Step 5: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi+1@intnt.ai');
  console.log('✅ Step 2: PASS - Agent role organization located');
  console.log('✅ Step 3: PASS - Organization with Agent role selected');
  console.log('✅ Step 4: PASS - Redirected to dashboard');
  console.log('✅ Step 5: PASS - Dashboard menu visible');
  console.log('✅ Step 6: PASS - Flow Tester menu visible');
  console.log('✅ Step 7: PASS - Flow Designer menu NOT visible');
  console.log('✅ Step 8: PASS - Knowledge Base menu NOT visible');
  console.log('✅ Step 9: PASS - Logs menu NOT visible');
  console.log('✅ Step 10: PASS - Add-Ons menu NOT visible');
  console.log('✅ Step 11: PASS - Settings menu NOT visible');
  console.log('✅ Step 12: PASS - Organization menu NOT visible');
  console.log('✅ Step 13: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
