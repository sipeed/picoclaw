import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';
import Imap from 'imap';
import { simpleParser } from 'mailparser';
import * as dotenv from 'dotenv';
dotenv.config({ path: __dirname + '/../../.env' });

test('Invite member to organization flow', async ({ page }) => {
  test.setTimeout(180000);
  const primaryEmail = process.env.IMAP_USER || 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const randomSuffix = Math.floor(10000 + Math.random() * 90000);
  const invitedEmail = `heidi+${randomSuffix}@intnt.ai`;
  const invitedPassword = 'testing2026!!';
  const organizationName = 'Testing2026!';
  const adminRole = 'admin';

  // Helper function to retrieve invitation email using IMAP
  async function getInvitationEmail(): Promise<string> {
    return new Promise((resolve, reject) => {
      const imap = new Imap({
        user: process.env.IMAP_USER || primaryEmail,
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
          return reject(new Error('Timeout: Invitation email not received within 60 seconds'));
        }

        imap.openBox('INBOX', false, (err) => {
          if (err) return reject(err);

          // Search for unseen emails with "You have been invited" in subject
          imap.search(['UNSEEN', ['SUBJECT', 'You have been invited']], (err, uids) => {
            if (err) return reject(err);

            if (!uids || uids.length === 0) {
              // No email yet — wait and retry
              console.log('   Waiting for invitation email...');
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

                  // User-provided link example: /auth/set-password?type=invite&token=...
                  const linkMatch =
                    emailBody.match(/href="([^"]*set-password[^"]*token=[^"]+)"/i) ||
                    emailBody.match(/href="([^"]*invite[^"]*)"/i) ||
                    emailBody.match(/(https?:\/\/[^\s<>"]+set-password[^\s<>"]+token=[^\s<>"]+)/i) ||
                    emailBody.match(/https?:\/\/[^\s<>"]+dashboard[^\s<>"]+token=[^\s<>"]+/i);

                  if (!linkMatch) {
                    return reject(new Error('Invitation link not found in email body'));
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

  // Step 1: Login and select organization
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 3: Click Organization in the left sidebar
  console.log('\n📍 Step 3: Click Organization in the left sidebar');
  const organizationLink = page.locator('nav').locator('a:has-text("Organization")').first();
  await expect(organizationLink).toBeVisible();
  await organizationLink.click();
  console.log('✅ PASS: Step 3 - Organization settings page opened');

  // Step 4: Click Add A Member
  console.log('\n📍 Step 4: Click Add A Member');
  const addMemberButton = page.getByRole('button', { name: /Add a Member/i }).first();
  await expect(addMemberButton).toBeVisible();
  await addMemberButton.click();
  console.log('✅ PASS: Step 4 - Add A Member modal opened');

  // Step 5: Enter email
  console.log(`\n📍 Step 5: Enter email ${invitedEmail}`);
  /**
   * Use getByRole('dialog') to find the modal, then find the input.
   * Scoping prevents interference with background elements.
   */
  const addMemberModal = page.getByRole('dialog').filter({ hasText: /Add a Member/i });
  await expect(addMemberModal).toBeVisible({ timeout: 20000 });

  const memberEmailInput = addMemberModal.locator('input').first();
  await expect(memberEmailInput).toBeVisible();
  await memberEmailInput.fill(invitedEmail);
  console.log(`✅ PASS: Step 5 - Email entered: ${invitedEmail}`);

  // Step 6: Select role admin
  console.log(`\n📍 Step 6: Select role ${adminRole}`);
  const roleSelector = addMemberModal.locator('.v-select').first();
  await roleSelector.click();

  // Vuetify teleports menus to .v-overlay-container
  const adminOption = page.locator('.v-overlay-container .v-list-item').filter({
    hasText: new RegExp(`^${adminRole}$`, 'i')
  }).first();
  await expect(adminOption).toBeVisible({ timeout: 5000 });
  await adminOption.click();
  console.log(`✅ PASS: Step 6 - Role ${adminRole} selected`);

  // Step 7: Click Add
  console.log('\n📍 Step 7: Click Add');
  const addButton = addMemberModal.getByRole('button', { name: /^Add$/i }).first();
  await expect(addButton).toBeVisible();
  await addButton.click();
  console.log('✅ PASS: Step 7 - Add button clicked');

  // Step 8: Verify notification
  console.log('\n📍 Step 8: Verify notification "Member invite sent successfully"');
  const successSnackbar = page.locator('.v-snackbar__content', {
    hasText: /invite sent successfully/i
  });
  await expect(successSnackbar).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 8 - Success notification verified');

  // Step 9: Retrieve invitation email via IMAP
  console.log('\n📍 Step 9: Retrieve invitation email via IMAP');
  let inviteLink = '';
  try {
    inviteLink = await getInvitationEmail();
    console.log('✅ PASS: Step 9 - Invitation email retrieved via IMAP');
    console.log(`   Invite link: ${inviteLink.substring(0, 60)}...`);
  } catch (error: any) {
    console.log(`❌ FAIL: Step 9 - ${error.message || error}`);
    throw error;
  }

  // Step 10: Verify extraction of email in the list
  console.log(`\n📍 Step 10: Verify ${invitedEmail} in Organization Team list`);
  const memberInList = page.locator('.organization-table td', { hasText: invitedEmail });
  await expect(memberInList).toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 10 - Invited email verified in team list');

  // Step 11: Logout Admin
  console.log('\n📍 Step 11: Logout Admin');
  const profileMenu = page.locator('#menu-activator');
  await profileMenu.click();
  const logoutBtn = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /Logout/i }).first();
  await logoutBtn.click();
  await page.waitForURL('**/login', { timeout: 30000 });
  console.log('✅ PASS: Step 11 - Admin logged out successfully');

  // Step 12: Open invitation link
  console.log('\n📍 Step 12: Open invitation link');
  await page.goto(inviteLink, { waitUntil: 'networkidle' });
  await expect(page).toHaveURL(/.*set-password/);
  console.log('✅ PASS: Step 12 - Invitation link opened');

  // Step 13: Set Password
  console.log('\n📍 Step 13: Set Password for the new user');
  const passwordField = page.locator('input[type="password"]').first();
  const confirmPasswordField = page.locator('input[type="password"]').nth(1);
  const setPasswordButton = page.getByRole('button', { name: /Set Password/i });

  await expect(passwordField).toBeVisible();
  await passwordField.fill(invitedPassword);
  await confirmPasswordField.fill(invitedPassword);
  await setPasswordButton.click();
  console.log('✅ PASS: Step 13 - Password set and submitted');

  // Step 14: Verify success and click Go To Dashboard
  console.log('\n📍 Step 14: Verify success and click Go To Dashboard');
  await expect(page.getByText(/Password set successfully/i)).toBeVisible({ timeout: 20000 });
  const goToDashboardBtn = page.getByRole('button', { name: /Go To Dashboard/i });
  await goToDashboardBtn.click();
  console.log('✅ PASS: Step 14 - Go To Dashboard clicked');

  // Step 15: Verify redirection to login page
  console.log('\n📍 Step 15: Verify redirection to login page');
  await page.waitForURL('**/login', { timeout: 30000 });
  await expect(page).toHaveURL(/.*login/);
  console.log('✅ PASS: Step 15 - Redirected to login page');

  // Step 16: Login with invited user
  console.log(`\n📍 Step 16: Login with ${invitedEmail}`);
  const loginEmailInput = page.locator('.v-form').locator('input').nth(0);
  const loginPassInput = page.locator('.v-form').locator('input').nth(1);
  await loginEmailInput.fill(invitedEmail);
  await loginPassInput.fill(invitedPassword);
  await page.locator('button[type="submit"]').click();
  console.log('✅ PASS: Step 16 - Login submitted');

  // Step 17: Verify redirection to selection page
  console.log('\n📍 Step 17: Verify redirection to selection page');
  await page.waitForURL('**/?select_org', { timeout: 30000 });
  console.log('✅ PASS: Step 17 - Redirected to organization selection page');

  // Step 18: Verify organization visibility
  console.log(`\n📍 Step 18: Verify ${organizationName} visible in selector`);
  // Wait for loader to disappear if it's there
  if (await page.locator('.loading-container').isVisible().catch(() => false)) {
    await expect(page.locator('.loading-container')).not.toBeVisible({ timeout: 15000 });
  }
  const orgCard = page.locator('.organization-card').filter({ hasText: organizationName });
  await expect(orgCard).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 18 - Organization visible');

  // Step 19: Select organization
  console.log('\n📍 Step 19: Select organization');
  await orgCard.click();
  console.log('✅ PASS: Step 19 - Organization card clicked');

  // Step 20: Verify dashboard access
  console.log('\n📍 Step 20: Verify dashboard access');
  await page.waitForURL(url => url.pathname === '/' && !url.searchParams.has('select_org'), { timeout: 30000 });
  console.log('✅ PASS: Step 20 - Dashboard access confirmed');

  // Step 21: Report results
  console.log('\n📍 Step 21: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1 & 2: PASS - Login and select organization');
  console.log('✅ Step 3: PASS - Organization settings page opened');
  console.log('✅ Step 4: PASS - Add A Member button clicked');
  console.log('✅ Step 5: PASS - Email entered: ' + invitedEmail);
  console.log('✅ Step 6: PASS - Role admin selected');
  console.log('✅ Step 7: PASS - Add button clicked');
  console.log('✅ Step 8: PASS - Success notification verified');
  console.log('✅ Step 9: PASS - Invitation email retrieved via IMAP');
  console.log('✅ Step 10: PASS - Invited email verified in team list');
  console.log('✅ Step 11: PASS - Admin logged out successfully');
  console.log('✅ Step 12: PASS - Invitation link opened');
  console.log('✅ Step 13: PASS - Password set for new user');
  console.log('✅ Step 14: PASS - Go To Dashboard clicked');
  console.log('✅ Step 15: PASS - Redirected to login page');
  console.log('✅ Step 16: PASS - Login successful with invited member');
  console.log('✅ Step 17: PASS - Redirected to organization selection page');
  console.log('✅ Step 18: PASS - Organization Testing2026 visible');
  console.log('✅ Step 19: PASS - Organization Testing2026 selected');
  console.log('✅ Step 20: PASS - Redirected to dashboard');
  console.log('✅ Step 21: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
