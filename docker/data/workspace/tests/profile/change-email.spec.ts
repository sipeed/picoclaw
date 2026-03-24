import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Change email flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';
  const newEmail = 'heidi2@intnt.ai';

  // Step 1-2: Perform the login and select org flow
  console.log('\n📍 Step 1-2: Perform login and select organization Testing2026');
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);
  console.log('✅ PASS: Step 1-2 - Login and organization selection completed');

  // Step 3: Verify redirect to dashboard
  console.log('\n📍 Step 3: Verify redirect to https://dashboard.int3nt.info/');
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
  console.log('✅ PASS: Step 3 - User redirected to dashboard');

  // Step 4: Open profile dropdown
  console.log('\n📍 Step 4: Click the profile avatar dropdown');
  const profileActivator = page.locator('#menu-activator');
  await expect(profileActivator).toBeVisible({ timeout: 10000 });
  await profileActivator.click();
  console.log('✅ PASS: Step 4 - Profile dropdown clicked');

  // Step 5: Click Profile option
  console.log('\n📍 Step 5: Click Profile option');
  const profileOption = page.locator('.v-overlay-container .v-list-item-title').filter({
    hasText: /^Profile$/
  }).first();
  await expect(profileOption).toBeVisible({ timeout: 5000 });
  await profileOption.click();
  await page.waitForURL('**/profile', { timeout: 10000 });
  console.log('✅ PASS: Step 5 - Navigated to Profile page');

  // Step 6: Verify Profile page content
  console.log('\n📍 Step 6: Verify Profile page loaded');
  await expect(page.locator('.profile-container')).toBeVisible({ timeout: 10000 });
  await expect(page.locator('.loading-state')).not.toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 6 - Profile page content visible');

  // Step 7: Click "Change email" link
  console.log('\n📍 Step 7: Click Change Email link');
  const changeEmailLink = page.locator('.change-email-link');
  await expect(changeEmailLink).toBeVisible();
  await changeEmailLink.click();
  await page.waitForURL('**/change-email', { timeout: 10000 });
  console.log('✅ PASS: Step 7 - Navigated to Change Email page');

  // Step 8: (Optional/Implicit) Verify form is ready
  console.log('\n📍 Step 8: Verify Change Email form is ready');
  await expect(page.locator('.email-form')).toBeVisible();
  console.log('✅ PASS: Step 8 - Change Email form visible');

  // Step 9: Filling new email fields
  console.log('\n📍 Step 9: Filling new email fields');
  /**
   * FIX: Use getByPlaceholder because GlobalFormField sets the 
   * placeholder attribute on the inner <input>.
   */
  const newEmailInput = page.getByPlaceholder('New Email', { exact: true });
  const confirmEmailInput = page.getByPlaceholder('Confirm New Email', { exact: true });

  await expect(newEmailInput).toBeVisible({ timeout: 10000 });
  await newEmailInput.fill(newEmail);
  await confirmEmailInput.fill(newEmail);
  console.log('✅ PASS: Step 9 - New email fields filled');

  // Step 10: Click "Confirm" and handle the Modal
  console.log('\n📍 Step 10: Clicking Confirm and handling modal');

  // 1. Click the button on the main page
  const confirmBtnOnPage = page.locator('button').filter({ hasText: /^Confirm$/i });
  await confirmBtnOnPage.click();

  // 2. Locate the Confirmation Modal using its ARIA role and Title
  const alertModal = page.getByRole('dialog').filter({ hasText: /Confirm Email Change/i });
  await expect(alertModal).toBeVisible({ timeout: 10000 });

  // 3. Click the "Yes, change" button inside that specific modal
  const modalActionBtn = alertModal.getByRole('button', { name: /Yes, change/i });
  await expect(modalActionBtn).toBeVisible();
  await modalActionBtn.click();

  console.log('✅ PASS: Step 10 - Email change confirmed in modal');

  // Step 11: Verify success snackbar
  console.log('\n📍 Step 11: Verify success snackbar');
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /Email updated successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 11 - Email successfully updated');

  // Step 12: Revert email change (for cleanup)
  console.log('\n📍 Step 12: Reverting email back to primaryEmail');
  // Wait for the previous snackbar to disappear to avoid confusion
  await expect(successSnackbar).toBeHidden({ timeout: 15000 });

  await newEmailInput.fill(primaryEmail);
  await confirmEmailInput.fill(primaryEmail);
  await confirmBtnOnPage.click();
  await expect(alertModal).toBeVisible({ timeout: 10000 });
  await modalActionBtn.click();
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 12 - Email reverted successfully');

  // Step 13: Report results
  console.log('\n📍 Step 13: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1-2: PASS - Login and organization selection completed');
  console.log('✅ Step 3: PASS - User redirected to dashboard');
  console.log('✅ Step 4: PASS - Profile dropdown clicked');
  console.log('✅ Step 5: PASS - Profile option clicked');
  console.log('✅ Step 6: PASS - Profile page or dialog displayed');
  console.log('✅ Step 7: PASS - Change email button clicked');
  console.log('✅ Step 8: PASS - Change email form or popup appeared');
  console.log('✅ Step 9: PASS - New Email fields filled');
  console.log('✅ Step 10: PASS - Save Changes button clicked');
  console.log('✅ Step 11: PASS - Email updated successfully');
  console.log('✅ Step 12: PASS - Email reverted successfully');
  console.log('✅ Step 13: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
