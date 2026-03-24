import { useAtom } from "jotai"
import { useCallback, useEffect } from "react"

import { authStatusAtom, isAuthenticatedAtom, needsLoginAtom, needsSetupAtom } from "./store"
import type { AuthStatus, ChangePasswordRequest, LoginRequest, SetupRequest } from "./types"

const API_BASE = "/api/auth"

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  })
  return response.json()
}

export function useAuth() {
  const [status, setStatus] = useAtom(authStatusAtom)
  const isAuthenticated = useAtom(isAuthenticatedAtom)[0]
  const needsLogin = useAtom(needsLoginAtom)[0]
  const needsSetup = useAtom(needsSetupAtom)[0]

  const checkStatus = useCallback(async () => {
    try {
      const result = await fetchJSON<AuthStatus>(`${API_BASE}/status`)
      setStatus(result)
      return result
    } catch {
      return status
    }
  }, [setStatus, status])

  const login = useCallback(
    async (username: string, password: string) => {
      const result = await fetchJSON<{ success: boolean; message?: string }>(`${API_BASE}/login`, {
        method: "POST",
        body: JSON.stringify({ username, password } as LoginRequest),
      })

      if (result.success) {
        await checkStatus()
      }

      return result
    },
    [checkStatus]
  )

  const logout = useCallback(async () => {
    await fetchJSON<{ success: boolean }>(`${API_BASE}/logout`, {
      method: "POST",
    })
    await checkStatus()
  }, [checkStatus])

  const setup = useCallback(
    async (username: string, password: string) => {
      const result = await fetchJSON<{ success: boolean; message?: string }>(`${API_BASE}/setup`, {
        method: "POST",
        body: JSON.stringify({ username, password } as SetupRequest),
      })

      if (result.success) {
        await checkStatus()
      }

      return result
    },
    [checkStatus]
  )

  const changePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    const result = await fetchJSON<{ success: boolean; error?: string }>(`${API_BASE}/change-password`, {
      method: "POST",
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      } as ChangePasswordRequest),
    })
    return result
  }, [])

  useEffect(() => {
    checkStatus()
  }, [checkStatus])

  return {
    status,
    isAuthenticated,
    needsLogin,
    needsSetup,
    checkStatus,
    login,
    logout,
    setup,
    changePassword,
  }
}
