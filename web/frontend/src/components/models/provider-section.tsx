import { IconChevronDown } from "@tabler/icons-react"
import { useState } from "react"

import type { ModelInfo } from "@/api/models"

import { ModelCard } from "./model-card"
import { ProviderIcon } from "./provider-icon"

interface ProviderSectionProps {
  provider: string
  providerKey: string
  models: ModelInfo[]
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  settingDefaultIndex: number | null
}

export function ProviderSection({
  provider,
  providerKey,
  models,
  onEdit,
  onSetDefault,
  onDelete,
  settingDefaultIndex,
}: ProviderSectionProps) {
  const [open, setOpen] = useState(true)

  return (
    <section className="my-8">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="mb-3 grid w-full grid-cols-[1fr_auto_1fr_auto] items-center gap-2 px-1 py-1.5 text-left"
        aria-expanded={open}
      >
        <div className="border-border/40 border-t" />
        <span className="text-foreground/80 text-center text-xs font-semibold tracking-wide uppercase">
          <span className="bg-background inline-flex items-center gap-1.5 px-2">
            <ProviderIcon providerKey={providerKey} providerLabel={provider} />
            {provider}
          </span>
        </span>
        <div className="border-border/40 border-t" />
        <span className="flex justify-end">
          <IconChevronDown
            className={[
              "text-muted-foreground size-4 transition-transform",
              open ? "rotate-180" : "",
            ].join(" ")}
          />
        </span>
      </button>

      {open && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {models.map((model) => (
            <ModelCard
              key={model.model_name}
              model={model}
              onEdit={onEdit}
              onSetDefault={onSetDefault}
              onDelete={onDelete}
              settingDefault={settingDefaultIndex === model.index}
            />
          ))}
        </div>
      )}
    </section>
  )
}
