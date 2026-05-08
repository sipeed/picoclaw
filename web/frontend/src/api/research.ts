import { launcherFetch } from "@/api/http"

export interface ResearchAgent {
  id: string
  name: string
  active: boolean
  progress: number
  ram: string
  type: string
}

export interface ResearchNode {
  name: string
  abbr: string
  x: number
  y: number
}

export interface ResearchReport {
  id: string
  title: string
  pages: number
  words: number
  status: "in-progress" | "complete"
  progress?: number
}

export interface ResearchConfig {
  type: string
  depth: string
  restrictToGraph: boolean
}

// API Functions (TanStack Query compatible)
export async function listResearchAgents(): Promise<ResearchAgent[]> {
  const res = await launcherFetch("/api/research/agents")
  return res.json() as Promise<ResearchAgent[]>
}

export async function toggleResearchAgent(id: string): Promise<void> {
  await launcherFetch(`/api/research/agents/${id}/toggle`, { method: "PUT" })
}

export async function listResearchGraph(): Promise<ResearchNode[]> {
  const res = await launcherFetch("/api/research/graph")
  const data = await res.json() as { nodes: ResearchNode[] }
  return data.nodes
}

export async function listResearchReports(): Promise<ResearchReport[]> {
  const res = await launcherFetch("/api/research/reports")
  const data = await res.json() as { reports: ResearchReport[] }
  return data.reports
}

export async function updateResearchConfig(config: ResearchConfig): Promise<void> {
  await launcherFetch("/api/research/config", {
    method: "PUT",
    body: JSON.stringify(config),
  })
}