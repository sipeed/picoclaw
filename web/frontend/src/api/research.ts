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
  return launcherFetch<ResearchAgent[]>("/api/research/agents")
}

export async function toggleResearchAgent(id: string): Promise<void> {
  await launcherFetch(`/api/research/agents/${id}/toggle`, { method: "PUT" })
}

export async function listResearchGraph(): Promise<ResearchNode[]> {
  const response = await launcherFetch<{ nodes: ResearchNode[] }>("/api/research/graph")
  return response.nodes
}

export async function listResearchReports(): Promise<ResearchReport[]> {
  const response = await launcherFetch<{ reports: ResearchReport[] }>("/api/research/reports")
  return response.reports
}

export async function updateResearchConfig(config: ResearchConfig): Promise<void> {
  await launcherFetch("/api/research/config", {
    method: "PUT",
    body: JSON.stringify(config),
  })
}