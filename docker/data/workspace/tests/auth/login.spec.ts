import { test, expect } from '@playwright/test';

test('Login flow - User login and redirect to organization selection', async ({ page }) => {
  // Step 1: Open the page and verify it loads
  console.log('\n📍 Step 1: Open the page and verify it loads');
  await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });
  await expect(page).toHaveURL(/.*login/);
  console.log('✅ PASS: Step 1 - Login page loaded successfully');

  // Step 2: Check the login form is visible
  console.log('\n📍 Step 2: Check the login form is visible');
  const emailInput = page.locator('.v-text-field').nth(0).locator('input');
  const passwordInput = page.locator('.v-text-field').nth(1).locator('input');
  const loginButton = page.locator('button:has-text("Login")');

  await expect(emailInput).toBeVisible();
  await expect(passwordInput).toBeVisible();
  await expect(loginButton).toBeVisible();
  console.log('✅ PASS: Step 2 - Login form is visible');

  // Step 3: Input email and password
  console.log('\n📍 Step 3: Input email heidi@intnt.ai and password testing2026!');
  await emailInput.fill('heidi@intnt.ai');
  await expect(emailInput).toHaveValue('heidi@intnt.ai');
  console.log('  ✓ Email entered: heidi@intnt.ai');

  await passwordInput.fill('testing2026!');
  await expect(passwordInput).toHaveValue('testing2026!');
  console.log('  ✓ Password entered: testing2026!');
  console.log('✅ PASS: Step 3 - Credentials entered successfully');

  // Step 4: Click the Login button
  console.log('\n📍 Step 4: Click the Login button');
  await loginButton.click();
  console.log('✅ PASS: Step 4 - Login button clicked');

  // Step 5: Confirm the login is successful and verify redirect
  console.log('\n📍 Step 5: Confirm login success and verify redirect to ?select_org');
  await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 20000 });
  await expect(page).toHaveURL('https://dashboard.int3nt.info/?select_org');
  console.log('✅ PASS: Step 5 - Login successful, redirected to organization selection page');

  // Step 6: Report results
  console.log('\n📍 Step 6: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Page loaded');
  console.log('✅ Step 2: PASS - Login form visible');
  console.log('✅ Step 3: PASS - Credentials entered');
  console.log('✅ Step 4: PASS - Login button clicked');
  console.log('✅ Step 5: PASS - Login successful, redirected to ?select_org');
  console.log('✅ Step 6: PASS - All steps passed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
