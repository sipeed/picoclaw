/**
 * Dashboard launcher auth API.
 * Uses plain fetch (not launcherFetch) to avoid redirect loops on auth pages.
 */
export async function postLauncherDashboardLogin(
  password: string,
): Promise<boolean> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: JSON.stringify({ password: password.trim() }),
  })
  return res.ok
}

export type LauncherAuthStatus = {
  authenticated: boolean
  /** true when a bcrypt password has been stored in the DB */
  initialized: boolean
}

export async function getLauncherAuthStatus(): Promise<LauncherAuthStatus> {
  const res = await fetch("/api/auth/status", {
    method: "GET",
    credentials: "same-origin",
  })
  if (!res.ok) {
    throw new Error(`status ${res.status}`)
  }
  return (await res.json()) as LauncherAuthStatus
}

export async function postLauncherDashboardLogout(): Promise<boolean> {
  const res = await fetch("/api/auth/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: "{}",
  })
  return res.ok
}

export type SetupResult =
  | { ok: true }
  | { ok: false; error: string }

export async function postLauncherDashboardSetup(
  password: string,
  confirm: string,
): Promise<SetupResult> {
  const res = await fetch("/api/auth/setup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: JSON.stringify({ password: password.trim(), confirm: confirm.trim() }),
  })
  if (res.ok) return { ok: true }
  let msg = "Unknown error"
  try {
    const j = (await res.json()) as { error?: string }
    if (j.error) msg = j.error
  } catch {
    /* ignore */
  }
  return { ok: false, error: msg }
}
