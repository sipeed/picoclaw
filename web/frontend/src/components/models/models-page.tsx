import { IconDatabase, IconLoader2, IconPlus, IconStar } from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ModelInfo, getModels, setDefaultModel } from "@/api/models"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

import { AddModelSheet } from "./add-model-sheet"
import { CatalogDialog } from "./catalog-dialog"
import { DeleteModelDialog } from "./delete-model-dialog"
import { EditModelSheet } from "./edit-model-sheet"
import { getProviderKey, getProviderLabel } from "./provider-label"
import { PROVIDER_PRIORITY } from "./provider-registry"
import { ProviderSection } from "./provider-section"

interface ProviderGroup {
  key: string
  label: string
  models: ModelInfo[]
  hasDefault: boolean
  availableCount: number
}

export function ModelsPage() {
  const { t } = useTranslation()
  const [models, setModels] = useState<ModelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState("")

  const [editingModel, setEditingModel] = useState<ModelInfo | null>(null)
  const [deletingModel, setDeletingModel] = useState<ModelInfo | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [catalogOpen, setCatalogOpen] = useState(false)
  const [settingDefaultIndex, setSettingDefaultIndex] = useState<number | null>(
    null,
  )

  const fetchModels = useCallback(async () => {
    try {
      const data = await getModels()
      const sorted = [...data.models].sort((a, b) => {
        if (a.is_default && !b.is_default) return -1
        if (!a.is_default && b.is_default) return 1
        if (a.available && !b.available) return -1
        if (!a.available && b.available) return 1
        return a.model_name.localeCompare(b.model_name)
      })
      setModels(sorted)
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
    if (model.is_default) return

    setSettingDefaultIndex(model.index)
    try {
      await setDefaultModel(model.model_name)
      await fetchModels()
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("models.defaultChangeSuccess"),
        model.model_name,
        gateway?.restartRequired === true,
      )
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setSettingDefaultIndex(null)
    }
  }

  const grouped: Record<string, { label: string; models: ModelInfo[] }> = {}
  for (const model of models) {
    const providerKey = getProviderKey(model.provider)
    if (!grouped[providerKey]) {
      grouped[providerKey] = {
        label: getProviderLabel(model.provider),
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
        label: group.label,
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

      const aPriority = PROVIDER_PRIORITY[a.key] ?? Number.MAX_SAFE_INTEGER
      const bPriority = PROVIDER_PRIORITY[b.key] ?? Number.MAX_SAFE_INTEGER
      if (aPriority !== bPriority) {
        return aPriority - bPriority
      }

      return a.label.localeCompare(b.label)
    })

  const defaultModel = models.find((model) => model.is_default)

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.models")}>
        <div className="flex items-center gap-3">
          <Button
            size="sm"
            variant="outline"
            onClick={() => setCatalogOpen(true)}
          >
            <IconDatabase className="size-4" />
            {t("models.catalog.button")}
          </Button>
          <Button size="sm" variant="outline" onClick={() => setAddOpen(true)}>
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
        </div>

        {loading && (
          <div className="flex items-center justify-center py-20">
            <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        )}

        {fetchError && (
          <div className="text-destructive bg-destructive/10 rounded-lg px-4 py-3 text-sm">
            {fetchError}
          </div>
        )}

        {!loading && !fetchError && (
          <div className="pb-8">
            {providerGroups.map((providerGroup) => (
              <ProviderSection
                key={providerGroup.key}
                provider={providerGroup.label}
                providerKey={providerGroup.key}
                models={providerGroup.models}
                onEdit={setEditingModel}
                onSetDefault={handleSetDefault}
                onDelete={setDeletingModel}
                settingDefaultIndex={settingDefaultIndex}
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
      />

      <AddModelSheet
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSaved={fetchModels}
        existingModelNames={models.map((model) => model.model_name)}
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
      />
    </div>
  )
}
