export interface MediaCacheEntry {
  hash: string
  type: "image_desc" | "pdf_ocr"
  result: string
  file_path?: string
  pages?: number
  created_at: string
  accessed_at: string
}

export interface MediaCacheContent {
  hash: string
  type: string
  content: string
  file_path?: string
  pages?: number
}

async function request<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) {
    throw new Error(`API error: ${res.status}`)
  }
  return res.json() as Promise<T>
}

export async function getMediaCacheEntries(
  type?: string,
): Promise<MediaCacheEntry[]> {
  const params = type ? `?type=${encodeURIComponent(type)}` : ""
  return request<MediaCacheEntry[]>(`/api/media-cache${params}`)
}

export async function getMediaCacheContent(
  hash: string,
): Promise<MediaCacheContent> {
  return request<MediaCacheContent>(
    `/api/media-cache/${encodeURIComponent(hash)}`,
  )
}

export async function deleteMediaCacheEntry(hash: string): Promise<void> {
  const res = await fetch(`/api/media-cache/${encodeURIComponent(hash)}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
}

export async function deleteAllMediaCache(): Promise<{ deleted: number }> {
  const res = await fetch("/api/media-cache", { method: "DELETE" })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json() as Promise<{ deleted: number }>
}
