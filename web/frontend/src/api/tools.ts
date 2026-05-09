import { launcherFetch } from "@/api/http"

export interface ToolSupportItem {
  name: string
  description: string
  category: string
  config_key: string
  status: "enabled" | "disabled" | "blocked"
  reason_code?: string
}

interface ToolsResponse {
  tools: ToolSupportItem[]
}

interface ToolActionResponse {
  status: string
}

export interface WebSearchProviderOption {
  id: string
  label: string
  configured: boolean
  current: boolean
  requires_auth: boolean
}

export interface WebSearchProviderConfig {
  enabled: boolean
  max_results: number
  base_url?: string
  api_key?: string
  api_key_set?: boolean
}

export interface WebSearchConfigResponse {
  provider: string
  current_service: string
  prefer_native: boolean
  proxy?: string
  providers: WebSearchProviderOption[]
  settings: Record<string, WebSearchProviderConfig>
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as {
        error?: string
        errors?: string[]
      }
      if (Array.isArray(body.errors) && body.errors.length > 0) {
        message = body.errors.join("; ")
      } else if (typeof body.error === "string" && body.error.trim() !== "") {
        message = body.error
      }
    } catch {
      // ignore invalid body
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function getTools(): Promise<ToolsResponse> {
  return request<ToolsResponse>("/api/tools")
}

export async function setToolEnabled(
  name: string,
  enabled: boolean,
): Promise<ToolActionResponse> {
  return request<ToolActionResponse>(
    `/api/tools/${encodeURIComponent(name)}/state`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    },
  )
}

export async function getWebSearchConfig(): Promise<WebSearchConfigResponse> {
  return request<WebSearchConfigResponse>("/api/tools/web-search-config")
}

export async function updateWebSearchConfig(
  payload: WebSearchConfigResponse,
): Promise<WebSearchConfigResponse> {
  return request<WebSearchConfigResponse>("/api/tools/web-search-config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
}

export interface GrantPermissionRequest {
  path: string
  command: string
  duration: "once" | "session"
}

export interface GrantPermissionResponse {
  status: string
  message?: string
}

export async function grantPermission(
  path: string,
  command: string,
  duration: "once" | "session"
): Promise<GrantPermissionResponse> {
  return request<GrantPermissionResponse>("/api/permission/grant", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, command, duration }),
  })
}
