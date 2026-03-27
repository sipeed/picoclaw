---
name: app-selectors
description: Global login and organization selection flow for dashboard.int3nt.info. Use this when writing Playwright tests that require authentication.
---

# Global Login & Org Selection

> Generated: 2026-03-27T09:36:49.125Z

## Test Credentials

| Field | Value |
|-------|-------|
| Email | `heidi@intnt.ai` |
| Password | `testing2026!` |
| Organization | `Testing2026!` |

## Login Flow (copy-paste ready)

```typescript
// Step 1 — Navigate
await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

// Step 2 — Fill credentials (use EXACTLY these selectors)
await page.locator('.v-text-field').nth(0).locator('input').fill('heidi@intnt.ai');
await page.locator('.v-text-field').nth(1).locator('input').fill('testing2026!');
await page.getByRole('button', { name: /login/i }).click();

// Step 3 — Wait for redirect to org selection
await page.waitForURL(/\?select_org/, { timeout: 20000 });

// Step 4 — Wait for loader to disappear, then select organization
const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
if (await loader.first().isVisible().catch(() => false)) {
  await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
}
await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```

## Key Selectors

| Element | Selector |
|---------|----------|
| Email input | `.v-text-field:nth(0) input` |
| Password input | `.v-text-field:nth(1) input` |
| Login button | `getByRole('button', { name: /login/i })` |
| Org card | `.organization-card` |
| Org name in card | `.organization-name` |
| Org dropdown trigger | `.org-dropdown-trigger` |
| Org dropdown item | `.org-dropdown-item` |

**Known orgs:** "Testing2026!", "Testing"
