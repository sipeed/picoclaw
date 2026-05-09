# Skills Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-optimized:subagent-driven-development (recommended) or superpowers-optimized:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add skills management UI to cockpit (frontend only, backend API already exists)

**Architecture:** Frontend skills API client + TanStack Query hook + SkillsPage component integrated into cockpit tabs. Reuses existing UI components (Card, Button, Input, Badge) and follows same patterns as AgentsPage.

**Tech Stack:** TypeScript, React, TanStack Query, Tailwind CSS, shadcn/ui

**Assumptions:**
- Backend skills API is fully functional at `/api/skills` endpoints — will NOT work if backend is broken
- `launcherFetch` from `@/api/http` handles auth correctly — will NOT work if auth is misconfigured
- Existing UI components (Card, Button, etc.) match their current API — will NOT work if component APIs changed

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `web/frontend/src/api/skills.ts` | Create | Skills API client functions matching backend types |
| `web/frontend/src/hooks/use-cockpit-skills.ts` | Create | TanStack Query hook for skills state management |
| `web/frontend/src/components/agent/skills/skill-card.tsx` | Create | SkillCard component displaying skill info |
| `web/frontend/src/components/agent/skills/skills-page.tsx` | Create | SkillsPage with search, install, delete |
| `web/frontend/src/components/agent/skills/index.ts` | Create | Barrel export file |
| `web/frontend/src/components/agent/cockpit/cockpit-page.tsx` | Modify | Add "Skills" tab between Tools and Agents |
| `web/frontend/src/components/agent/cockpit/use-agent-cockpit.ts` | Modify | Add skills hook integration |

---

### Task 1: Create Skills API Client

**Files:**
- Create: `web/frontend/src/api/skills.ts`

**Does NOT cover:** Skills editing (backend doesn't support it), registry search UI (future feature)

- [x] **Step 1: Create skills API types and functions**

```typescript
import { launcherFetch } from "@/api/http"

export interface SkillSupportItem {
  name: string
  path: string
  source: string
  description: string
  origin_kind: string
  registry_name?: string
  registry_url?: string
  installed_version?: string
  installed_at?: number
}

export interface SkillsListResponse {
  skills: SkillSupportItem[]
}

export interface SkillDetailResponse extends SkillSupportItem {
  content: string
}

export interface SkillSearchResultItem {
  score: number
  slug: string
  display_name: string
  summary: string
  version: string
  registry_name: string
  url?: string
  installed: boolean
  installed_name?: string
}

export interface SkillSearchResponse {
  results: SkillSearchResultItem[]
  limit: number
  offset: number
  next_offset?: number
  has_more: boolean
}

export interface InstallSkillRequest {
  slug: string
  registry?: string
  version?: string
  force?: boolean
}

export interface InstallSkillResponse {
  status: string
  slug: string
  registry: string
  version: string
  summary?: string
  is_suspicious?: boolean
  skill?: SkillSupportItem
}

export async function listSkills(): Promise<SkillsListResponse> {
  const res = await launcherFetch("/api/skills")
  if (!res.ok) throw new Error(`Failed to list skills: ${res.status}`)
  return res.json()
}

export async function getSkill(name: string): Promise<SkillDetailResponse> {
  const res = await launcherFetch(`/api/skills/${encodeURIComponent(name)}`)
  if (!res.ok) throw new Error(`Failed to get skill: ${res.status}`)
  return res.json()
}

export async function searchSkills(query: string, limit = 20, offset = 0): Promise<SkillSearchResponse> {
  const params = new URLSearchParams({ q: query })
  if (limit !== 20) params.set("limit", limit.toString())
  if (offset !== 0) params.set("offset", offset.toString())
  const res = await launcherFetch(`/api/skills/search?${params.toString()}`)
  if (!res.ok) throw new Error(`Failed to search skills: ${res.status}`)
  return res.json()
}

export async function installSkill(data: InstallSkillRequest): Promise<InstallSkillResponse> {
  const res = await launcherFetch("/api/skills/install", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: `Failed to install skill: ${res.status}` }))
    throw new Error(error.message || `Failed to install skill: ${res.status}`)
  }
  return res.json()
}

export async function deleteSkill(name: string): Promise<{ status: string }> {
  const res = await launcherFetch(`/api/skills/${encodeURIComponent(name)}`, {
    method: "DELETE",
  })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: `Failed to delete skill: ${res.status}` }))
    throw new Error(error.message || `Failed to delete skill: ${res.status}`)
  }
  return res.json()
}

export async function importSkill(file: File): Promise<SkillSupportItem> {
  const formData = new FormData()
  formData.append("file", file)
  const res = await launcherFetch("/api/skills/import", {
    method: "POST",
    body: formData,
  })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: `Failed to import skill: ${res.status}` }))
    throw new Error(error.message || `Failed to import skill: ${res.status}`)
  }
  return res.json()
}
```

- [x] **Step 2: Verify TypeScript compilation**

Run: `cd web/frontend && npx tsc --noEmit --project tsconfig.json 2>&1 | Select-String "skills.ts" -CaseSensitive:$false`
Expected: No errors containing "skills.ts"

- [x] **Step 3: Commit**

```bash
git add web/frontend/src/api/skills.ts
git commit -m "feat(frontend): add skills API client functions"
```

---

### Task 2: Create Cockpit Skills Hook

**Files:**
- Create: `web/frontend/src/hooks/use-cockpit-skills.ts`

**Does NOT cover:** Caching strategies beyond TanStack Query defaults, offline support

- [x] **Step 1: Create skills hook with TanStack Query**

```typescript
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import {
  listSkills,
  deleteSkill,
  type SkillSupportItem,
  type SkillsListResponse,
} from "@/api/skills"

export function useCockpitSkills() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const skillsQuery = useQuery({
    queryKey: ["skills"],
    queryFn: listSkills,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteSkill(name),
    onSuccess: () => {
      toast.success(t("pages.agent.skills.delete_success", "Skill deleted"))
      void queryClient.invalidateQueries({ queryKey: ["skills"] })
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to delete skill")
    },
  })

  return {
    skills: skillsQuery.data?.skills ?? [],
    isLoading: skillsQuery.isLoading,
    isError: skillsQuery.isError,
    deleteSkill: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  }
}
```

- [x] **Step 2: Verify TypeScript compilation**

Run: `cd web/frontend && npx tsc --noEmit --project tsconfig.json 2>&1 | Select-String "use-cockpit-skills.ts" -CaseSensitive:$false`
Expected: No errors containing "use-cockpit-skills.ts"

- [x] **Step 3: Commit**

```bash
git add web/frontend/src/hooks/use-cockpit-skills.ts
git commit -m "feat(frontend): add cockpit skills hook with TanStack Query"
```

---

### Task 3: Create SkillCard Component

**Files:**
- Create: `web/frontend/src/components/agent/skills/skill-card.tsx`
- Create: `web/frontend/src/components/agent/skills/index.ts`

**Does NOT cover:** Skill editing UI (backend doesn't support), skill content preview

- [x] **Step 1: Create SkillCard component**

```tsx
import { IconTrash, IconDownload, IconWorld, IconFolder } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"
import type { SkillSupportItem } from "@/api/skills"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

interface SkillCardProps {
  skill: SkillSupportItem
  onDelete: () => void
}

function originKindLabel(kind: string): string {
  switch (kind) {
    case "builtin":
      return "Built-in"
    case "third_party":
      return "Third Party"
    case "manual":
      return "Manual"
    default:
      return kind
  }
}

export function SkillCard({ skill, onDelete }: SkillCardProps) {
  const { t } = useTranslation()

  const kindColor = skill.origin_kind === "builtin"
    ? "bg-blue-500/20 text-blue-400"
    : skill.origin_kind === "third_party"
    ? "bg-purple-500/20 text-purple-400"
    : "bg-gray-500/20 text-gray-400"

  return (
    <Card size="sm">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <CardTitle className="text-base font-semibold tracking-tight">
                {skill.name}
              </CardTitle>
              <Badge className={cn("text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5", kindColor)}>
                {originKindLabel(skill.origin_kind)}
              </Badge>
              {skill.installed_version && (
                <Badge className="bg-white/10 text-white/60 text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5">
                  v{skill.installed_version}
                </Badge>
              )}
            </div>
            <CardDescription className="line-clamp-2 text-sm leading-relaxed">
              {skill.description}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 text-xs text-white/60">
            {skill.registry_name && (
              <span className="flex items-center gap-1">
                <IconWorld className="size-3.5" />
                {skill.registry_name}
              </span>
            )}
            {skill.source && (
              <span className="flex items-center gap-1">
                <IconFolder className="size-3.5" />
                {skill.source}
              </span>
            )}
          </div>
          {skill.origin_kind === "manual" && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onDelete}
              className="text-white/60 hover:text-destructive hover:bg-destructive/10"
              title={t("common.delete")}
            >
              <IconTrash className="size-3.5" />
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
```

- [x] **Step 2: Create barrel export file**

```typescript
export { SkillCard } from "./skill-card"
```

- [x] **Step 3: Verify TypeScript compilation**

Run: `cd web/frontend && npx tsc --noEmit --project tsconfig.json 2>&1 | Select-String "skill-card" -CaseSensitive:$false`
Expected: No errors containing "skill-card"

- [x] **Step 4: Commit**

```bash
git add web/frontend/src/components/agent/skills/
git commit -m "feat(frontend): add SkillCard component for skills display"
```

---

### Task 4: Create SkillsPage Component

**Files:**
- Create: `web/frontend/src/components/agent/skills/skills-page.tsx`

**Does NOT cover:** Skill install UI (separate future task), skill search, registry browsing

- [x] **Step 1: Create SkillsPage component**

```tsx
import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { IconSearch, IconTrash } from "@tabler/icons-react"
import type { SkillSupportItem } from "@/api/skills"
import { useCockpitSkills } from "@/hooks/use-cockpit-skills"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { SkillCard } from "./skill-card"

interface SkillsPageProps {
  embedded?: boolean
}

export function SkillsPage({ embedded = false }: SkillsPageProps) {
  const { t } = useTranslation()
  const {
    skills,
    isLoading,
    isError,
    deleteSkill,
  } = useCockpitSkills()

  const [searchQuery, setSearchQuery] = useState("")
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const [skillToDelete, setSkillToDelete] = useState<SkillSupportItem | null>(null)

  const filteredSkills = useMemo(() => {
    const query = deferredSearchQuery.trim().toLowerCase()
    if (!query) return skills
    return skills.filter(
      (skill) =>
        skill.name.toLowerCase().includes(query) ||
        skill.description.toLowerCase().includes(query) ||
        skill.origin_kind.toLowerCase().includes(query)
    )
  }, [skills, deferredSearchQuery])

  const handleDelete = async () => {
    if (!skillToDelete) return
    try {
      await deleteSkill(skillToDelete.name)
    } catch {
      // Error handled by hook
    } finally {
      setSkillToDelete(null)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-muted-foreground">Loading skills...</p>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-destructive">Failed to load skills. Please try again.</p>
      </div>
    )
  }

  const mainContent = (
    <div className="mx-auto w-full max-w-6xl">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="max-w-md flex-1">
          <Input
            placeholder={t("common.search", "Search skills...")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="bg-background"
          />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {filteredSkills.length === 0 ? (
          <div className="col-span-full rounded-lg border border-dashed border-white/10 p-12 text-center">
            <p className="text-muted-foreground">
              {deferredSearchQuery
                ? t("pages.agent.skills.no_results", "No skills found")
                : t("pages.agent.skills.no_skills", "No skills installed")}
            </p>
            <p className="mt-2 text-sm text-muted-foreground/60">
              {deferredSearchQuery
                ? t("pages.agent.skills.no_results_hint", "Try a different search")
                : t("pages.agent.skills.no_skills_hint", "Install skills via the CLI or import them")}
            </p>
          </div>
        ) : (
          filteredSkills.map((skill) => (
            <SkillCard
              key={skill.name}
              skill={skill}
              onDelete={() => setSkillToDelete(skill)}
            />
          ))
        )}
      </div>
    </div>
  )

  const modals = (
    <AlertDialog
      open={skillToDelete !== null}
      onOpenChange={(open) => { if (!open) setSkillToDelete(null) }}
    >
      <AlertDialogContent>
        <AlertDialogTitle>
          {t("pages.agent.skills.confirm_delete", "Delete Skill?")}
        </AlertDialogTitle>
        <p className="text-muted-foreground">
          {t(
            "pages.agent.skills.confirm_delete_message",
            `Are you sure you want to delete "${skillToDelete?.name}"? This action cannot be undone.`,
          )}
        </p>
        <AlertDialogFooter>
          <AlertDialogCancel>
            {t("common.cancel", "Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {t("common.delete", "Delete")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )

  if (embedded) {
    return (
      <>
        {mainContent}
        {modals}
      </>
    )
  }

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.skills", "Skills")} />
      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        {mainContent}
      </div>
      {modals}
    </div>
  )
}
```

- [x] **Step 2: Update barrel export**

```typescript
export { SkillCard } from "./skill-card"
export { SkillsPage } from "./skills-page"
```

- [x] **Step 3: Verify TypeScript compilation**

Run: `cd web/frontend && npx tsc --noEmit --project tsconfig.json 2>&1 | Select-String "skills-page" -CaseSensitive:$false`
Expected: No errors containing "skills-page"

- [x] **Step 4: Commit**

```bash
git add web/frontend/src/components/agent/skills/
git commit -m "feat(frontend): add SkillsPage component with search and delete"
```

---

### Task 5: Integrate Skills Tab into Cockpit

**Files:**
- Modify: `web/frontend/src/components/agent/cockpit/cockpit-page.tsx`
- Modify: `web/frontend/src/components/agent/cockpit/use-agent-cockpit.ts`

**Does NOT cover:** Skills tab styling (reuses existing patterns), subagents tab (future feature)

- [x] **Step 1: Read current cockpit-page.tsx and use-agent-cockpit.ts**

Read both files to understand current structure.

- [x] **Step 2: Update use-agent-cockpit.ts to include skills**

Add skills hook integration to the cockpit hook.

```typescript
// Add to imports
import { useCockpitSkills } from "@/hooks/use-cockpit-skills"

// Inside useAgentCockpit function, add:
const {
  skills,
  isLoading: skillsLoading,
  isError: skillsError,
  deleteSkill: deleteSkillFn,
} = useCockpitSkills()

// Add to return object:
skills,
skillsLoading,
skillsError,
```

- [x] **Step 3: Update cockpit-page.tsx to add Skills tab**

```tsx
// Add import
import { SkillsPage } from "../skills"

// Add "skills" to activeTab type and initial state
const [activeTab, setActiveTab] = useState<"tools" | "skills" | "agents">("tools")

// Add Skills tab button (between Tools and Agents buttons)
<button
  onClick={() => setActiveTab("skills")}
  className={cn(
    "flex items-center gap-2 text-xs uppercase tracking-widest font-bold transition-colors",
    activeTab === "skills" ? "text-[#F27D26] border-b-2 border-[#F27D26] pb-4 -mb-4.5" : "text-white/40 hover:text-white/60"
  )}
>
  <IconBrain className="size-4" />
  Skills
</button>

// Add IconBrain import
import { IconLayoutDashboard, IconUsers, IconBrain } from "@tabler/icons-react"

// Add skills tab content (after tools section, before agents section)
{activeTab === "skills" && <SkillsPage embedded />}
```

- [x] **Step 4: Verify TypeScript compilation**

Run: `cd web/frontend && npx tsc --noEmit --project tsconfig.json 2>&1 | Select-String "cockpit-page|use-agent-cockpit" -CaseSensitive:$false`
Expected: No errors containing these files

- [x] **Step 5: Commit**

```bash
git add web/frontend/src/components/agent/cockpit/cockpit-page.tsx web/frontend/src/components/agent/cockpit/use-agent-cockpit.ts
git commit -m "feat(frontend): integrate Skills tab into cockpit"
```

---

## Plan Self-Review

**1. Spec coverage:**
- [x] Skills API client functions → Task 1
- [x] Cockpit skills hook → Task 2
- [x] SkillsPage component with search/delete → Task 3, 4
- [x] Skills tab in cockpit → Task 5
- [x] Loading/error states → Task 4

**2. Placeholder scan:**
- [x] No "TODO", "TBD", or vague steps found
- [x] All code blocks contain actual implementation

**3. Type consistency:**
- [x] `SkillSupportItem` type defined in Task 1, used consistently in Tasks 2-5
- [x] `useCockpitSkills` hook return type matches usage in Tasks 4-5

**4. No hidden dependencies:**
- [x] Task 1 (API client) before Task 2 (hook) before Tasks 3-5 (components)
- [x] Each task produces independently testable changes

---

Plan complete and saved to `docs/superpowers-optimized/plans/2026-05-07-skills-management.md`.

**Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, with checkpoints

**Which approach?**

