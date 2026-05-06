import { launcherFetch } from "@/api/http"

// API client for Pico Channel configuration.

interface PicoInfoResponse {
  ws_url: string
  enabled: boolean
  configured?: boolean
}

interface PicoSetupResponse {
  ws_url: string
  enabled: boolean
  configured?: boolean
  changed: boolean
}

export interface PicoSubagentStatusItem {
  id: string
  label?: string
  status: "running" | "completed" | "failed" | "canceled" | string
  created: number
  result?: string
}

export interface PicoSubagentStatusResponse {
  session_id: string
  channel: string
  chat_id: string
  tasks: PicoSubagentStatusItem[]
}

export interface PicoMemoryGraphNode {
  id: string
  label: string
  kind: string
  group: string
  preview?: string
  weight?: number
}

export interface PicoMemoryGraphEdge {
  source: string
  target: string
  kind: string
}

export interface PicoMemoryGraphResponse {
  session_id: string
  generated_at: string
  nodes: PicoMemoryGraphNode[]
  edges: PicoMemoryGraphEdge[]
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getPicoInfo(): Promise<PicoInfoResponse> {
  return request<PicoInfoResponse>("/api/pico/info")
}

export async function regenPicoToken(): Promise<PicoInfoResponse> {
  return request<PicoInfoResponse>("/api/pico/token", { method: "POST" })
}

export async function setupPico(): Promise<PicoSetupResponse> {
  return request<PicoSetupResponse>("/api/pico/setup", { method: "POST" })
}

export async function getPicoSubagents(
  sessionId: string,
): Promise<PicoSubagentStatusResponse> {
  const params = new URLSearchParams({ session_id: sessionId })
  return request<PicoSubagentStatusResponse>(
    `/api/pico/subagents?${params.toString()}`,
  )
}

export async function getPicoMemoryGraph(
  sessionId: string,
): Promise<PicoMemoryGraphResponse> {
  const params = new URLSearchParams({ session_id: sessionId })
  return request<PicoMemoryGraphResponse>(
    `/api/pico/memory-graph?${params.toString()}`,
  )
}

export type { PicoInfoResponse, PicoSetupResponse }
