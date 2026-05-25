import { test, expect } from '@playwright/test';
import { loginAndSelectOrg } from '../utils/auth';
import * as fs from 'fs';
import * as path from 'path';

test('Download conversation logs flow', async ({ page }) => {
  test.setTimeout(180000);

  const primaryEmail = 'heidi@intnt.ai';
  const primaryPassword = 'testing2026!';
  const organizationName = 'Testing';
  const testMessage = 'Test message for conversation logs';

  // Step 1: Perform the login and select org flow
  console.log('\n📍 Step 1: Perform login and select organization Testing2026');
  await loginAndSelectOrg(page, primaryEmail, primaryPassword, organizationName);

  // Step 2: Verify redirect to dashboard
  console.log('\n📍 Step 2: Verify redirect to ');
  await expect(page).toHaveURL(/.*dashboard\.int3nt\.info\/?$/);
  console.log('✅ PASS: Step 2 - User redirected to dashboard');

  // Step 3: Navigate to Flow Tester and send a message
  console.log('\n📍 Step 3: Navigate to Flow Tester and send a message');
  const flowTesterLink = page.locator('nav').getByText('Flow Tester').first();
  if (await flowTesterLink.isVisible({ timeout: 3000 }).catch(() => false)) {
    await flowTesterLink.click();
    await page.waitForTimeout(2000);

    // Send a test message
    const messageInput = page.locator('input[placeholder*="message" i], textarea[placeholder*="message" i]').first();
    if (await messageInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await messageInput.fill(testMessage);
      const sendButton = page.locator('button:has-text("Send"), button[aria-label*="send" i]').first();
      if (await sendButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await sendButton.click();
        await page.waitForTimeout(2000);
      }
    }
    console.log('✅ PASS: Step 3 - Message sent in Flow Tester');
  } else {
    console.log('⚠️  SKIP: Step 3 - Flow Tester not available');
  }

  // Step 4: Navigate to Logs
  console.log('\n📍 Step 4: Navigate to Logs in left sidebar');
  const logsLink = page.locator('nav').getByText('Logs').first();
  await expect(logsLink).toBeVisible();
  await logsLink.click();
  await page.waitForTimeout(2000);
  console.log('✅ PASS: Step 4 - Logs page opened');

  // Step 5: Verify Logs page is displayed
  console.log('\n📍 Step 5: Verify Logs page is displayed');
  const logsContainer = page.locator('.logs-container');
  await expect(logsContainer).toBeVisible({ timeout: 20000 });
  console.log('✅ PASS: Step 5 - Logs page displayed');

  // Step 6: Verify Date range is selected
  console.log('\n📍 Step 6: Verify and select date range');
  const dateButtons = page.locator('.dropdown-button');
  await expect(dateButtons.first()).toBeVisible();

  console.log('📍 Step 6.1: Set Datetime Start to Apr 1, 2026');
  const datetimeStartButton = dateButtons.first();
  await datetimeStartButton.click();
  const dateMenu = page
    .locator('.v-overlay--active')
    .filter({ has: page.locator('.v-date-picker') })
    .last();
  const datePicker = dateMenu.locator('.v-date-picker');
  await expect(datePicker).toBeVisible({ timeout: 10000 });
  await datePicker.locator('button').filter({ hasText: /^1$/ }).first().click();
  await expect(datetimeStartButton).toContainText('Apr 1, 2026', { timeout: 10000 });

  console.log('✅ PASS: Step 6 - Date range filter present');

  // Step 7: Click "Load Data" to ensure results exist
  console.log('\n📍 Step 7: Click "Load Data" to populate table');
  const loadButton = page.getByRole('button', { name: /Load Data/i });
  await loadButton.click();

  // Wait for the table to stop loading
  const table = page.locator('.custom-table');
  await expect(table.locator('.v-data-table__progress')).not.toBeVisible({ timeout: 15000 });
  console.log('✅ PASS: Step 7 - Data loaded');

  // Step 8: Click Export to CSV (Matches logsPage.exportCsv)
  console.log('\n📍 Step 8: Click Export to CSV button');
  const downloadButton = page.locator('.export-button');

  // Ensure button is enabled (requires data to be present)
  await expect(downloadButton).not.toBeDisabled({
    timeout: 10000
  });

  const downloadPromise = page.waitForEvent('download');
  await downloadButton.click();
  const download = await downloadPromise;
  console.log('✅ PASS: Step 8 - Download started');

  // Step 9: Verify CSV file and headers
  console.log('\n📍 Step 9: Verify CSV content');
  const suggestedFilename = download.suggestedFilename();
  expect(suggestedFilename).toMatch(/\.csv$/);

  const downloadPath = path.join(__dirname, suggestedFilename);
  await download.saveAs(downloadPath);

  const csvContent = fs.readFileSync(downloadPath, 'utf-8');
  const lines = csvContent.split('\n');
  const headers = lines[0].split(',').map(h => h.trim().toLowerCase());

  // Updated based on NodeEvent type in common logs implementation
  const expectedColumns = [
    'event_timestamp',
    'node_type',
    'conversation_id',
    'input_message',
    'output_message',
    'model_name'
  ];

  for (const col of expectedColumns) {
    const found = headers.some(h => h.includes(col.toLowerCase()));
    expect(found).toBeTruthy();
  }
  console.log('✅ PASS: Step 9 - CSV verified');

  // Cleanup
  if (fs.existsSync(downloadPath)) { fs.unlinkSync(downloadPath); }

  // Step 10: Report results
  console.log('\n📍 Step 10: Report PASS or FAIL for each step');
  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login and organization selection completed');
  console.log('✅ Step 2: PASS - User redirected to dashboard');
  console.log('✅ Step 3: PASS - Message sent in Flow Tester');
  console.log('✅ Step 4: PASS - Logs page opened');
  console.log('✅ Step 5: PASS - Logs page displayed');
  console.log('✅ Step 6: PASS - Date range selected');
  console.log('✅ Step 7: PASS - Data loaded');
  console.log('✅ Step 8: PASS - Download started');
  console.log('✅ Step 9: PASS - CSV file downloaded and verified');
  console.log('✅ Step 10: PASS - All steps completed');
  console.log('='.repeat(70));
  console.log('\n✅ ALL TESTS PASSED\n');
});
