import { expect, type Locator, type Page } from '@playwright/test';

export const createFlowDesignerCanvasHelpers = (page: Page) => {
  const dismissVisibleModals = async () => {
    const visibleModals = page.locator('.modal-dialog:visible');

    for (let attempt = 1; attempt <= 3; attempt++) {
      const visibleCount = await visibleModals.count();
      if (visibleCount === 0) return;

      const activeModal = visibleModals.last();
      await page.keyboard.press('Escape').catch(() => {});
      await page.waitForTimeout(300);

      if (await activeModal.isVisible().catch(() => false)) {
        await activeModal.locator('button').first().click({ timeout: 2000 }).catch(() => {});
        await page.waitForTimeout(300);
      }

      if (await activeModal.isVisible().catch(() => false)) {
        await activeModal.getByRole('button', { name: /^Cancel$/ }).click({ timeout: 2000 }).catch(() => {});
        await page.waitForTimeout(300);
      }
    }

    const remainingVisible = await visibleModals.count();
    if (remainingVisible > 0) {
      throw new Error('Modal dialog is still visible and blocking canvas interactions');
    }
  };

  const connectEdge = async (
    label: string,
    sourceHandle: Locator,
    targetHandle: Locator,
    options?: { normalizeZoom?: boolean },
  ) => {
    await dismissVisibleModals();

    const blockingOverlay = page.locator('.v-overlay--active').filter({
      has: page.locator('.monaco-editor, .view-lines'),
    });
    if (await blockingOverlay.first().isVisible().catch(() => false)) {
      await blockingOverlay.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
    }

    const flowCanvas = page.locator('.vue-flow');
    await flowCanvas.waitFor({ state: 'visible', timeout: 60000 });
    const canvasBox = await flowCanvas.boundingBox();
    if (!canvasBox) throw new Error('Flow canvas bounding box not found');

    const canvasCenterX = canvasBox.x + canvasBox.width / 2;
    const canvasCenterY = canvasBox.y + canvasBox.height / 2;
    await page.mouse.move(canvasCenterX, canvasCenterY);
    await page.waitForTimeout(200);

    if (options?.normalizeZoom) {
      await page.keyboard.down('Control');
      for (let i = 0; i < 16; i++) {
        await page.mouse.move(canvasCenterX, canvasCenterY);
        await page.mouse.wheel(0, -120);
      }
      for (let i = 0; i < 10; i++) {
        await page.mouse.move(canvasCenterX, canvasCenterY);
        await page.mouse.wheel(0, 120);
      }
      await page.keyboard.up('Control');
      await page.waitForTimeout(400);
    }

    const edgeLocator = page.locator('.vue-flow__edge[data-id]');
    const edgesBefore = await edgeLocator.count();

    await sourceHandle.waitFor({ state: 'visible', timeout: 30000 });
    await targetHandle.waitFor({ state: 'visible', timeout: 30000 });

    let lastEdges = edgesBefore;

    for (let attempt = 1; attempt <= 3; attempt++) {
      await sourceHandle.hover({ timeout: 2000 }).catch(() => {});
      await targetHandle.hover({ timeout: 2000 }).catch(() => {});

      const sourceBox = await sourceHandle.boundingBox();
      const targetBox = await targetHandle.boundingBox();

      if (!sourceBox || !targetBox) {
        await page.waitForTimeout(300);
        continue;
      }

      const sourceX = sourceBox.x + sourceBox.width / 2;
      const sourceY = sourceBox.y + sourceBox.height / 2;
      const targetX = targetBox.x + targetBox.width / 2;
      const targetY = targetBox.y + targetBox.height / 2;

      await page.mouse.move(sourceX, sourceY);
      await page.waitForTimeout(200);
      await page.mouse.down();
      await page.mouse.move(targetX, targetY, { steps: 100 });
      await page.waitForTimeout(350);
      await page.mouse.move(targetX, targetY, { steps: 3 });
      await page.waitForTimeout(250);
      await page.mouse.up();
      await page.waitForTimeout(300);

      try {
        await expect.poll(async () => edgeLocator.count(), { timeout: 15000 }).toBeGreaterThan(edgesBefore);
        await page.waitForTimeout(700);
        const stableEdgesAfter = await edgeLocator.count();
        if (stableEdgesAfter > edgesBefore) {
          return;
        }
        lastEdges = stableEdgesAfter;
      } catch {
        lastEdges = await edgeLocator.count();
        await page.waitForTimeout(800);
      }
    }

    throw new Error(`Edge ${label} NOT created — before: ${edgesBefore}, after: ${lastEdges}`);
  };

  const dragNodeToScreenPosition = async (
    label: string,
    nodeWrapper: Locator,
    targetCenterX: number,
    targetCenterY: number,
  ) => {
    await dismissVisibleModals();

    const flowCanvas = page.locator('.vue-flow');
    await flowCanvas.waitFor({ state: 'visible', timeout: 60000 });

    for (let attempt = 1; attempt <= 3; attempt++) {
      await nodeWrapper.waitFor({ state: 'visible', timeout: 30000 });
      const nodeBox = await nodeWrapper.boundingBox();
      if (!nodeBox) throw new Error(`${label} node bounding box not found`);

      await page.mouse.move(nodeBox.x + nodeBox.width / 2, nodeBox.y + nodeBox.height / 2);
      await page.mouse.down();
      await page.mouse.move(targetCenterX, targetCenterY, { steps: 60 });
      await page.mouse.up();
      await page.waitForTimeout(500);

      if (!(await nodeWrapper.first().isVisible().catch(() => false))) {
        await page.waitForTimeout(300);
        continue;
      }

      const updatedBox = await nodeWrapper.boundingBox();
      if (!updatedBox) throw new Error(`${label} node bounding box not found after drag`);

      const deltaX = Math.abs(updatedBox.x + updatedBox.width / 2 - targetCenterX);
      const deltaY = Math.abs(updatedBox.y + updatedBox.height / 2 - targetCenterY);
      if (deltaX <= 80 && deltaY <= 80) return;
    }

    throw new Error(`${label} node did not land near target position`);
  };

  const ensureNodeIdOnCanvas = async (
    desiredId: string,
    fallbackId: string,
  ) => {
    const nodeWrapper = page.locator('.vue-flow__node')
      .filter({ has: page.locator(`.node-container#${desiredId}, .node-container#${fallbackId}`) });

    if (await page.locator(`.node-container#${desiredId}`).first().isVisible().catch(() => false)) {
      return nodeWrapper;
    }

    const fallbackNodeWrapper = page.locator('.vue-flow__node')
      .filter({ has: page.locator(`.node-container#${fallbackId}`) });
    await fallbackNodeWrapper.waitFor({ state: 'visible', timeout: 15000 });
    await page.locator(`.node-container#${fallbackId}`).click();

    const retryModal = page.locator('.modal-dialog.v-overlay--active').last();
    await retryModal.waitFor({ state: 'visible', timeout: 30000 });
    const retryNodeIdField = retryModal.locator('.field-container')
      .filter({ has: page.locator('label', { hasText: /^Node ID/ }) })
      .locator('.v-field__input')
      .first();
    const retryNodeIdValue = await retryNodeIdField.inputValue();
    if (retryNodeIdValue !== desiredId) {
      await retryNodeIdField.click();
      await retryNodeIdField.fill(desiredId);
      await retryNodeIdField.press('Tab');
    }

    const retrySaveButton = retryModal.getByRole('button', { name: /^Save$/ });
    const retryCloseButton = retryModal.locator('button').first();
    if (await retrySaveButton.isEnabled().catch(() => false)) {
      await retrySaveButton.click();
      await page.waitForTimeout(500);
    }
    await page.keyboard.press('Escape').catch(() => {});
    await retryCloseButton.click({ timeout: 2000 }).catch(() => {});
    await dismissVisibleModals();
    await nodeWrapper.waitFor({ state: 'visible', timeout: 15000 });
    return nodeWrapper;
  };

  return {
    dismissVisibleModals,
    connectEdge,
    dragNodeToScreenPosition,
    ensureNodeIdOnCanvas,
  };
};
