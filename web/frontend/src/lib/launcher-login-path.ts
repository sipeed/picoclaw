/** Normalize URL pathname for comparisons (trailing slashes, empty). */
export function normalizePathname(p: string): string {
  const t = p.replace(/\/+$/, "")
  return t === "" ? "/" : t
}

export function isLauncherLoginPathname(pathname: string): boolean {
  return normalizePathname(pathname) === "/launcher-login"
}
