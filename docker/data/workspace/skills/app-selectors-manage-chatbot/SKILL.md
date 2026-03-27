---
name: app-selectors-manage-chatbot
description: DOM selectors and component map for the Manage Chatbot page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Manage Chatbot — Component Map

> Generated: 2026-03-27T09:37:00.639Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Manage Chatbot
**URL:** `/manage-chatbot`

**Headings:**
- `h3` — "Chatbot"

**Dropdowns / Selects (1):**
- **Select** (select)
  - Selector: `.v-select:nth(0)`
  - Open: `page.locator('.v-select:nth(0)').click()`
  - Options: `No data available`
  - Pick: `page.locator('.v-list-item:has-text("OPTION")').click()`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')`
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (5):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | ChatbotSelectSelectDashboardFlow DesignerFlow TesterKnowledg |
| `` | `label` | `` | Select |
| `` | `input` | `` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `` | `div` | `` |  |

---

