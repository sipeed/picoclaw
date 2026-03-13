// API client for Pico Channel configuration.

interface PicoTokenResponse {
  token: string
  ws_url: string
  enabled: boolean
}

interface PicoSetupResponse {
  token: string
  ws_url: string
  enabled: boolean
  changed: boolean
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getPicoToken(): Promise<PicoTokenResponse> {
  return request<PicoTokenResponse>("/api/pico/token")
}

export async function regenPicoToken(): Promise<PicoTokenResponse> {
  return request<PicoTokenResponse>("/api/pico/token", { method: "POST" })
}

export async function setupPico(): Promise<PicoSetupResponse> {
  return request<PicoSetupResponse>("/api/pico/setup", { method: "POST" })
}

export type { PicoTokenResponse, PicoSetupResponse }
