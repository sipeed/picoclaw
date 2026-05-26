import { test, expect } from '@playwright/test';
import { createFlowDesignerCanvasHelpers } from './helpers/flow-designer-canvas';

test('Create new flow with Knowledge Base web crawler node', async ({ page }) => {
  test.setTimeout(300000);
  page.setDefaultTimeout(60000);
  page.setDefaultNavigationTimeout(60000);

  const { dismissVisibleModals, connectEdge, dragNodeToScreenPosition } = createFlowDesignerCanvasHelpers(page);

  // ============================================================================
  // PHASE 1: LOGIN
  // ============================================================================

  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.locator('.login-card').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 1 - Login page loaded');

  console.log('📍 Step 2: Fill email and password');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  console.log('✅ PASS: Step 2 - Credentials entered');

  console.log('📍 Step 3: Click login button');
  await page.getByRole('button', { name: /login/i }).click();
  await page.waitForURL(/\?select_org/, { timeout: 60000 });
  console.log('✅ PASS: Step 3 - Redirected to org selection');

  // ============================================================================
  // PHASE 2: ORGANIZATION SELECTION
  // ============================================================================

  console.log('📍 Step 4: Select organization Testing');
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 30000 });
  }
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 60000 });
  await page.locator('.organization-card').filter({ has: page.locator(':text-is("Testing")') }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 30000 });
  console.log('✅ PASS: Step 4 - Organization selected, redirected to dashboard');

  // ============================================================================
  // PHASE 3: NAVIGATE TO FLOW DESIGNER
  // ============================================================================

  console.log('📍 Step 5: Click Flow Designer in sidebar');
  try {
    await page.locator('a[href*="flow-designer"]').first().click({ timeout: 60000 });
  } catch {
    await page.goto('/flow-designer', { waitUntil: 'networkidle' });
  }
  await page.waitForURL(/\/flow-designer$/, { timeout: 60000 });
  console.log('✅ PASS: Step 5 - Flow Designer page loaded');

  console.log('📍 Step 6: Click Add New button to create flow');
  await page.locator('button').filter({ hasText: /Add New/ }).first().click();
  await page.waitForURL(/\/flow-designer\/\d+/, { timeout: 30000 });
  await page.locator('.vue-flow').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 6 - Flow canvas opened with START and END nodes');

  // ============================================================================
  // PHASE 4: RENAME FLOW (EARLY)
  // ============================================================================

  console.log('📍 Step 7: Click flow name field to rename');
  const flowNameText = page.locator('.panel-container p.text-secondary').first();
  await flowNameText.click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 7 - Flow name field clicked');

  console.log('📍 Step 8: Enter flow name Web Crawler');
  const flowNameInput = page.locator('.panel-container input').first();
  await flowNameInput.waitFor({ state: 'visible', timeout: 15000 });
  await flowNameInput.fill('Web Crawler');
  await flowNameInput.press('Enter');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 8 - Flow renamed to Web Crawler');

  // ============================================================================
  // PHASE 5: ADD KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 9: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 9 - Add Nodes menu opened');

  console.log('📍 Step 10: Select Knowledge Base Node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Knowledge Base Node/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 10 - Knowledge Base Node selected');

  // ============================================================================
  // PHASE 6: POSITION KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 11: Position Knowledge Base node below START');
  const kbWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }) })
    .first();
  await kbWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const kbBBox = await kbWrapper.boundingBox();
  if (!kbBBox) throw new Error('Knowledge Base node not found');
  const startNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') });
  const startNodeBox = await startNodeWrapper.boundingBox();
  if (!startNodeBox) throw new Error('START node not found on canvas');
  await dragNodeToScreenPosition(
    'KnowledgeBaseNode',
    kbWrapper,
    startNodeBox.x + startNodeBox.width / 2,
    startNodeBox.y + startNodeBox.height / 2 + 120,
  );
  console.log('✅ PASS: Step 11 - Knowledge Base node positioned');

  // ============================================================================
  // PHASE 7: CONFIGURE KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 12: Click KB node to open modal and verify Is Tool is True');
  await page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  const isToolSelectInitial = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0);
  await expect(isToolSelectInitial).toContainText(/True/);
  console.log('✅ PASS: Step 12 - KB node modal opened and Is Tool verified as True');

  console.log('📍 Step 13: Change Node ID to wc');
  const nodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput.click();
  await nodeIdInput.fill('wc');
  await nodeIdInput.press('Tab');
  console.log('✅ PASS: Step 13 - Node ID changed to wc');

  console.log('📍 Step 14: Verify Size is 5');
  const sizeInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Size/ }) })
    .locator('input[type="number"]');
  const sizeValue = await sizeInput.inputValue();
  if (sizeValue !== '5') {
    throw new Error(`Size is ${sizeValue}, expected 5`);
  }
  console.log('✅ PASS: Step 14 - Size verified as 5');

  console.log('📍 Step 15: Verify Response Fields has "context" pre-filled');
  const responseFieldInput = page.locator('.modal-dialog .array-item-content input[type="text"]').last();
  await responseFieldInput.waitFor({ state: 'visible', timeout: 15000 });
  const responseFieldValue = await responseFieldInput.inputValue();
  if (responseFieldValue !== 'context') {
    throw new Error(`Response Fields expected "context", got "${responseFieldValue}"`);
  }
  console.log('✅ PASS: Step 15 - Response Fields verified with "context" pre-filled');

  console.log('📍 Step 16: Verify Source is Document Search Options');
  const sourceSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1);
  await expect(sourceSelect).toContainText(/Document Search/);
  console.log('✅ PASS: Step 16 - Source verified as Document Search Options');

  console.log('📍 Step 17: Select Document Search Cluster testing web crawler');
  const clusterSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2);
  await clusterSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /testing web crawler/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 17 - Document Search Cluster set to testing web crawler');

  console.log('📍 Step 18: Set Is Tool to False');
  const isToolSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0);
  await isToolSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /False/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 18 - Is Tool set to False');

  console.log('📍 Step 19: Add Search Field body - click Add Item');
  const addItemButtons = page.locator('.modal-dialog button').filter({ hasText: /Add Item/ });
  const firstAddButton = addItemButtons.first();
  await firstAddButton.click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 19 - Add Item clicked for Search Fields');

  console.log('📍 Step 20: Fill search field with body');
  const searchFieldInput = page.locator('.modal-dialog .array-item-content input[type="text"]').first();
  await searchFieldInput.waitFor({ state: 'visible', timeout: 15000 });
  await searchFieldInput.fill('body');
  await searchFieldInput.press('Tab');
  console.log('✅ PASS: Step 20 - Search Field filled with body');

  console.log('📍 Step 21: Set Min Score to 0');
  const minScoreInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /minimum|min.*score/i }) })
    .locator('.v-field__input');
  await minScoreInput.fill('0');
  await minScoreInput.press('Tab');
  console.log('✅ PASS: Step 21 - Min Score set to 0');

  console.log('📍 Step 22: Set Inner Hits Size to 10');
  const innerHitsInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Inner Hits/ }) })
    .locator('.v-field__input');
  await innerHitsInput.fill('10');
  await innerHitsInput.press('Tab');
  console.log('✅ PASS: Step 22 - Inner Hits Size set to 10');

  console.log('📍 Step 23: Set Size to 10');
  await sizeInput.fill('10');
  await sizeInput.press('Tab');
  console.log('✅ PASS: Step 23 - Size set to 10');

  console.log('📍 Step 24: Set Arguments to Knowledge Search Arguments');
  const argumentsSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(3);
  await argumentsSelect.scrollIntoViewIfNeeded();
  await argumentsSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Knowledge Search Arguments/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 24 - Arguments set to Knowledge Search Arguments');

  console.log('📍 Step 25: Verify Arguments is set to Knowledge Search Arguments');
  await expect(argumentsSelect).toContainText(/Knowledge Search Arguments/);
  console.log('✅ PASS: Step 25 - Arguments verified as Knowledge Search Arguments');

  console.log('📍 Step 26: Enter Query value intent');
  const queryInput = page.locator('.modal-dialog').getByRole('textbox').last();
  await queryInput.scrollIntoViewIfNeeded();
  await queryInput.fill('intent');
  await queryInput.press('Tab');
  console.log('✅ PASS: Step 26 - Query set to intent');

  console.log('📍 Step 27: Click Save button');
  const saveButton = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton.click();
  await page.waitForTimeout(500);
  await dismissVisibleModals();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 27 - Knowledge Base node saved');

  // ============================================================================
  // PHASE 8: CONNECT START → wc
  // ============================================================================

  console.log('📍 Step 28: Connect START → wc');

  const startHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') })
    .locator('.vue-flow__handle-bottom');
  const wcHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#wc') })
    .locator('.vue-flow__handle-top');
  await connectEdge('START → wc', startHandle, wcHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 28 - Edge START → wc created');

  // ============================================================================
  // PHASE 9: ADD REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 29: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 29 - Add Nodes menu opened');

  console.log('📍 Step 30: Select Reply Message node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 30 - Reply Message node selected');

  // ============================================================================
  // PHASE 10: POSITION REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 31: Position Reply Message node below wc');
  const replyWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  await replyWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const replyBBox = await replyWrapper.boundingBox();
  if (!replyBBox) throw new Error('Reply Message node not found');
  const wcNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#wc') });
  const wcNodeBox = await wcNodeWrapper.boundingBox();
  if (!wcNodeBox) throw new Error('wc node not found on canvas');
  await dragNodeToScreenPosition(
    'ReplyMessage',
    replyWrapper,
    wcNodeBox.x + wcNodeBox.width / 2,
    wcNodeBox.y + wcNodeBox.height / 2 + 120,
  );
  console.log('✅ PASS: Step 31 - Reply Message node positioned');

  // ============================================================================
  // PHASE 11: CONFIGURE REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 32: Click Reply Message node to open modal');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 32 - Reply Message node modal opened');

  console.log('📍 Step 33: Verify Node Version is Version 2.0.0');
  const nodeVersionField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await nodeVersionField.waitFor({ state: 'visible', timeout: 15000 });
  await expect(nodeVersionField).toContainText(/2\.0\.0/);
  console.log('✅ PASS: Step 33 - Node Version verified as Version 2.0.0');

  console.log('📍 Step 34: Verify Receiver Channel is None');
  const receiverChannelField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await receiverChannelField.waitFor({ state: 'visible', timeout: 15000 });
  await expect(receiverChannelField).toContainText(/None/);
  console.log('✅ PASS: Step 34 - Receiver Channel verified as None');

  console.log('📍 Step 35: Verify Content Type is Text Message');
  const contentTypeField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await contentTypeField.waitFor({ state: 'visible', timeout: 15000 });
  await expect(contentTypeField).toContainText(/Text Message/);
  console.log('✅ PASS: Step 35 - Content Type verified as Text Message');

  console.log('📍 Step 36: Change Node ID to ReplyMessage');
  const replyNodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await replyNodeIdInput.click();
  await replyNodeIdInput.fill('ReplyMessage');
  await replyNodeIdInput.press('Tab');
  console.log('✅ PASS: Step 36 - Node ID changed to ReplyMessage');

  console.log('📍 Step 37: Set Message field with template expression');
  const messageInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput.click();
  await messageInput.fill("{{ state['nodes']['wc']['output']['results'][0].context }}");
  await messageInput.press('Tab');
  console.log('✅ PASS: Step 37 - Message field set with template expression');

  console.log('📍 Step 38: Click Save button');
  const replySaveButton = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await replySaveButton.click();
  await page.waitForTimeout(500);
  await dismissVisibleModals();
  console.log('✅ PASS: Step 38 - Reply Message node saved');

  // ============================================================================
  // PHASE 12: CONNECT wc → REPLY MESSAGE
  // ============================================================================

  console.log('📍 Step 39: Connect wc → ReplyMessage');
  const wcSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#wc') })
    .locator('.vue-flow__handle-bottom');
  const replyTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-top');
  await connectEdge('wc → ReplyMessage', wcSourceHandle, replyTargetHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 39 - Edge wc → ReplyMessage created');

  // ============================================================================
  // PHASE 13: CONNECT REPLY MESSAGE → END
  // ============================================================================

  console.log('📍 Step 40: Connect ReplyMessage → END');
  const replySourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-bottom');
  const endTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');
  await connectEdge('ReplyMessage → END', replySourceHandle, endTargetHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 40 - Edge ReplyMessage → END created');

  // ============================================================================
  // PHASE 14: VERIFY FLOW STRUCTURE
  // ============================================================================

  console.log('📍 Step 41: Verify flow structure START → wc → ReplyMessage → END');
  const finalEdgeCount = await page.locator('.vue-flow__edge[data-id]').count();
  if (finalEdgeCount < 3) {
    throw new Error(`Expected at least 3 edges, found ${finalEdgeCount}`);
  }
  console.log('✅ PASS: Step 41 - Flow structure verified with all edges');

  // ============================================================================
  // PHASE 15: SAVE FLOW VERSION
  // ============================================================================

  console.log('📍 Step 42: Click Save button (disk icon)');
  await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
  const saveModal = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
  await saveModal.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 42 - Save button clicked and Save Flow Version modal appeared');

  console.log('📍 Step 43: Enter version name Webcrawler and save');
  const versionNameInput = saveModal.locator('.v-field__input').first();
  await versionNameInput.fill('Webcrawler');
  const modalSaveButton = saveModal.locator('button').filter({ hasText: /^Save$/ });
  await modalSaveButton.click();
  await saveModal.waitFor({ state: 'hidden', timeout: 30000 });
  await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i, { timeout: 15000 });
  console.log('✅ PASS: Step 43 - Flow version saved and success toast appeared');

  // ============================================================================
  // TEST SUMMARY
  // ============================================================================

  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Login page loaded');
  console.log('✅ Step 2: PASS - Credentials entered');
  console.log('✅ Step 3: PASS - Redirected to org selection');
  console.log('✅ Step 4: PASS - Organization selected');
  console.log('✅ Step 5: PASS - Flow Designer page loaded');
  console.log('✅ Step 6: PASS - Flow canvas opened');
  console.log('✅ Step 7: PASS - Flow name field clicked');
  console.log('✅ Step 8: PASS - Flow renamed to Web Crawler');
  console.log('✅ Step 9: PASS - Add Nodes menu opened');
  console.log('✅ Step 10: PASS - Knowledge Base Node selected');
  console.log('✅ Step 11: PASS - Knowledge Base node positioned');
  console.log('✅ Step 12: PASS - KB node modal opened, Is Tool verified as True');
  console.log('✅ Step 13: PASS - Node ID changed to wc');
  console.log('✅ Step 14: PASS - Size verified as 5');
  console.log('✅ Step 15: PASS - Response Fields verified with context pre-filled');
  console.log('✅ Step 16: PASS - Source verified as Document Search Options');
  console.log('✅ Step 17: PASS - Document Search Cluster set to testing web crawler');
  console.log('✅ Step 18: PASS - Is Tool set to False');
  console.log('✅ Step 19: PASS - Add Item clicked for Search Fields');
  console.log('✅ Step 20: PASS - Search Field filled with body');
  console.log('✅ Step 21: PASS - Min Score set to 0');
  console.log('✅ Step 22: PASS - Inner Hits Size set to 10');
  console.log('✅ Step 23: PASS - Size set to 10');
  console.log('✅ Step 24: PASS - Arguments set to Knowledge Search Arguments');
  console.log('✅ Step 25: PASS - Arguments verified');
  console.log('✅ Step 26: PASS - Query set to intent');
  console.log('✅ Step 27: PASS - Knowledge Base node saved');
  console.log('✅ Step 28: PASS - Edge START → wc created');
  console.log('✅ Step 29: PASS - Add Nodes menu opened');
  console.log('✅ Step 30: PASS - Reply Message node selected');
  console.log('✅ Step 31: PASS - Reply Message node positioned');
  console.log('✅ Step 32: PASS - Reply Message node modal opened');
  console.log('✅ Step 33: PASS - Node Version verified as Version 2.0.0');
  console.log('✅ Step 34: PASS - Receiver Channel verified as None');
  console.log('✅ Step 35: PASS - Content Type verified as Text Message');
  console.log('✅ Step 36: PASS - Node ID changed to ReplyMessage');
  console.log('✅ Step 37: PASS - Message field set with template expression');
  console.log('✅ Step 38: PASS - Reply Message node saved');
  console.log('✅ Step 39: PASS - Edge wc → ReplyMessage created');
  console.log('✅ Step 40: PASS - Edge ReplyMessage → END created');
  console.log('✅ Step 41: PASS - Flow structure verified');
  console.log('✅ Step 42: PASS - Save button clicked and modal appeared');
  console.log('✅ Step 43: PASS - Flow version saved and success toast appeared');
  console.log('='.repeat(70));
  console.log('✅ TEST COMPLETE - All 43 steps passed successfully');
  console.log('='.repeat(70));
});
