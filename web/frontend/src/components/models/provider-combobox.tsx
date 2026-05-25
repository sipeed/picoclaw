import { IconCheck, IconChevronDown } from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { cn } from "@/lib/utils"

import { ProviderIcon } from "./provider-icon"
import {
  getCanonicalProviderKey,
  type ProviderCatalogEntry,
  getProviderCatalog,
} from "./provider-registry"
import type { ModelProviderOption } from "@/api/models"

interface ProviderComboboxProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  backendOptions?: ModelProviderOption[]
  /** When true, only show providers with create_allowed from the backend. */
  filterCreateAllowed?: boolean
  /** Container element for the popover portal. Use to avoid scroll conflicts inside dialogs/sheets. */
  containerRef?: React.RefObject<HTMLElement | null>
}

export function ProviderCombobox({
  value,
  onChange,
  placeholder,
  backendOptions,
  filterCreateAllowed,
  containerRef,
}: ProviderComboboxProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [containerEl, setContainerEl] = useState<HTMLElement | null>(null)

  useEffect(() => {
    setContainerEl(containerRef?.current ?? null)
  }, [containerRef])

  const canonicalValue = getCanonicalProviderKey(value, backendOptions)
  const allProviders: ProviderCatalogEntry[] = getProviderCatalog(backendOptions)
  const visible = filterCreateAllowed
    ? allProviders.filter((p) => p.createAllowed || p.key === canonicalValue)
    : allProviders
  const allKeys = new Set(allProviders.map((p) => p.key))
  const selected = allProviders.find((p) => p.key === canonicalValue)
  const showUnknownValue = value && !allKeys.has(canonicalValue)

  const handleSelect = (currentValue: string) => {
    onChange(currentValue === canonicalValue ? "" : currentValue)
    setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(isOpen: boolean) => {
        setOpen(isOpen)
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
        >
          {selected ? (
            <span className="flex items-center gap-2">
              <ProviderIcon
                provider={selected}
              />
              {selected.label}
            </span>
          ) : showUnknownValue ? (
            <span className="flex items-center gap-2 font-mono text-sm">
              {value}
            </span>
          ) : (
            <span className="text-muted-foreground">
              {placeholder || t("models.combobox.selectProvider")}
            </span>
          )}
          <IconChevronDown className="ml-2 size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0" container={containerEl}>
        <Command>
          <CommandInput placeholder={t("models.combobox.searchProvider")} />
          <CommandList>
            <CommandEmpty>
              {backendOptions && backendOptions.length > 0
                ? t("models.combobox.noProvider")
                : t("models.combobox.noCatalog")}
            </CommandEmpty>
            <CommandGroup>
              {visible.map((provider) => {
                const disabled = !provider.createAllowed && provider.key !== value

                return (
                  <CommandItem
                    key={provider.key}
                    value={provider.key}
                    keywords={[
                      provider.label,
                      ...provider.aliases,
                    ]}
                    onSelect={handleSelect}
                    disabled={disabled}
                  >
                    <span className="flex items-center gap-2">
                      <ProviderIcon
                        provider={provider}
                      />
                      <span>{provider.label}</span>
                      {provider.isLocal && (
                         <span className="text-muted-foreground text-xs">
                           {t("models.combobox.local")}
                         </span>
                      )}
                    </span>
                    <IconCheck
                      className={cn(
                        "ml-auto size-4",
                        canonicalValue === provider.key ? "opacity-100" : "opacity-0",
                      )}
                    />
                  </CommandItem>
                )
              })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
