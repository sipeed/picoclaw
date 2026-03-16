import { Page, expect } from '@playwright/test';

/**
 * Helper to perform the login flow.
 * @param page - Playwright Page object.
 * @param email - User email.
 * @param password - User password.
 */
export async function performLogin(page: Page, email: string, password: string) {
    console.log(`\n📍 Performing login flow for ${email}`);
    await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/.*login/);

    const emailInput = page.locator('.v-text-field').nth(0).locator('input');
    const passwordInput = page.locator('.v-text-field').nth(1).locator('input');
    const loginButton = page.getByRole('button', { name: /login/i });

    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
    await expect(loginButton).toBeVisible();

    await emailInput.fill(email);
    await passwordInput.fill(password);
    await loginButton.click();

    await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 20000 });
    await expect(page).toHaveURL(/.*\?select_org/);
    console.log(`✅ PASS: Login successful with ${email}`);
}

/**
 * Helper to select an organization from the selection page.
 * @param page - Playwright Page object.
 * @param organizationName - Name of the organization to select.
 */
export async function selectOrganization(page: Page, organizationName: string) {
    console.log(`\n📍 Selecting organization: ${organizationName}`);

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

    console.log(`✅ PASS: Organization ${organizationName} selected`);
}

/**
 * Helper to perform login and select an organization.
 */
export async function loginAndSelectOrg(page: Page, email: string, password: string, organizationName: string) {
    await performLogin(page, email, password);
    await selectOrganization(page, organizationName);
}
