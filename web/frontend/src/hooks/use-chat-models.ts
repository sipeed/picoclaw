import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ModelInfo, getModels, setDefaultModel } from "@/api/models"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

interface UseChatModelsOptions {
  isConnected: boolean
}

function isLocalModel(model: ModelInfo): boolean {
  const isLocalHostBase = Boolean(
    model.api_base?.includes("localhost") ||
    model.api_base?.includes("127.0.0.1"),
  )

  return (
    model.auth_method === "local" || (!model.auth_method && isLocalHostBase)
  )
}

export function useChatModels({ isConnected }: UseChatModelsOptions) {
  const { t } = useTranslation()
  const [modelList, setModelList] = useState<ModelInfo[]>([])
  const [defaultModelName, setDefaultModelName] = useState("")
  const setDefaultRequestIdRef = useRef(0)

  const loadModels = useCallback(async () => {
    try {
      const data = await getModels()
      setModelList(data.models)
      if (data.models.some((m) => m.model_name === data.default_model)) {
        setDefaultModelName(data.default_model)
      }
    } catch {
      // silently fail
    }
  }, [])

  useEffect(() => {
    const timerId = setTimeout(() => {
      void loadModels()
    }, 0)

    return () => clearTimeout(timerId)
  }, [isConnected, loadModels])

  const handleSetDefault = useCallback(
    async (modelName: string) => {
      if (modelName === defaultModelName) return
      const requestId = ++setDefaultRequestIdRef.current

      try {
        await setDefaultModel(modelName)
        const data = await getModels()
        if (requestId !== setDefaultRequestIdRef.current) {
          return
        }

        setModelList(data.models)
        if (data.models.some((m) => m.model_name === data.default_model)) {
          setDefaultModelName(data.default_model)
        }
        const gateway = await refreshGatewayState({ force: true })
        showSaveSuccessOrRestartToast(
          t,
          t("models.defaultChangeSuccess"),
          modelName,
          gateway?.restartRequired === true,
        )
      } catch (err) {
        console.error("Failed to set default model:", err)
        toast.error(err instanceof Error ? err.message : t("models.loadError"))
      }
    },
    [defaultModelName, t],
  )

  const hasAvailableModels = useMemo(
    () => modelList.some((m) => m.available),
    [modelList],
  )

  const oauthModels = useMemo(
    () => modelList.filter((m) => m.available && m.auth_method === "oauth"),
    [modelList],
  )

  const localModels = useMemo(
    () => modelList.filter((m) => m.available && isLocalModel(m)),
    [modelList],
  )

  const apiKeyModels = useMemo(
    () =>
      modelList.filter(
        (m) => m.available && m.auth_method !== "oauth" && !isLocalModel(m),
      ),
    [modelList],
  )

  return {
    defaultModelName,
    hasAvailableModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
  }
}
