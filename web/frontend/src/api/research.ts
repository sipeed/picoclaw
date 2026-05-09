import { launcherFetch } from "@/api/http"

export interface ResearchAgent {
  id: string
  name: string
  active: boolean
  progress: number
  ram: string
  type: string
  status?: string
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
  restrict_to_graph: boolean
}

// Default/fallback data for offline mode
const DEFAULT_AGENTS: ResearchAgent[] = [
  { id: "1", name: "Literature Analyzer", active: false, progress: 0, ram: "2GB", type: "literature-analyzer" },
  { id: "2", name: "Data Extractor", active: false, progress: 0, ram: "4GB", type: "data-extractor" },
  { id: "3", name: "Fact Validator", active: false, progress: 0, ram: "1GB", type: "fact-validator" },
  { id: "4", name: "Synthesizer", active: false, progress: 0, ram: "3GB", type: "synthesizer" },
]

const DEFAULT_NODES: ResearchNode[] = [
  { name: "Research Topic", abbr: "RT", x: 400, y: 300 },
  { name: "Literature", abbr: "Lit", x: 200, y: 150 },
  { name: "Data Sources", abbr: "DS", x: 600, y: 150 },
  { name: "Analysis", abbr: "An", x: 400, y: 450 },
]

const DEFAULT_REPORTS: ResearchReport[] = [
  { id: "1", title: "Initial Research Report", pages: 0, words: 0, status: "in-progress", progress: 0 },
]

const DEFAULT_CONFIG: ResearchConfig = {
  type: "comprehensive",
  depth: "deep",
  restrict_to_graph: false,
}

// API Functions with offline fallback

/**
 * Fetches research agents with offline fallback
 */
export async function listResearchAgents(): Promise<ResearchAgent[]> {
  try {
    const res = await launcherFetch("/api/research/agents")
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as { agents: ResearchAgent[] }
    return data.agents || []
  } catch (error) {
    console.warn("[Research API] Failed to fetch agents, using offline fallback:", error)
    return DEFAULT_AGENTS
  }
}

/**
 * Toggles a research agent's active state
 */
export async function toggleResearchAgent(id: string): Promise<void> {
  const res = await launcherFetch(`/api/research/agents/${id}/toggle`, { method: "PUT" })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

/**
 * Fetches research graph nodes with offline fallback
 */
export async function listResearchGraph(): Promise<ResearchNode[]> {
  try {
    const res = await launcherFetch("/api/research/graph")
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as { nodes: ResearchNode[] }
    return data.nodes || []
  } catch (error) {
    console.warn("[Research API] Failed to fetch graph, using offline fallback:", error)
    return DEFAULT_NODES
  }
}

/**
 * Fetches research reports with offline fallback
 */
export async function listResearchReports(): Promise<ResearchReport[]> {
  try {
    const res = await launcherFetch("/api/research/reports")
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as { reports: ResearchReport[] }
    return data.reports || []
  } catch (error) {
    console.warn("[Research API] Failed to fetch reports, using offline fallback:", error)
    return DEFAULT_REPORTS
  }
}

/**
 * Gets current research configuration with offline fallback
 */
export async function getResearchConfig(): Promise<ResearchConfig> {
  try {
    const res = await launcherFetch("/api/research/config")
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return await res.json() as Promise<ResearchConfig>
  } catch (error) {
    console.warn("[Research API] Failed to fetch config, using offline fallback:", error)
    return DEFAULT_CONFIG
  }
}

/**
 * Updates research configuration
 */
export async function updateResearchConfig(config: ResearchConfig): Promise<void> {
  const res = await launcherFetch("/api/research/config", {
    method: "PUT",
    body: JSON.stringify(config),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

/**
 * Exports a research report in the specified format
 * @param reportId - The ID of the report to export
 * @param format - Export format: "markdown" or "pdf"
 * @returns The blob content
 */
export async function exportReport(reportId: string, format: "markdown" | "pdf" = "markdown"): Promise<Blob> {
  const res = await launcherFetch(`/api/research/export?id=${reportId}&format=${format}`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.blob()
}

/**
 * Downloads a report as a file
 */
export async function downloadReport(reportId: string, title: string, format: "markdown" | "pdf" = "markdown"): Promise<void> {
  const blob = await exportReport(reportId, format)
  const extension = format === "pdf" ? "txt" : "md" // PDF actually returns txt for now
  const filename = `${title.replace(/[^a-z0-9]/gi, "_")}.${extension}`

  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}