# Single page:

docker exec $(docker compose --env-file .env --profile gateway ps -q picoclaw-gateway) \
 node /home/picoclaw/.picoclaw/workspace/inspect-scripts/inspect-knowledge-base.js

# All pages (shared browser, efficient):

docker exec $(docker compose --env-file .env --profile gateway ps -q picoclaw-gateway) \
 node /home/picoclaw/.picoclaw/workspace/inspect-scripts/run-all.js

# Creating Playwright Script Prompt:

docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m "$(cat <<'EOF'
Before generating anything, read and follow:

- /skills/playwright/SKILL.md
- /skills/app-selectors/SKILL.md
- /skills/app-selectors-knowledge-base/SKILL.md

IMPORTANT RULES:

1. You MUST only use selectors from the SKILL.md files above. Copy them exactly.
2. NEVER guess or invent Vuetify selectors like .v-dialog, .v-dialog--active, .v-overlay, .v-card, etc.
3. For login, use the EXACT credentials and selectors from /skills/app-selectors/SKILL.md. Do NOT use env vars or placeholder credentials.
4. For modals/dialogs, use the actual custom CSS classes from the Discovered Modals or Custom Elements section. NEVER use generic Vuetify wrappers.
5. TypeScript-safe Playwright only: NEVER use .filter({ hasAttribute: ... }) because it is invalid in Playwright.
6. Logging rule: print "✅ PASS" only AFTER the step assertion succeeds. Never print optimistic PASS logs.
7. Loading states in modals: after a modal/overlay opens, ALWAYS wait for any "Loading..." text to disappear before asserting content. Use: await page.locator('.v-overlay--active').locator('text=/loading/i').waitFor({ state: 'hidden', timeout: 10000 }).catch(() => {});
8. NEVER guess component types (e.g. .v-radio-group, .v-tabs). Always check the SKILL.md "Discovered Modals / Dialogs" section for the EXACT elements and their selectors. If a toggle, mode selector, or form field is not documented in the SKILL.md, use getByRole('button') or getByText() with exact visible text.

Create a Playwright test file at /tests/knowledge-base/edit-kb-schedule.spec.ts.

After creating the test, auto-run it and include the terminal output + result summary:

- Command: npx playwright test tests/knowledge-base/edit-kb-schedule.spec.ts

Test credentials: email=heidi@intnt.ai password=testing2026! org=Testing2026!

1. Perform case "Login"
2. On Select Organization page, select organization "Testing2026!"
3. User redirected to https://dashboard.int3nt.info/
4. Click "Knowledge Base" on the left sidebar
5. Locate knowledge base bucket "Picotest1"
6. Click "Schedule"
7. Manage Schedule modal appears showing the existing schedule
8. Verify the modal displays:

- Sync Type
- Cron Expression

9. Click "Edit Schedule"
10. Schedule configuration screen appears
11. Ensure Cron Expression mode = SIMPLE
12. In Frequency, select Monthly
13. Under Select day(s) choose Day 1
14. In At time, enter 00:00
15. Verify the Cron Expression is automatically updated
16. Click Save

Expected result:

1. Existing schedule information is displayed correctly
2. User able to click Edit Schedule
3. Schedule configuration screen appears
4. User able to modify schedule using Simple mode
5. Cron expression updates automatically based on selected configuration
6. Schedule saved successfully
7. Notification appears: "Schedule updated successfully"
8. Manage Schedule modal now displays the updated Cron Expression
   EOF
   )"
