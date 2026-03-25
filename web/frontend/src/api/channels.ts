// API client for channels navigation and channel-specific config flows.

export type ChannelConfig = Record<string, unknown>
export type AppConfig = Record<string, unknown>

export interface SupportedChannel {
  name: string
  display_name?: string
  config_key: string
  variant?: string
}

interface ChannelsCatalogResponse {
  channels: SupportedChannel[]
}

interface ConfigActionResponse {
  status: string
  errors?: string[]
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as {
        error?: string
        errors?: string[]
        status?: string
      }
      if (Array.isArray(body.errors) && body.errors.length > 0) {
        message = body.errors.join("; ")
      } else if (typeof body.error === "string" && body.error.trim() !== "") {
        message = body.error
      }
    } catch {
      // Keep default fallback message if response body is not JSON.
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function getChannelsCatalog(): Promise<ChannelsCatalogResponse> {
  return request<ChannelsCatalogResponse>("/api/channels/catalog")
}

export async function getAppConfig(): Promise<AppConfig> {
  return request<AppConfig>("/api/config")
}

export async function patchAppConfig(
  patch: Record<string, unknown>,
): Promise<ConfigActionResponse> {
  return request<ConfigActionResponse>("/api/config", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  })
}

// WeChat QR login flow API

export interface WeixinFlowResponse {
  flow_id: string
  status: "wait" | "scaned" | "confirmed" | "expired" | "error"
  qr_data_uri?: string
  account_id?: string
  error?: string
}

export async function startWeixinFlow(): Promise<WeixinFlowResponse> {
  return request<WeixinFlowResponse>("/api/weixin/flows", { method: "POST" })
}

export async function pollWeixinFlow(flowID: string): Promise<WeixinFlowResponse> {
  return request<WeixinFlowResponse>(`/api/weixin/flows/${encodeURIComponent(flowID)}`)
}

export type { ChannelsCatalogResponse, ConfigActionResponse }
