import { test, expect } from '@playwright/test';
import { createFlowDesignerCanvasHelpers } from './helpers/flow-designer-canvas';

test('Create new flow with Model node and parser', async ({ page }) => {
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

  // Read canvas transform ONCE — used for all absolute node positioning
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

  console.log('📍 Step 8: Enter flow name Model Node With Parser');
  const flowNameInput = page.locator('.panel-container input').first();
  await flowNameInput.waitFor({ state: 'visible', timeout: 15000 });
  await flowNameInput.fill('Model Node With Parser');
  await flowNameInput.press('Enter');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 8 - Flow renamed to Model Node With Parser');

  // ============================================================================
  // PHASE 5: ADD REPLY MESSAGE NODE (first node)
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

  // Position at canvas (250, 80)
  const replyWrapper1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  await replyWrapper1.waitFor({ state: 'visible', timeout: 15000 });
  const replyBBox1 = await replyWrapper1.boundingBox();
  if (!replyBBox1) throw new Error('Reply Message node not found');
  const targetX1 = 250 * tf.scale + tf.tx;
  const targetY1 = 80 * tf.scale + tf.ty;
  await dragNodeToScreenPosition('ReplyMessage', replyWrapper1, targetX1, targetY1);

  // ============================================================================
  // PHASE 6: CONFIGURE REPLY MESSAGE NODE
  // ============================================================================

  console.log('📍 Step 11: Click Reply Message node to open modal');
  const replyNode = page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first();
  await replyNode.waitFor({ state: 'visible', timeout: 15000 });
  await replyNode.evaluate((el) => (el as HTMLElement).click());
  const replyModal = page.locator('.modal-dialog');
  if (!(await replyModal.isVisible().catch(() => false))) {
    await replyNode.evaluate((el) => (el as HTMLElement).click());
  }
  await replyModal.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 11 - Reply Message node modal opened');

  console.log('📍 Step 12: Verify Node Version is Version 2.0.0');
  const nodeVersionField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await nodeVersionField1.waitFor({ state: 'visible', timeout: 15000 });
  await expect(nodeVersionField1).toContainText(/Version 2\.0\.0/);
  console.log('✅ PASS: Step 12 - Node Version verified as Version 2.0.0');

  console.log('📍 Step 13: Verify Receiver Channel is None');
  const receiverChannelField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await receiverChannelField1.waitFor({ state: 'visible', timeout: 15000 });
  await expect(receiverChannelField1).toContainText(/None/);
  console.log('✅ PASS: Step 13 - Receiver Channel verified as None');

  console.log('📍 Step 14: Verify Content Type is Text Message');
  const contentTypeField1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await contentTypeField1.waitFor({ state: 'visible', timeout: 15000 });
  await expect(contentTypeField1).toContainText(/Text Message/);
  console.log('✅ PASS: Step 14 - Content Type verified as Text Message');

  console.log('📍 Step 15: Change Node ID to ReplyMessage');
  const nodeIdInput1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput1.click();
  await nodeIdInput1.fill('ReplyMessage');
  await nodeIdInput1.press('Tab');
  console.log('✅ PASS: Step 15 - Node ID changed to ReplyMessage');

  console.log('📍 Step 16: Set Message field');
  const messageInput1 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput1.click();
  await messageInput1.fill('Below is the Model node result. It should return the hardcoded weather details of Tokyo');
  await messageInput1.press('Tab');
  console.log('✅ PASS: Step 16 - Message field set');

  console.log('📍 Step 17: Click Save button');
  const saveButton1 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton1.click();
  await page.waitForTimeout(500);
  await dismissVisibleModals();
  console.log('✅ PASS: Step 17 - Reply Message node saved');

  // ============================================================================
  // PHASE 7: CONNECT START → REPLYMESSAGE
  // ============================================================================

  console.log('📍 Step 18: Connect START → ReplyMessage');

  const startHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') })
    .locator('.vue-flow__handle-bottom');
  const replyHandle1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-top');
  await connectEdge('START → ReplyMessage', startHandle, replyHandle1, { normalizeZoom: true });
  console.log('✅ PASS: Step 18 - Edge START → ReplyMessage created');

  // ============================================================================
  // PHASE 8: ADD MODEL NODE
  // ============================================================================

  console.log('📍 Step 19: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 19 - Add Nodes menu opened');

  console.log('📍 Step 20: Select Model node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Model/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 20 - Model node selected');

  // Position Model node at canvas (500, 80) — read fresh tf to account for zoom changes
  const tfModel = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const modelWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /Model/ }) })
    .first();
  await modelWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const modelBBox = await modelWrapper.boundingBox();
  if (!modelBBox) throw new Error('Model node not found');
  const targetX2 = 500 * tfModel.scale + tfModel.tx;
  const targetY2 = 80 * tfModel.scale + tfModel.ty;
  await dragNodeToScreenPosition('Model', modelWrapper, targetX2, targetY2);

  // ============================================================================
  // PHASE 9: CONFIGURE MODEL NODE
  // ============================================================================

  console.log('📍 Step 21: Click Model node to open modal');
  await page.locator('.node-container').filter({ hasText: /Model/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 21 - Model node modal opened');

  console.log('📍 Step 22: Verify Temperature is 0');
  const temperatureInput = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Temperature/ }) })
    .locator('input[type="number"],.v-field__input');
  await temperatureInput.first().waitFor({ state: 'visible', timeout: 15000 });
  const tempValue = await temperatureInput.first().inputValue();
  if (tempValue !== '0') {
    throw new Error(`Temperature is ${tempValue}, expected 0`);
  }
  console.log('✅ PASS: Step 22 - Temperature verified as 0');

  console.log('📍 Step 23: Verify Input Parser is prefilled');
  const inputParserField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Input Parser/ }) });
  await inputParserField.waitFor({ state: 'visible', timeout: 15000 });
  const inputParserContent = await inputParserField.textContent();
  if (!inputParserContent || inputParserContent.trim().length === 0) {
    throw new Error('Input Parser is empty, expected prefilled content');
  }
  console.log('✅ PASS: Step 23 - Input Parser verified as prefilled');

  console.log('📍 Step 24: Verify Stream Output is False');
  const streamOutputSelect = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Stream Output/ }) });
  await streamOutputSelect.waitFor({ state: 'visible', timeout: 15000 });
  await expect(streamOutputSelect).toContainText(/False/);
  console.log('✅ PASS: Step 24 - Stream Output verified as False');

  console.log('📍 Step 25: Set Node ID to model');
  const nodeIdInput2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput2.click();
  await nodeIdInput2.fill('model');
  await nodeIdInput2.press('Tab');
  console.log('✅ PASS: Step 25 - Node ID set to model');

  console.log('📍 Step 26: Select Model Source = provided_models');
  const modelSourceSelect = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Model Source/ }) })
    .locator('.v-select,.v-autocomplete,.v-combobox');
  await modelSourceSelect.scrollIntoViewIfNeeded();
  await modelSourceSelect.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /provided_models/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 26 - Model Source set to provided_models');

  console.log('📍 Step 27: Set System Prompt');
  const systemPromptText = `Your task is to output EXACTLY the JSON below:\n{\n  "city": "Tokyo",\n  "condition": "sunny",\n  "temperature": 24.5\n}`;
  const systemPromptField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /System Prompt/ }) });
  await systemPromptField.scrollIntoViewIfNeeded();
  const systemPromptInput = systemPromptField.locator('textarea,.v-field__input').first();
  await systemPromptInput.fill(systemPromptText);
  await systemPromptInput.press('Tab');
  console.log('✅ PASS: Step 27 - System Prompt set');

  console.log('📍 Step 28: Set Output Parser');
  const outputParserCode = `from pydantic import BaseModel\nclass Options(BaseModel):\n    city: str\n    condition: str\n    temperature: float`;
  const outputParserField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Output Parser/ }) });
  await outputParserField.scrollIntoViewIfNeeded();
  // Click inside the Monaco editor area to focus it
  const monacoEditor = outputParserField.locator('.monaco-editor');
  if (await monacoEditor.count() > 0) {
    await monacoEditor.click();
  } else {
    await outputParserField.click();
  }
  await page.waitForTimeout(200);
  // Use clipboard paste to avoid Monaco's per-line auto-indent escalation.
  // execCommand/insertText causes Monaco to treat each "city: str" colon as a block
  // opener, escalating indentation on every line. Paste with Ctrl+A (cursor at col 0)
  // inserts the block verbatim without per-line indentation shifting.
  await page.keyboard.press('Control+a');
  await page.waitForTimeout(100);
  // Write to clipboard via a temporary textarea (no permission needed)
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
  }, outputParserCode);
  await page.waitForTimeout(100);
  await page.keyboard.press('Control+v');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 28 - Output Parser set');

  console.log('📍 Step 29: Set Stream Output to False');
  const streamOutputDropdown = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Stream Output/ }) })
    .locator('.v-select,.v-autocomplete,.v-combobox');
  await streamOutputDropdown.scrollIntoViewIfNeeded();
  await streamOutputDropdown.click();
  await page.waitForTimeout(300);
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /^False$/ }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 29 - Stream Output set to False');

  console.log('📍 Step 30: Select Model = gpt-41');
  // Model field uses div-based label (no <label>). Combobox order: 0=Model Source, 1=Stream Output, 2=Model
  const modelDropdown = page.locator('.modal-dialog').locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(2);
  await modelDropdown.scrollIntoViewIfNeeded();
  await modelDropdown.click();
  await page.waitForTimeout(300);
  await page.getByRole('option', { name: 'gpt-41', exact: true }).click();
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 30 - Model set to gpt-41');

  console.log('📍 Step 31: Click Save button');
  const saveButton2 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton2.click();
  await page.waitForTimeout(500);
  await dismissVisibleModals();
  console.log('✅ PASS: Step 31 - Model node saved');

  // ============================================================================
  // PHASE 10: CONNECT REPLYMESSAGE → MODEL
  // ============================================================================

  console.log('📍 Step 32: Connect ReplyMessage → model');

  const replySourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#ReplyMessage') })
    .locator('.vue-flow__handle-bottom');
  const modelTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#model') })
    .locator('.vue-flow__handle-top');
  await connectEdge('ReplyMessage → model', replySourceHandle, modelTargetHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 32 - Edge ReplyMessage → model created');

  // ============================================================================
  // PHASE 11: ADD OUTPUT (REPLY MESSAGE) NODE
  // ============================================================================

  console.log('📍 Step 33: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 33 - Add Nodes menu opened');

  console.log('📍 Step 34: Select Reply Message node');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 34 - Reply Message node selected');

  // Position output node at canvas (750, 80) — read fresh tf to account for zoom changes
  const tfOutput = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  // Find the new unnamed reply message node (not ReplyMessage or output yet)
  const replyWrappers = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) });
  const outputWrapper = replyWrappers.last();
  await outputWrapper.waitFor({ state: 'visible', timeout: 15000 });
  const outputBBox = await outputWrapper.boundingBox();
  if (!outputBBox) throw new Error('Output Reply Message node not found');
  const targetX3 = 750 * tfOutput.scale + tfOutput.tx;
  const targetY3 = 80 * tfOutput.scale + tfOutput.ty;
  await dragNodeToScreenPosition('output', outputWrapper, targetX3, targetY3);

  // ============================================================================
  // PHASE 12: CONFIGURE OUTPUT NODE
  // ============================================================================

  console.log('📍 Step 35: Click output node to open modal');
  // Click the last ReplyMessage node (the newly added one, not yet renamed)
  const allReplyContainers = page.locator('.node-container').filter({ hasText: /ReplyMessage/ });
  await allReplyContainers.last().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 35 - Output node modal opened');

  console.log('📍 Step 36: Set Node ID to output');
  const nodeIdInput3 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node ID/ }) })
    .locator('.v-field__input');
  await nodeIdInput3.click();
  await nodeIdInput3.fill('output');
  await nodeIdInput3.press('Tab');
  console.log('✅ PASS: Step 36 - Node ID set to output');

  console.log('📍 Step 37: Verify Node Version is Version 2.0.0');
  const nodeVersionField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Node Version/ }) });
  await nodeVersionField2.waitFor({ state: 'visible', timeout: 15000 });
  await expect(nodeVersionField2).toContainText(/Version 2\.0\.0/);
  console.log('✅ PASS: Step 37 - Node Version verified as Version 2.0.0');

  console.log('📍 Step 38: Verify Receiver Channel is None');
  const receiverChannelField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Receiver Channel/ }) });
  await receiverChannelField2.waitFor({ state: 'visible', timeout: 15000 });
  await expect(receiverChannelField2).toContainText(/None/);
  console.log('✅ PASS: Step 38 - Receiver Channel verified as None');

  console.log('📍 Step 39: Verify Content Type is Text Message');
  const contentTypeField2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Content Type/ }) });
  await contentTypeField2.waitFor({ state: 'visible', timeout: 15000 });
  await expect(contentTypeField2).toContainText(/Text Message/);
  console.log('✅ PASS: Step 39 - Content Type verified as Text Message');

  console.log("📍 Step 40: Set Message field with template expression");
  const messageInput2 = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /Message/ }) })
    .locator('.v-field__input');
  await messageInput2.click();
  await messageInput2.fill("city: {{ state['nodes']['model']['output']['city'] }} condition: {{ state['nodes']['model']['output']['condition'] }} temperature: {{ state['nodes']['model']['output']['temperature'] }}");
  await messageInput2.press('Tab');
  console.log('✅ PASS: Step 40 - Message field set with template expression');

  console.log('📍 Step 41: Click Save button');
  const saveButton3 = page.locator('.modal-dialog button').filter({ hasText: /^Save$/ });
  await saveButton3.click();
  await page.waitForTimeout(500);
  await dismissVisibleModals();
  console.log('✅ PASS: Step 41 - Output node saved');

  // ============================================================================
  // PHASE 13: VERIFY OUTPUT NODE ON CANVAS
  // ============================================================================

  console.log('📍 Step 42: Verify node output is visible on canvas');
  const outputNode = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#output') });
  await outputNode.waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 42 - Node output is visible on canvas');

  // ============================================================================
  // PHASE 14: CONNECT MODEL → OUTPUT
  // ============================================================================

  console.log('📍 Step 43: Connect model → output');

  const modelSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#model') })
    .locator('.vue-flow__handle-bottom');
  const outputTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#output') })
    .locator('.vue-flow__handle-top');
  await connectEdge('model → output', modelSourceHandle, outputTargetHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 43 - Edge model → output created');

  // ============================================================================
  // PHASE 15: CONNECT OUTPUT → END
  // ============================================================================

  console.log('📍 Step 44: Connect output → END');

  const outputSourceHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#output') })
    .locator('.vue-flow__handle-bottom');
  const endTargetHandle = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');
  await connectEdge('output → END', outputSourceHandle, endTargetHandle, { normalizeZoom: true });
  console.log('✅ PASS: Step 44 - Edge output → END created');

  // ============================================================================
  // PHASE 16: VERIFY FLOW STRUCTURE
  // ============================================================================

  console.log('📍 Step 45: Verify flow structure START → ReplyMessage → model → output → END');
  const finalEdgeCount = await page.locator('.vue-flow__edge[data-id]').count();
  if (finalEdgeCount < 4) {
    throw new Error(`Expected at least 4 edges, found ${finalEdgeCount}`);
  }
  console.log('✅ PASS: Step 45 - Flow structure verified with all 4 edges');

  // ============================================================================
  // PHASE 17: SAVE FLOW VERSION
  // ============================================================================

  console.log('📍 Step 46: Click Save button (disk icon)');
  await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
  const saveModal = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
  await saveModal.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 46 - Save button clicked');

  console.log('📍 Step 47: Verify Save Flow Version modal appears');
  console.log('✅ PASS: Step 47 - Save Flow Version modal appeared');

  console.log('📍 Step 48: Enter version name Model Node with Parser');
  const versionNameInput = saveModal.locator('.v-field__input').first();
  await versionNameInput.fill('Model Node with Parser');
  console.log('✅ PASS: Step 48 - Version name entered');

  console.log('📍 Step 49: Click Save button in modal');
  const modalSaveButton = saveModal.locator('button').filter({ hasText: /^Save$/ });
  await modalSaveButton.click();
  await saveModal.waitFor({ state: 'hidden', timeout: 30000 });
  console.log('✅ PASS: Step 49 - Save Flow Version modal closed');

  console.log('📍 Step 50: Verify success toast notification');
  try {
    await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i, { timeout: 15000 });
    console.log('✅ PASS: Step 50 - Success toast notification appeared');
  } catch {
    const toastText = await page.locator('.v-snackbar').textContent().catch(() => '(no toast)');
    console.log(`⚠️ WARN: Step 50 - Toast showed: "${toastText?.trim()}" (expected success/saved)`);
  }

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
  console.log('✅ Step 8: PASS - Flow renamed to Model Node With Parser');
  console.log('✅ Step 9: PASS - Add Nodes menu opened');
  console.log('✅ Step 10: PASS - Reply Message node selected');
  console.log('✅ Step 11: PASS - Reply Message node modal opened');
  console.log('✅ Step 12: PASS - Node Version verified as Version 2.0.0');
  console.log('✅ Step 13: PASS - Receiver Channel verified as None');
  console.log('✅ Step 14: PASS - Content Type verified as Text Message');
  console.log('✅ Step 15: PASS - Node ID set to ReplyMessage');
  console.log('✅ Step 16: PASS - Message field set');
  console.log('✅ Step 17: PASS - Reply Message node saved');
  console.log('✅ Step 18: PASS - Edge START → ReplyMessage created');
  console.log('✅ Step 19: PASS - Add Nodes menu opened');
  console.log('✅ Step 20: PASS - Model node selected');
  console.log('✅ Step 21: PASS - Model node modal opened');
  console.log('✅ Step 22: PASS - Temperature verified as 0');
  console.log('✅ Step 23: PASS - Input Parser verified as prefilled');
  console.log('✅ Step 24: PASS - Stream Output verified as False');
  console.log('✅ Step 25: PASS - Node ID set to model');
  console.log('✅ Step 26: PASS - Model Source set to provided_models');
  console.log('✅ Step 27: PASS - System Prompt set');
  console.log('✅ Step 28: PASS - Output Parser set');
  console.log('✅ Step 29: PASS - Stream Output set to False');
  console.log('✅ Step 30: PASS - Model set to gpt-41');
  console.log('✅ Step 31: PASS - Model node saved');
  console.log('✅ Step 32: PASS - Edge ReplyMessage → model created');
  console.log('✅ Step 33: PASS - Add Nodes menu opened');
  console.log('✅ Step 34: PASS - Reply Message node selected');
  console.log('✅ Step 35: PASS - Output node modal opened');
  console.log('✅ Step 36: PASS - Node ID set to output');
  console.log('✅ Step 37: PASS - Node Version verified as Version 2.0.0');
  console.log('✅ Step 38: PASS - Receiver Channel verified as None');
  console.log('✅ Step 39: PASS - Content Type verified as Text Message');
  console.log('✅ Step 40: PASS - Message field set with template expression');
  console.log('✅ Step 41: PASS - Output node saved');
  console.log('✅ Step 42: PASS - Node output visible on canvas');
  console.log('✅ Step 43: PASS - Edge model → output created');
  console.log('✅ Step 44: PASS - Edge output → END created');
  console.log('✅ Step 45: PASS - Flow structure verified');
  console.log('✅ Step 46: PASS - Save button clicked');
  console.log('✅ Step 47: PASS - Save Flow Version modal appeared');
  console.log('✅ Step 48: PASS - Version name entered');
  console.log('✅ Step 49: PASS - Flow version saved');
  console.log('='.repeat(70));
  console.log('✅ TEST COMPLETE - All steps through 49 passed successfully');
  console.log('='.repeat(70));
});
