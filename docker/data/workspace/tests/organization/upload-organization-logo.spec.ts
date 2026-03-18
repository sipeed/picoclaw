import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';
import path from 'path';

test('Organization logo upload flow', async ({ page }) => {
  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing2026';
  const downloadsDir = path.join(process.env.HOME || '/home/picoclaw', 'Downloads');
  const logoFile1 = path.join(downloadsDir, 'test2.png');
  const logoFile2 = path.join(downloadsDir, 'test.png');

  // Step 1 & 2: Login and select organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click the pencil icon to open the upload dialog
  console.log('\n📍 Step 3: Click on the change logo button to open upload dialog');
  const changeLogoButton = page.locator('.change-logo-btn');
  // The button only appears for Admins
  await expect(changeLogoButton).toBeVisible({
    timeout: 10000
  });
  await changeLogoButton.click();

  // Wait for the dialog to appear
  const logoDialog = page.locator('.v-card', { hasText: /Change Logo/i });
  await expect(logoDialog).toBeVisible({ timeout: 5000 });
  console.log('✅ PASS: Step 3 - Change logo dialog opened');

  // Step 4: Upload test2.jpg
  console.log('\n📍 Step 4: Selecting test2.jpg');
  /**
   * NOTE: We skip toBeVisible() because the <input type="file"> 
   * has style="display: none" in LogoUploadDialog.vue.
   */
  const fileInput = page.locator('input[type="file"]').first();
  await fileInput.setInputFiles(logoFile1);
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 4 - test2.jpg file selected');

  // Step 5: Click the "Upload Logo" button in the dialog
  console.log('\n📍 Step 5: Confirming upload');
  const uploadConfirmBtn = page.locator('button', { hasText: /Upload Logo/i }).first();
  await expect(uploadConfirmBtn).toBeEnabled({ timeout: 5000 });
  await uploadConfirmBtn.click();

  // Wait for the success snackbar
  const successToast = page.locator('.v-snackbar__content', { hasText: /Logo uploaded successfully/i });
  await expect(successToast).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 5 - Logo updated to test2.jpg');

  // Step 6: Trigger the next upload
  console.log('\n📍 Step 6: Opening dialog again for second upload');
  await changeLogoButton.click();
  await expect(logoDialog).toBeVisible();

  // Step 7: Upload test.jpg
  console.log('\n📍 Step 7: Selecting test.jpg');
  const fileInput2 = page.locator('input[type="file"]').first();
  await fileInput2.setInputFiles(logoFile2);
  await page.waitForTimeout(1000);

  // Step 8: Confirm and final verification
  console.log('\n📍 Step 8: Confirming second upload');
  const uploadConfirmBtn2 = page.locator('button', { hasText: /Upload Logo/i }).first();
  await uploadConfirmBtn2.click();

  await expect(successToast).toBeVisible({ timeout: 10000 });
  console.log('✅ PASS: Step 8 - Final logo updated to test.jpg');


  // Step 9: Report results
  console.log('\n📍 Step 9: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
  console.log('✅ Step 2: PASS - Organization selected and redirected to dashboard');
  console.log('✅ Step 3: PASS - Organization logo clicked');
  console.log('✅ Step 4: PASS - test2.jpg file uploaded');
  console.log('✅ Step 5: PASS - Organization logo updated to test2.jpg');
  console.log('✅ Step 6: PASS - Organization logo clicked again');
  console.log('✅ Step 7: PASS - test.jpg file uploaded');
  console.log('✅ Step 8: PASS - Organization logo updated to test.jpg');
  console.log('✅ Step 9: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
