import {
  IconEdit,
  IconKey,
  IconLoader2,
  IconStar,
  IconStarFilled,
  IconTrash,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"

interface ModelCardProps {
  model: ModelInfo
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  settingDefault: boolean
}

export function ModelCard({
  model,
  onEdit,
  onSetDefault,
  onDelete,
  settingDefault,
}: ModelCardProps) {
  const { t } = useTranslation()
  const isOAuth = model.auth_method === "oauth"
  const canSetDefault = model.configured && !model.is_default && !model.is_virtual

  return (
    <div
      className={[
        "group/card hover:bg-muted/30 relative flex w-full max-w-[36rem] flex-col gap-3 justify-self-start rounded-xl border p-4 transition-colors hover:shadow-xs",
        model.configured
          ? "border-border/60 bg-card"
          : "border-border/50 bg-card/60",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={[
              "mt-0.5 h-2 w-2 shrink-0 rounded-full",
              model.is_default
                ? "bg-green-400 shadow-[0_0_0_2px_rgba(74,222,128,0.35)]"
                : model.configured
                  ? "bg-green-500"
                  : "bg-muted-foreground/25",
            ].join(" ")}
            title={
              model.configured
                ? t("models.status.configured")
                : t("models.status.unconfigured")
            }
          />
          <span className="text-foreground truncate text-sm font-semibold">
            {model.model_name}
          </span>
          {model.is_default && (
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
          {model.is_default ? (
            <span
              className="text-primary p-1"
              title={t("models.badge.default")}
            >
              <IconStarFilled className="size-3.5" />
            </span>
          ) : (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => onSetDefault(model)}
              disabled={settingDefault || !canSetDefault}
              title={t("models.action.setDefault")}
            >
              {settingDefault ? (
                <IconLoader2 className="size-3.5 animate-spin" />
              ) : (
                <IconStar className="size-3.5" />
              )}
            </Button>
          )}

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(model)}
            title={t("models.action.edit")}
          >
            <IconEdit className="size-3.5" />
          </Button>

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(model)}
            disabled={model.is_default}
            title={t("models.action.delete")}
            className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            <IconTrash className="size-3.5" />
          </Button>
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
        ) : model.configured && model.api_key ? (
          <span className="text-muted-foreground/70 flex items-center gap-1 font-mono text-[11px]">
            <IconKey className="size-3" />
            {model.api_key}
          </span>
        ) : (
          <span className="text-muted-foreground/50 text-[11px]">
            {t("models.status.unconfigured")}
          </span>
        )}
      </div>
    </div>
  )
}
