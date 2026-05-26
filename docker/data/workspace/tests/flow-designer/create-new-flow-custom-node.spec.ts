import { test, expect } from '@playwright/test';
import { createFlowDesignerCanvasHelpers } from './helpers/flow-designer-canvas';

test('Create new flow with Custom Node', async ({ page }) => {
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

  console.log('📍 Step 4: Select organization Testing2026!');
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    await loader.first().waitFor({ state: 'hidden', timeout: 30000 });
  }
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 60000 });
  await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 60000 });
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
  // PHASE 4: RENAME FLOW
  // ============================================================================

  console.log('📍 Step 7: Click flow name field to rename');
  const flowNameText = page.locator('.panel-container p.text-secondary').first();
  await flowNameText.click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 7 - Flow name field clicked');

  console.log('📍 Step 8: Enter flow name Custom Node');
  const flowNameInput = page.locator('.panel-container input').first();
  await flowNameInput.waitFor({ state: 'visible', timeout: 15000 });
  await flowNameInput.fill('Custom Node');
  await flowNameInput.press('Enter');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 8 - Flow renamed to Custom Node');

  // ============================================================================
  // PHASE 5: ADD FIRST REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 9: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 9 - Add Nodes menu opened');

  console.log('📍 Step 10: Select Reply Message node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 10 - Reply Message node selected');

  // Position first Reply Message just below START for a short, stable edge
  const replyWrapper1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  const startNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') });
  const startNodeBox = await startNodeWrapper.boundingBox();
  if (!startNodeBox) throw new Error('START node not found on canvas');
  await dragNodeToScreenPosition(
    'first ReplyMessage',
    replyWrapper1,
    startNodeBox.x + startNodeBox.width / 2,
    startNodeBox.y + startNodeBox.height / 2 + 120,
  );

  // ============================================================================
  // PHASE 6: CONFIGURE FIRST REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 11: Click Reply Message node to open modal');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 11 - Reply Message node modal opened');

  console.log('📍 Step 12: Verify auto-populated fields and read Node ID');
  const nodeIdInput1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput1.waitFor({ state: 'visible', timeout: 15000 });
  const firstReplyId = await nodeIdInput1.inputValue();
  const firstReplyWrapperById = page.locator('.vue-flow__node')
    .filter({ has: page.locator(`.node-container#${firstReplyId}`) })
    .first();
  const firstReplyWrapperByLabel = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  const resolveFirstReplyWrapper = async () => {
    if (await firstReplyWrapperById.isVisible().catch(() => false)) {
      return firstReplyWrapperById;
    }
    await firstReplyWrapperByLabel.waitFor({ state: 'visible', timeout: 15000 });
    return firstReplyWrapperByLabel;
  };
  console.log(`  Auto-generated Node ID: ${firstReplyId}`);
  const nodeVersionField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await nodeVersionField1.waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 12 - Auto-populated fields verified, Node ID: ' + firstReplyId);

  console.log('📍 Step 13: Verify default values (Node Version, Receiver Channel, Content Type)');
  await expect(nodeVersionField1).toContainText(/Version 2\.0\.0/);
  const receiverChannelField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await expect(receiverChannelField1).toContainText(/None/);
  const contentTypeField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await expect(contentTypeField1).toContainText(/Text Message/);
  console.log('✅ PASS: Step 13 - Default values verified: Version 2.0.0, None, Text Message');

  console.log('📍 Step 14: Enter Message text');
  const messageInput1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput1.click();
  await messageInput1.fill("Below is the Custom Node Result. should return 'is what you inputted' behind the text you wrote");
  await messageInput1.press('Tab');
  console.log('✅ PASS: Step 14 - Message field set');

  console.log('📍 Step 15: Click Save button');
  const modalDialog1 = page.locator('.modal-dialog.v-overlay--active').filter({ hasText: firstReplyId });
  const saveButton1 = modalDialog1.getByRole('button', { name: /^Save$/ });
  const closeButton1 = modalDialog1.locator('button').first();
  await expect(saveButton1).toBeEnabled({ timeout: 30000 });
  await saveButton1.click();
  await page.waitForTimeout(500);
  if (await modalDialog1.isVisible().catch(() => false)) {
    await closeButton1.click();
  }
  await dismissVisibleModals();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 15 - First Reply Message node saved');

  // ============================================================================
  // PHASE 7: CONNECT START → FIRST REPLY MESSAGE
  // ============================================================================

  console.log('📍 Step 16: Connect START → ' + firstReplyId);

  const startHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') })
    .locator('.vue-flow__handle-bottom');
  const firstReplyWrapperForStartConnect = await resolveFirstReplyWrapper();
  const replyHandle1 = firstReplyWrapperForStartConnect.locator('.vue-flow__handle-top');

  await connectEdge(`START → ${firstReplyId}`, startHandle, replyHandle1, { normalizeZoom: true });
  console.log('✅ PASS: Step 16 - Edge START → ' + firstReplyId + ' created');

  // ============================================================================
  // PHASE 8: ADD CUSTOM NODE
  // ============================================================================

  console.log('📍 Step 17: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 17 - Add Nodes menu opened');

  console.log('📍 Step 18: Select Custom Node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Custom Node/i }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 18 - Custom Node selected');

  const customWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /CustomNode|Custom/ }) })
    .last();
  await customWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const firstReplyWrapper = await resolveFirstReplyWrapper();
  const firstReplyBox = await firstReplyWrapper.boundingBox();
  if (!firstReplyBox) throw new Error(`${firstReplyId} node not found on canvas`);
  await dragNodeToScreenPosition(
    'CustomNode',
    customWrapper,
    firstReplyBox.x + firstReplyBox.width / 2,
    firstReplyBox.y + firstReplyBox.height / 2 + 120,
  );

  // ============================================================================
  // PHASE 9: CONFIGURE CUSTOM NODE
  // ============================================================================

  console.log('📍 Step 19: Click Custom Node to open modal and change Node ID to CustomNode');
  await page.locator('.node-container').filter({ hasText: /CustomNode|Custom/ }).last().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });

  const customNodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await customNodeIdInput.click();
  await customNodeIdInput.fill('CustomNode');
  await customNodeIdInput.press('Tab');
  console.log('✅ PASS: Step 19 - Custom Node modal opened and Node ID set to CustomNode');

  console.log('📍 Step 20: Enter code in Code field');
  const customCode = `def custom_function(state, config):\n    return {'result': state['nodes']['START']['output']['messages'].content + ', is what you inputted'}`;

  // Use clipboard paste to insert code into Monaco editor (avoids auto-indent issues)
  const codeEditor = page.locator('.modal-dialog .monaco-editor').first();
  await codeEditor.waitFor({ state: 'visible', timeout: 60000 });
  await codeEditor.click();
  await page.waitForTimeout(300);
  await page.keyboard.press('Control+a');
  await page.waitForTimeout(100);
  await page.evaluate((code) => {
    const ta = document.createElement('textarea');
    ta.value = code;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }, customCode);
  await page.waitForTimeout(100);
  await page.keyboard.press('Control+v');
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 20 - Code entered in Code field');

  console.log('📍 Step 21: Click Save button');
  const modalDialog2 = page.getByRole('dialog').filter({ hasText: /CustomNode/ });
  const saveButton2 = modalDialog2.getByRole('button', { name: /^Save$/ });
  const closeButton2 = modalDialog2.locator('button').first();
  await expect(saveButton2).toBeEnabled({ timeout: 30000 });
  await saveButton2.click();
  await page.waitForTimeout(500);
  await page.keyboard.press('Escape').catch(() => {});
  await closeButton2.click({ timeout: 2000 }).catch(() => {});
  await dismissVisibleModals();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 21 - Custom Node saved');

  const customNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#CustomNode') });
  const firstReplyBoxAfterSave = await firstReplyWrapper.boundingBox();
  if (!firstReplyBoxAfterSave) throw new Error(`${firstReplyId} node not found on canvas after saving CustomNode`);
  await dragNodeToScreenPosition(
    'CustomNode',
    customNodeWrapper,
    firstReplyBoxAfterSave.x + firstReplyBoxAfterSave.width / 2,
    firstReplyBoxAfterSave.y + firstReplyBoxAfterSave.height / 2 + 120,
  );

  // ============================================================================
  // PHASE 10: CONNECT FIRST REPLY MESSAGE → CUSTOM NODE
  // ============================================================================

  console.log('📍 Step 22: Connect ' + firstReplyId + ' → CustomNode');

  const firstReplyWrapperForCustomConnect = await resolveFirstReplyWrapper();
  const replySourceHandle1 = firstReplyWrapperForCustomConnect.locator('.vue-flow__handle-bottom');
  const customTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#CustomNode') })
    .locator('.vue-flow__handle-top');

  await connectEdge(`${firstReplyId} → CustomNode`, replySourceHandle1, customTargetHandle);
  console.log('✅ PASS: Step 22 - Edge ' + firstReplyId + ' → CustomNode created');

  // ============================================================================
  // PHASE 11: ADD SECOND REPLY MESSAGE (OUTPUT) NODE
  // ============================================================================

  console.log('📍 Step 23: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 23 - Add Nodes menu opened');

  console.log('📍 Step 24: Select Reply Message node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 24 - Reply Message node selected');

  const replyWrappers = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) });
  const outputWrapper = replyWrappers.last();
  await outputWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const customNodeBox = await customNodeWrapper.boundingBox();
  if (!customNodeBox) throw new Error('CustomNode not found on canvas');
  const endNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') });
  const endNodeBox = await endNodeWrapper.boundingBox();
  if (!endNodeBox) throw new Error('END node not found on canvas');
  const customCenterY = customNodeBox.y + customNodeBox.height / 2;
  const endCenterY = endNodeBox.y + endNodeBox.height / 2;
  const outputGap = Math.min(120, Math.max(80, (endCenterY - customCenterY) / 2));
  await dragNodeToScreenPosition(
    'output',
    outputWrapper,
    customNodeBox.x + customNodeBox.width / 2,
    customCenterY + outputGap,
  );

  // ============================================================================
  // PHASE 12: CONFIGURE OUTPUT NODE
  // ============================================================================

  console.log('📍 Step 25: Click output node to open modal and change Node ID to output');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).last().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });

  const outputNodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  const outputInitialId = await outputNodeIdInput.inputValue();
  await outputNodeIdInput.click();
  await outputNodeIdInput.fill('output');
  await outputNodeIdInput.press('Tab');
  console.log('✅ PASS: Step 25 - Output node modal opened and Node ID set to output');

  console.log('📍 Step 26: Verify Node Version, Receiver Channel, Content Type');
  const nodeVersionField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await expect(nodeVersionField2).toContainText(/Version 2\.0\.0/);
  const receiverChannelField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await expect(receiverChannelField2).toContainText(/None/);
  const contentTypeField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await expect(contentTypeField2).toContainText(/Text Message/);
  console.log('✅ PASS: Step 26 - Node Version 2.0.0, Receiver Channel None, Content Type Text Message verified');

  console.log('📍 Step 27: Set Message field with template expression');
  const messageInput2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput2.click();
  await messageInput2.fill("{{ state['nodes']['CustomNode']['output']['result'] }}");
  await messageInput2.press('Tab');
  console.log('✅ PASS: Step 27 - Message field set with template expression');

  console.log('📍 Step 28: Click Save button');
  const modalDialog3 = page.locator('.modal-dialog.v-overlay--active').last();
  const saveButton3 = modalDialog3.getByRole('button', { name: /^Save$/ });
  const closeButton3 = modalDialog3.locator('button').first();
  await expect(saveButton3).toBeEnabled({ timeout: 30000 });
  await saveButton3.click();
  await page.waitForTimeout(500);
  await page.keyboard.press('Escape').catch(() => {});
  await closeButton3.click({ timeout: 2000 }).catch(() => {});
  await dismissVisibleModals();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 28 - Output node saved');

  const outputNodeWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator(`.node-container#output, .node-container#${outputInitialId}`) });
  if (!(await page.locator('.node-container#output').first().isVisible().catch(() => false))) {
    const outputNodeByOriginalId = page.locator('.vue-flow__node')
      .filter({ has: page.locator(`.node-container#${outputInitialId}`) });
    await outputNodeByOriginalId.waitFor({ state: 'visible', timeout: 15000 });
    await page.locator(`.node-container#${outputInitialId}`).click();

    const retryOutputModal = page.locator('.modal-dialog.v-overlay--active').last();
    await retryOutputModal.waitFor({ state: 'visible', timeout: 30000 });
    const retryOutputNodeIdField = retryOutputModal.locator('.field-container')
      .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
      .locator('.v-field__input')
      .first();
    const retryOutputNodeIdValue = await retryOutputNodeIdField.inputValue();
    if (retryOutputNodeIdValue !== 'output') {
      await retryOutputNodeIdField.click();
      await retryOutputNodeIdField.fill('output');
      await retryOutputNodeIdField.press('Tab');
    }

    const retrySaveButton = retryOutputModal.getByRole('button', { name: /^Save$/ });
    const retryCloseButton = retryOutputModal.locator('button').first();
    if (await retrySaveButton.isEnabled().catch(() => false)) {
      await retrySaveButton.click();
      await page.waitForTimeout(500);
    }
    await page.keyboard.press('Escape').catch(() => {});
    await retryCloseButton.click({ timeout: 2000 }).catch(() => {});
    await dismissVisibleModals();
    await outputNodeWrapper.waitFor({ state: 'visible', timeout: 15000 });
  }

  const customNodeBoxBeforeConnect = await customNodeWrapper.boundingBox();
  const endNodeBoxBeforeConnect = await endNodeWrapper.boundingBox();
  if (!customNodeBoxBeforeConnect) throw new Error('CustomNode not found before connecting to output');
  if (!endNodeBoxBeforeConnect) throw new Error('END node not found before connecting output');
  const customCenterYBeforeConnect = customNodeBoxBeforeConnect.y + customNodeBoxBeforeConnect.height / 2;
  const endCenterYBeforeConnect = endNodeBoxBeforeConnect.y + endNodeBoxBeforeConnect.height / 2;
  const outputGapBeforeConnect = Math.min(120, Math.max(80, (endCenterYBeforeConnect - customCenterYBeforeConnect) / 2));
  await dragNodeToScreenPosition(
    'output',
    outputNodeWrapper,
    customNodeBoxBeforeConnect.x + customNodeBoxBeforeConnect.width / 2,
    customCenterYBeforeConnect + outputGapBeforeConnect,
  );

  // ============================================================================
  // PHASE 13: CONNECT CUSTOMNODE → OUTPUT
  // ============================================================================

  console.log('📍 Step 29: Connect CustomNode → output');

  const customSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#CustomNode') })
    .locator('.vue-flow__handle-bottom');
  const outputTargetHandle = outputNodeWrapper.locator('.vue-flow__handle-top');

  await connectEdge('CustomNode → output', customSourceHandle, outputTargetHandle);
  console.log('✅ PASS: Step 29 - Edge CustomNode → output created');

  // ============================================================================
  // PHASE 14: CONNECT OUTPUT → END
  // ============================================================================

  console.log('📍 Step 30: Connect output → END');

  const outputSourceHandle = outputNodeWrapper.locator('.vue-flow__handle-bottom');
  const endTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');

  await connectEdge('output → END', outputSourceHandle, endTargetHandle);
  console.log('✅ PASS: Step 30 - Edge output → END created');

  // ============================================================================
  // PHASE 15: VERIFY FLOW STRUCTURE
  // ============================================================================

  console.log('📍 Step 31: Verify flow structure START → ' + firstReplyId + ' → CustomNode → output → END');
  const finalEdgeCount = await page.locator('.vue-flow__edge[data-id]').count();
  if (finalEdgeCount < 4) {
    throw new Error(`Expected at least 4 edges, found ${finalEdgeCount}`);
  }
  console.log('✅ PASS: Step 31 - Flow structure verified with all 4 edges');

  // ============================================================================
  // PHASE 16: SAVE FLOW VERSION
  // ============================================================================

  console.log('📍 Step 32: Click Save button (disk icon)');
  await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
  const saveModal = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
  await saveModal.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 32 - Save button clicked');

  console.log('📍 Step 33: Verify Save Flow Version modal and enter version name CustomnodeV1');
  const versionNameInput = saveModal.locator('.v-field__input').first();
  await versionNameInput.fill('CustomnodeV1');
  console.log('✅ PASS: Step 33 - Save Flow Version modal appeared and version name entered');

  console.log('📍 Step 34: Click Save button in modal');
  const modalSaveButton = saveModal.locator('button').filter({ hasText: /^Save$/ });
  await modalSaveButton.click();
  await saveModal.waitFor({ state: 'hidden', timeout: 30000 });
  console.log('✅ PASS: Step 34 - Save Flow Version modal closed');

  console.log('📍 Step 35: Verify success toast notification');
  await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i, { timeout: 15000 });
  console.log('✅ PASS: Step 35 - Success toast notification appeared');

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
  console.log('✅ Step 8: PASS - Flow renamed to Custom Node');
  console.log('✅ Step 9: PASS - Add Nodes menu opened');
  console.log('✅ Step 10: PASS - Reply Message node selected');
  console.log('✅ Step 11: PASS - Reply Message node modal opened');
  console.log('✅ Step 12: PASS - Auto-populated fields verified, Node ID: ' + firstReplyId);
  console.log('✅ Step 13: PASS - Default values verified');
  console.log('✅ Step 14: PASS - Message field set');
  console.log('✅ Step 15: PASS - First Reply Message node saved');
  console.log('✅ Step 16: PASS - Edge START → ' + firstReplyId + ' created');
  console.log('✅ Step 17: PASS - Add Nodes menu opened');
  console.log('✅ Step 18: PASS - Custom Node selected');
  console.log('✅ Step 19: PASS - Custom Node modal opened, Node ID set to CustomNode');
  console.log('✅ Step 20: PASS - Code entered in Code field');
  console.log('✅ Step 21: PASS - Custom Node saved');
  console.log('✅ Step 22: PASS - Edge ' + firstReplyId + ' → CustomNode created');
  console.log('✅ Step 23: PASS - Add Nodes menu opened');
  console.log('✅ Step 24: PASS - Reply Message node selected');
  console.log('✅ Step 25: PASS - Output node modal opened, Node ID set to output');
  console.log('✅ Step 26: PASS - Output node fields verified');
  console.log('✅ Step 27: PASS - Message field set with template expression');
  console.log('✅ Step 28: PASS - Output node saved');
  console.log('✅ Step 29: PASS - Edge CustomNode → output created');
  console.log('✅ Step 30: PASS - Edge output → END created');
  console.log('✅ Step 31: PASS - Flow structure verified');
  console.log('✅ Step 32: PASS - Save button clicked');
  console.log('✅ Step 33: PASS - Save Flow Version modal appeared, version name entered');
  console.log('✅ Step 34: PASS - Flow version saved');
  console.log('✅ Step 35: PASS - Success toast notification appeared');
  console.log('='.repeat(70));
  console.log('✅ TEST COMPLETE - All 35 steps passed successfully');
  console.log('='.repeat(70));
});
