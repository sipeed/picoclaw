export interface ResearchTask {
  id: string
  title: string
  slug: string
  description: string
  status: "pending" | "active" | "completed" | "failed" | "canceled"
  output_dir: string
  created_at: string
  updated_at: string
  completed_at?: string
  document_count: number
}

export interface ResearchDocument {
  id: string
  task_id: string
  title: string
  file_path: string
  doc_type: "finding" | "summary" | "note"
  seq: number
  summary: string
  created_at: string
}

export interface ResearchTaskDetail extends ResearchTask {
  documents: ResearchDocument[]
}

export interface ResearchDocContent {
  id: string
  title: string
  doc_type: string
  content: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: string }
      if (typeof body.error === "string" && body.error.trim() !== "") {
        message = body.error
      }
    } catch {
      // ignore
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function getResearchTasks(
  status?: string,
): Promise<ResearchTask[]> {
  const params = status ? `?status=${encodeURIComponent(status)}` : ""
  return request<ResearchTask[]>(`/api/research${params}`)
}

export async function getResearchTask(
  id: string,
): Promise<ResearchTaskDetail> {
  return request<ResearchTaskDetail>(
    `/api/research/${encodeURIComponent(id)}`,
  )
}

export async function createResearchTask(
  title: string,
  description: string,
): Promise<ResearchTask> {
  return request<ResearchTask>("/api/research", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, description }),
  })
}

export async function researchTaskAction(
  id: string,
  action: "cancel" | "reopen" | "update",
  data?: { title?: string; description?: string },
): Promise<ResearchTaskDetail> {
  return request<ResearchTaskDetail>(
    `/api/research/${encodeURIComponent(id)}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, ...data }),
    },
  )
}

export async function getResearchDocContent(
  taskId: string,
  docId: string,
): Promise<ResearchDocContent> {
  return request<ResearchDocContent>(
    `/api/research/${encodeURIComponent(taskId)}/doc/${encodeURIComponent(docId)}`,
  )
}
