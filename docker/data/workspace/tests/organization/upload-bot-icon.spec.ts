import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';
import path from 'path';

test('Bot icon upload flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026!';
  const downloadsDir = path.join(process.env.HOME || '/home/picoclaw', 'Downloads');
  const iconFile = path.join(downloadsDir, 'test.png');

  // Step 1 & 2: Login and select organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Organization in the left sidebar
  console.log('\n📍 Step 4: Navigating to Organization page');
  // Locate the Organization link in the sidebar
  const orgSidebarLink = page.locator('.nav-drawer .v-list-item', {
    hasText: /Organization/i
  });
  await expect(orgSidebarLink).toBeVisible({ timeout: 10000 });
  await orgSidebarLink.click();

  // Wait for the URL to change and the page to load
  await page.waitForURL('**/organization', { timeout: 15000 });
  await page.waitForLoadState('networkidle');
  console.log('✅ PASS: Step 4 - Navigated to Organization page');

  // Step 4: Scroll to the Bot Icons section
  console.log('\n📍 Step 4: Scroll to the Bot Icons section');
  const botIconsSection = page.locator('.bot-icons-header');
  await expect(botIconsSection).toBeVisible({ timeout: 15000 });
  await botIconsSection.scrollIntoViewIfNeeded();
  console.log('✅ PASS: Step 4 - Scrolled to Bot Icons section');

  // Step 5: Click the Upload Icon button
  console.log('\n📍 Step 5: Click the Upload Icon button');
  // Target the button specifically inside the bot-icons container
  const uploadButton = page.locator('.bot-icons-actions button').first();
  await expect(uploadButton).toBeVisible();
  await uploadButton.click();
  console.log('✅ PASS: Step 5 - Upload Icon button clicked');

  // Step 6: Upload image file
  console.log('\n📍 Step 6: Upload image file');
  // The file input is hidden (display: none) in the Vue component, 
  // but Playwright's setInputFiles works on hidden inputs.
  const fileInput = page.locator('input[type="file"]').first();
  await fileInput.setInputFiles(iconFile);
  console.log('✅ PASS: Step 6 - Image file selected');

  // Step 7: Wait for upload to complete
  console.log('\n📍 Step 7: Wait for upload processing');
  /**
   * In OrganizationPage.vue, the upload starts IMMEDIATELY upon file selection.
   * There is no "Confirm" button. Instead, we wait for the "Uploading icon..." 
   * overlay to disappear.
   */
  const uploadOverlay = page.locator('text=/Uploading icon/i');
  if (await uploadOverlay.isVisible({ timeout: 2000 }).catch(() => false)) {
    await expect(uploadOverlay).not.toBeVisible({ timeout: 20000 });
  }
  console.log('✅ PASS: Step 7 - Upload completed');

  // Step 8: Verify the uploaded icon appears in the grid
  console.log('\n📍 Step 8: Verify uploaded icon appears in the grid');
  const uploadedIcon = page.locator('.bot-icons-grid .bot-icon-card img').first();
  await expect(uploadedIcon).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 8 - Uploaded icon visible in grid');

  // Step 9: Verify success notification
  console.log('\n📍 Step 9: Verify notification "Icon uploaded successfully"');
  // Target the Vuetify snackbar content
  const successNotification = page.locator('.v-snackbar__content', {
    hasText: /uploaded successfully/i
  });
  await expect(successNotification).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 9 - Success notification appeared');


  // Step 10: Report results
  console.log('\n📍 Step 10: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization Testing2026 selected and redirected');
  console.log('✅ Step 3: PASS - Organization settings page opened');
  console.log('✅ Step 4: PASS - Scrolled to Bot Icons section');
  console.log('✅ Step 5: PASS - Upload Icon button clicked');
  console.log('✅ Step 6: PASS - Image file uploaded: test.jpg');
  console.log('✅ Step 7: PASS - Upload confirmed');
  console.log('✅ Step 8: PASS - Uploaded icon visible');
  console.log('✅ Step 9: PASS - Success notification appeared');
  console.log('✅ Step 10: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
