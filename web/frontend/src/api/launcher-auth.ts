/**
 * Dashboard launcher token login. Uses plain fetch (not launcherFetch) to avoid
 * redirect loops on 401 while on the login page.
 */
export async function postLauncherDashboardLogin(
  token: string,
): Promise<boolean> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: JSON.stringify({ token: token.trim() }),
  })
  return res.ok
}

export type LauncherAuthTokenHelp = {
  env_var_name: string
  log_file?: string
  config_file?: string
  tray_copy_menu: boolean
  console_stdout: boolean
}

export type LauncherAuthStatus = {
  authenticated: boolean
  token_help?: LauncherAuthTokenHelp
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
