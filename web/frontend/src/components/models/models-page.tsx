import {
  IconArrowsSort,
  IconDatabase,
  IconLoader2,
  IconPlus,
  IconStar,
} from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ModelInfo,
  type ModelProviderOption,
  getDefaultChain,
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

export function ModelsPage() {
  const { t } = useTranslation()
  const [models, setModels] = useState<ModelInfo[]>([])
  const [providerOptions, setProviderOptions] = useState<ModelProviderOption[]>(
    [],
  )
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState("")

  const [editingModel, setEditingModel] = useState<ModelInfo | null>(null)
  const [deletingModel, setDeletingModel] = useState<ModelInfo | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [catalogOpen, setCatalogOpen] = useState(false)
  const [defaultChainOpen, setDefaultChainOpen] = useState(false)
  const [settingDefaultIndex, setSettingDefaultIndex] = useState<number | null>(
    null,
  )
  const [defaultModelDraft, setDefaultModelDraft] = useState("")
  const [defaultModelBaseline, setDefaultModelBaseline] = useState("")
  const [fallbackChainDraft, setFallbackChainDraft] = useState<string[]>([])
  const [fallbackChainBaseline, setFallbackChainBaseline] = useState<string[]>(
    [],
  )
  const [savingChain, setSavingChain] = useState(false)
  const providerMap = getProviderCatalogMap(providerOptions)

  const fetchModels = useCallback(async () => {
    setLoading(true)
    try {
      const [data, chain] = await Promise.all([getModels(), getDefaultChain()])
      const sorted = [...data.models].sort((a, b) => {
        if (a.is_default && !b.is_default) return -1
        if (!a.is_default && b.is_default) return 1
        if (a.available && !b.available) return -1
        if (!a.available && b.available) return 1
        return a.model_name.localeCompare(b.model_name)
      })
      setModels(sorted)
      setProviderOptions(data.provider_options || [])
      setDefaultModelDraft(chain.default_model || "")
      setDefaultModelBaseline(chain.default_model || "")
      setFallbackChainDraft(chain.fallback_chain || [])
      setFallbackChainBaseline(chain.fallback_chain || [])
      setFetchError("")
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  const handleSetDefault = async (model: ModelInfo) => {
    if (defaultModelDraft === model.model_name) return
    setSettingDefaultIndex(model.index)
    setDefaultModelDraft(model.model_name)
    setFallbackChainDraft((prev) =>
      prev.filter((item) => item !== model.model_name),
    )
    setTimeout(() => {
      setSettingDefaultIndex((current) =>
        current === model.index ? null : current,
      )
    }, 150)
  }

  const handleToggleFallback = (model: ModelInfo) => {
    if (model.model_name === defaultModelDraft) {
      return
    }
    setFallbackChainDraft((prev) =>
      prev.includes(model.model_name)
        ? prev.filter((item) => item !== model.model_name)
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
    setSavingChain(true)
    try {
      const saved = await updateDefaultChain({
        default_model: defaultModelDraft,
        fallback_chain: fallbackChainDraft,
      })
      setDefaultModelBaseline(saved.default_model || "")
      setFallbackChainBaseline(saved.fallback_chain || [])
      await fetchModels()
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("models.defaultChain.saveSuccess"),
        defaultModelDraft || t("navigation.models"),
        gateway?.restartRequired === true,
      )
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setSavingChain(false)
    }
  }

  const handleResetChain = () => {
    setDefaultModelDraft(defaultModelBaseline)
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
        hasDefault: group.models.some((model) => model.is_default),
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
  const fallbackModels = fallbackChainDraft
    .map((modelName) => models.find((model) => model.model_name === modelName))
    .filter((model): model is ModelInfo => model != null)
  const chainDirty =
    defaultModelDraft !== defaultModelBaseline ||
    JSON.stringify(fallbackChainDraft) !== JSON.stringify(fallbackChainBaseline)

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.models")}>
        <div className="flex items-center gap-3">
          <Button
            size="sm"
            variant="outline"
            onClick={() => setDefaultChainOpen(true)}
            disabled={models.length === 0}
          >
            <IconArrowsSort className="size-4" />
            {t("models.defaultChain.button")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setCatalogOpen(true)}
            disabled={providerOptions.length === 0}
          >
            <IconDatabase className="size-4" />
            {t("models.catalog.button")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setAddOpen(true)}
            disabled={providerOptions.length === 0}
          >
            <IconPlus className="size-4" />
            {t("models.add.button")}
          </Button>
        </div>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        <div className="pt-2">
          {!defaultModel && (
            <div className="text-muted-foreground flex items-center gap-1.5 text-sm">
              <span>{t("models.noDefaultHintPrefix")}</span>
              <IconStar className="size-3.5 shrink-0" />
              <span>{t("models.noDefaultHintSuffix")}</span>
            </div>
          )}
          <p className="text-muted-foreground mt-1 text-sm">
            {t("models.description")}
          </p>
          {!loading && models.length > 0 && (
            <div className="bg-muted/30 mt-4 rounded-xl border px-4 py-3">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm font-medium">
                    {t("models.defaultChain.title")}
                  </p>
                  <p className="text-muted-foreground mt-1 text-sm">
                    {defaultModel
                      ? t("models.defaultChain.summary", {
                          model: defaultModel.model_name,
                          count: fallbackModels.length,
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

        {loading && (
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

        {!loading && !fetchError && (
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
                settingDefaultIndex={settingDefaultIndex}
                fallbackChain={fallbackChainDraft}
                defaultModelName={defaultModelDraft}
              />
            ))}
          </div>
        )}
      </div>

      <EditModelSheet
        model={editingModel}
        open={editingModel !== null}
        onClose={() => setEditingModel(null)}
        onSaved={fetchModels}
        providerOptions={providerOptions}
      />

      <AddModelSheet
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSaved={fetchModels}
        existingModelNames={models.map((model) => model.model_name)}
        providerOptions={providerOptions}
      />

      <DeleteModelDialog
        model={deletingModel}
        onClose={() => setDeletingModel(null)}
        onDeleted={fetchModels}
      />

      <CatalogDialog
        open={catalogOpen}
        onClose={() => setCatalogOpen(false)}
        onModelAdded={fetchModels}
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
                disabled={savingChain}
              >
                {t("common.reset")}
              </Button>
              <Button onClick={handleSaveChain} disabled={savingChain}>
                {savingChain ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
