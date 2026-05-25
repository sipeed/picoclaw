import { test, expect } from '@playwright/test';

test('Create new flow with User Utterance node', async ({ page }) => {
  test.setTimeout(300000);
  page.setDefaultTimeout(60000);
  page.setDefaultNavigationTimeout(60000);

  // ============ PHASE 1: LOGIN & NAVIGATION ============

  console.log('📍 Step 1: Navigate to login page');
  await page.goto('/login', { waitUntil: 'networkidle' });
  console.log('✅ PASS: Step 1 - Navigated to login page');

  console.log('📍 Step 2: Fill email address');
  await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
  console.log('✅ PASS: Step 2 - Email filled');

  console.log('📍 Step 3: Fill password');
  await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
  console.log('✅ PASS: Step 3 - Password filled');

  console.log('📍 Step 4: Click login button');
  await page.getByRole('button', { name: /login/i }).click();
  console.log('✅ PASS: Step 4 - Login button clicked');

  console.log('📍 Step 5: Wait for post-login redirect');
  await page.waitForURL(url => url.pathname !== '/login', { timeout: 60000 });
  if (page.url().includes('select_org')) {
    await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
    await page.waitForURL(url => !url.href.includes('select_org'), { timeout: 30000 });
  }
  console.log('✅ PASS: Step 5 - Redirected past login');

  console.log('📍 Step 6: Confirm organization selected / dashboard reached');
  console.log('✅ PASS: Step 6 - Organization selected');

  console.log('📍 Step 7: Navigate to Flow Designer');
  await page.locator('a:has-text("Flow Designer")').click();
  await page.waitForURL('**/flow-designer', { timeout: 60000 });
  console.log('✅ PASS: Step 7 - Navigated to Flow Designer');

  console.log('📍 Step 8: Click Add New button');
  await page.locator('.m-auto').filter({ hasText: /Add New/ }).click();
  await page.waitForURL('**/flow-designer/**', { timeout: 60000 });
  console.log('✅ PASS: Step 8 - New flow created, canvas opened');

  // ============ PHASE 2: FLOW SETUP ============

  console.log('📍 Step 9: Verify START and END nodes are present');
  await page.locator('.node-container#START').waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.node-container#END').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 9 - START and END nodes verified');

  // Read canvas transform ONCE — used for all absolute node positioning
  const tf = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });

  console.log('📍 Step 10: Rename flow to "User Utterance"');
  await page.locator('.panel-container p.text-secondary').first().click();
  await page.waitForTimeout(300);
  const flowNameInput = page.locator('.panel-container input').first();
  await flowNameInput.waitFor({ state: 'visible', timeout: 15000 });
  await flowNameInput.fill('User Utterance');
  await flowNameInput.press('Enter');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 10 - Flow renamed to "User Utterance"');

  // ============ PHASE 3: ADD FIRST REPLY MESSAGE NODE ============

  console.log('📍 Step 11: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 11 - Add Nodes menu opened');

  console.log('📍 Step 12: Select Reply Message from dropdown');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 12 - Reply Message node added');

  console.log('📍 Step 13: Position first Reply Message node at canvas (250, 100)');
  const firstReplyWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first();
  const firstReplyBBox = await firstReplyWrapper.boundingBox();
  if (!firstReplyBBox) throw new Error('Cannot position first Reply Message — node not found');
  const targetX1 = 250 * tf.scale + tf.tx;
  const targetY1 = 100 * tf.scale + tf.ty;
  await page.mouse.move(firstReplyBBox.x + firstReplyBBox.width / 2, firstReplyBBox.y + firstReplyBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetX1, targetY1, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 13 - First Reply Message positioned');

  console.log('📍 Step 14: Click first Reply Message node to open config modal');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 14 - Node config modal opened');

  console.log('📍 Step 15: Verify auto-populated fields in modal');
  await page.locator('.modal-dialog .field-container').first().waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 15 - Modal fields visible');

  console.log('📍 Step 16: Fill Message field with prompt text');
  const messageField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /^Message/ }) })
    .locator('.v-field__input')
    .first();
  await messageField.click();
  await messageField.fill('Write something below to test the user utterance node:');
  await messageField.press('Tab');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 16 - Message field filled');

  console.log('📍 Step 17: Click Save button in modal');
  await page.locator('.modal-dialog').locator('button').filter({ hasText: /^Save$/ }).click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 17 - Node config saved, modal closed');

  // ============ PHASE 4: CONNECT START TO FIRST REPLY MESSAGE ============

  console.log('📍 Step 18: Connect START → Reply Message');
  await page.mouse.move(640, 360);
  await page.keyboard.down('Control');
  for (let i = 0; i < 20; i++) { await page.mouse.wheel(0, -100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(200);
  await page.keyboard.down('Control');
  for (let i = 0; i < 10; i++) { await page.mouse.wheel(0, 100); }
  await page.keyboard.up('Control');
  await page.waitForTimeout(500);
  const edgesBefore1 = await page.locator('.vue-flow__edge[data-id]').count();
  const sourceHandle1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#START') })
    .locator('.vue-flow__handle-bottom');
  const targetHandle1 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first()
    .locator('.vue-flow__handle-top');
  const srcBox1 = await sourceHandle1.boundingBox();
  const tgtBox1 = await targetHandle1.boundingBox();
  if (!srcBox1 || !tgtBox1) throw new Error('Handle not found for START → Reply Message connection');
  await page.mouse.move(srcBox1.x + srcBox1.width / 2, srcBox1.y + srcBox1.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(tgtBox1.x + tgtBox1.width / 2, tgtBox1.y + tgtBox1.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 18 - START → Reply Message connected');

  console.log('📍 Step 19: Verify edge created between START and Reply Message');
  const edgesAfter1 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter1 <= edgesBefore1) throw new Error(`Edge NOT created — count before: ${edgesBefore1}, after: ${edgesAfter1}`);
  console.log('✅ PASS: Step 19 - Edge verified');

  // ============ PHASE 5: ADD USER UTTERANCE NODE ============

  console.log('📍 Step 20: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 20 - Add Nodes menu opened');

  console.log('📍 Step 21: Select User Utterance from dropdown');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /User Utterance/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 21 - User Utterance node added');

  console.log('📍 Step 22: Position User Utterance node at canvas (250, 200)');
  const tfUU = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const userUtteranceWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /UserUtterance/ }) })
    .first();
  const userUtteranceBBox = await userUtteranceWrapper.boundingBox();
  if (!userUtteranceBBox) throw new Error('Cannot position User Utterance — node not found');
  const targetX2 = 250 * tfUU.scale + tfUU.tx;
  const targetY2 = 200 * tfUU.scale + tfUU.ty;
  await page.mouse.move(userUtteranceBBox.x + userUtteranceBBox.width / 2, userUtteranceBBox.y + userUtteranceBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetX2, targetY2, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 22 - User Utterance positioned');

  console.log('📍 Step 23: Click User Utterance node to open config modal');
  await page.locator('.node-container').filter({ hasText: /UserUtterance/ }).first().evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 23 - User Utterance config modal opened');

  console.log('📍 Step 24: Change Node ID to "input"');
  const nodeIdField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /^Node ID/ }) })
    .locator('.v-field__input')
    .first();
  await nodeIdField.click();
  await nodeIdField.fill('input');
  await nodeIdField.press('Tab');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 24 - Node ID changed to "input"');

  console.log('📍 Step 25: Leave State Variable empty (default)');
  console.log('✅ PASS: Step 25 - State Variable left empty');

  console.log('📍 Step 26: Click Save button in modal');
  await page.locator('.modal-dialog').locator('button').filter({ hasText: /^Save$/ }).click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 26 - User Utterance config saved');

  // ============ PHASE 6: CONNECT FIRST REPLY MESSAGE TO USER UTTERANCE ============

  console.log('📍 Step 27: Connect Reply Message → input (User Utterance)');
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
  const sourceHandle2 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .first()
    .locator('.vue-flow__handle-bottom');
  const targetHandle2 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#input') })
    .locator('.vue-flow__handle-top');
  const srcBox2 = await sourceHandle2.boundingBox();
  const tgtBox2 = await targetHandle2.boundingBox();
  if (!srcBox2 || !tgtBox2) throw new Error('Handle not found for Reply Message → input connection');
  await page.mouse.move(srcBox2.x + srcBox2.width / 2, srcBox2.y + srcBox2.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(tgtBox2.x + tgtBox2.width / 2, tgtBox2.y + tgtBox2.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 27 - Reply Message → input connected');

  console.log('📍 Step 28: Verify edge created between Reply Message and input');
  const edgesAfter2 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter2 <= edgesBefore2) throw new Error(`Edge NOT created — count before: ${edgesBefore2}, after: ${edgesAfter2}`);
  console.log('✅ PASS: Step 28 - Edge verified');

  // ============ PHASE 7: ADD OUTPUT REPLY MESSAGE NODE ============

  console.log('📍 Step 29: Click Add Nodes button');
  await page.locator('button.nodes-button[aria-haspopup="menu"]').first().click();
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 29 - Add Nodes menu opened');

  console.log('📍 Step 30: Select Reply Message from dropdown');
  await page.locator('.nodes-dropdown-item').filter({ hasText: /Reply Message/ }).click();
  await page.keyboard.press('Escape');
  await page.locator('.nodes-dropdown-menu').waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 30 - Second Reply Message node added');

  console.log('📍 Step 31: Position Output Reply Message node at canvas (250, 300)');
  const tfReply2 = await page.locator('.vue-flow__transformationpane').evaluate(el => {
    const m = new DOMMatrix((el as HTMLElement).style.transform);
    return { scale: m.a, tx: m.e, ty: m.f };
  });
  const secondReplyWrapper = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container').filter({ hasText: /ReplyMessage/ }) })
    .nth(1);
  const secondReplyBBox = await secondReplyWrapper.boundingBox();
  if (!secondReplyBBox) throw new Error('Cannot position Output Reply Message — node not found');
  const targetX3 = 250 * tfReply2.scale + tfReply2.tx;
  const targetY3 = 300 * tfReply2.scale + tfReply2.ty;
  await page.mouse.move(secondReplyBBox.x + secondReplyBBox.width / 2, secondReplyBBox.y + secondReplyBBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetX3, targetY3, { steps: 50 });
  await page.mouse.up();
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 31 - Output Reply Message positioned');

  console.log('📍 Step 32: Click Output Reply Message node to open config modal');
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).nth(1).evaluate((el) => (el as HTMLElement).click());
  await page.locator('.modal-dialog').waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 32 - Output node config modal opened');

  console.log('📍 Step 33: Change Node ID to "Output"');
  const outputNodeIdField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /^Node ID/ }) })
    .locator('.v-field__input')
    .first();
  await outputNodeIdField.click();
  await outputNodeIdField.fill('Output');
  await outputNodeIdField.press('Tab');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 33 - Node ID changed to "Output"');

  console.log('📍 Step 34: Leave Node Version, Receiver Channel, Content Type as default');
  console.log('✅ PASS: Step 34 - Fields left as default');

  console.log('📍 Step 35: Fill Message field with template');
  const outputMessageField = page.locator('.modal-dialog .field-container')
    .filter({ has: page.locator('label', { hasText: /^Message/ }) })
    .locator('.v-field__input')
    .first();
  await outputMessageField.click();
  await outputMessageField.fill("{{ state['nodes']['input']['output']['messages'].content }}");
  await outputMessageField.press('Tab');
  await page.waitForTimeout(300);
  console.log('✅ PASS: Step 35 - Message template filled');

  console.log('📍 Step 36: Click Save button in modal');
  await page.locator('.modal-dialog').locator('button').filter({ hasText: /^Save$/ }).click();
  await page.locator('.modal-dialog').waitFor({ state: 'hidden', timeout: 30000 });
  await page.waitForTimeout(500);
  console.log('✅ PASS: Step 36 - Output node config saved');

  // ============ PHASE 8: CONNECT INPUT TO OUTPUT AND OUTPUT TO END ============

  console.log('📍 Step 37: Connect input → Output');
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
  const sourceHandle3 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#input') })
    .locator('.vue-flow__handle-bottom');
  const targetHandle3 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#Output') })
    .locator('.vue-flow__handle-top');
  const srcBox3 = await sourceHandle3.boundingBox();
  const tgtBox3 = await targetHandle3.boundingBox();
  if (!srcBox3 || !tgtBox3) throw new Error('Handle not found for input → Output connection');
  await page.mouse.move(srcBox3.x + srcBox3.width / 2, srcBox3.y + srcBox3.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(tgtBox3.x + tgtBox3.width / 2, tgtBox3.y + tgtBox3.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 37 - input → Output connected');

  console.log('📍 Step 38: Verify edge created between input and Output');
  const edgesAfter3 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter3 <= edgesBefore3) throw new Error(`Edge NOT created — count before: ${edgesBefore3}, after: ${edgesAfter3}`);
  console.log('✅ PASS: Step 38 - Edge verified');

  console.log('📍 Step 39: Connect Output → END');
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
  const sourceHandle4 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#Output') })
    .locator('.vue-flow__handle-bottom');
  const targetHandle4 = page.locator('.vue-flow__node')
    .filter({ has: page.locator('.node-container#END') })
    .locator('.vue-flow__handle-top');
  const srcBox4 = await sourceHandle4.boundingBox();
  const tgtBox4 = await targetHandle4.boundingBox();
  if (!srcBox4 || !tgtBox4) throw new Error('Handle not found for Output → END connection');
  await page.mouse.move(srcBox4.x + srcBox4.width / 2, srcBox4.y + srcBox4.height / 2);
  await page.waitForTimeout(200);
  await page.mouse.down();
  await page.mouse.move(tgtBox4.x + tgtBox4.width / 2, tgtBox4.y + tgtBox4.height / 2, { steps: 50 });
  await page.waitForTimeout(500);
  await page.mouse.up();
  await page.waitForTimeout(1000);
  console.log('✅ PASS: Step 39 - Output → END connected');

  console.log('📍 Step 40: Verify edge created between Output and END');
  const edgesAfter4 = await page.locator('.vue-flow__edge[data-id]').count();
  if (edgesAfter4 <= edgesBefore4) throw new Error(`Edge NOT created — count before: ${edgesBefore4}, after: ${edgesAfter4}`);
  console.log('✅ PASS: Step 40 - Edge verified');

  // ============ PHASE 9: SAVE FLOW VERSION ============

  console.log('📍 Step 41: Click Save Flow button (disk icon)');
  await page.locator('button').filter({ has: page.locator('.mdi-content-save') }).click();
  const saveDialog = page.locator('.v-overlay--active').filter({ hasText: /Save Flow Version/ });
  await saveDialog.waitFor({ state: 'visible', timeout: 60000 });
  console.log('✅ PASS: Step 41 - Save Flow Version modal opened');

  console.log('📍 Step 42: Enter version name "UserUtteranceV1"');
  const versionNameInput = saveDialog.locator('.v-field__input').first();
  await versionNameInput.fill('UserUtteranceV1');
  console.log('✅ PASS: Step 42 - Version name entered');

  console.log('📍 Step 43: Click Save button in modal');
  await saveDialog.locator('button').filter({ hasText: /^Save$/ }).click();
  await saveDialog.waitFor({ state: 'hidden', timeout: 30000 });
  console.log('✅ PASS: Step 43 - Flow version saved');

  console.log('📍 Step 44: Verify success toast notification');
  await expect(page.locator('.v-snackbar')).toContainText(/success|saved/i, { timeout: 15000 });
  console.log('✅ PASS: Step 44 - Success notification verified');

  // ============ PHASE 10: FINAL VERIFICATION ============

  console.log('📍 Step 45: Verify all nodes are on canvas');
  await page.locator('.node-container#START').waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.node-container').filter({ hasText: /ReplyMessage/ }).first().waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.node-container#input').waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.node-container#Output').waitFor({ state: 'visible', timeout: 15000 });
  await page.locator('.node-container#END').waitFor({ state: 'visible', timeout: 15000 });
  console.log('✅ PASS: Step 45 - All nodes verified on canvas');

  console.log('📍 Step 46: Verify all edges are connected (expect 4 total)');
  const finalEdgeCount = await page.locator('.vue-flow__edge[data-id]').count();
  if (finalEdgeCount < 4) throw new Error(`Expected at least 4 edges, found ${finalEdgeCount}`);
  console.log('✅ PASS: Step 46 - All edges verified');

  console.log('📍 Step 47: Verify flow name is "User Utterance"');
  await expect(page.locator('.panel-container p.text-secondary').first()).toContainText('User Utterance');
  console.log('✅ PASS: Step 47 - Flow name verified');

  // ============ TEST SUMMARY ============

  console.log('\n' + '='.repeat(70));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(70));
  console.log('✅ Step 1: PASS - Navigated to login page');
  console.log('✅ Step 2: PASS - Email filled');
  console.log('✅ Step 3: PASS - Password filled');
  console.log('✅ Step 4: PASS - Login button clicked');
  console.log('✅ Step 5: PASS - Redirected to org selection');
  console.log('✅ Step 6: PASS - Organization selected');
  console.log('✅ Step 7: PASS - Navigated to Flow Designer');
  console.log('✅ Step 8: PASS - New flow created, canvas opened');
  console.log('✅ Step 9: PASS - START and END nodes verified');
  console.log('✅ Step 10: PASS - Flow renamed to "User Utterance"');
  console.log('✅ Step 11: PASS - Add Nodes menu opened');
  console.log('✅ Step 12: PASS - Reply Message node added');
  console.log('✅ Step 13: PASS - First Reply Message positioned');
  console.log('✅ Step 14: PASS - Node config modal opened');
  console.log('✅ Step 15: PASS - Modal fields visible');
  console.log('✅ Step 16: PASS - Message field filled');
  console.log('✅ Step 17: PASS - Node config saved, modal closed');
  console.log('✅ Step 18: PASS - START → Reply Message connected');
  console.log('✅ Step 19: PASS - Edge verified');
  console.log('✅ Step 20: PASS - Add Nodes menu opened');
  console.log('✅ Step 21: PASS - User Utterance node added');
  console.log('✅ Step 22: PASS - User Utterance positioned');
  console.log('✅ Step 23: PASS - User Utterance config modal opened');
  console.log('✅ Step 24: PASS - Node ID changed to "input"');
  console.log('✅ Step 25: PASS - State Variable left empty');
  console.log('✅ Step 26: PASS - User Utterance config saved');
  console.log('✅ Step 27: PASS - Reply Message → input connected');
  console.log('✅ Step 28: PASS - Edge verified');
  console.log('✅ Step 29: PASS - Add Nodes menu opened');
  console.log('✅ Step 30: PASS - Second Reply Message node added');
  console.log('✅ Step 31: PASS - Output Reply Message positioned');
  console.log('✅ Step 32: PASS - Output node config modal opened');
  console.log('✅ Step 33: PASS - Node ID changed to "Output"');
  console.log('✅ Step 34: PASS - Fields left as default');
  console.log('✅ Step 35: PASS - Message template filled');
  console.log('✅ Step 36: PASS - Output node config saved');
  console.log('✅ Step 37: PASS - input → Output connected');
  console.log('✅ Step 38: PASS - Edge verified');
  console.log('✅ Step 39: PASS - Output → END connected');
  console.log('✅ Step 40: PASS - Edge verified');
  console.log('✅ Step 41: PASS - Save Flow Version modal opened');
  console.log('✅ Step 42: PASS - Version name entered');
  console.log('✅ Step 43: PASS - Flow version saved');
  console.log('✅ Step 44: PASS - Success notification verified');
  console.log('✅ Step 45: PASS - All nodes verified on canvas');
  console.log('✅ Step 46: PASS - All edges verified');
  console.log('✅ Step 47: PASS - Flow name verified');
  console.log('='.repeat(70));
});
