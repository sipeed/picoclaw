import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Change password flow', async ({ page }) => {
  test.setTimeout(180000); // 3 minutes timeout
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';
  const newPassword = 'testing2027!';

  // Step 1-2: Perform the login and select org flow
  console.log('\n📍 Step 1-2: Perform login and select organization Testing2026');
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);
  console.log('✅ PASS: Step 1-2 - Login and organization selection completed');

  // Step 3: Verify redirect to dashboard
  console.log('\n📍 Step 3: Verify redirect to ');
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
  console.log('✅ PASS: Step 3 - User redirected to dashboard');

  // Step 4: Open profile dropdown
  console.log('\n📍 Step 4: Click the profile avatar dropdown');
  const profileActivator = page.locator('#menu-activator');
  await expect(profileActivator).toBeVisible({ timeout: 20000 });
  await profileActivator.click();
  console.log('✅ PASS: Step 4 - Profile dropdown clicked');

  // Step 5: Click Profile option
  console.log('\n📍 Step 5: Click Profile option');
  const profileOption = page.locator('.v-overlay-container .v-list-item-title').filter({
    hasText: /^Profile$/
  }).first();
  await expect(profileOption).toBeVisible({ timeout: 5000 });
  await profileOption.click();
  await page.waitForURL('**/profile', { timeout: 60000 });
  console.log('✅ PASS: Step 5 - Navigated to Profile page');

  // Step 6: Verify Profile page content
  console.log('\n📍 Step 6: Verify Profile page loaded');
  await expect(page.locator('.profile-container')).toBeVisible({ timeout: 20000 });
  await expect(page.locator('.loading-state')).not.toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 6 - Profile page content visible');

  // Step 7: Click "Change password" link
  console.log('\n📍 Step 7: Click Change Password link');
  const changePasswordLink = page.locator('.change-password-link');
  await expect(changePasswordLink).toBeVisible();
  await changePasswordLink.click();
  await page.waitForURL('**/change-password', { timeout: 60000 });
  console.log('✅ PASS: Step 7 - Navigated to Change Password page');

  // Step 8: Verify Change Password form is ready
  console.log('\n📍 Step 8: Verify Change Password form is ready');
  await expect(page.locator('.password-form')).toBeVisible();
  console.log('✅ PASS: Step 8 - Change Password form visible');

  // Step 9: Filling new password fields
  console.log('\n📍 Step 9: Filling new password fields');
  // Using placeholders defined in ChangePasswordPage.vue
  const newPassInput = page.getByPlaceholder('New Password', { exact: true });
  const confirmPassInput = page.getByPlaceholder('Confirm New Password', { exact: true });

  await expect(newPassInput).toBeVisible({ timeout: 20000 });
  await newPassInput.fill(newPassword);
  await confirmPassInput.fill(newPassword);
  console.log('✅ PASS: Step 9 - New password fields filled');

  // Step 10: Click "Confirm" and handle the Modal
  console.log('\n📍 Step 10: Clicking Confirm and handling modal');

  // 1. Click the button on the main page
  const confirmBtnOnPage = page.locator('button').filter({ hasText: /^Confirm$/i });
  await confirmBtnOnPage.click();

  // 2. Locate the Confirmation Modal using its ARIA role and Title
  const alertModal = page.getByRole('dialog').filter({ hasText: /Confirm Password Change/i });
  await expect(alertModal).toBeVisible({ timeout: 20000 });

  // 3. Click the "Yes, change" button inside that specific modal
  const modalActionBtn = alertModal.getByRole('button', { name: /Yes, change/i });
  await expect(modalActionBtn).toBeVisible();
  await modalActionBtn.click();

  console.log('✅ PASS: Step 10 - Password change confirmed in modal');

  // Step 11: Verify success snackbar
  console.log('\n📍 Step 11: Verify success snackbar');
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /Password updated successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 11 - Password successfully updated');

  // Step 12: Revert password change (for cleanup)
  console.log('\n📍 Step 12: Reverting password back to primaryPassword');
  // Wait for the previous snackbar to disappear to avoid confusion
  await expect(successSnackbar).toBeHidden({ timeout: 15000 });

  await newPassInput.fill(primaryPassword);
  await confirmPassInput.fill(primaryPassword);
  await confirmBtnOnPage.click();
  await expect(alertModal).toBeVisible({ timeout: 20000 });
  await modalActionBtn.click();
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 12 - Password reverted successfully');

  // Step 13: Report results
  console.log('\n📍 Step 13: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1-2: PASS - Login and organization selection completed');
  console.log('✅ Step 3: PASS - User redirected to dashboard');
  console.log('✅ Step 4: PASS - Profile dropdown clicked');
  console.log('✅ Step 5: PASS - Profile option clicked');
  console.log('✅ Step 6: PASS - Profile page content visible');
  console.log('✅ Step 7: PASS - Change password button clicked');
  console.log('✅ Step 8: PASS - Change password form visible');
  console.log('✅ Step 9: PASS - New Password fields filled');
  console.log('✅ Step 10: PASS - Password change confirmed in modal');
  console.log('✅ Step 11: PASS - Password updated successfully');
  console.log('✅ Step 12: PASS - Password reverted successfully');
  console.log('✅ Step 13: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
