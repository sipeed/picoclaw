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
    const orgOption = page.getByText(organizationName).first();
    if (await orgOption.isVisible({ timeout: 3000 }).catch(() => false)) {
        await orgOption.click();
    } else {
        const selector = page.locator('[class*="dropdown"], [class*="select"], button[class*="org"]').first();
        if (await selector.isVisible({ timeout: 3000 }).catch(() => false)) {
            await selector.click();
            await page.waitForTimeout(500);
            const orgItem = page.getByText(organizationName).first();
            await orgItem.click();
        }
    }

    await page.waitForURL('**/dashboard.int3nt.info/', { timeout: 15000 });
    await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
    console.log(`✅ PASS: Organization ${organizationName} selected`);
}

/**
 * Helper to perform login and select an organization.
 */
export async function loginAndSelectOrg(page: Page, email: string, password: string, organizationName: string) {
    await performLogin(page, email, password);
    await selectOrganization(page, organizationName);
}
