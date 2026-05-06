import {
  IconChevronDown,
  IconChevronRight,
  IconLoader2,
  IconTrash,
} from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type CatalogEntry,
  type CatalogModel,
  addModel,
  deleteCatalog,
  getCatalogs,
} from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { refreshGatewayState } from "@/store/gateway"

import { getProviderLabel } from "./provider-label"

interface CatalogDialogProps {
  open: boolean
  onClose: () => void
  onModelAdded: () => void
}

export function CatalogDialog({
  open,
  onClose,
  onModelAdded,
}: CatalogDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [selected, setSelected] = useState<Map<string, Set<string>>>(new Map())
  const [adding, setAdding] = useState(false)
  const [filter, setFilter] = useState("")

  const loadCatalogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getCatalogs()
      setEntries(res.entries || [])
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load catalogs")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) {
      loadCatalogs()
      setExpandedId(null)
      setSelected(new Map())
      setFilter("")
    }
  }, [open, loadCatalogs])

  const toggleExpand = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id))
  }

  const toggleModel = (catalogId: string, modelId: string) => {
    setSelected((prev) => {
      const next = new Map(prev)
      const set = new Set(next.get(catalogId) || [])
      if (set.has(modelId)) set.delete(modelId)
      else set.add(modelId)
      next.set(catalogId, set)
      return next
    })
  }

  const toggleAll = (catalogId: string, models: CatalogModel[]) => {
    setSelected((prev) => {
      const next = new Map(prev)
      const current = next.get(catalogId) || new Set()
      const filtered = filter
        ? models.filter((m) => m.id.toLowerCase().includes(filter.toLowerCase()))
        : models
      if (filtered.every((m) => current.has(m.id))) {
        next.set(catalogId, new Set())
      } else {
        next.set(catalogId, new Set(filtered.map((m) => m.id)))
      }
      return next
    })
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteCatalog(id)
      setEntries((prev) => prev.filter((e) => e.id !== id))
      setSelected((prev) => {
        const next = new Map(prev)
        next.delete(id)
        return next
      })
      if (expandedId === id) setExpandedId(null)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to delete catalog")
    }
  }

  const handleAddSelected = async (entry: CatalogEntry) => {
    const catalogSelected = selected.get(entry.id) || new Set()
    if (catalogSelected.size === 0) return

    setAdding(true)
    try {
      const modelsToAdd = entry.models.filter((m) => catalogSelected.has(m.id))
      for (const model of modelsToAdd) {
        await addModel({
          model_name: model.id,
          provider: entry.provider || undefined,
          model: model.id,
          api_base: entry.api_base || undefined,
        })
      }
      await refreshGatewayState({ force: true })
      toast.success(
        t("models.catalog.addSuccess", { count: modelsToAdd.length }),
      )
      onModelAdded()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to add models")
    } finally {
      setAdding(false)
    }
  }

  const getFilteredModels = (models: CatalogModel[]) =>
    filter
      ? models.filter((m) => m.id.toLowerCase().includes(filter.toLowerCase()))
      : models

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("models.catalog.title")}</DialogTitle>
          <DialogDescription>
            {t("models.catalog.description")}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {loading && (
            <div className="flex items-center justify-center gap-2 py-8 text-muted-foreground">
              <IconLoader2 className="size-5 animate-spin" />
              <span>{t("models.catalog.loading")}</span>
            </div>
          )}

          {!loading && entries.length === 0 && (
            <div className="py-8 text-center text-sm text-muted-foreground">
              {t("models.catalog.empty")}
            </div>
          )}

          {entries.length > 0 && (
            <Input
              placeholder={t("models.catalog.filterPlaceholder")}
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="h-8"
            />
          )}

          <div className="max-h-[400px] space-y-2 overflow-y-auto">
            {entries.map((entry) => {
              const isExpanded = expandedId === entry.id
              const entrySelected = selected.get(entry.id) || new Set()
              const filteredModels = getFilteredModels(entry.models)

              return (
                <div
                  key={entry.id}
                  className="rounded-lg border bg-card text-card-foreground"
                >
                  <div
                    className="flex cursor-pointer items-center gap-3 px-3 py-2.5 hover:bg-accent/50"
                    onClick={() => toggleExpand(entry.id)}
                  >
                    {isExpanded ? (
                      <IconChevronDown className="size-4 shrink-0 text-muted-foreground" />
                    ) : (
                      <IconChevronRight className="size-4 shrink-0 text-muted-foreground" />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium">
                          {getProviderLabel(entry.provider)}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">
                          {entry.api_key_mask}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <span>
                          {entry.models.length} {t("models.catalog.models")}
                        </span>
                        {entry.api_base && (
                          <>
                            <span>|</span>
                            <span className="truncate">{entry.api_base}</span>
                          </>
                        )}
                        {entry.fetched_at && (
                          <>
                            <span>|</span>
                            <span>
                              {t("models.catalog.fetchedAt")}{" "}
                              {new Date(entry.fetched_at).toLocaleDateString()}
                            </span>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-7 text-destructive"
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDelete(entry.id)
                        }}
                        title={t("models.catalog.delete")}
                      >
                        <IconTrash className="size-3.5" />
                      </Button>
                    </div>
                  </div>

                  {isExpanded && (
                    <div className="border-t px-3 py-2">
                      <div className="mb-1.5 flex items-center justify-between text-xs text-muted-foreground">
                        <span>
                          {t("models.catalog.found", {
                            count: filteredModels.length,
                          })}
                        </span>
                        <button
                          type="button"
                          onClick={() => toggleAll(entry.id, entry.models)}
                          className="text-primary hover:underline"
                        >
                          {filteredModels.every((m) =>
                            entrySelected.has(m.id),
                          )
                            ? t("models.catalog.deselectAll")
                            : t("models.catalog.selectAll")}
                        </button>
                      </div>
                      <div className="max-h-[200px] space-y-0.5 overflow-y-auto">
                        {filteredModels.map((m) => (
                          <label
                            key={m.id}
                            className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1 text-sm hover:bg-accent"
                          >
                            <input
                              type="checkbox"
                              checked={entrySelected.has(m.id)}
                              onChange={() => toggleModel(entry.id, m.id)}
                              className="size-3.5"
                            />
                            <span className="font-mono text-xs">{m.id}</span>
                            {m.owned_by && (
                              <span className="ml-auto text-xs text-muted-foreground">
                                {m.owned_by}
                              </span>
                            )}
                          </label>
                        ))}
                      </div>
                      {entrySelected.size > 0 && (
                        <div className="mt-2 flex justify-end">
                          <Button
                            size="sm"
                            onClick={() => handleAddSelected(entry)}
                            disabled={adding}
                          >
                            {adding && (
                              <IconLoader2 className="mr-1 size-3 animate-spin" />
                            )}
                            {t("models.catalog.addSelected", {
                              count: entrySelected.size,
                            })}
                          </Button>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
