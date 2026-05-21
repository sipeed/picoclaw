import { launcherFetch } from "@/api/http"

export interface TurnProfilesResponse {
  profiles: string[]
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(path, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getTurnProfiles(): Promise<TurnProfilesResponse> {
  return request<TurnProfilesResponse>("/api/config/turn-profiles")
}
