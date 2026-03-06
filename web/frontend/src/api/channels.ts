// API client for channel management.

export type ChannelConfig = Record<string, unknown>

export interface ChannelInfo {
  name: string
  display_name: string
  enabled: boolean
  configured: boolean
  config: ChannelConfig
}

interface ChannelsListResponse {
  channels: ChannelInfo[]
}

interface ChannelActionResponse {
  status: string
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getChannels(): Promise<ChannelsListResponse> {
  return request<ChannelsListResponse>("/api/channels")
}

export async function updateChannel(
  name: string,
  config: ChannelConfig,
): Promise<ChannelActionResponse> {
  return request<ChannelActionResponse>(`/api/channels/${name}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
}

export async function toggleChannel(
  name: string,
  enabled: boolean,
): Promise<ChannelActionResponse> {
  return request<ChannelActionResponse>(`/api/channels/${name}/toggle`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  })
}

export type { ChannelsListResponse, ChannelActionResponse }
