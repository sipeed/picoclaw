export interface SkillSupportItem {
  name: string
  path: string
  source: "workspace" | "global" | "builtin" | string
  description: string
}

export interface SkillDetailResponse extends SkillSupportItem {
  content: string
}

interface SkillsResponse {
  skills: SkillSupportItem[]
}

interface SkillActionResponse {
  status?: string
  name?: string
  path?: string
  source?: string
  description?: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    throw new Error(await extractErrorMessage(res))
  }
  return res.json() as Promise<T>
}

export async function getSkills(): Promise<SkillsResponse> {
  return request<SkillsResponse>("/api/skills")
}

export async function getSkill(name: string): Promise<SkillDetailResponse> {
  return request<SkillDetailResponse>(`/api/skills/${encodeURIComponent(name)}`)
}

export async function importSkill(file: File): Promise<SkillActionResponse> {
  const formData = new FormData()
  formData.set("file", file)

  const res = await fetch("/api/skills/import", {
    method: "POST",
    body: formData,
  })
  if (!res.ok) {
    throw new Error(await extractErrorMessage(res))
  }
  return res.json() as Promise<SkillActionResponse>
}

export async function deleteSkill(name: string): Promise<SkillActionResponse> {
  return request<SkillActionResponse>(
    `/api/skills/${encodeURIComponent(name)}`,
    {
      method: "DELETE",
    },
  )
}

async function extractErrorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as {
      error?: string
      errors?: string[]
    }
    if (Array.isArray(body.errors) && body.errors.length > 0) {
      return body.errors.join("; ")
    }
    if (typeof body.error === "string" && body.error.trim() !== "") {
      return body.error
    }
  } catch {
    // ignore invalid body
  }
  return `API error: ${res.status} ${res.statusText}`
}
