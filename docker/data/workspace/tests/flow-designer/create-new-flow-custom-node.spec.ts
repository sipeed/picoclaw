import { test, expect } from '@playwright/test';

test('Create new flow with Custom Node', async ({ page }) => {
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

  // Read canvas transform ONCE for node positioning
  const tf = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });

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

  // Position first Reply Message at canvas (200, 80)
  const replyWrapper1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  await replyWrapper1.waitFor({ state: 'visible', timeout: 15000 });
  const replyBBox1 = await replyWrapper1.boundingBox();
  if (!replyBBox1) throw new Error('First Reply Message node not found');
  await page.mouse.move(replyBBox1.x + replyBBox1.width / 2, replyBBox1.y + replyBBox1.height / 2);
  await page.mouse.down();
  await page.mouse.move(200 * tf.scale + tf.tx, 80 * tf.scale + tf.ty, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);

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
  const saveButton1 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton1.click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 15 - First Reply Message node saved');

  // ============================================================================
  // PHASE 7: CONNECT START → FIRST REPLY MESSAGE
  // ============================================================================

  console.log('📍 Step 16: Connect START → ' + firstReplyId);

  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); } // zoom in to max
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
  const replyHandle1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator(`.node-container#${firstReplyId}`) })
    .locator('.vue-flow__handle-top');

  const startBox = await startHandle.boundingBox();
  const replyBox1 = await replyHandle1.boundingBox();
  if (!startBox || !replyBox1) throw new Error(`Handles not found for START → ${firstReplyId}`);

  await page.mouse.move(startBox.x + startBox.width / 2, startBox.y + startBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(replyBox1.x + replyBox1.width / 2, replyBox1.y + replyBox1.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter1 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter1 <= edgesBefore1) {
    throw new Error(`Edge START → ${firstReplyId} NOT created — before: ${edgesBefore1}, after: ${edgesAfter1}`);
  }
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

  // Position Custom Node at canvas (450, 80) — re-read transform after zoom changes
  const tfCustom = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const customWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /CustomNode|Custom/ }) })
    .last();
  await customWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const customBBox = await customWrapper.boundingBox();
  if (!customBBox) throw new Error('Custom Node not found on canvas');
  await page.mouse.move(customBBox.x + customBBox.width / 2, customBBox.y + customBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(450 * tfCustom.scale + tfCustom.tx, 80 * tfCustom.scale + tfCustom.ty, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);

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
  const saveButton2 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton2.click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 21 - Custom Node saved');

  // ============================================================================
  // PHASE 10: CONNECT FIRST REPLY MESSAGE → CUSTOM NODE
  // ============================================================================

  console.log('📍 Step 22: Connect ' + firstReplyId + ' → CustomNode');

  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  const edgesBefore2 = await page.locator('.vue-flow__edge[data-id]').count();

  const replySourceHandle1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator(`.node-container#${firstReplyId}`) })
    .locator('.vue-flow__handle-bottom');
  const customTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#CustomNode') })
    .locator('.vue-flow__handle-top');

  const replySourceBox1 = await replySourceHandle1.boundingBox();
  const customTargetBox = await customTargetHandle.boundingBox();
  if (!replySourceBox1 || !customTargetBox) throw new Error(`Handles not found for ${firstReplyId} → CustomNode`);

  await page.mouse.move(replySourceBox1.x + replySourceBox1.width / 2, replySourceBox1.y + replySourceBox1.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(customTargetBox.x + customTargetBox.width / 2, customTargetBox.y + customTargetBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter2 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter2 <= edgesBefore2) {
    throw new Error(`Edge ${firstReplyId} → CustomNode NOT created — before: ${edgesBefore2}, after: ${edgesAfter2}`);
  }
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

  // Position output node at canvas (700, 80) — re-read transform after zoom changes
  const tfOutput = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const replyWrappers = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) });
  const outputWrapper = replyWrappers.last();
  await outputWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const outputBBox = await outputWrapper.boundingBox();
  if (!outputBBox) throw new Error('Second Reply Message node not found');
  await page.mouse.move(outputBBox.x + outputBBox.width / 2, outputBBox.y + outputBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(700 * tfOutput.scale + tfOutput.tx, 80 * tfOutput.scale + tfOutput.ty, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);

  // ============================================================================
  // PHASE 12: CONFIGURE OUTPUT NODE
  // ============================================================================

  console.log('📍 Step 25: Click output node to open modal and change Node ID to output');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).last().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });

  const outputNodeIdInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
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
  const saveButton3 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton3.click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 28 - Output node saved');

  // ============================================================================
  // PHASE 13: CONNECT CUSTOMNODE → OUTPUT
  // ============================================================================

  console.log('📍 Step 29: Connect CustomNode → output');

  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  const edgesBefore3 = await page.locator('.vue-flow__edge[data-id]').count();

  const customSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#CustomNode') })
    .locator('.vue-flow__handle-bottom');
  const outputTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#output') })
    .locator('.vue-flow__handle-top');

  const customSourceBox = await customSourceHandle.boundingBox();
  const outputTargetBox = await outputTargetHandle.boundingBox();
  if (!customSourceBox || !outputTargetBox) throw new Error('Handles not found for CustomNode → output');

  await page.mouse.move(customSourceBox.x + customSourceBox.width / 2, customSourceBox.y + customSourceBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(outputTargetBox.x + outputTargetBox.width / 2, outputTargetBox.y + outputTargetBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter3 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter3 <= edgesBefore3) {
    throw new Error(`Edge CustomNode → output NOT created — before: ${edgesBefore3}, after: ${edgesAfter3}`);
  }
  console.log('✅ PASS: Step 29 - Edge CustomNode → output created');

  // ============================================================================
  // PHASE 14: CONNECT OUTPUT → END
  // ============================================================================

  console.log('📍 Step 30: Connect output → END');

  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);

  const edgesBefore4 = await page.locator('.vue-flow__edge[data-id]').count();

  const outputSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#output') })
    .locator('.vue-flow__handle-bottom');
  const endTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');

  const outputSourceBox = await outputSourceHandle.boundingBox();
  const endTargetBox = await endTargetHandle.boundingBox();
  if (!outputSourceBox || !endTargetBox) throw new Error('Handles not found for output → END');

  await page.mouse.move(outputSourceBox.x + outputSourceBox.width / 2, outputSourceBox.y + outputSourceBox.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(endTargetBox.x + endTargetBox.width / 2, endTargetBox.y + endTargetBox.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);

  const edgesAfter4 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter4 <= edgesBefore4) {
    throw new Error(`Edge output → END NOT created — before: ${edgesBefore4}, after: ${edgesAfter4}`);
  }
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
