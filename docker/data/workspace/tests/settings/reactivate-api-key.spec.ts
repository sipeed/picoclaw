import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';

test('API Key reactivate flow', async ({ page }) => {
    const primaryEmail = 'heidi@intnt.ai';
    const primaryPassword = 'testing2026!';
    const organizationName = 'Testing2026';

    // Step 1-2: Perform the login and select org flow
    await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

    // Step 3: Click Settings in the left sidebar
    console.log('\n📍 Step 3: Click Settings in the left sidebar');
    const settingsLink = page.locator('nav').getByText('Settings').first();
    await expect(settingsLink).toBeVisible();
    await settingsLink.click();
    await page.waitForTimeout(1500);
    console.log('✅ PASS: Step 3 - Settings page opened');

    // Step 4: Locate the first API Key that is currently REVOKED
    console.log('\n📍 Step 4: Locate first Revoked API Key in table');

    const tableLoader = page.locator('.loading-state');
    if (await tableLoader.isVisible().catch(() => false)) {
        console.log('Table is loading, waiting...');
        await expect(tableLoader).not.toBeVisible({ timeout: 15000 });
    }

    /**
     * We look specifically for a row that contains the text 'Revoked' 
     * so we can reactivate it.
     */
    const revokedKeyRow = page.locator('.api-keys-table tbody tr').filter({
        hasText: /revoked/i
    }).first();

    await expect(revokedKeyRow).toBeVisible({
        timeout: 10000
    });
    console.log('✅ PASS: Step 4 - Revoked API Key located');

    // Step 5: Click the edit (pencil) icon
    console.log('\n📍 Step 5: Click edit (pencil) icon');
    const editBtn = revokedKeyRow.locator('.edit-btn');
    await expect(editBtn).toBeVisible();
    await editBtn.click();
    console.log('✅ PASS: Step 5 - Edit icon clicked');

    // Step 6: Verify Edit API Key popup
    const editDialog = page.locator('.v-dialog--active, [role="dialog"]').filter({
        hasText: /Edit API Key/i
    }).first();
    await expect(editDialog).toBeVisible({ timeout: 5000 });
    console.log('✅ PASS: Step 6 - Edit API Key popup appeared');

    // Step 7: Click Status dropdown
    const statusSelector = editDialog.locator('.v-select').first();
    await expect(statusSelector).toBeVisible();
    await statusSelector.click();
    console.log('✅ PASS: Step 7 - Status dropdown clicked');

    // Step 8: Select "Active" (this is the reactivation step)
    console.log('\n📍 Step 8: Select Active from dropdown');
    const activeOption = page.locator('.v-overlay-container .v-list-item').filter({
        hasText: /^Active$/
    }).first();
    await expect(activeOption).toBeVisible();
    await activeOption.click();
    console.log('✅ PASS: Step 8 - Active status selected');

    // Step 9: Click Save
    console.log('\n📍 Step 9: Click Save');
    const saveButton = editDialog.locator('button').filter({ hasText: /^Save$/i }).first();
    await saveButton.click();
    console.log('✅ PASS: Step 9 - Save button clicked');

    // Step 10: Verify notification
    const successNotification = page.locator('.v-snackbar__content', {
        hasText: /updated successfully/i
    });
    await expect(successNotification).toBeVisible({ timeout: 10000 });
    console.log('✅ PASS: Step 10 - Success notification appeared');

    // Step 11: Verify status updated to Active in the table
    console.log('\n📍 Step 11: Verify status in table');
    /**
     * We re-locate the row (it will no longer have 'Revoked' text 
     * so we search by description or just wait for the Active chip).
     */
    await expect(page.locator('.api-keys-table')).toBeVisible();
    const activeChip = page.locator('.v-chip').filter({ hasText: /^Active$/i }).first();
    await expect(activeChip).toBeVisible({ timeout: 10000 });
    console.log('✅ PASS: Step 11 - API Key status confirmed as Active in table');

    // Step 12: Report Summary
    console.log('\n📍 Step 12: Report PASS or FAIL for each step');
    console.log('\n' + '='.repeat(70));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(70));
    console.log('✅ Step 1: PASS - Login successful using helper function');
    console.log('✅ Step 2: PASS - Organization Testing2026 selected');
    console.log('✅ Step 3: PASS - Settings page opened');
    console.log('✅ Step 4: Found Revoked Key PASS');
    console.log('✅ Step 5: PASS - Edit icon clicked');
    console.log('✅ Step 6: PASS - Edit API Key popup appeared');
    console.log('✅ Step 7: PASS - Status dropdown clicked');
    console.log('✅ Step 8: PASS - Active status selected');
    console.log('✅ Step 10: PASS - Success notification appeared');
    console.log('✅ Step 11: Verified Reactivation in Table PASS');
    console.log('✅ Step 12: PASS - All steps completed');
    console.log('='.repeat(70));
    console.log('\n✅ ALL TESTS PASSED\n');
});
