import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { type ModelInfo, getModels, setDefaultModel } from "@/api/models"

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
      } catch (err) {
        console.error("Failed to set default model:", err)
      }
    },
    [defaultModelName],
  )

  const hasConfiguredModels = useMemo(
    () => modelList.some((m) => m.configured),
    [modelList],
  )

  const oauthModels = useMemo(
    () => modelList.filter((m) => m.configured && m.auth_method === "oauth"),
    [modelList],
  )

  const localModels = useMemo(
    () => modelList.filter((m) => m.configured && isLocalModel(m)),
    [modelList],
  )

  const apiKeyModels = useMemo(
    () =>
      modelList.filter(
        (m) => m.configured && m.auth_method !== "oauth" && !isLocalModel(m),
      ),
    [modelList],
  )

  return {
    defaultModelName,
    hasConfiguredModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
  }
}
