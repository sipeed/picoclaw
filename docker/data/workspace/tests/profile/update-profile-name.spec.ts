import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Update profile name flow', async ({ page }) => {
  test.setTimeout(180000); // 3 minutes timeout
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';
  const firstName = 'heidi';
  const lastName = 'lau';

  // Step 1-2: Perform the login and select org flow
  console.log('\n📍 Step 1-2: Perform login and select organization Testing2026');
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);
  console.log('✅ PASS: Step 1-2 - Login and organization selection completed');

  // Step 3: Verify redirect to dashboard
  console.log('\n📍 Step 3: Verify redirect to ');
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
  console.log('✅ PASS: Step 3 - User redirected to dashboard');

  // Step 4: Click the avatar in the top right to open the menu
  console.log('\n📍 Step 4: Click the profile avatar dropdown');
  // Using the specific ID used in DefaultLayoutDrawer.vue
  const profileActivator = page.locator('#menu-activator');
  await expect(profileActivator).toBeVisible({ timeout: 10000 });
  await profileActivator.click();
  console.log('✅ PASS: Step 4 - Profile dropdown clicked');

  // Step 5: Click "Profile" option in the teleported menu
  console.log('\n📍 Step 5: Click Profile option');
  /**
   * In Vuetify, menu items are teleported to .v-overlay-container.
   * We use a strict regex to avoid matching breadcrumbs or other "Profile" text.
   */
  const profileOption = page.locator('.v-overlay-container .v-list-item-title').filter({
    hasText: /^Profile$/
  }).first();

  await expect(profileOption).toBeVisible({ timeout: 5000 });
  await profileOption.click();

  // Wait for navigation to complete
  await page.waitForURL('**/profile', { timeout: 10000 });
  console.log('✅ PASS: Step 5 - Navigated to Profile page');

  // Step 6: Verify Profile page content is loaded
  console.log('\n📍 Step 6: Verify Profile page loaded');
  const profileContainer = page.locator('.profile-container');
  await expect(profileContainer).toBeVisible({ timeout: 10000 });
  // Wait for the inner loading spinner to disappear
  await expect(page.locator('.loading-state')).not.toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 6 - Profile page content visible');

  // Step 7: Locate and fill First Name and Last Name
  console.log('\n📍 Step 7: Update First and Last Name');
  // Locating inputs by the labels defined in ProfilePage.vue
  const firstNameInput = page.locator('.name-field', { hasText: /First Name/i }).locator('input');
  const lastNameInput = page.locator('.name-field', { hasText: /Last Name/i }).locator('input');

  await firstNameInput.fill(firstName);
  await lastNameInput.fill(lastName);
  console.log('✅ PASS: Step 7 - Name fields updated');

  // Step 8: Click Save Changes (triggers confirmation modal)
  console.log('\n📍 Step 8: Click Save Changes');
  const saveChangesBtn = page.getByRole('button', { name: /Save Changes/i });
  await expect(saveChangesBtn).toBeVisible();
  await saveChangesBtn.click();
  console.log('✅ PASS: Step 8 - Save Changes button clicked');

  // Step 9: Handle the Confirmation Modal
  console.log('\n📍 Step 9: Confirming update in modal');
  /**
   * Use getByRole('dialog') which is the standard accessibility role for modals.
   * We filter by the title seen in your pages.json: "Confirm Profile Update"
   */
  const confirmModal = page.getByRole('dialog').filter({ hasText: /Confirm Profile Update/i });
  await expect(confirmModal).toBeVisible({ timeout: 10000 });

  // Find and click the "Yes, update" button inside the modal
  const confirmBtn = confirmModal.getByRole('button', { name: /Yes, update/i });
  await expect(confirmBtn).toBeVisible();
  await confirmBtn.click();
  console.log('✅ PASS: Step 9 - Update confirmed in modal');

  // Step 10: Verify success
  console.log('\n📍 Step 10: Verify update success');
  /**
   * Snackbars are also teleported. We search for the content text.
   */
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /updated successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 10 - Profile updated successfully and snackbar appeared');

  // Step 11: Report results
  console.log('\n📍 Step 11: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1-2: PASS - Login and organization selection completed');
  console.log('✅ Step 3: PASS - User redirected to dashboard');
  console.log('✅ Step 4: PASS - Profile dropdown clicked');
  console.log('✅ Step 5: PASS - Profile option clicked');
  console.log('✅ Step 6: PASS - Profile page or dialog displayed');
  console.log('✅ Step 7: PASS - First Name and Last Name fields filled');
  console.log('✅ Step 8: PASS - Save Changes button clicked');
  console.log('✅ Step 9: PASS - Confirm update in Confirmation Modal');
  console.log('✅ Step 10: PASS - Name updated successfully');
  console.log('✅ Step 11: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
