import { launcherFetch } from "@/api/http"

export interface Agent {
  slug: string
  name: string
  description: string
  system_prompt: string
  model: string
  tool_permissions: string[]
  status: "enabled" | "disabled"
  created_at: number
  updated_at: number
}

export interface AgentListResponse {
  agents: Agent[]
}

export interface AgentCreateRequest {
  name: string
  description?: string
  system_prompt: string
  model: string
  tool_permissions?: string[]
}

export interface AgentUpdateRequest {
  name?: string
  description?: string
  system_prompt?: string
  model?: string
  tool_permissions?: string[]
  status?: string
}

export async function listAgents(): Promise<AgentListResponse> {
  const res = await launcherFetch("/api/agents")
  if (!res.ok) throw new Error(`Failed to list agents: ${res.status}`)
  return res.json()
}

export async function getAgent(slug: string): Promise<Agent> {
  const params = new URLSearchParams({ slug })
  const res = await launcherFetch(`/api/agent?${params.toString()}`)
  if (!res.ok) throw new Error(`Failed to get agent: ${res.status}`)
  return (await res.json()).agent
}

export async function createAgent(data: AgentCreateRequest): Promise<Agent> {
  const res = await launcherFetch("/api/agent/create", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to create agent: ${res.status}`)
  return (await res.json()).agent
}

export async function updateAgent(slug: string, data: AgentUpdateRequest): Promise<Agent> {
  const params = new URLSearchParams({ slug })
  const res = await launcherFetch(`/api/agent/update?${params.toString()}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to update agent: ${res.status}`)
  return (await res.json()).agent
}

export async function deleteAgent(slug: string): Promise<{ status: string }> {
  const params = new URLSearchParams({ slug })
  const res = await launcherFetch(`/api/agent/delete?${params.toString()}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(`Failed to delete agent: ${res.status}`)
  return res.json()
}

export async function importAgent(content: string): Promise<Agent> {
  const res = await launcherFetch("/api/agent/import", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  })
  if (!res.ok) throw new Error(`Failed to import agent: ${res.status}`)
  return (await res.json()).agent
}
