import { test, expect } from '@playwright/test';
import { performLogin } from './utils/auth';
import Imap from 'imap';
import { simpleParser } from 'mailparser';
import * as dotenv from 'dotenv';
dotenv.config({ path: __dirname + '/../.env' });

test('Password reset flow - Reset password and verify login', async ({ page }) => {
    test.setTimeout(120000);
    const testEmail = 'test@intnt.ai';
    const newPassword = 'testing2027!';
    const originalPassword = 'testing2026!';

    // Helper function to retrieve password reset email using IMAP
    async function getPasswordResetEmail(): Promise<string> {
        return new Promise((resolve, reject) => {
            const imap = new Imap({
                user: process.env.IMAP_USER || testEmail,
                password: process.env.IMAP_PASSWORD || '',
                host: process.env.IMAP_HOST || 'imap.gmail.com',
                port: Number(process.env.IMAP_PORT) || 993,
                tls: true,
                tlsOptions: { rejectUnauthorized: false }
            });

            const timeoutMs = 60000; // 60 seconds max wait
            const pollInterval = 5000; // check every 5 seconds
            const deadline = Date.now() + timeoutMs;

            function poll() {
                if (Date.now() > deadline) {
                    imap.end();
                    return reject(new Error('Timeout: password reset email not received within 60 seconds'));
                }

                imap.openBox('INBOX', false, (err) => {
                    if (err) return reject(err);

                    // Search for unseen emails with reset subject
                    imap.search(['UNSEEN', ['SUBJECT', 'Reset Your Password']], (err, uids) => {
                        if (err) return reject(err);

                        if (!uids || uids.length === 0) {
                            // No email yet — wait and retry
                            console.log('   Waiting for reset email...');
                            setTimeout(poll, pollInterval);
                            return;
                        }

                        // Fetch the latest matching email
                        const latestUid = uids[uids.length - 1];
                        const fetch = imap.fetch([latestUid], { bodies: '' });

                        fetch.on('message', (msg) => {
                            msg.on('body', (stream) => {
                                simpleParser(stream, (err, mail) => {
                                    if (err) return reject(err);

                                    const emailBody = mail.html?.toString() || mail.text?.toString() || '';

                                    // Extract reset link from email body
                                    const linkMatch =
                                        emailBody.match(/href="([^"]*token=[^"]+)"/i) ||
                                        emailBody.match(/href="([^"]*reset[^"]*)"/i) ||
                                        emailBody.match(/(https?:\/\/[^\s<>"]+token=[^\s<>"]+)/i) ||
                                        emailBody.match(/https?:\/\/[^\s<>"]+reset[^\s<>"]+/i);

                                    if (!linkMatch) {
                                        return reject(new Error('Reset link not found in email body'));
                                    }

                                    // Mark email as seen so next search won't find it again
                                    imap.addFlags([latestUid], ['\\Seen'], () => {
                                        imap.end();
                                        resolve(linkMatch[1]);
                                    });
                                });
                            });
                        });

                        fetch.once('error', reject);
                    });
                });
            }

            imap.once('ready', poll);
            imap.once('error', reject);
            imap.connect();
        });
    }

    // Step 1: Open the page and verify it loads
    console.log('\n📍 Step 1: Open the page and verify it loads');
    await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/.*login/);
    console.log('✅ PASS: Step 1 - Login page loaded successfully');

    // Step 2: Click the Forgot password link
    console.log('\n📍 Step 2: Click the Forgot password link');
    const forgotPasswordLink = page.locator('button:has-text("Forgot Password"), a:has-text("Forgot Password"), button:has-text("Forgot password")').first();
    await expect(forgotPasswordLink).toBeVisible();
    await forgotPasswordLink.click();
    console.log('✅ PASS: Step 2 - Forgot password link clicked');

    // Step 3: Enter email and click Send Email
    console.log('\n📍 Step 3: Enter email test@intnt.ai and click Send Email');
    const emailInputField = page.locator('.forgot-password-card .v-input input').first();
    await expect(emailInputField).toBeVisible();
    await emailInputField.fill(testEmail);

    const sendEmailButton = page.locator('button:has-text("Send Email"), button:has-text("Send"), button[type="submit"]').first();
    await expect(sendEmailButton).toBeVisible();
    await sendEmailButton.click();
    await page.waitForTimeout(2000);
    console.log('✅ PASS: Step 3 - Email entered and Send Email clicked');

    // Step 4: Retrieve the password reset email via IMAP
    console.log('\n📍 Step 4: Retrieve password reset email via IMAP');
    let resetLink = '';
    try {
        resetLink = await getPasswordResetEmail();
        console.log('✅ PASS: Step 4 - Password reset email retrieved via IMAP');
        console.log(`   Reset link: ${resetLink.substring(0, 60)}...`);
    } catch (error) {
        console.log(`❌ FAIL: Step 4 - ${error}`);
        throw error;
    }

    // Step 5: Extract reset link and open it
    console.log('\n📍 Step 5: Extract password reset link and open it');
    await page.goto(resetLink, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    console.log('✅ PASS: Step 5 - Password reset link opened');

    // Step 6: Fill new password
    console.log('\n📍 Step 6: Fill Password testing2027! and Confirm Password testing2027!');
    const passwordField = page.locator('input[type="password"]').first();
    const confirmPasswordField = page.locator('input[type="password"]').nth(1);

    await expect(passwordField).toBeVisible();
    await expect(confirmPasswordField).toBeVisible();

    await passwordField.fill(newPassword);
    await confirmPasswordField.fill(newPassword);
    console.log('✅ PASS: Step 6 - New passwords entered');

    // Step 7: Click Set Password
    console.log('\n📍 Step 7: Click Set Password');
    const setPasswordButton = page.locator('button:has-text("Set Password"), button:has-text("Update Password"), button[type="submit"]').first();
    await expect(setPasswordButton).toBeVisible();
    await setPasswordButton.click();
    console.log('✅ PASS: Step 7 - Set Password clicked');

    // Step 8: Verify success message and click Go To Dashboard
    console.log('\n📍 Step 8: Verify success message and click Go To Dashboard');
    await expect(page.getByText(/Your password was reset successfully/i)).toBeVisible({ timeout: 10000 });
    console.log('   ✓ Success message visible');
    const goToDashboardButton = page.locator('button:has-text("Go To Dashboard"), a:has-text("Go To Dashboard")').first();
    await expect(goToDashboardButton).toBeVisible();
    await goToDashboardButton.click();
    await page.waitForURL('**/dashboard.int3nt.info/login', { timeout: 15000 });
    await expect(page).toHaveURL(/.*login/);
    console.log('✅ PASS: Step 8 - Clicked Go To Dashboard, redirected to login page');

    // Step 9: Login with new password and verify
    console.log(`\n📍 Step 9: Login with new password ${newPassword} and verify`);
    await performLogin(page, testEmail, newPassword);
    console.log('✅ PASS: Step 9 - Login successful with new password, redirected to org selection');

    // Step 10: Logout before reverting password
    console.log('\n📍 Step 10: Click the logout icon at upper right');
    const logoutButton = page.locator('[class*="logout"], [class*="user-menu"], button:has-text("Logout"), button:has-text("Sign out"), [aria-label*="logout" i], [aria-label*="sign out" i]').first();
    const profileMenu = page.locator('button[class*="profile"], button[class*="user"], [class*="avatar"]').first();

    let logoutFound = false;

    // Try clicking logout button directly
    if (await logoutButton.isVisible({ timeout: 10000 }).catch(() => false)) {
        await logoutButton.click();
        logoutFound = true;
        console.log('   ✓ Logout button clicked directly');
    }
    // Try clicking profile menu first, then logout
    else if (await profileMenu.isVisible({ timeout: 10000 }).catch(() => false)) {
        await profileMenu.click();
        await page.waitForTimeout(500);

        const logoutInMenu = page.locator('[class*="logout"], button:has-text("Logout"), button:has-text("Sign out")').first();
        if (await logoutInMenu.isVisible({ timeout: 10000 }).catch(() => false)) {
            await logoutInMenu.click();
            logoutFound = true;
            console.log('   ✓ Profile menu clicked and logout selected');
        }
    }

    if (logoutFound) {
        await page.waitForURL('**/dashboard.int3nt.info/login', { timeout: 10000 });
        console.log('✅ PASS: Step 10 - Logged out successfully, redirected to login');
    } else {
        console.log('❌ FAIL: Step 10 - Logout button not found');
        throw new Error('Logout button not found');
    }

    // Step 11: Revert password - Click Forgot password again
    console.log('\n📍 Step 11: Revert password - Click Forgot password');
    await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });
    const forgotPasswordLink2 = page.locator('button:has-text("Forgot Password"), a:has-text("Forgot Password"), button:has-text("Forgot password")').first();
    await expect(forgotPasswordLink2).toBeVisible();
    await forgotPasswordLink2.click();
    console.log('✅ PASS: Step 11 - Forgot password link clicked again');

    // Step 12: Enter email and send reset email again
    console.log('\n📍 Step 12: Enter email and send reset email again');
    const emailInputField2 = page.locator('.forgot-password-card .v-input input').first();
    await expect(emailInputField2).toBeVisible();
    await emailInputField2.fill(testEmail);

    const sendEmailButton2 = page.locator('button:has-text("Send Email"), button:has-text("Send"), button[type="submit"]').first();
    await expect(sendEmailButton2).toBeVisible();
    await sendEmailButton2.click();
    await page.waitForTimeout(2000);

    // Retrieve the second reset email via IMAP
    let resetLink2 = '';
    try {
        resetLink2 = await getPasswordResetEmail();
        console.log('✅ PASS: Step 12 - Reset email sent and link retrieved via IMAP');
    } catch (error) {
        console.log(`❌ FAIL: Step 12 - ${error}`);
        throw error;
    }

    // Step 12: Open reset link and fill original password
    console.log('\n📍 Step 13: Fill original password testing2026! and click Set Password');
    await page.goto(resetLink2, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const passwordField2 = page.locator('input[type="password"]').first();
    const confirmPasswordField2 = page.locator('input[type="password"]').nth(1);

    await expect(passwordField2).toBeVisible();
    await expect(confirmPasswordField2).toBeVisible();

    await passwordField2.fill(originalPassword);
    await confirmPasswordField2.fill(originalPassword);

    const setPasswordButton2 = page.locator('button:has-text("Set Password"), button:has-text("Update Password"), button[type="submit"]').first();
    await expect(setPasswordButton2).toBeVisible();
    await setPasswordButton2.click();

    await expect(page.getByText('Your password was reset successfully')).toBeVisible({ timeout: 10000 });
    const goToDashboardButton2 = page.locator('button:has-text("Go To Dashboard"), a:has-text("Go To Dashboard")').first();
    await expect(goToDashboardButton2).toBeVisible();
    await goToDashboardButton2.click();
    await page.waitForURL('**/dashboard.int3nt.info/login', { timeout: 15000 });
    await expect(page).toHaveURL(/.*login/);
    console.log('✅ PASS: Step 13 - Password reverted, clicked Go To Dashboard, redirected to login');

    // Step 14: Login with original password and verify
    console.log(`\n📍 Step 14: Login with original password ${originalPassword} and verify`);
    await performLogin(page, testEmail, originalPassword);
    console.log('✅ PASS: Step 14 - Login successful with original password, redirected to org selection');

    // Step 14: Report results
    console.log('\n📍 Step 15: Report PASS or FAIL for each step');
    console.log('\n' + '='.repeat(70));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(70));
    console.log('✅ Step 1:  PASS - Login page loaded');
    console.log('✅ Step 2:  PASS - Forgot password link clicked');
    console.log('✅ Step 3:  PASS - Email sent');
    console.log('✅ Step 4:  PASS - Password reset email retrieved via IMAP');
    console.log('✅ Step 5:  PASS - Reset link opened');
    console.log('✅ Step 6:  PASS - New password entered');
    console.log('✅ Step 7:  PASS - Set Password clicked');
    console.log('✅ Step 8:  PASS - Redirected to login');
    console.log('✅ Step 9:  PASS - Login successful with new password');
    console.log('✅ Step 11: PASS - Forgot password clicked again');
    console.log('✅ Step 12: PASS - Reset email retrieved again via IMAP');
    console.log('✅ Step 13: PASS - Password reverted and redirected');
    console.log('✅ Step 14: PASS - Login successful with original password');
    console.log('✅ Step 15: PASS - All steps completed');
    console.log('='.repeat(70));
    console.log('\n✅ ALL TESTS PASSED\n');
});