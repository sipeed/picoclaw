import {
  IconArrowsSort,
  IconDatabase,
  IconLoader2,
  IconPlus,
  IconStar,
} from "@tabler/icons-react"
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ModelInfo,
  type ModelProviderOption,
  type ModelReferenceRename,
  getModels,
  updateDefaultChain,
} from "@/api/models"
import { ConfigChangeNotice } from "@/components/config-change-notice"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

import { AddModelSheet } from "./add-model-sheet"
import { CatalogDialog } from "./catalog-dialog"
import { DefaultChainDialog } from "./default-chain-dialog"
import { DeleteModelDialog } from "./delete-model-dialog"
import { EditModelSheet } from "./edit-model-sheet"
import {
  getCanonicalProviderKey,
  getProviderCatalogMap,
  getProviderDefaultAPIBase,
} from "./provider-registry"
import type { ProviderCatalogEntry } from "./provider-registry"
import { ProviderSection } from "./provider-section"

interface ProviderGroup {
  key: string
  provider: Pick<ProviderCatalogEntry, "key" | "label" | "iconSlug" | "domain">
  models: ModelInfo[]
  hasDefault: boolean
  availableCount: number
}

function modelEntryAllowedInDefaultChain(model: ModelInfo) {
  return (
    !model.is_virtual &&
    model.default_model_allowed !== false &&
    model.status !== "unconfigured"
  )
}

function sortModelList(models: ModelInfo[], defaultModelName: string) {
  return [...models].sort((a, b) => {
    const aDefault = a.model_name === defaultModelName
    const bDefault = b.model_name === defaultModelName
    if (aDefault && !bDefault) return -1
    if (!aDefault && bDefault) return 1
    if (a.available && !b.available) return -1
    if (!a.available && b.available) return 1
    return a.model_name.localeCompare(b.model_name)
  })
}

function modelNameAllowedInDefaultChain(
  modelName: string,
  models: ModelInfo[],
) {
  const matchingModels = models.filter(
    (model) => model.model_name === modelName,
  )
  return (
    matchingModels.length > 0 &&
    matchingModels.every(modelEntryAllowedInDefaultChain)
  )
}

function modelReferenceAllowedAfterModelChange(
  modelName: string,
  previousModels: ModelInfo[],
  models: ModelInfo[],
  providerOptions: ModelProviderOption[],
  defaultProvider: string,
) {
  const matchingModels = models.filter(
    (model) => model.model_name === modelName,
  )
  if (matchingModels.length > 0) {
    return matchingModels.every(modelEntryAllowedInDefaultChain)
  }

  if (
    previousModels.some((model) => model.model_name === modelName) ||
    modelName.trim() === ""
  ) {
    return false
  }

  const raw = modelName.trim()
  const slashIndex = raw.indexOf("/")
  const prefix = slashIndex > 0 ? raw.slice(0, slashIndex) : ""
  const explicitProvider = providerOptions.find(
    (option) =>
      getCanonicalProviderKey(option.id, providerOptions) ===
      getCanonicalProviderKey(prefix, providerOptions),
  )
  if (explicitProvider) {
    const exactLegacyMatches = models.filter(
      (model) => !model.is_virtual && model.model.trim() === raw,
    )
    if (exactLegacyMatches.length > 0) {
      const selected = selectBestRawModelMatch(
        exactLegacyMatches,
        providerOptions,
      )
      return (
        selected.default_model_allowed !== false &&
        modelInfoCanConfigureRawTemplate(selected, providerOptions)
      )
    }
  }
  const provider = explicitProvider ? prefix : defaultProvider || "openai"
  const modelId = explicitProvider ? raw.slice(slashIndex + 1).trim() : raw
  if (!modelId) return false

  const providerKey = getCanonicalProviderKey(provider, providerOptions)
  const matchingRawModels = models.filter(
    (model) =>
      !model.is_virtual &&
      getCanonicalProviderKey(model.provider, providerOptions) ===
        providerKey &&
      model.model.trim() === modelId,
  )
  if (matchingRawModels.length > 0) {
    const selected = selectBestRawModelMatch(matchingRawModels, providerOptions)
    return (
      selected.default_model_allowed !== false &&
      modelInfoCanConfigureRawTemplate(selected, providerOptions)
    )
  }

  const providerAllowed = providerOptions.some(
    (option) =>
      getCanonicalProviderKey(option.id, providerOptions) === providerKey &&
      option.default_model_allowed !== false,
  )
  return (
    providerAllowed &&
    models.some(
      (model) =>
        !model.is_virtual &&
        getCanonicalProviderKey(model.provider, providerOptions) ===
          providerKey &&
        modelInfoCanConfigureRawTemplate(model, providerOptions),
    )
  )
}

function modelInfoCanConfigureRawTemplate(
  model: ModelInfo,
  providerOptions: ModelProviderOption[],
) {
  if (model.status !== "unconfigured") return true
  const authMethod = model.auth_method?.trim().toLowerCase()
  if (authMethod === "oauth" || authMethod === "token") return false
  if (!model.api_base?.trim()) return false
  const provider = getCanonicalProviderKey(model.provider, providerOptions)
  if (
    ["anthropic", "anthropic-messages", "alibaba-coding-anthropic"].includes(
      provider,
    )
  ) {
    return false
  }
  const normalizeBase = (value: string) =>
    value.trim().replace(/\/+$/, "").toLowerCase()
  const defaultBase = getProviderDefaultAPIBase(provider, providerOptions)
  return (
    !defaultBase || normalizeBase(model.api_base) !== normalizeBase(defaultBase)
  )
}

function rawModelTemplateScore(
  model: ModelInfo,
  providerOptions: ModelProviderOption[],
) {
  let score = 0
  if (model.api_key?.trim()) score += 32
  const authMethod = model.auth_method?.trim().toLowerCase()
  const provider = getCanonicalProviderKey(model.provider, providerOptions)
  const normalizeBase = (value: string) =>
    value.trim().replace(/\/+$/, "").toLowerCase()
  const defaultBase = getProviderDefaultAPIBase(provider, providerOptions)
  if (
    authMethod !== "oauth" &&
    authMethod !== "token" &&
    model.api_base?.trim() &&
    (!defaultBase ||
      normalizeBase(model.api_base) !== normalizeBase(defaultBase))
  ) {
    score += 16
  }
  if (model.auth_method?.trim() || model.connect_mode?.trim()) score += 8
  if (model.enabled) score += 1
  return score
}

function selectBestRawModelMatch(
  models: ModelInfo[],
  providerOptions: ModelProviderOption[],
) {
  return [...models].sort(
    (a, b) =>
      rawModelTemplateScore(b, providerOptions) -
        rawModelTemplateScore(a, providerOptions) || a.index - b.index,
  )[0]
}

function modelReferenceIdentity(
  modelName: string,
  models: ModelInfo[],
  providerOptions: ModelProviderOption[],
  defaultProvider: string,
) {
  const aliasMatches = models.filter(
    (model) => model.model_name === modelName.trim(),
  )
  if (aliasMatches.length > 0) return `model_name:${modelName.trim()}`

  const raw = modelName.trim()
  if (!raw) return ""
  const slashIndex = raw.indexOf("/")
  const prefix = slashIndex > 0 ? raw.slice(0, slashIndex) : ""
  const explicitProvider = providerOptions.find(
    (option) =>
      getCanonicalProviderKey(option.id, providerOptions) ===
      getCanonicalProviderKey(prefix, providerOptions),
  )
  if (explicitProvider) {
    const exactLegacyMatches = models.filter(
      (model) => !model.is_virtual && model.model.trim() === raw,
    )
    const exactLegacyMatch = exactLegacyMatches.length
      ? selectBestRawModelMatch(exactLegacyMatches, providerOptions)
      : undefined
    if (exactLegacyMatch) {
      return `model_name:${exactLegacyMatch.model_name}`
    }
  }
  const provider = explicitProvider ? prefix : defaultProvider || "openai"
  const modelId = explicitProvider ? raw.slice(slashIndex + 1).trim() : raw
  const providerKey = getCanonicalProviderKey(provider, providerOptions)
  const configuredMatches = models.filter(
    (model) =>
      !model.is_virtual &&
      getCanonicalProviderKey(model.provider, providerOptions) ===
        providerKey &&
      model.model.trim() === modelId,
  )
  const configuredMatch = configuredMatches.length
    ? selectBestRawModelMatch(configuredMatches, providerOptions)
    : undefined
  if (configuredMatch) return `model_name:${configuredMatch.model_name}`
  return `${providerKey}/${modelId}`
}

function applyReferenceRename(
  modelName: string,
  renames: ModelReferenceRename[],
  role: "default" | "fallback",
) {
  let renamedModelName = modelName
  for (const rename of renames) {
    if (renamedModelName === rename.from && rename[role]) {
      renamedModelName = rename.to
    }
  }
  return renamedModelName
}

function applyReferenceRenames(
  names: string[],
  renames: ModelReferenceRename[],
) {
  const renamed: string[] = []
  for (const modelName of names) {
    const nextModelName = applyReferenceRename(modelName, renames, "fallback")
    if (!renamed.includes(nextModelName)) renamed.push(nextModelName)
  }
  return renamed
}

function deriveModelUpdateRenames(
  updates: {
    index: number
    from: string
    stableAliases: { index: number; modelName: string }[]
  }[],
  models: ModelInfo[],
): ModelReferenceRename[] {
  const aliases = new Set(models.map((model) => model.model_name))
  return updates.flatMap((update) => {
    const updated = models.find((model) => model.index === update.index)
    const indexesStayedStable = update.stableAliases.every((stable) =>
      models.some(
        (model) =>
          model.index === stable.index && model.model_name === stable.modelName,
      ),
    )
    if (
      !updated ||
      !indexesStayedStable ||
      updated.model_name === update.from ||
      aliases.has(update.from)
    ) {
      return []
    }
    return [
      {
        from: update.from,
        to: updated.model_name,
        default: true,
        fallback: true,
      },
    ]
  })
}

function insertByPersistedOrder(
  result: string[],
  modelName: string,
  persistedChain: string[],
) {
  const persistedIndex = persistedChain.indexOf(modelName)
  for (let index = persistedIndex + 1; index < persistedChain.length; index++) {
    const nextIndex = result.indexOf(persistedChain[index])
    if (nextIndex !== -1) {
      result.splice(nextIndex, 0, modelName)
      return
    }
  }
  for (let index = persistedIndex - 1; index >= 0; index--) {
    const previousIndex = result.indexOf(persistedChain[index])
    if (previousIndex !== -1) {
      result.splice(previousIndex + 1, 0, modelName)
      return
    }
  }
  result.push(modelName)
}

function rebaseDefaultChainDraft(
  previousBaseline: { defaultModelName: string; fallbackChain: string[] },
  draft: { defaultModelName: string; fallbackChain: string[] },
  persisted: { defaultModelName: string; fallbackChain: string[] },
  previousModels: ModelInfo[],
  models: ModelInfo[],
  referenceRenames: ModelReferenceRename[],
  providerOptions: ModelProviderOption[],
  defaultProvider: string,
) {
  const previousDefaultModelName = applyReferenceRename(
    previousBaseline.defaultModelName,
    referenceRenames,
    "default",
  )
  const draftDefaultModelName = applyReferenceRename(
    draft.defaultModelName,
    referenceRenames,
    "default",
  )
  const defaultModelChanged = draftDefaultModelName !== previousDefaultModelName
  const keepDraftDefault =
    defaultModelChanged &&
    modelReferenceAllowedAfterModelChange(
      draftDefaultModelName,
      previousModels,
      models,
      providerOptions,
      defaultProvider,
    )
  const defaultModelName = keepDraftDefault
    ? draftDefaultModelName
    : persisted.defaultModelName
  if (!defaultModelName) {
    return { defaultModelName: "", fallbackChain: [] }
  }

  const previousFallbackChain = applyReferenceRenames(
    previousBaseline.fallbackChain,
    referenceRenames,
  )
  const draftFallbackChain = applyReferenceRenames(
    draft.fallbackChain,
    referenceRenames,
  )
  const previousBaselineNames = new Set(previousFallbackChain)
  const persistedNames = new Set(persisted.fallbackChain)
  const userRemovedNames = new Set(
    previousFallbackChain.filter(
      (modelName) => !draftFallbackChain.includes(modelName),
    ),
  )
  const seenFallbackIdentities = new Set<string>()
  const fallbackChain = draftFallbackChain.filter((modelName, index, names) => {
    const identity = modelReferenceIdentity(
      modelName,
      models,
      providerOptions,
      defaultProvider,
    )
    if (identity && seenFallbackIdentities.has(identity)) return false
    const keep =
      modelName !== defaultModelName &&
      identity !==
        modelReferenceIdentity(
          defaultModelName,
          models,
          providerOptions,
          defaultProvider,
        ) &&
      names.indexOf(modelName) === index &&
      (keepDraftDefault ||
        persistedNames.has(modelName) ||
        !previousBaselineNames.has(modelName)) &&
      modelReferenceAllowedAfterModelChange(
        modelName,
        previousModels,
        models,
        providerOptions,
        defaultProvider,
      )
    if (keep && identity) seenFallbackIdentities.add(identity)
    return keep
  })

  for (const modelName of persisted.fallbackChain) {
    const identity = modelReferenceIdentity(
      modelName,
      models,
      providerOptions,
      defaultProvider,
    )
    if (
      modelName === defaultModelName ||
      fallbackChain.includes(modelName) ||
      userRemovedNames.has(modelName) ||
      (identity && seenFallbackIdentities.has(identity))
    ) {
      continue
    }
    insertByPersistedOrder(fallbackChain, modelName, persisted.fallbackChain)
    if (identity) seenFallbackIdentities.add(identity)
  }

  return { defaultModelName, fallbackChain }
}

export function ModelsPage() {
  const { t } = useTranslation()
  const [models, setModels] = useState<ModelInfo[]>([])
  const [providerOptions, setProviderOptions] = useState<ModelProviderOption[]>(
    [],
  )
  const [defaultProvider, setDefaultProvider] = useState("openai")
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState("")

  const [editingModel, setEditingModel] = useState<ModelInfo | null>(null)
  const [deletingModel, setDeletingModel] = useState<ModelInfo | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [catalogOpen, setCatalogOpen] = useState(false)
  const [defaultChainOpen, setDefaultChainOpen] = useState(false)
  const [defaultModelDraft, setDefaultModelDraft] = useState("")
  const [defaultModelBaseline, setDefaultModelBaseline] = useState("")
  const [fallbackChainDraft, setFallbackChainDraft] = useState<string[]>([])
  const [fallbackChainBaseline, setFallbackChainBaseline] = useState<string[]>(
    [],
  )
  const [savingChain, setSavingChain] = useState(false)
  const [modelUpdatePending, setModelUpdatePending] = useState(false)
  const refreshRequestSequence = useRef(0)
  const savingChainRef = useRef(false)
  const refreshQueuedDuringSave = useRef(false)
  const modelUpdateInFlight = useRef(0)
  const modelReconcileInFlight = useRef(false)
  const modelReconcileSequence = useRef(0)
  const pendingReferenceRenames = useRef<ModelReferenceRename[]>([])
  const pendingModelUpdates = useRef<
    {
      index: number
      from: string
      stableAliases: { index: number; modelName: string }[]
    }[]
  >([])
  const providerMap = getProviderCatalogMap(providerOptions)
  const chainDirty =
    defaultModelDraft !== defaultModelBaseline ||
    JSON.stringify(fallbackChainDraft) !== JSON.stringify(fallbackChainBaseline)
  const latestEditorState = useRef({
    defaultModelDraft,
    defaultModelBaseline,
    fallbackChainDraft,
    fallbackChainBaseline,
    models,
    providerOptions,
    defaultProvider,
    chainDirty,
  })
  useLayoutEffect(() => {
    latestEditorState.current = {
      defaultModelDraft,
      defaultModelBaseline,
      fallbackChainDraft,
      fallbackChainBaseline,
      models,
      providerOptions,
      defaultProvider,
      chainDirty,
    }
  }, [
    chainDirty,
    defaultModelBaseline,
    defaultModelDraft,
    fallbackChainBaseline,
    fallbackChainDraft,
    models,
    providerOptions,
    defaultProvider,
  ])

  const fetchModels = useCallback(async () => {
    if (modelReconcileInFlight.current) {
      return
    }
    if (savingChainRef.current || modelUpdateInFlight.current > 0) {
      refreshQueuedDuringSave.current = true
      return
    }
    const requestSequence = ++refreshRequestSequence.current
    setLoading(true)
    try {
      const data = await getModels()
      if (requestSequence !== refreshRequestSequence.current) return
      const persisted = {
        defaultModelName: data.default_model || "",
        fallbackChain: data.fallback_chain || [],
      }
      const latest = latestEditorState.current
      const renameCount = pendingReferenceRenames.current.length
      const modelUpdateCount = pendingModelUpdates.current.length
      const referenceRenames = [
        ...pendingReferenceRenames.current.slice(0, renameCount),
        ...deriveModelUpdateRenames(
          pendingModelUpdates.current.slice(0, modelUpdateCount),
          data.models,
        ),
      ]
      const rebased = latest.chainDirty
        ? rebaseDefaultChainDraft(
            {
              defaultModelName: latest.defaultModelBaseline,
              fallbackChain: latest.fallbackChainBaseline,
            },
            {
              defaultModelName: latest.defaultModelDraft,
              fallbackChain: latest.fallbackChainDraft,
            },
            persisted,
            latest.models,
            data.models,
            referenceRenames,
            data.provider_options || [],
            data.default_provider || "openai",
          )
        : persisted
      pendingReferenceRenames.current.splice(0, renameCount)
      pendingModelUpdates.current.splice(0, modelUpdateCount)
      setModels(sortModelList(data.models, rebased.defaultModelName))
      setProviderOptions(data.provider_options || [])
      setDefaultProvider(data.default_provider || "openai")
      setDefaultModelDraft(rebased.defaultModelName)
      setDefaultModelBaseline(persisted.defaultModelName)
      setFallbackChainDraft(rebased.fallbackChain)
      setFallbackChainBaseline(persisted.fallbackChain)
      setFetchError("")
    } catch (e) {
      if (requestSequence !== refreshRequestSequence.current) return
      setFetchError(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      if (requestSequence === refreshRequestSequence.current) {
        setLoading(false)
      }
    }
  }, [t])

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  const refreshAfterModelChange = useCallback(async () => {
    if (savingChainRef.current || modelUpdateInFlight.current > 0) {
      refreshQueuedDuringSave.current = true
      return
    }
    const requestSequence = ++refreshRequestSequence.current
    setLoading(true)
    try {
      const data = await getModels()
      if (requestSequence !== refreshRequestSequence.current) return
      const persisted = {
        defaultModelName: data.default_model || "",
        fallbackChain: data.fallback_chain || [],
      }
      const latest = latestEditorState.current
      const renameCount = pendingReferenceRenames.current.length
      const modelUpdateCount = pendingModelUpdates.current.length
      const referenceRenames = [
        ...pendingReferenceRenames.current.slice(0, renameCount),
        ...deriveModelUpdateRenames(
          pendingModelUpdates.current.slice(0, modelUpdateCount),
          data.models,
        ),
      ]
      const rebased = latest.chainDirty
        ? rebaseDefaultChainDraft(
            {
              defaultModelName: latest.defaultModelBaseline,
              fallbackChain: latest.fallbackChainBaseline,
            },
            {
              defaultModelName: latest.defaultModelDraft,
              fallbackChain: latest.fallbackChainDraft,
            },
            persisted,
            latest.models,
            data.models,
            referenceRenames,
            data.provider_options || [],
            data.default_provider || "openai",
          )
        : persisted
      pendingReferenceRenames.current.splice(0, renameCount)
      pendingModelUpdates.current.splice(0, modelUpdateCount)
      setModels(sortModelList(data.models, rebased.defaultModelName))
      setProviderOptions(data.provider_options || [])
      setDefaultProvider(data.default_provider || "openai")
      setDefaultModelDraft(rebased.defaultModelName)
      setDefaultModelBaseline(persisted.defaultModelName)
      setFallbackChainDraft(rebased.fallbackChain)
      setFallbackChainBaseline(persisted.fallbackChain)
      setFetchError("")
    } catch (e) {
      if (requestSequence !== refreshRequestSequence.current) return
      setFetchError(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      if (requestSequence === refreshRequestSequence.current) {
        setLoading(false)
      }
    }
  }, [t])

  const beginModelMutation = useCallback(() => {
    modelUpdateInFlight.current += 1
    ++modelReconcileSequence.current
    setModelUpdatePending(true)
    refreshQueuedDuringSave.current = true
    ++refreshRequestSequence.current
  }, [])

  const finishModelMutation = useCallback(() => {
    modelUpdateInFlight.current = Math.max(0, modelUpdateInFlight.current - 1)
    if (modelUpdateInFlight.current !== 0) return

    refreshQueuedDuringSave.current = false
    modelReconcileInFlight.current = true
    const reconcileSequence = ++modelReconcileSequence.current
    void refreshAfterModelChange().finally(() => {
      if (reconcileSequence !== modelReconcileSequence.current) return
      modelReconcileInFlight.current = false
      if (modelUpdateInFlight.current === 0) {
        setModelUpdatePending(false)
      }
    })
  }, [refreshAfterModelChange])

  useEffect(() => {
    if (savingChain || !refreshQueuedDuringSave.current) return
    refreshQueuedDuringSave.current = false
    void refreshAfterModelChange()
  }, [refreshAfterModelChange, savingChain])

  const handleSetDefault = (model: ModelInfo) => {
    if (
      !model.available ||
      !modelNameAllowedInDefaultChain(model.model_name, models)
    ) {
      return
    }
    if (defaultModelDraft === model.model_name) return
    setDefaultModelDraft(model.model_name)
    setModels((prev) => sortModelList(prev, model.model_name))
    const defaultIdentity = modelReferenceIdentity(
      model.model_name,
      models,
      providerOptions,
      defaultProvider,
    )
    setFallbackChainDraft((prev) =>
      prev.filter(
        (item) =>
          item !== model.model_name &&
          modelReferenceIdentity(
            item,
            models,
            providerOptions,
            defaultProvider,
          ) !== defaultIdentity,
      ),
    )
  }

  const handleToggleFallback = (model: ModelInfo) => {
    const defaultIdentity = modelReferenceIdentity(
      defaultModelDraft,
      models,
      providerOptions,
      defaultProvider,
    )
    if (
      model.model_name === defaultModelDraft ||
      modelReferenceIdentity(
        model.model_name,
        models,
        providerOptions,
        defaultProvider,
      ) === defaultIdentity ||
      !model.available ||
      !modelNameAllowedInDefaultChain(model.model_name, models)
    ) {
      return
    }
    setFallbackChainDraft((prev) =>
      prev.includes(model.model_name)
        ? prev.filter((item) => item !== model.model_name)
        : prev.some(
              (item) =>
                modelReferenceIdentity(
                  item,
                  models,
                  providerOptions,
                  defaultProvider,
                ) ===
                modelReferenceIdentity(
                  model.model_name,
                  models,
                  providerOptions,
                  defaultProvider,
                ),
            )
          ? prev
          : [...prev, model.model_name],
    )
  }

  const moveFallback = (modelName: string, targetIndex: number) => {
    setFallbackChainDraft((prev) => {
      const currentIndex = prev.indexOf(modelName)
      if (currentIndex === -1) {
        return prev
      }

      const next = [...prev]
      next.splice(currentIndex, 1)
      next.splice(targetIndex, 0, modelName)
      return next
    })
  }

  const handleSaveChain = async () => {
    if (modelUpdateInFlight.current > 0 || modelReconcileInFlight.current) {
      return
    }
    // Invalidate refreshes that captured the pre-save chain. Refreshes requested
    // while saving are queued and restarted after the saved draft is reconciled.
    savingChainRef.current = true
    refreshQueuedDuringSave.current = true
    ++refreshRequestSequence.current
    const submitted = {
      defaultModelName: defaultModelDraft,
      fallbackChain: fallbackChainDraft,
    }
    setSavingChain(true)
    try {
      const saved = await updateDefaultChain({
        default_model: submitted.defaultModelName,
        fallback_chain: submitted.fallbackChain,
      })
      const persisted = {
        defaultModelName: saved.default_model || "",
        fallbackChain: saved.fallback_chain || [],
      }
      const latest = latestEditorState.current
      const rebased = rebaseDefaultChainDraft(
        submitted,
        {
          defaultModelName: latest.defaultModelDraft,
          fallbackChain: latest.fallbackChainDraft,
        },
        persisted,
        latest.models,
        latest.models,
        [],
        latest.providerOptions,
        latest.defaultProvider,
      )
      setModels((current) => sortModelList(current, rebased.defaultModelName))
      setDefaultModelBaseline(persisted.defaultModelName)
      setDefaultModelDraft(rebased.defaultModelName)
      setFallbackChainBaseline(persisted.fallbackChain)
      setFallbackChainDraft(rebased.fallbackChain)
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("models.defaultChain.saveSuccess"),
        submitted.defaultModelName || t("navigation.models"),
        gateway?.restartRequired === true,
      )
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      savingChainRef.current = false
      setSavingChain(false)
    }
  }

  const handleResetChain = () => {
    setDefaultModelDraft(defaultModelBaseline)
    setModels((prev) => sortModelList(prev, defaultModelBaseline))
    setFallbackChainDraft(fallbackChainBaseline)
  }

  const grouped: Record<
    string,
    {
      provider: Pick<
        ProviderCatalogEntry,
        "key" | "label" | "iconSlug" | "domain"
      >
      models: ModelInfo[]
    }
  > = {}
  for (const model of models) {
    const providerKey = getCanonicalProviderKey(model.provider, providerOptions)
    const providerDef = providerKey ? providerMap.get(providerKey) : undefined
    if (!grouped[providerKey]) {
      grouped[providerKey] = {
        provider: {
          key: providerKey,
          label: providerDef?.label || providerKey,
          iconSlug: providerDef?.iconSlug,
          domain: providerDef?.domain,
        },
        models: [],
      }
    }
    grouped[providerKey].models.push(model)
  }

  const providerGroups: ProviderGroup[] = Object.entries(grouped)
    .map(([key, group]) => {
      const availableCount = group.models.filter(
        (model) => model.available,
      ).length
      return {
        key,
        provider: group.provider,
        models: group.models,
        hasDefault: group.models.some(
          (model) => model.model_name === defaultModelDraft,
        ),
        availableCount,
      }
    })
    .sort((a, b) => {
      if (a.hasDefault && !b.hasDefault) return -1
      if (!a.hasDefault && b.hasDefault) return 1

      if (a.availableCount !== b.availableCount) {
        return b.availableCount - a.availableCount
      }

      const aPriority = -(providerMap.get(a.key)?.priority ?? 0)
      const bPriority = -(providerMap.get(b.key)?.priority ?? 0)
      if (aPriority !== bPriority) {
        return aPriority - bPriority
      }

      return a.provider.label.localeCompare(b.provider.label)
    })

  const defaultModel = models.find(
    (model) => model.model_name === defaultModelDraft,
  )
  const defaultModelEntryCount = models.filter(
    (model) => model.model_name === defaultModelDraft,
  ).length
  const defaultModelIdentity = modelReferenceIdentity(
    defaultModelDraft,
    models,
    providerOptions,
    defaultProvider,
  )
  const fallbackIdentities = new Set(
    fallbackChainDraft.map((modelName) =>
      modelReferenceIdentity(
        modelName,
        models,
        providerOptions,
        defaultProvider,
      ),
    ),
  )
  const fallbackDefaultConflictModelNames = new Set(
    models
      .filter((model) => {
        const identity = modelReferenceIdentity(
          model.model_name,
          models,
          providerOptions,
          defaultProvider,
        )
        return (
          identity === defaultModelIdentity ||
          (!fallbackChainDraft.includes(model.model_name) &&
            fallbackIdentities.has(identity))
        )
      })
      .map((model) => model.model_name),
  )
  const defaultModelLabel = defaultModel?.model_name ?? defaultModelDraft
  const defaultChainAllowedModelNames = new Set(
    models
      .filter((model) =>
        modelNameAllowedInDefaultChain(model.model_name, models),
      )
      .map((model) => model.model_name),
  )
  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.models")}>
        <div className="flex items-center gap-3">
          <Button
            size="sm"
            variant="outline"
            onClick={() => setDefaultChainOpen(true)}
            disabled={
              savingChain ||
              modelUpdatePending ||
              (models.length === 0 && !defaultModelDraft)
            }
          >
            <IconArrowsSort className="size-4" />
            {t("models.defaultChain.button")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setCatalogOpen(true)}
            disabled={
              savingChain || modelUpdatePending || providerOptions.length === 0
            }
          >
            <IconDatabase className="size-4" />
            {t("models.catalog.button")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setAddOpen(true)}
            disabled={
              savingChain || modelUpdatePending || providerOptions.length === 0
            }
          >
            <IconPlus className="size-4" />
            {t("models.add.button")}
          </Button>
        </div>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        <div className="pt-2">
          {!defaultModelDraft && (
            <div className="text-muted-foreground flex items-center gap-1.5 text-sm">
              <span>{t("models.noDefaultHintPrefix")}</span>
              <IconStar className="size-3.5 shrink-0" />
              <span>{t("models.noDefaultHintSuffix")}</span>
            </div>
          )}
          <p className="text-muted-foreground mt-1 text-sm">
            {t("models.description")}
          </p>
          {!loading && (models.length > 0 || defaultModelDraft) && (
            <div className="bg-muted/30 mt-4 rounded-xl border px-4 py-3">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm font-medium">
                    {t("models.defaultChain.title")}
                  </p>
                  <p className="text-muted-foreground mt-1 text-sm">
                    {defaultModelLabel
                      ? t("models.defaultChain.summary", {
                          model: defaultModelLabel,
                          count: fallbackChainDraft.length,
                        })
                      : t("models.defaultChain.noDefault")}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setDefaultChainOpen(true)}
                >
                  <IconArrowsSort className="size-4" />
                  {t("models.defaultChain.manage")}
                </Button>
              </div>
            </div>
          )}
          {!loading && providerOptions.length === 0 && (
            <p className="text-muted-foreground mt-1 text-sm">
              {t("models.providerCatalogUnavailable")}
            </p>
          )}
        </div>

        {(loading || savingChain || modelUpdatePending) && (
          <div className="flex items-center justify-center py-20">
            <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        )}

        {fetchError && (
          <div className="bg-destructive/10 rounded-lg px-4 py-3 text-sm">
            <p className="text-destructive">{fetchError}</p>
            <div className="mt-3 flex items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  void fetchModels()
                }}
              >
                {t("models.retry")}
              </Button>
            </div>
          </div>
        )}

        {!loading && !savingChain && !modelUpdatePending && !fetchError && (
          <div className="pb-8">
            {providerGroups.map((providerGroup) => (
              <ProviderSection
                key={providerGroup.key}
                provider={providerGroup.provider}
                models={providerGroup.models}
                onEdit={setEditingModel}
                onSetDefault={handleSetDefault}
                onToggleFallback={handleToggleFallback}
                onDelete={setDeletingModel}
                fallbackChain={fallbackChainDraft}
                defaultModelName={defaultModelDraft}
                defaultModelEntryCount={defaultModelEntryCount}
                defaultChainAllowedModelNames={defaultChainAllowedModelNames}
                fallbackDefaultConflictModelNames={
                  fallbackDefaultConflictModelNames
                }
              />
            ))}
          </div>
        )}
      </div>

      <EditModelSheet
        model={editingModel}
        open={editingModel !== null}
        onClose={() => setEditingModel(null)}
        onUpdateStarted={() => {
          beginModelMutation()
          if (
            editingModel &&
            !pendingModelUpdates.current.some(
              (update) => update.index === editingModel.index,
            )
          ) {
            pendingModelUpdates.current.push({
              index: editingModel.index,
              from: editingModel.model_name,
              stableAliases: models
                .filter((model) => model.index !== editingModel.index)
                .map((model) => ({
                  index: model.index,
                  modelName: model.model_name,
                })),
            })
          }
        }}
        onUpdated={(referenceRename) => {
          if (referenceRename) {
            pendingReferenceRenames.current.push(referenceRename)
          }
          void refreshAfterModelChange()
        }}
        onSaved={() => {
          void refreshAfterModelChange()
        }}
        onUpdateSettled={finishModelMutation}
        providerOptions={providerOptions}
      />

      <AddModelSheet
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSaved={() => {
          void refreshAfterModelChange()
        }}
        onMutationStarted={beginModelMutation}
        onMutationSettled={finishModelMutation}
        existingModelNames={models.map((model) => model.model_name)}
        providerOptions={providerOptions}
      />

      <DeleteModelDialog
        model={deletingModel}
        isDefaultModel={
          deletingModel?.model_name === defaultModelDraft &&
          defaultModelEntryCount <= 1
        }
        onClose={() => setDeletingModel(null)}
        onDeleted={() => {
          void refreshAfterModelChange()
        }}
        onMutationStarted={beginModelMutation}
        onMutationSettled={finishModelMutation}
      />

      <CatalogDialog
        open={catalogOpen}
        onClose={() => setCatalogOpen(false)}
        onModelAdded={() => {
          void refreshAfterModelChange()
        }}
        onMutationStarted={beginModelMutation}
        onMutationSettled={finishModelMutation}
        providerOptions={providerOptions}
      />

      <DefaultChainDialog
        open={defaultChainOpen}
        onClose={() => setDefaultChainOpen(false)}
        models={models}
        defaultModelName={defaultModelDraft}
        fallbackChain={fallbackChainDraft}
        onMoveUp={(modelName) => {
          const index = fallbackChainDraft.indexOf(modelName)
          if (index > 0) {
            moveFallback(modelName, index - 1)
          }
        }}
        onMoveDown={(modelName) => {
          const index = fallbackChainDraft.indexOf(modelName)
          if (index !== -1 && index < fallbackChainDraft.length - 1) {
            moveFallback(modelName, index + 1)
          }
        }}
        onMoveTop={(modelName) => moveFallback(modelName, 0)}
        onMoveBottom={(modelName) =>
          moveFallback(modelName, Math.max(fallbackChainDraft.length - 1, 0))
        }
        onRemove={(modelName) =>
          setFallbackChainDraft((prev) =>
            prev.filter((item) => item !== modelName),
          )
        }
      />

      {chainDirty && (
        <div className="border-border/70 bg-background/95 supports-backdrop-filter:bg-background/80 shrink-0 border-t px-4 py-3 shadow-[0_-12px_30px_rgba(15,23,42,0.10)] backdrop-blur">
          <div className="mx-auto flex w-full max-w-[1400px] flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex-1">
              <ConfigChangeNotice
                kind="save"
                title={t("common.saveChangesTitle")}
                description={t("models.defaultChain.unsaved")}
              />
            </div>
            <div className="flex items-center justify-end gap-2">
              <Button
                variant="outline"
                onClick={handleResetChain}
                disabled={savingChain || modelUpdatePending}
              >
                {t("common.reset")}
              </Button>
              <Button
                onClick={handleSaveChain}
                disabled={savingChain || modelUpdatePending}
              >
                {savingChain ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
