import { isLauncherAuthPathname } from "@/lib/launcher-login-path"

function isLauncherAuthPath(): boolean {
  if (typeof globalThis.location === "undefined") {
    return false
  }
  if (isLauncherAuthPathname(globalThis.location.pathname || "/")) {
    return true
  }
  try {
    return isLauncherAuthPathname(
      new URL(globalThis.location.href).pathname || "/",
    )
  } catch {
    return false
  }
}

/**
 * Same-origin fetch that sends cookies; redirects to launcher login on 401 JSON responses.
 * Skips redirect while already on an auth page (login or setup) to avoid reload loops.
 * Adds X-Requested-With header on mutating requests for CSRF protection.
 */
export async function launcherFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const method = init?.method?.toUpperCase() || (typeof input === "string" || input instanceof URL ? "GET" : input instanceof Request ? input.method : "GET")
  const isMutating = method === "POST" || method === "PUT" || method === "DELETE"
  const headers = new Headers(init?.headers)
  if (isMutating) {
    headers.set("X-Requested-With", "XMLHttpRequest")
  }
  const res = await fetch(input, {
    credentials: "same-origin",
    ...init,
    headers,
  })
  if (res.status === 401) {
    const ct = res.headers.get("content-type") || ""
    if (
      ct.includes("application/json") &&
      typeof globalThis.location !== "undefined" &&
      !isLauncherAuthPath()
    ) {
      globalThis.location.assign("/launcher-login")
    }
  }
  return res
}
