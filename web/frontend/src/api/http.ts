import { isLauncherLoginPathname } from "@/lib/launcher-login-path"

function isLauncherLoginPath(): boolean {
  if (typeof globalThis.location === "undefined") {
    return false
  }
  if (isLauncherLoginPathname(globalThis.location.pathname || "/")) {
    return true
  }
  try {
    return isLauncherLoginPathname(
      new URL(globalThis.location.href).pathname || "/",
    )
  } catch {
    return false
  }
}

/**
 * Same-origin fetch that sends cookies; redirects to launcher login on 401 JSON responses.
 * Skips redirect while already on the login page to avoid reload loops (e.g. gateway poll).
 */
export async function launcherFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const res = await fetch(input, {
    credentials: "same-origin",
    ...init,
  })
  if (res.status === 401) {
    const ct = res.headers.get("content-type") || ""
    if (
      ct.includes("application/json") &&
      typeof globalThis.location !== "undefined" &&
      !isLauncherLoginPath()
    ) {
      globalThis.location.assign("/launcher-login")
    }
  }
  return res
}
