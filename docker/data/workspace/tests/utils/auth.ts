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

    // Wait for loader to disappear
    const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
    if (await loader.first().isVisible().catch(() => false)) {
        await expect(loader.first()).not.toBeVisible({ timeout: 15000 });
    }

    /**
     * FIX: Find the .organization-card that has an internal .organization-name span
     * that matches the name exactly (ignoring surrounding whitespace).
     */
    const nameRegex = new RegExp(`^\\s*${organizationName}\\s*$`, 'i');

    // This finds the card containing the EXACT name element
    const orgCard = page.locator('.organization-card').filter({
        has: page.locator('.organization-name', { hasText: nameRegex })
    }).first();

    if (await orgCard.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log(`Found correct organization card for: ${organizationName}`);
        await orgCard.click();
    } else {
        // Fallback for Sidebar dropdown
        console.log('Main card not found, checking sidebar dropdown...');
        const trigger = page.locator('.org-dropdown-trigger');
        await expect(trigger).toBeVisible({ timeout: 5000 });

        const currentOrgLabel = await trigger.locator('.org-name').innerText();
        if (currentOrgLabel.trim().toLowerCase() === organizationName.toLowerCase()) {
            console.log(`✅ ${organizationName} is already selected.`);
            return;
        }

        await trigger.click();

        /**
         * FIX: Look for the specific .org-name span inside the list item
         */
        const dropdownItem = page.locator('.org-dropdown-item').filter({
            has: page.locator('.org-name', { hasText: nameRegex })
        }).first();

        await expect(dropdownItem).toBeVisible({ timeout: 3000 });
        await dropdownItem.click();
    }

    // Wait for the URL/UI update
    await page.waitForLoadState('networkidle');
    // FIX: Target the parent container instead of multiple individual elements
    const selectOrgContainer = page.locator('.select-org-container');

    // Wait for the URL to normalize (no more query params like ?select_org)
    await page.waitForURL(url => !url.searchParams.has('select_org'), { timeout: 15000 });

    // Ensure the container itself is gone
    await expect(selectOrgContainer).not.toBeVisible({ timeout: 10000 });

    console.log(`✅ PASS: Organization ${organizationName} selected`);
}

/**
 * Helper to perform login and select an organization.
 */
export async function loginAndSelectOrg(page: Page, email: string, password: string, organizationName: string) {
    await performLogin(page, email, password);
    await selectOrganization(page, organizationName);
}
