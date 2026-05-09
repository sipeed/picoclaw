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

export async function getSkills(): Promise<SkillsListResponse> {
  return listSkills()
}

export async function getSkill(name: string): Promise<SkillDetailResponse> {
  const res = await launcherFetch(`/api/skills/${encodeURIComponent(name)}`)
  if (!res.ok) throw new Error(`Failed to get skill: ${res.status}`)
  return res.json()
}

export interface SkillRegistrySearchResult {
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
