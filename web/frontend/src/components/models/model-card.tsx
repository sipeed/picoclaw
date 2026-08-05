import {
  IconArrowAutofitDown,
  IconEdit,
  IconKey,
  IconStar,
  IconStarFilled,
  IconTrash,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface ModelCardProps {
  model: ModelInfo
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onToggleFallback: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  inFallbackChain: boolean
  isDefault: boolean
  defaultChainAllowed: boolean
  fallbackDefaultConflict: boolean
  deleteDisabled: boolean
}

export function ModelCard({
  model,
  onEdit,
  onSetDefault,
  onToggleFallback,
  onDelete,
  inFallbackChain,
  isDefault,
  defaultChainAllowed,
  fallbackDefaultConflict,
  deleteDisabled,
}: ModelCardProps) {
  const { t } = useTranslation()
  const isOAuth = model.auth_method === "oauth"
  const status = model.status
  const statusLabel = t(`models.status.${status}`)
  const canSetDefault = model.available && !isDefault && defaultChainAllowed
  const canUseInFallback =
    model.available &&
    !isDefault &&
    !fallbackDefaultConflict &&
    defaultChainAllowed

  const setDefaultLabel = t("models.action.setDefault")
  const setDefaultDisabledReason = (() => {
    if (!model.available)
      return t("models.action.setDefaultDisabled.unavailable")
    if (isDefault) return t("models.action.setDefaultDisabled.isDefault")
    if (model.is_virtual) return t("models.action.setDefaultDisabled.isVirtual")
    if (!defaultChainAllowed) {
      return t("models.action.setDefaultDisabled.unsupportedProvider")
    }
    return setDefaultLabel
  })()

  const editLabel = t("models.action.edit")
  const toggleFallbackLabel = inFallbackChain
    ? t("models.action.removeFromFallback")
    : t("models.action.addToFallback")
  const toggleFallbackDisabledReason = (() => {
    if (!model.available)
      return t("models.action.setDefaultDisabled.unavailable")
    if (model.is_virtual) return t("models.action.setDefaultDisabled.isVirtual")
    if (!defaultChainAllowed) {
      return t("models.action.setDefaultDisabled.unsupportedProvider")
    }
    if (isDefault || fallbackDefaultConflict)
      return t("models.action.addToFallbackDisabled.isDefault")
    return toggleFallbackLabel
  })()
  const deleteLabel = t("models.action.delete")
  const deleteDisabledReason = deleteDisabled
    ? t("models.action.deleteDisabled.isDefault")
    : deleteLabel

  return (
    <div
      className={[
        "group/card hover:bg-muted/30 relative flex w-full max-w-[36rem] flex-col gap-3 justify-self-start rounded-xl border p-4 transition-colors hover:shadow-xs",
        model.available
          ? "border-border/60 bg-card"
          : "border-border/50 bg-card/60",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={[
              "mt-0.5 h-2 w-2 shrink-0 rounded-full",
              isDefault
                ? "bg-green-400 shadow-[0_0_0_2px_rgba(74,222,128,0.35)]"
                : status === "available"
                  ? "bg-green-500"
                  : status === "unreachable"
                    ? "bg-amber-500"
                    : "bg-muted-foreground/25",
            ].join(" ")}
            title={statusLabel}
          />
          <span className="text-foreground truncate text-sm font-semibold">
            {model.model_name}
          </span>
          {isDefault && (
            <span className="bg-primary/10 text-primary shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-none font-medium">
              {t("models.badge.default")}
            </span>
          )}
          {model.is_virtual && (
            <span className="bg-muted text-muted-foreground shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-none font-medium">
              {t("models.badge.virtual")}
            </span>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-0.5">
          {isDefault ? (
            <span
              className="text-primary p-1"
              title={t("models.badge.default")}
            >
              <IconStarFilled className="size-3.5" />
            </span>
          ) : (
            <Tooltip delayDuration={!canSetDefault ? 0 : 700}>
              <TooltipTrigger asChild>
                <span
                  className={!canSetDefault ? "cursor-not-allowed" : undefined}
                  tabIndex={!canSetDefault ? 0 : undefined}
                  role={!canSetDefault ? "button" : undefined}
                  aria-disabled={!canSetDefault ? true : undefined}
                  aria-label={!canSetDefault ? setDefaultLabel : undefined}
                  title={!canSetDefault ? setDefaultLabel : undefined}
                >
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => onSetDefault(model)}
                    disabled={!canSetDefault}
                    aria-label={setDefaultLabel}
                    title={setDefaultLabel}
                  >
                    <IconStar className="size-3.5" />
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{setDefaultDisabledReason}</TooltipContent>
            </Tooltip>
          )}

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(model)}
            aria-label={editLabel}
            title={editLabel}
          >
            <IconEdit className="size-3.5" />
          </Button>

          <Tooltip delayDuration={canUseInFallback ? 700 : 0}>
            <TooltipTrigger asChild>
              <span
                className={!canUseInFallback ? "cursor-not-allowed" : undefined}
                tabIndex={!canUseInFallback ? 0 : undefined}
                role={!canUseInFallback ? "button" : undefined}
                aria-disabled={!canUseInFallback ? true : undefined}
                aria-label={!canUseInFallback ? toggleFallbackLabel : undefined}
                title={!canUseInFallback ? toggleFallbackLabel : undefined}
              >
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onToggleFallback(model)}
                  disabled={!canUseInFallback}
                  aria-label={toggleFallbackLabel}
                  title={toggleFallbackLabel}
                >
                  <IconArrowAutofitDown
                    className={[
                      "size-3.5",
                      inFallbackChain ? "text-primary" : "",
                    ].join(" ")}
                  />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{toggleFallbackDisabledReason}</TooltipContent>
          </Tooltip>

          <Tooltip delayDuration={deleteDisabled ? 0 : 700}>
            <TooltipTrigger asChild>
              <span
                className={deleteDisabled ? "cursor-not-allowed" : undefined}
                tabIndex={deleteDisabled ? 0 : undefined}
                role={deleteDisabled ? "button" : undefined}
                aria-disabled={deleteDisabled ? true : undefined}
                aria-label={deleteDisabled ? deleteLabel : undefined}
                title={deleteDisabled ? deleteLabel : undefined}
              >
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onDelete(model)}
                  disabled={deleteDisabled}
                  aria-label={deleteLabel}
                  title={deleteLabel}
                  className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                >
                  <IconTrash className="size-3.5" />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{deleteDisabledReason}</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <p className="text-muted-foreground truncate font-mono text-xs leading-snug">
        {model.model}
      </p>

      <div className="flex items-center gap-2">
        {isOAuth ? (
          <span className="text-muted-foreground bg-muted rounded px-1.5 py-0.5 text-[10px] font-medium">
            OAuth
          </span>
        ) : status === "available" && model.api_key ? (
          <span className="text-muted-foreground/70 flex items-center gap-1 font-mono text-[11px]">
            <IconKey className="size-3" />
            {model.api_key}
          </span>
        ) : (
          <span className="text-muted-foreground/50 text-[11px]">
            {statusLabel}
          </span>
        )}
      </div>
    </div>
  )
}
