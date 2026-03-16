import { test, expect } from '@playwright/test';
import { performLogin } from './utils/auth';

test('Logout flow - User logout and redirect to login page', async ({ page }) => {
  // Step 1: Perform the login flow
  await performLogin(page, 'test@intnt.ai', 'testing2026!');
  console.log('✅ PASS: Step 1 - Login flow completed');

  // Step 2: Verify login succeeds and redirect
  console.log('\n📍 Step 2: Verify login succeeds and redirect');
  try {
    await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 15000 }).catch(() => { });
    const currentUrl = page.url();

    if (currentUrl.includes('?select_org') || currentUrl.includes('dashboard.int3nt.info')) {
      console.log(`  ✓ Redirected to: ${currentUrl}`);
      console.log('✅ PASS: Step 2 - Login successful and redirected');
    } else {
      console.log(`  ✗ Unexpected URL: ${currentUrl}`);
      console.log('❌ FAIL: Step 2 - Login failed or unexpected redirect');
      throw new Error('Login redirect failed');
    }
  } catch (error) {
    console.log('❌ FAIL: Step 2 - Login verification failed');
    throw error;
  }

  // Step 3: Click the logout icon located at the upper right
  console.log('\n📍 Step 3: Click the logout icon at upper right');
  try {
    // Look for logout button/icon in the upper right
    // Common selectors: profile menu, user menu, logout button
    const logoutButton = page.locator('[class*="logout"], [class*="user-menu"], button:has-text("Logout"), button:has-text("Sign out"), [aria-label*="logout" i], [aria-label*="sign out" i]').first();

    // If not found, try to find the profile/user menu button first
    const profileMenu = page.locator('button[class*="profile"], button[class*="user"], [class*="avatar"]').first();

    let logoutFound = false;

    // Try clicking logout button directly
    if (await logoutButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await logoutButton.click();
      logoutFound = true;
      console.log('  ✓ Logout button clicked directly');
    }
    // Try clicking profile menu first, then logout
    else if (await profileMenu.isVisible({ timeout: 3000 }).catch(() => false)) {
      await profileMenu.click();
      await page.waitForTimeout(500);

      const logoutInMenu = page.locator('[class*="logout"], button:has-text("Logout"), button:has-text("Sign out")').first();
      if (await logoutInMenu.isVisible({ timeout: 3000 }).catch(() => false)) {
        await logoutInMenu.click();
        logoutFound = true;
        console.log('  ✓ Profile menu clicked and logout selected');
      }
    }

    if (logoutFound) {
      console.log('✅ PASS: Step 3 - Logout icon clicked');
    } else {
      console.log('❌ FAIL: Step 3 - Logout button not found');
      throw new Error('Logout button not found');
    }
  } catch (error) {
    console.log(`❌ FAIL: Step 3 - ${error}`);
    throw error;
  }

  // Step 4: Confirm logout and verify redirect to login page
  console.log('\n📍 Step 4: Confirm logout and verify redirect to login page');
  try {
    await page.waitForURL('**/dashboard.int3nt.info/login', { timeout: 15000 }).catch(() => { });
    await expect(page).toHaveURL(/.*login/);
    console.log('  ✓ Redirected to login page');
    console.log('✅ PASS: Step 4 - Logout successful, redirected to login page');
  } catch (error) {
    console.log('❌ FAIL: Step 4 - Logout redirect failed');
    throw error;
  }

  // Step 5: Report results
  console.log('\n📍 Step 5: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login flow completed');
  console.log('✅ Step 2: PASS - Login successful and redirected');
  console.log('✅ Step 3: PASS - Logout icon clicked');
  console.log('✅ Step 4: PASS - Logout successful, redirected to login');
  console.log('✅ Step 5: PASS - All steps passed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
