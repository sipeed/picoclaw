# Skills Management Design

## Scope
Add skills management UI to the cockpit to view/install/delete skills (frontend only, backend API already exists).

## Non-Goals
- Subagents tab (separate feature)
- Skill editing (backend doesn't support it, only install/delete/import)

## Architecture
1. **Frontend API Client** (`web/frontend/src/api/skills.ts`):
   - `listSkills()`: GET /api/skills → returns `SkillSupportItem[]`
   - `getSkill(name)`: GET /api/skills/{name} → returns `SkillDetailResponse`
   - `searchSkills(query, limit?, offset?)`: GET /api/skills/search?q=... → returns `SkillSearchResponse`
   - `installSkill(slug, registry?)`: POST /api/skills/install → returns `InstallSkillResponse`
   - `deleteSkill(name)`: DELETE /api/skills/{name} → returns `{status: "ok"}`
   - `importSkill(file)`: POST /api/skills/import (multipart form) → returns `SkillSupportItem`

2. **Cockpit Skills Hook** (`web/frontend/src/hooks/use-cockpit-skills.ts`):
   - Uses TanStack Query to manage skills state (consistent with agents)
   - Exposes: `skillsQuery`, `searchQuery`, `installSkill`, `deleteSkill`
   - Handles loading/error states

3. **Skills UI Integration**:
   - Add "Skills" tab to `cockpit-page.tsx` (between "Tools" and "Agents")
   - Create `SkillsPage` component (similar to `AgentsPage`):
     - Search bar
     - Grid of `SkillCard` components (name, description, origin kind, version)
     - Install/Delete actions
   - Reuse existing UI components: `Card`, `Button`, `Input`, `Badge`

## Data Flow
Frontend → `launcherFetch` → Backend `/api/skills` → `pkg/skills` loader → returns installed skills

## Error Handling
- Query errors: Show error message (similar to agents fix)
- Install failures: Toast error with message
- Delete confirm dialog (similar to agents)

## Testing
- Verify API client functions match backend API types
- Test skills query loading/error states
- Test install/delete flows

## Failure-Mode Check
1. **Backend skills tool disabled**: `ensureSkillRegistryToolEnabled` returns error → Frontend shows error, disable install actions. *Severity: Minor, document as known limitation.*
2. **Large skill list**: Pagination not implemented in backend list → Load all skills at once. *Severity: Minor, acceptable for now.*
3. **Install conflicts**: Skill already exists → Backend returns 409, frontend shows error. *Severity: Minor, handled by error message.*

## Next Steps
After user approval, invoke `writing-plans` to decompose into implementation tasks.
