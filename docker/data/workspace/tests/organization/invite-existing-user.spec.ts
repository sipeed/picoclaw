import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('Invite existing user to organization flow', async ({ page }) => {
    const primaryEmail = 'heidi@intnt.ai';
    const primaryPassword = 'testing2026!';
    const secondaryEmail = 'heidi+1@intnt.ai';
    const secondaryPassword = 'testing2026!!';
    const organizationName = 'Testing2026!';
    const adminRole = 'admin';

    // Step 1 & 2: Login and select organization
    await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

    // Step 3: Click Organization in the left sidebar
    console.log('\n📍 Step 3: Click Organization in the left sidebar');
    const organizationLink = page.locator('nav').locator('a:has-text("Organization"), button:has-text("Organization"), [class*="sidebar"] >> text=Organization').first();
    await expect(organizationLink).toBeVisible();
    await organizationLink.click();
    await page.waitForTimeout(1000);
    console.log('✅ PASS: Step 3 - Organization menu clicked');

    // Step 4: Click Add a Member
    console.log('\n📍 Step 4: Click Add a Member');
    const addMemberButton = page.getByRole('button', { name: /Add a Member/i }).first();
    await expect(addMemberButton).toBeVisible({ timeout: 10000 });
    await addMemberButton.click();
    console.log('✅ PASS: Step 4 - Add a Member modal opened');

    // Step 5: Enter email heidi+1@intnt.ai
    console.log(`\n📍 Step 5: Enter email ${secondaryEmail}`);
    /**
     * Use getByRole('dialog') to find the modal, then find the input.
     * Since GlobalFormField doesn't use standard labels, we find the input 
     * by its position or by finding the div that contains the "Email" text.
     */
    const addMemberModal = page.getByRole('dialog').filter({ hasText: /Add a Member/i });
    await expect(addMemberModal).toBeVisible({ timeout: 10000 });

    const memberEmailInput = addMemberModal.locator('input').first();
    await expect(memberEmailInput).toBeVisible();
    await memberEmailInput.fill(secondaryEmail);
    console.log(`✅ PASS: Step 5 - Email entered: ${secondaryEmail}`);

    // Step 6: Select role admin
    console.log(`\n📍 Step 6: Select role ${adminRole}`);
    // The role selector is a v-select inside the modal
    const roleSelector = addMemberModal.locator('.v-select').first();
    await roleSelector.click();

    // Vuetify teleports the dropdown list to the global overlay container
    const adminOption = page.locator('.v-overlay-container .v-list-item').filter({
        hasText: new RegExp(`^${adminRole}$`, 'i')
    }).first();
    await expect(adminOption).toBeVisible({ timeout: 5000 });
    await adminOption.click();
    console.log(`✅ PASS: Step 6 - Role ${adminRole} selected`);

    // Step 7: Click Add and verify user is added
    console.log('\n📍 Step 7: Click Add and verify user is added');
    const addButton = addMemberModal.getByRole('button', { name: /^Add$/i }).first();
    await expect(addButton).toBeVisible();
    await addButton.click();

    // Wait for modal to close and table to refresh
    await expect(addMemberModal).not.toBeVisible({ timeout: 10000 });
    const memberInList = page.locator('.organization-table td', { hasText: secondaryEmail });
    await expect(memberInList).toBeVisible({ timeout: 15000 });
    console.log(`✅ PASS: Step 7 - User ${secondaryEmail} added to table`);

    // Step 8: Logout from current account
    console.log('\n📍 Step 8: Logout from current account');
    const profileMenu = page.locator('#menu-activator');
    await expect(profileMenu).toBeVisible();
    await profileMenu.click();

    const logoutBtn = page.locator('.v-overlay-container .v-list-item').filter({ hasText: /Logout/i }).first();
    await expect(logoutBtn).toBeVisible();
    await logoutBtn.click();

    await page.waitForURL('**/login', { timeout: 15000 });
    console.log('✅ PASS: Step 8 - Logged out successfully');

    // Step 9: Login with the invited user
    console.log(`\n📍 Step 9: Login with ${secondaryEmail}`);
    // Target the inputs specifically inside the login card
    const loginEmailInput = page.locator('.v-form').locator('input').nth(0);
    const loginPassInput = page.locator('.v-form').locator('input').nth(1);
    const loginButton = page.locator('button[type="submit"]');

    await loginEmailInput.fill(secondaryEmail);
    await loginPassInput.fill(secondaryPassword);
    await loginButton.click();

    // Wait for the redirect to the organization selection query
    await page.waitForURL('**/?select_org', { timeout: 15000 });
    console.log('✅ PASS: Step 9 - Login successful and redirected to selection page');

    // Step 10: Verify organization is available
    console.log(`\n📍 Step 10: selecting organization: ${organizationName}`);

    // 1. Check if we are on the "Select Organization" Page (rendered as a grid of cards)
    const orgCard = page.locator('.organization-card', { hasText: organizationName });

    // Wait for the loader to clear if it's there
    const loader = page.locator('.loading-container, .loading-spinner');
    if (await loader.first().isVisible().catch(() => false)) {
        await expect(loader.first()).not.toBeVisible({ timeout: 10000 });
    }
    if (await orgCard.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log(`Found organization card for: ${organizationName}`);
        await orgCard.click();
    } else {
        // 2. We are likely already inside the dashboard. Use the sidebar dropdown.
        console.log('Organization card not found, checking sidebar dropdown...');

        // Find the dropdown trigger in the navigation drawer
        const trigger = page.locator('.org-dropdown-trigger');
        await expect(trigger).toBeVisible({ timeout: 5000 });

        // Check if the target org is already selected (trigger displays the current org name)
        const currentOrgName = await trigger.locator('.org-name').innerText();
        if (currentOrgName.trim() === organizationName) {
            console.log(`✅ Organization ${organizationName} is already selected.`);
            return;
        }
        // Open the dropdown
        await trigger.click();

        // Find the item in the teleported menu (which is appended directly to body)
        // We look for .org-dropdown-item that contains the target name
        const dropdownItem = page.locator('.org-dropdown-menu .org-dropdown-item', {
            hasText: organizationName
        });

        await expect(dropdownItem).toBeVisible({ timeout: 3000 });
        await dropdownItem.click();
    }
    // 3. Wait for navigation/reload to complete
    // Instead of a hardcoded prod URL, we wait for the "select_org" query to disappear 
    // or specifically for the dashboard components to load.
    await page.waitForLoadState('networkidle');

    // Verify we are not on the select_org page anymore
    await expect(page.locator('.welcome-title')).not.toBeVisible({ timeout: 20000 });

    console.log(`✅ PASS: Step 10 - Organization ${organizationName} selected`);

    // Step 11: Confirm user is logged in
    console.log('\n📍 Step 11: Confirm profile menu is visible');
    const userProfile = page.locator('#menu-activator');
    await expect(userProfile).toBeVisible({ timeout: 10000 });
    console.log('✅ PASS: Step 11 - User confirmed logged in after org selection');

    // Step 12: Report results
    console.log('\n📍 Step 12: Report PASS or FAIL for each step');
    console.log('\n' + '='.repeat(70));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(70));
    console.log('✅ Step 1: PASS - Login successful with heidi@intnt.ai');
    console.log('✅ Step 2: PASS - Organization selected and redirected to dashboard');
    console.log('✅ Step 3: PASS - Organization menu clicked');
    console.log('✅ Step 4: PASS - Add a Member button clicked');
    console.log('✅ Step 5: PASS - Email entered: heidi+1@intnt.ai');
    console.log('✅ Step 6: PASS - Role admin selected');
    console.log('✅ Step 7: PASS - User heidi+1@intnt.ai added to organization');
    console.log('✅ Step 8: PASS - Logged out successfully');
    console.log('✅ Step 9: PASS - Login successful with heidi+1@intnt.ai');
    console.log('✅ Step 10: PASS - Organization testing2026 available');
    console.log('✅ Step 11: PASS - User heidi+1@intnt.ai confirmed logged in');
    console.log('✅ Step 12: PASS - All steps completed');
    console.log('='.repeat(70));
    console.log('\n✅ ALL TESTS PASSED\n');
});