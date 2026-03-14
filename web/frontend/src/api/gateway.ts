// API client for gateway process management.

interface GatewayStatusResponse {
  gateway_status: "running" | "starting" | "restarting" | "stopped" | "error"
  gateway_start_allowed?: boolean
  gateway_start_reason?: string
  passphrase_state?: "" | "pending" | "failed"
  gateway_restart_required?: boolean
  pid?: number
  boot_default_model?: string
  config_default_model?: string
  [key: string]: unknown
}

interface GatewayLogsResponse {
  logs?: string[]
  log_total?: number
  log_run_id?: number
}

interface GatewayActionResponse {
  status: string
  pid?: number
  log_total?: number
  log_run_id?: number
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getGatewayStatus(): Promise<GatewayStatusResponse> {
  return request<GatewayStatusResponse>("/api/gateway/status")
}

export async function getGatewayLogs(options?: {
  log_offset?: number
  log_run_id?: number
}): Promise<GatewayLogsResponse> {
  const params = new URLSearchParams()
  if (options?.log_offset !== undefined) {
    params.set("log_offset", options.log_offset.toString())
  }
  if (options?.log_run_id !== undefined) {
    params.set("log_run_id", options.log_run_id.toString())
  }
  const queryString = params.toString() ? `?${params.toString()}` : ""
  return request<GatewayLogsResponse>(`/api/gateway/logs${queryString}`)
}

export async function startGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/start", {
    method: "POST",
  })
}

export async function stopGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/stop", {
    method: "POST",
  })
}

export async function restartGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/restart", {
    method: "POST",
  })
}

export async function clearGatewayLogs(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/logs/clear", {
    method: "POST",
  })
}

export type {
  GatewayStatusResponse,
  GatewayLogsResponse,
  GatewayActionResponse,
}
