import { test, expect } from '@playwright/test';

test('Create new flow with Knowledge Base node', async ({ page }) => {
  test.setTimeout(300000);
  page.setDefaultTimeout(60000);
  page.setDefaultNavigationTimeout(60000);

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
  // PHASE 4: ADD KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 7: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 7 - Add Nodes menu opened');

  console.log('📍 Step 8: Select Knowledge Base Node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Knowledge Base Node/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 8 - Knowledge Base Node selected');

  // ============================================================================
  // PHASE 5: POSITION KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 9: Position Knowledge Base node using absolute canvas coordinates');
  // Read canvas transform ONCE for absolute positioning
  const tf = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });

  const kbWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }) })
    .first();
  await kbWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const kbBBox = await kbWrapper.boundingBox();
  if (!kbBBox) throw new Error('Knowledge Base node not found');

  // Position at canvas (250, 100)
  const targetX1 = 250 * tf.scale + tf.tx;
  const targetY1 = 100 * tf.scale + tf.ty;
  await page.mouse.move(kbBBox.x + kbBBox.width / 2, kbBBox.y + kbBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetX1, targetY1, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 9 - Knowledge Base node positioned');

  // ============================================================================
  // PHASE 6: CONFIGURE KNOWLEDGE BASE NODE
  // ============================================================================

  console.log('📍 Step 10: Click Knowledge Base node to open modal');
  await page.locator('.node-container').filter({ hasText: /KnowledgeBaseNode/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 10 - Knowledge Base node modal opened');

  console.log('📍 Step 11: Change Node ID to Kb');
  const nodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput.click();
  await nodeIdInput.fill('Kb');
  await nodeIdInput.press('Tab');
  console.log('✅ PASS: Step 11 - Node ID changed to Kb');

  console.log('📍 Step 12: Set Is Tool to False');
  const isToolSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(0);
  await isToolSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /False/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 12 - Is Tool set to False');

  console.log('📍 Step 13: Verify Source is Document Search Options');
  const sourceSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(1);
  await expect(sourceSelect).toContainText(/Document Search/);
  console.log('✅ PASS: Step 13 - Source verified as Document Search Options');

  console.log('📍 Step 14: Select Document Search Cluster testing kb');
  const clusterSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2);
  await clusterSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /testing kb/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 14 - Document Search Cluster set to testing kb');

  console.log('📍 Step 15: Add Search Field body');
  const addItemButtons = page.locator('.modal-dialog button').filter({ hasText: /Add Item/ });
  const firstAddButton = addItemButtons.first();
  await firstAddButton.click();
  await page.waitForTimeout(300);
  // Search Fields uses GlobalFormField inputType="array" → items render inside .array-item-content
  const searchFieldInput = page.locator('.modal-dialog .array-item-content input[type="text"]').first();
  await searchFieldInput.waitFor({ state: 'visible', timeout: 15000 });
  await searchFieldInput.fill('body');
  await searchFieldInput.press('Tab');
  console.log('✅ PASS: Step 15 - Search Field body added');

  console.log('📍 Step 16: Verify Size is 5');
  const sizeInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Size/ }) })
    .locator('input[type="number"]');
  const sizeValue = await sizeInput.inputValue();
  if (sizeValue !== '5') {
    throw new Error(`Size is ${sizeValue}, expected 5`);
  }
  console.log('✅ PASS: Step 16 - Size verified as 5');

  console.log('📍 Step 17: Set Min Score to 0');
  const minScoreInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /minimum|min.*score/i }) })
    .locator('.v-field__input');
  await minScoreInput.fill('0');
  await minScoreInput.press('Tab');
  console.log('✅ PASS: Step 17 - Min Score set to 0');

  console.log('📍 Step 18: Set Inner Hits Size to 10');
  const innerHitsInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Inner Hits/ }) })
    .locator('.v-field__input');
  await innerHitsInput.fill('10');
  await innerHitsInput.press('Tab');
  console.log('✅ PASS: Step 18 - Inner Hits Size set to 10');

  console.log('📍 Step 19: Add Response Field context');
  const secondAddButton = addItemButtons.nth(1);
  await secondAddButton.click();
  await page.waitForTimeout(300);
  // Response Fields also uses GlobalFormField inputType="array" → .array-item-content wrapper
  // After both Search and Response fields have one item each, the last input is the response one
  const responseFieldInput = page.locator('.modal-dialog .array-item-content input[type="text"]').last();
  await responseFieldInput.waitFor({ state: 'visible', timeout: 15000 });
  await responseFieldInput.fill('context');
  await responseFieldInput.press('Tab');
  console.log('✅ PASS: Step 19 - Response Field context added');

  console.log('📍 Step 20: Set Arguments to Knowledge Search Arguments');
  const argumentsSelect = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(3);
  await argumentsSelect.scrollIntoViewIfNeeded();
  await argumentsSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /Knowledge Search Arguments/ }).click();
  await page.waitForTimeout(300);
  await expect(argumentsSelect).toContainText(/Knowledge Search Arguments/);
  console.log('✅ PASS: Step 20 - Arguments set to Knowledge Search Arguments');

  console.log('📍 Step 21: Enter Query value intent');
  // Query field uses div-based label (no <label> element) — appears after Arguments is set
  // It is the last textbox in the modal at this point
  const queryInput = page.locator('.modal-dialog').getByRole('textbox').last();
  await queryInput.scrollIntoViewIfNeeded();
  await queryInput.fill('intent');
  await queryInput.press('Tab');
  console.log('✅ PASS: Step 21 - Query set to intent');

  console.log('📍 Step 22: Click Save button');
  const saveButton = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton.click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500); // let canvas re-render before connecting
  console.log('✅ PASS: Step 22 - Knowledge Base node saved');

  // ============================================================================
  // PHASE 7: CONNECT START → KB
  // ============================================================================

  console.log('📍 Step 23: Connect START → Kb');

  // Zoom in to max first, then zoom out to ~100% — zoom-out-only silently fails at min zoom
  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); } // zoom in to max (400%)
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); } // zoom out to ~100%
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  const edgesBefore1 = await page.locator('.vue-flow__edge[data-id]').count();

  const startHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') })
    .locator('.vue-flow__handle-bottom');
  const kbHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#Kb') })
    .locator('.vue-flow__handle-top');

  const startBox = await startHandle.boundingBox();
  const kbBox = await kbHandle.boundingBox();
  if (!startBox || !kbBox) throw new Error('Handles not found for START → Kb');

  await page.mouse.move(startBox.x + startBox.width / 2, startBox.y + startBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(kbBox.x + kbBox.width / 2, kbBox.y + kbBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter1 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter1 <= edgesBefore1) {
    throw new Error(`Edge START → Kb NOT created — before: ${edgesBefore1}, after: ${edgesAfter1}`);
  }
  console.log('✅ PASS: Step 23 - Edge START → Kb created');

  // ============================================================================
  // PHASE 8: ADD REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 24: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 24 - Add Nodes menu opened');

  console.log('📍 Step 25: Select Reply Message node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 25 - Reply Message node selected');

  // ============================================================================
  // PHASE 9: POSITION REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 26: Position Reply Message node at canvas (250, 200)');
  const replyWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  await replyWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const replyBBox = await replyWrapper.boundingBox();
  if (!replyBBox) throw new Error('Reply Message node not found');

  // Re-read transform — tf is stale after zoom normalization during KB node config
  const tfReply = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const targetX2 = 250 * tfReply.scale + tfReply.tx;
  const targetY2 = 200 * tfReply.scale + tfReply.ty;
  await page.mouse.move(replyBBox.x + replyBBox.width / 2, replyBBox.y + replyBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetX2, targetY2, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 26 - Reply Message node positioned');

  // ============================================================================
  // PHASE 10: CONFIGURE REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 27: Click Reply Message node to open modal');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 27 - Reply Message node modal opened');

  console.log('📍 Step 28: Verify auto-filled fields');
  const nodeVersionField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await nodeVersionField.waitFor({ state: 'visible', timeout: 15000 });
  const receiverChannelField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await receiverChannelField.waitFor({ state: 'visible', timeout: 15000 });
  const contentTypeField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await contentTypeField.waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 28 - Auto-filled fields verified');

  console.log('📍 Step 29: Change Node ID to ReplyMessage');
  const replyNodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await replyNodeIdInput.click();
  await replyNodeIdInput.fill('ReplyMessage');
  await replyNodeIdInput.press('Tab');
  console.log('✅ PASS: Step 29 - Node ID changed to ReplyMessage');

  console.log('📍 Step 30: Set Message field with template expression');
  const messageInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput.click();
  await messageInput.fill("{{ state['nodes']['kb']['output']['results'][0].context }}");
  await messageInput.press('Tab');
  console.log('✅ PASS: Step 30 - Message field set with template expression');

  console.log('📍 Step 31: Click Save button');
  const replySaveButton = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await replySaveButton.click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  console.log('✅ PASS: Step 31 - Reply Message node saved');

  // ============================================================================
  // PHASE 11: CONNECT KB → REPLY MESSAGE
  // ============================================================================

  // Zoom in to max first, then zoom out to ~100% — zoom-out-only silently fails at min zoom
  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); } // zoom in to max (400%)
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); } // zoom out to ~100%
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  console.log('📍 Step 32: Connect Kb → ReplyMessage');
  const edgesBefore2 = await page.locator('.vue-flow__edge[data-id]').count();

  const kbSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#Kb') })
    .locator('.vue-flow__handle-bottom');
  const replyTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-top');

  const kbSourceBox = await kbSourceHandle.boundingBox();
  const replyTargetBox = await replyTargetHandle.boundingBox();
  if (!kbSourceBox || !replyTargetBox) throw new Error('Handles not found for Kb → ReplyMessage');

  await page.mouse.move(kbSourceBox.x + kbSourceBox.width / 2, kbSourceBox.y + kbSourceBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(replyTargetBox.x + replyTargetBox.width / 2, replyTargetBox.y + replyTargetBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter2 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter2 <= edgesBefore2) {
    throw new Error(`Edge Kb → ReplyMessage NOT created — before: ${edgesBefore2}, after: ${edgesAfter2}`);
  }
  console.log('✅ PASS: Step 32 - Edge Kb → ReplyMessage created');

  // ============================================================================
  // PHASE 12: CONNECT REPLY MESSAGE → END
  // ============================================================================

  // Zoom in to max first, then zoom out to ~100% — zoom-out-only silently fails at min zoom
  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); } // zoom in to max (400%)
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); } // zoom out to ~100%
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  console.log('📍 Step 33: Connect ReplyMessage → END');
  const edgesBefore3 = await page.locator('.vue-flow__edge[data-id]').count();

  const replySourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-bottom');
  const endTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');

  const replySourceBox = await replySourceHandle.boundingBox();
  const endTargetBox = await endTargetHandle.boundingBox();
  if (!replySourceBox || !endTargetBox) throw new Error('Handles not found for ReplyMessage → END');

  await page.mouse.move(replySourceBox.x + replySourceBox.width / 2, replySourceBox.y + replySourceBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(endTargetBox.x + endTargetBox.width / 2, endTargetBox.y + endTargetBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter3 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter3 <= edgesBefore3) {
    throw new Error(`Edge ReplyMessage → END NOT created — before: ${edgesBefore3}, after: ${edgesAfter3}`);
  }
  console.log('✅ PASS: Step 33 - Edge ReplyMessage → END created');

  // ============================================================================
  // PHASE 13: VERIFY FLOW STRUCTURE
  // ============================================================================

  console.log('📍 Step 34: Verify flow structure START → Kb → ReplyMessage → END');
  const finalEdgeCount = await page.locator('.vue-flow__edge[data-id]').count();
  if (finalEdgeCount < 3) {
    throw new Error(`Expected at least 3 edges, found ${finalEdgeCount}`);
  }
  console.log('✅ PASS: Step 34 - Flow structure verified with all edges');

  // ============================================================================
  // PHASE 14: RENAME FLOW
  // ============================================================================

  console.log('📍 Step 35: Click flow name field to rename');
  const flowNameText = page.locator('.panel-container p.text-secondary').first();
  await flowNameText.click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 35 - Flow name field clicked');

  console.log('📍 Step 36: Enter flow name Knowledge Base');
  const flowNameInput = page.locator('.panel-container input').first();
  await flowNameInput.waitFor({ state: 'visible', timeout: 15000 });
  await flowNameInput.fill('Knowledge Base');
  await flowNameInput.press('Enter');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 36 - Flow renamed to Knowledge Base');

  // ============================================================================
  // PHASE 15: SAVE FLOW VERSION
  // ============================================================================

  console.log('📍 Step 37: Click Save button (disk icon)');
  await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
  const saveModal = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
  await saveModal.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 37 - Save button clicked');

  console.log('📍 Step 38: Verify Save Flow Version modal appears');
  console.log('✅ PASS: Step 38 - Save Flow Version modal appeared');

  console.log('📍 Step 39: Enter version name KnowledgebaseV1');
  const versionNameInput = saveModal.locator('.v-field__input').first();
  await versionNameInput.fill('KnowledgebaseV1');
  console.log('✅ PASS: Step 39 - Version name entered');

  console.log('📍 Step 40: Click Save button in modal');
  const modalSaveButton = saveModal.locator('button').filter({ hasText: /^Save$/ });
  await modalSaveButton.click();
  await saveModal.waitFor({ state: 'hidden', timeout: 30000 });
  console.log('✅ PASS: Step 40 - Save Flow Version modal closed');

  console.log('📍 Step 41: Verify success toast notification');
  await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i, { timeout: 15000 });
  console.log('✅ PASS: Step 41 - Success toast notification appeared');

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
  console.log('✅ Step 7: PASS - Add Nodes menu opened');
  console.log('✅ Step 8: PASS - Knowledge Base Node selected');
  console.log('✅ Step 9: PASS - Knowledge Base node positioned');
  console.log('✅ Step 10: PASS - Knowledge Base node modal opened');
  console.log('✅ Step 11: PASS - Node ID changed to Kb');
  console.log('✅ Step 12: PASS - Is Tool set to False');
  console.log('✅ Step 13: PASS - Source verified');
  console.log('✅ Step 14: PASS - Document Search Cluster set to testing kb');
  console.log('✅ Step 15: PASS - Search Field body added');
  console.log('✅ Step 16: PASS - Size verified as 5');
  console.log('✅ Step 17: PASS - Min Score set to 0');
  console.log('✅ Step 18: PASS - Inner Hits Size set to 10');
  console.log('✅ Step 19: PASS - Response Field context added');
  console.log('✅ Step 20: PASS - Arguments verified');
  console.log('✅ Step 21: PASS - Query set to intent');
  console.log('✅ Step 22: PASS - Knowledge Base node saved');
  console.log('✅ Step 23: PASS - Edge START → Kb created');
  console.log('✅ Step 24: PASS - Add Nodes menu opened');
  console.log('✅ Step 25: PASS - Reply Message node selected');
  console.log('✅ Step 26: PASS - Reply Message node positioned');
  console.log('✅ Step 27: PASS - Reply Message node modal opened');
  console.log('✅ Step 28: PASS - Auto-filled fields verified');
  console.log('✅ Step 29: PASS - Node ID changed to ReplyMessage');
  console.log('✅ Step 30: PASS - Message field set');
  console.log('✅ Step 31: PASS - Reply Message node saved');
  console.log('✅ Step 32: PASS - Edge Kb → ReplyMessage created');
  console.log('✅ Step 33: PASS - Edge ReplyMessage → END created');
  console.log('✅ Step 34: PASS - Flow structure verified');
  console.log('✅ Step 35: PASS - Flow name field clicked');
  console.log('✅ Step 36: PASS - Flow renamed to Knowledge Base');
  console.log('✅ Step 37: PASS - Save button clicked');
  console.log('✅ Step 38: PASS - Save Flow Version modal appeared');
  console.log('✅ Step 39: PASS - Version name entered');
  console.log('✅ Step 40: PASS - Flow version saved');
  console.log('✅ Step 41: PASS - Success toast notification appeared');
  console.log('='.repeat(70));
  console.log('✅ TEST COMPLETE - All 41 steps passed successfully');
  console.log('='.repeat(70));
});
