docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m "Before generating anything, read and follow:

- /skills/app-selector/SKILL.md
- /skills/playwright/SKILL.md

Create a Playwright test file at /tests/auth/login-test.spec.ts.

Test steps:

1. Open https://dashboard.int3nt.info/login
2. Fill Username 'heidi@intnt.ai'
3. Fill Password 'testing2026!'
4. Click Login

Expected result:

1. Login is successful
2. User is redirected to a URL that contains '?select_org'
   "

docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m 'Before generating anything, read and follow:

- /skills/app-selector-public/SKILL.md
- /skills/app-selector-protected/SKILL.md
- /skills/playwright/SKILL.md

Create a Playwright test file at /tests/knowledge-base/create-kb-bucket-gcs.spec.ts.

1. Perform case "Login" (Open https://dashboard.int3nt.info/login, Fill Username "heidi@intnt.ai", Fill Password "testing2026!", Click Login)
2. On Select Organization page, select organization "Testing2026!"
3. User redirected to https://dashboard.int3nt.info/
4. Click "Knowledge Base" in the left sidebar
5. Click "Create Knowledge Base Bucket" button
6. In Create KB Bucket Step 1: Source Settings, fill Knowledge Group Name with "Picotest1"
7. Leave LLM transformer model to parse documents as None
8. Click Source Type dropdown
9. Select Google Cloud Storage
10. Click Continue
11. In Create KB Bucket Step 2: Search Engine Configuration, verify the following default values:
12. Search Engine is set to Elasticsearch
13. Elasticsearch URL field is populated
14. Password/API Key field is populated
15. Leave all fields as default values
16. Click Submit

Expected result:

1. User successfully accesses Knowledge Base page
2. Create KB Bucket panel appears after clicking Create Knowledge Base Bucket
3. User able to enter Knowledge Group Name
4. User able to select Source Type: Google Cloud Storage
5. User able to proceed to Step 2: Search Engine Configuration
6. Default search engine configuration is displayed
7. New Knowledge Base Bucket "Picotest1" is successfully created
8. The new bucket appears in the Knowledge Base list'

docker compose --env-file .env --profile gateway run --rm picoclaw-agent -m 'Before generating anything, read and follow:

- /skills/app-selector-public/SKILL.md
- /skills/app-selector-protected/SKILL.md
- /skills/playwright/SKILL.md

Create a Playwright test file at /tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts.

1. Perform case "Login"
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

## Global Login Flow

```
Step 1 — Navigate to login:
  await page.goto('https://dashboard.int3nt.info/login', { waitUntil: 'networkidle' });

Step 2 — Fill credentials:
  await page.locator('.v-field__input').nth(0).fill('EMAIL');   // Email address field
  await page.locator('.v-field__input').nth(1).fill('PASSWORD'); // Password field
  await page.locator('button:has-text("Login")').click();

Step 3 — Wait for redirect after login (REQUIRED):
  // IMPORTANT: Use ONE single waitForURL for the final target.
  // Do NOT use waitForURL(/.*/) — it matches any URL including /login!
  // Do NOT chain multiple waitForURL calls — causes net::ERR_ABORTED!
  await page.waitForURL(/\\?select_org/, { timeout: 15000 });
  expect(page.url()).toContain('?select_org');

Step 4 — Select organization (if redirected to /?select_org):
  // Wait for org cards to render:
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  // MUST use .filter({ hasText: 'OrgName' }) with the EXACT org name from the prompt.
  // Do NOT use .first() without filtering — you must select the correct org!
  await page.locator('.organization-card').filter({ hasText: 'Testing2026!' }).click();
  // Known orgs: "Testing2026!", "Testing"
  // Wait for redirect to dashboard after selecting org:
  await page.waitForURL(/dashboard\.int3nt\.info\/(?!\?select_org)/, { timeout: 15000 });
```


docker exec $(docker compose --env-file .env --profile gateway ps -q picoclaw-gateway) \
  node /home/picoclaw/.picoclaw/workspace/inspect-all-pages.js