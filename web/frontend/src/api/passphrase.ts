// API client for in-memory passphrase management.

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(text || `API error: ${res.status}`)
  }
  return res.json() as Promise<T>
}

export interface PassphraseStatusResponse {
  passphrase_set: boolean
}

/** Returns whether a passphrase is currently held in the launcher. */
export async function getPassphraseStatus(): Promise<PassphraseStatusResponse> {
  return request<PassphraseStatusResponse>("/api/credential/passphrase/status")
}

/** Stores the passphrase in the launcher's in-memory SecureStore. */
export async function setPassphrase(passphrase: string): Promise<{ status: string }> {
  return request<{ status: string }>("/api/credential/passphrase", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ passphrase }),
  })
}
