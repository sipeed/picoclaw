import { useMemo, useState } from "react"

import type { ProviderCatalogEntry } from "./provider-registry"

interface ProviderIconProps {
  provider: Pick<ProviderCatalogEntry, "key" | "label" | "iconSlug" | "domain">
}

export function ProviderIcon({ provider }: ProviderIconProps) {
  const [sourceIndex, setSourceIndex] = useState(0)
  const [loadFailed, setLoadFailed] = useState(false)
  const initial = provider.label.trim().charAt(0).toUpperCase() || "?"
  const iconUrls = useMemo(() => {
    const slug = provider.iconSlug
    const domain = provider.domain
    const urls: string[] = []
    if (slug) {
      urls.push(`https://cdn.simpleicons.org/${slug}`)
    }
    if (domain) {
      urls.push(`https://www.google.com/s2/favicons?domain=${domain}&sz=64`)
    }
    return urls
  }, [provider.domain, provider.iconSlug])

  const iconUrl = iconUrls[sourceIndex]

  if (!iconUrl || loadFailed) {
    return (
      <span className="inline-flex size-4 shrink-0 items-center justify-center rounded-sm border border-black/10 bg-white text-[9px] font-semibold text-black/70 dark:border-white/20 dark:text-white/70">
        {initial}
      </span>
    )
  }

  return (
    <span className="inline-flex size-4 shrink-0 items-center justify-center overflow-hidden rounded-sm border border-black/10 bg-white p-0.5 dark:border-white/20">
      <img
        src={iconUrl}
        alt={`${provider.label} logo`}
        className="size-full object-contain"
        loading="lazy"
        referrerPolicy="no-referrer"
        onError={() => {
          if (sourceIndex < iconUrls.length - 1) {
            setSourceIndex((idx) => idx + 1)
            return
          }
          setLoadFailed(true)
        }}
      />
    </span>
  )
}
