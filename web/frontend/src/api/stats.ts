export interface RequestRecord {
  timestamp: string
  request_id: string
  channel: string
  sender_id: string
  sender_info: {
    platform?: string
    platform_id?: string
    canonical_id?: string
    username?: string
    display_name?: string
  }
  chat_id: string
  content: string
  content_length: number
  peer: {
    kind: string
    id: string
  }
  message_id: string
  media_count: number
  session_key: string
  processing_time_ms: number
}

export interface RequestStats {
  total: number
  by_channel: Record<string, number>
  by_day: Record<string, number>
  top_senders: Array<{
    sender: string
    channel: string
    count: number
  }>
}

export interface RequestLogsResponse {
  records: RequestRecord[]
  limit: number
  offset: number
}

export interface RequestLogConfig {
  enabled: boolean
  log_dir: string
  max_file_size_mb: number
  max_files: number
  retention_days: number
  archive_interval: string
  compress_archive: boolean
  log_content_max_length: number
  record_media: boolean
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getRequestStats(options?: {
  start?: string
  end?: string
}): Promise<RequestStats> {
  const params = new URLSearchParams()
  if (options?.start) {
    params.set("start", options.start)
  }
  if (options?.end) {
    params.set("end", options.end)
  }
  const queryString = params.toString() ? `?${params.toString()}` : ""
  return request<RequestStats>(`/api/stats/requests${queryString}`)
}

export async function getRequestLogs(options?: {
  start?: string
  end?: string
  channel?: string
  sender_id?: string
  limit?: number
  offset?: number
}): Promise<RequestLogsResponse> {
  const params = new URLSearchParams()
  if (options?.start) {
    params.set("start", options.start)
  }
  if (options?.end) {
    params.set("end", options.end)
  }
  if (options?.channel) {
    params.set("channel", options.channel)
  }
  if (options?.sender_id) {
    params.set("sender_id", options.sender_id)
  }
  if (options?.limit !== undefined) {
    params.set("limit", options.limit.toString())
  }
  if (options?.offset !== undefined) {
    params.set("offset", options.offset.toString())
  }
  const queryString = params.toString() ? `?${params.toString()}` : ""
  return request<RequestLogsResponse>(`/api/logs/requests${queryString}`)
}

export function getExportLogsUrl(options?: {
  start?: string
  end?: string
  channel?: string
  sender_id?: string
  format?: "json" | "csv"
}): string {
  const params = new URLSearchParams()
  if (options?.start) {
    params.set("start", options.start)
  }
  if (options?.end) {
    params.set("end", options.end)
  }
  if (options?.channel) {
    params.set("channel", options.channel)
  }
  if (options?.sender_id) {
    params.set("sender_id", options.sender_id)
  }
  if (options?.format) {
    params.set("format", options.format)
  }
  const queryString = params.toString() ? `?${params.toString()}` : ""
  return `/api/logs/requests/export${queryString}`
}

export async function getRequestLogConfig(): Promise<RequestLogConfig> {
  return request<RequestLogConfig>("/api/config/requestlog")
}

export async function updateRequestLogConfig(config: Partial<RequestLogConfig>): Promise<RequestLogConfig> {
  const res = await fetch(`${BASE_URL}/api/config/requestlog`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<RequestLogConfig>
}

export async function archiveNow(): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/logs/requests/archive-now`, {
    method: "POST",
  })
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
}
