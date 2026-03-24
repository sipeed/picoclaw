import { atom } from "jotai"

import type { AuthStatus } from "./types"

export const authStatusAtom = atom<AuthStatus>({
  enabled: false,
  configured: false,
  logged_in: false,
})

export const isAuthenticatedAtom = atom((get) => {
  const status = get(authStatusAtom)
  if (!status.enabled) return true
  return status.logged_in
})

export const needsSetupAtom = atom((get) => {
  const status = get(authStatusAtom)
  return status.enabled && !status.configured
})

export const needsLoginAtom = atom((get) => {
  const status = get(authStatusAtom)
  return status.enabled && status.configured && !status.logged_in
})
