import { IconDownload, IconLoader2 } from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { fetchUpstreamModels, type UpstreamModel } from "@/api/models"
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

import { PROVIDER_MAP } from "./provider-registry"

interface FetchModelsDialogProps {
  open: boolean
  onClose: () => void
  onFill: (models: string[]) => void
  provider: string
  apiKey: string
  apiBase: string
}

export function FetchModelsDialog({
  open,
  onClose,
  onFill,
  provider,
  apiKey,
  apiBase,
}: FetchModelsDialogProps) {
  const { t } = useTranslation()
  const [fetching, setFetching] = useState(false)
  const [models, setModels] = useState<UpstreamModel[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [error, setError] = useState("")
  const [filter, setFilter] = useState("")

  const providerDef = PROVIDER_MAP.get(provider)
  const needsKey = providerDef?.requiresApiKey !== false

  const handleFetch = useCallback(async () => {
    setFetching(true)
    setError("")
    setModels([])
    setSelected(new Set())
    try {
      const res = await fetchUpstreamModels({
        provider,
        api_key: apiKey,
        api_base: apiBase,
      })
      setModels(res.models)
      // Auto-select all by default
      setSelected(new Set(res.models.map((m) => m.id)))
    } catch (e) {
      setError(e instanceof Error ? e.message : t("models.fetch.failed"))
    } finally {
      setFetching(false)
    }
  }, [provider, apiKey, apiBase, t])

  // Auto-fetch when dialog opens
  useEffect(() => {
    if (open && provider) {
      handleFetch()
    }
  }, [open, provider, handleFetch])

  const handleFill = () => {
    onFill(Array.from(selected))
    handleClose()
  }

  const handleClose = () => {
    setModels([])
    setSelected(new Set())
    setError("")
    setFilter("")
    onClose()
  }

  const toggleModel = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    const filtered = models
      .map((m) => m.id)
      .filter(
        (id) => !filter || id.toLowerCase().includes(filter.toLowerCase()),
      )
    if (filtered.every((id) => selected.has(id))) {
      setSelected(new Set())
    } else {
      setSelected(new Set(filtered))
    }
  }

  const filteredModels = filter
    ? models.filter((m) => m.id.toLowerCase().includes(filter.toLowerCase()))
    : models

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconDownload className="size-5" />
            {t("models.fetch.title")}
          </DialogTitle>
          <DialogDescription>
            {t("models.fetch.description")}
            {provider && (
              <span className="mt-1 block font-mono text-xs">
                {t("models.fetch.providerLabel")} {provider}
                {apiBase && ` | ${apiBase}`}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {needsKey && !apiKey && (
            <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-3 text-sm text-yellow-700 dark:text-yellow-400">
              {t("models.fetch.needApiKey")}
            </div>
          )}

          {fetching && (
            <div className="flex items-center justify-center gap-2 py-8 text-muted-foreground">
              <IconLoader2 className="size-5 animate-spin" />
              <span>{t("models.fetch.fetching")}</span>
            </div>
          )}

          {error && (
            <div className="space-y-2">
              <div className="bg-destructive/10 text-destructive rounded-lg p-3 text-sm">
                {error}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleFetch}
                className="w-full"
              >
                {t("models.fetch.retry")}
              </Button>
            </div>
          )}

          {models.length > 0 && (
            <>
              <Input
                placeholder={t("models.fetch.filterPlaceholder")}
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="h-8"
              />
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>
                  {t("models.fetch.found", { count: models.length })}
                  {filter && ` ${t("models.fetch.shown", { count: filteredModels.length })}`}
                </span>
                <button
                  type="button"
                  onClick={toggleAll}
                  className="text-primary hover:underline"
                >
                  {filteredModels.every((m) => selected.has(m.id))
                    ? t("models.fetch.deselectAll")
                    : t("models.fetch.selectAll")}
                </button>
              </div>
              <div className="max-h-[300px] space-y-1 overflow-y-auto rounded-md border p-2">
                {filteredModels.map((m) => (
                  <label
                    key={m.id}
                    className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
                  >
                    <input
                      type="checkbox"
                      checked={selected.has(m.id)}
                      onChange={() => toggleModel(m.id)}
                      className="size-3.5"
                    />
                    <span className="font-mono text-xs">{m.id}</span>
                    {m.owned_by && (
                      <span className="text-muted-foreground ml-auto text-xs">
                        {m.owned_by}
                      </span>
                    )}
                  </label>
                ))}
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={handleClose}>
            {t("common.cancel")}
          </Button>
          {models.length > 0 && (
            <Button onClick={handleFill} disabled={selected.size === 0}>
              {t("models.fetch.fill", { count: selected.size })}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
