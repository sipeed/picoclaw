import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type OAuthFlowState,
  type OAuthProvider,
  type OAuthProviderStatus,
  getOAuthFlow,
  getOAuthProviders,
  loginOAuth,
  logoutOAuth,
  pollOAuthFlow,
} from "@/api/oauth"

type FlowWatchMode = "" | "status" | "poll"

function getProviderLabel(provider: OAuthProvider | ""): string {
  if (provider === "openai") return "OpenAI"
  if (provider === "anthropic") return "Anthropic"
  if (provider === "google-antigravity") return "Google Antigravity"
  return ""
}

export function useCredentialsPage() {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<OAuthProviderStatus[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const [activeAction, setActiveAction] = useState("")
  const [activeFlow, setActiveFlow] = useState<OAuthFlowState | null>(null)
  const actionTokenRef = useRef(0)

  const [watchFlowID, setWatchFlowID] = useState("")
  const [watchMode, setWatchMode] = useState<FlowWatchMode>("")
  const [pollIntervalMs, setPollIntervalMs] = useState(2000)

  const [openAIToken, setOpenAIToken] = useState("")
  const [anthropicToken, setAnthropicToken] = useState("")

  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false)
  const [logoutConfirmProvider, setLogoutConfirmProvider] = useState<
    OAuthProvider | ""
  >("")

  const [deviceSheetOpen, setDeviceSheetOpen] = useState(false)
  const [deviceFlow, setDeviceFlow] = useState<OAuthFlowState | null>(null)

  const loadProviders = useCallback(async () => {
    try {
      const data = await getOAuthProviders()
      setProviders(data.providers)
      setError("")
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("credentials.errors.loadFailed"),
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadProviders()
  }, [loadProviders])

  useEffect(() => {
    if (!watchFlowID || !watchMode) {
      return
    }

    let canceled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    const step = async () => {
      try {
        const flow =
          watchMode === "poll"
            ? await pollOAuthFlow(watchFlowID)
            : await getOAuthFlow(watchFlowID)

        if (canceled) {
          return
        }

        setActiveFlow(flow)
        setDeviceFlow((prev) =>
          prev?.flow_id === flow.flow_id ? { ...prev, ...flow } : prev,
        )

        if (flow.status === "pending") {
          timer = setTimeout(step, pollIntervalMs)
          return
        }

        if (watchMode === "poll") {
          setDeviceSheetOpen(false)
        }

        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        await loadProviders()
      } catch (err) {
        if (canceled) {
          return
        }
        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.flowFailed"),
        )
      }
    }

    void step()

    return () => {
      canceled = true
      if (timer) {
        clearTimeout(timer)
      }
    }
  }, [loadProviders, pollIntervalMs, t, watchFlowID, watchMode])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const flowID = params.get("oauth_flow_id")
    if (!flowID) {
      return
    }

    setWatchFlowID(flowID)
    setWatchMode("status")
    setPollIntervalMs(700)

    window.history.replaceState({}, "", window.location.pathname)
  }, [])

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const data = event.data as
        | { type?: string; flowId?: string; status?: string }
        | undefined
      if (!data || data.type !== "picoclaw-oauth-result" || !data.flowId) {
        return
      }

      setWatchFlowID(data.flowId)
      setWatchMode("status")
      setPollIntervalMs(700)
    }

    window.addEventListener("message", onMessage)
    return () => window.removeEventListener("message", onMessage)
  }, [])

  const providersMap = useMemo(() => {
    const map = new Map<OAuthProvider, OAuthProviderStatus>()
    for (const item of providers) {
      map.set(item.provider, item)
    }
    return map
  }, [providers])

  const openaiStatus = providersMap.get("openai")
  const anthropicStatus = providersMap.get("anthropic")
  const antigravityStatus = providersMap.get("google-antigravity")

  const bumpActionToken = useCallback(() => {
    actionTokenRef.current += 1
    return actionTokenRef.current
  }, [])

  const isActionTokenCurrent = useCallback((token: number) => {
    return actionTokenRef.current === token
  }, [])

  const startBrowserOAuth = useCallback(
    async (provider: OAuthProvider) => {
      const actionToken = bumpActionToken()
      setActiveAction(`${provider}:browser`)
      setError("")

      const authTab = window.open("", "_blank")
      if (!authTab) {
        if (!isActionTokenCurrent(actionToken)) {
          return
        }
        setActiveAction("")
        setError(t("credentials.errors.popupBlocked"))
        return
      }

      try {
        const resp = await loginOAuth({ provider, method: "browser" })
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        if (!resp.auth_url || !resp.flow_id) {
          throw new Error(t("credentials.errors.invalidBrowserResponse"))
        }

        authTab.location.href = resp.auth_url

        setActiveFlow({
          flow_id: resp.flow_id,
          provider,
          method: "browser",
          status: "pending",
          expires_at: resp.expires_at,
        })
        setWatchFlowID(resp.flow_id)
        setWatchMode("status")
        setPollIntervalMs(2000)
      } catch (err) {
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        authTab.close()
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      }
    },
    [bumpActionToken, isActionTokenCurrent, t],
  )

  const startOpenAIDeviceCode = useCallback(async () => {
    const actionToken = bumpActionToken()
    setActiveAction("openai:device")
    setError("")

    try {
      const resp = await loginOAuth({
        provider: "openai",
        method: "device_code",
      })
      if (!isActionTokenCurrent(actionToken)) {
        return
      }
      if (!resp.flow_id || !resp.user_code || !resp.verify_url) {
        throw new Error(t("credentials.errors.invalidDeviceResponse"))
      }

      const flow: OAuthFlowState = {
        flow_id: resp.flow_id,
        provider: "openai",
        method: "device_code",
        status: "pending",
        user_code: resp.user_code,
        verify_url: resp.verify_url,
        interval: resp.interval,
        expires_at: resp.expires_at,
      }

      setDeviceFlow(flow)
      setDeviceSheetOpen(true)
      setActiveFlow(flow)
      setWatchFlowID(resp.flow_id)
      setWatchMode("poll")
      setPollIntervalMs(Math.max(1000, (resp.interval ?? 5) * 1000))
    } catch (err) {
      if (!isActionTokenCurrent(actionToken)) {
        return
      }
      setActiveAction("")
      setError(
        err instanceof Error
          ? err.message
          : t("credentials.errors.loginFailed"),
      )
    }
  }, [bumpActionToken, isActionTokenCurrent, t])

  const saveToken = useCallback(
    async (provider: OAuthProvider, token: string) => {
      const actionID = `${provider}:token`
      setActiveAction(actionID)
      setError("")

      try {
        await loginOAuth({ provider, method: "token", token })
        if (provider === "openai") {
          setOpenAIToken("")
        }
        if (provider === "anthropic") {
          setAnthropicToken("")
        }
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, t],
  )

  const doLogout = useCallback(
    async (provider: OAuthProvider) => {
      const actionID = `${provider}:logout`
      setActiveAction(actionID)
      setError("")

      try {
        await logoutOAuth(provider)
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.logoutFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, t],
  )

  const askLogout = useCallback((provider: OAuthProvider) => {
    setLogoutConfirmProvider(provider)
    setLogoutDialogOpen(true)
  }, [])

  const handleConfirmLogout = useCallback(async () => {
    if (!logoutConfirmProvider) {
      return
    }
    await doLogout(logoutConfirmProvider)
    setLogoutDialogOpen(false)
    setLogoutConfirmProvider("")
  }, [doLogout, logoutConfirmProvider])

  const handleLogoutDialogOpenChange = useCallback((open: boolean) => {
    setLogoutDialogOpen(open)
    if (!open) {
      setLogoutConfirmProvider("")
    }
  }, [])

  const handleDeviceSheetOpenChange = useCallback(
    (open: boolean) => {
      setDeviceSheetOpen(open)
      if (open) {
        return
      }

      if (watchMode === "poll") {
        setWatchFlowID("")
        setWatchMode("")
        if (activeAction === "openai:device") {
          setActiveAction("")
        }
      }

      setDeviceFlow(null)
      if (
        activeFlow?.method === "device_code" &&
        activeFlow.status === "pending"
      ) {
        setActiveFlow(null)
      }
    },
    [activeAction, activeFlow, watchMode],
  )

  const stopLoading = useCallback(() => {
    bumpActionToken()
    setWatchFlowID("")
    setWatchMode("")
    setActiveAction("")
    setDeviceSheetOpen(false)
    setDeviceFlow(null)
    setActiveFlow((prev) => (prev?.status === "pending" ? null : prev))
  }, [bumpActionToken])

  const logoutProviderLabel = getProviderLabel(logoutConfirmProvider)

  const flowHint = useMemo(() => {
    if (!activeFlow) {
      return ""
    }
    if (activeFlow.status === "pending") {
      return t("credentials.flow.pending")
    }
    if (activeFlow.status === "success") {
      return t("credentials.flow.success")
    }
    if (activeFlow.status === "expired") {
      return t("credentials.flow.expired")
    }
    return activeFlow.error || t("credentials.flow.error")
  }, [activeFlow, t])

  return {
    loading,
    error,
    activeAction,
    activeFlow,
    flowHint,
    openAIToken,
    anthropicToken,
    openaiStatus,
    anthropicStatus,
    antigravityStatus,
    logoutDialogOpen,
    logoutConfirmProvider,
    logoutProviderLabel,
    deviceSheetOpen,
    deviceFlow,
    setOpenAIToken,
    setAnthropicToken,
    startBrowserOAuth,
    startOpenAIDeviceCode,
    stopLoading,
    saveToken,
    askLogout,
    handleConfirmLogout,
    handleLogoutDialogOpenChange,
    handleDeviceSheetOpenChange,
  }
}
