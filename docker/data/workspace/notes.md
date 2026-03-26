docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m 'Before generating anything, read and follow:

- /skills/playwright/SKILL.md
- /skills/app-selectors/SKILL.md
- /skills/app-selectors-knowledge-base/SKILL.md

IMPORTANT RULES:

1. You MUST only use selectors from the SKILL.md files above. Copy them exactly.
2. NEVER guess or invent Vuetify selectors like .v-dialog, .v-dialog--active, .v-overlay, .v-card, etc.
3. For login, use the EXACT credentials and selectors from /skills/app-selectors/SKILL.md. Do NOT use env vars or placeholder credentials.
4. For modals/dialogs, use the actual custom CSS classes from the Discovered Modals or Custom Elements section. NEVER use generic Vuetify wrappers.

Create a Playwright test file at /tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts.

After creating the test, auto-run it and include the terminal output + result summary:

- Command: npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts

Test credentials: email=heidi@intnt.ai password=testing2026! org=Testing2026!

1. Perform case "Login" with email heidi@intnt.ai and password testing2026!
2. On Select Organization page, select organization "Testing2026!"
3. User redirected to https://dashboard.int3nt.info/
4. Click "Knowledge Base" in the left sidebar
5. Locate knowledge base bucket "Picotest1"
6. Click "Schedule" on the bucket card
7. Manage Schedule modal appears
8. Click "Create Schedule"
9. Under Sync Type, select Full Sync
10. Ensure Cron Expression mode = SIMPLE
11. In Frequency, select Weekly
12. Verify the Cron Expression is automatically generated
13. Click Save

Expected result:

1. Manage Schedule modal appears
2. User able to select Full Sync
3. User able to configure schedule using Simple mode
4. System automatically generates a Cron Expression based on selected frequency
5. Schedule is saved successfully
6. Notification appears at the bottom of the page: "Schedule created successfully"'

# Single page:

docker exec $(docker compose --env-file .env --profile gateway ps -q picoclaw-gateway) \
 node /home/picoclaw/.picoclaw/workspace/inspect-scripts/inspect-knowledge-base.js

# All pages (shared browser, efficient):

docker exec $(docker compose --env-file .env --profile gateway ps -q picoclaw-gateway) \
 node /home/picoclaw/.picoclaw/workspace/inspect-scripts/run-all.js

docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m 'Before generating anything, read and follow:

- /skills/playwright/SKILL.md
- /skills/app-selectors/SKILL.md
- /skills/app-selectors-knowledge-base/SKILL.md

IMPORTANT RULES:

1. You MUST only use selectors from the SKILL.md files above. Copy them exactly.
2. NEVER guess or invent Vuetify selectors like .v-dialog, .v-dialog--active, .v-overlay, .v-card, etc.
3. For login, use the EXACT credentials and selectors from /skills/app-selectors/SKILL.md. Do NOT use env vars or placeholder credentials.
4. For modals/dialogs, use the actual custom CSS classes from the Discovered Modals or Custom Elements section. NEVER use generic Vuetify wrappers.

Create a Playwright test file at /tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts.

After creating the test, auto-run it and include the terminal output + result summary:

- Command: npx playwright test tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts

Test credentials: email=heidi@intnt.ai password=testing2026! org=Testing2026!

1. Perform case "Login"
2. On Select Organization page, select organization "Testing2026!"
3. User redirected to https://dashboard.int3nt.info/
4. Click "Knowledge Base" in the left sidebar
5. Click "Create Knowledge Base Bucket" button
6. In Create KB Bucket Step 1: Source Settings, fill Knowledge Group Name with "Picotest2"
7. Leave LLM transformer model to parse documents as None
8. Click Source Type dropdown
9. Select Website crawler
10. Click Continue
11. On the Website Crawler Configuration, fill the "Base url" field with "https://intentai.com"
12. Click the "Web crawler parameters", fill the "Seed URLs" field with "https://intentai.com/blog/"
13. Leave the "Sitemap URLs" and "Schedule (Cron Expression)" as it is
14. Click on "Crawl depth and limits", fill the field with:

- fill Max Crawl Depth: "2"
- Max Extracted Links Count: "1000"
- Max Unique URL Count: "25"

15. leave the rest as it is, click continue
16. Submit

Expected result:

1. User successfully accesses Knowledge Base page
2. Create KB Bucket panel appears after clicking Create Knowledge Base Bucket
3. User able to enter Knowledge Group Name
4. User able to select Source Type: Website Crawler
5. User able to proceed to Step 2: Search Engine Configuration
6. Default search engine configuration is displayed
7. New Knowledge Base Bucket "Picotest2" is successfully created
8. The new bucket appears in the Knowledge Base list'
