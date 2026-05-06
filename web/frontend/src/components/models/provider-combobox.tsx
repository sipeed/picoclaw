import { IconCheck, IconChevronDown } from "@tabler/icons-react"
import { useState } from "react"
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
import { Input } from "@/components/ui/input"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { cn } from "@/lib/utils"

import { ProviderIcon } from "./provider-icon"
import { KNOWN_PROVIDER_KEYS, PROVIDERS } from "./provider-registry"

interface ProviderComboboxProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

export function ProviderCombobox({
  value,
  onChange,
  placeholder,
}: ProviderComboboxProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [customMode, setCustomMode] = useState(false)
  const [customValue, setCustomValue] = useState("")

  const sorted = [...PROVIDERS].sort((a, b) => b.priority - a.priority)
  const selected = sorted.find((p) => p.key === value)
  const isCustom = value && !KNOWN_PROVIDER_KEYS.has(value)

  const handleSelect = (currentValue: string) => {
    if (currentValue === "__custom__") {
      setCustomMode(true)
      setCustomValue(isCustom ? value : "")
      return
    }
    onChange(currentValue === value ? "" : currentValue)
    setCustomMode(false)
    setOpen(false)
  }

  const handleCustomConfirm = () => {
    const trimmed = customValue.trim()
    if (trimmed) {
      onChange(trimmed)
    }
    setCustomMode(false)
    setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (!v) setCustomMode(false)
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
                providerKey={selected.key}
                providerLabel={selected.label}
              />
              {selected.labelZh || selected.label}
            </span>
          ) : isCustom ? (
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
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
        {customMode ? (
          <div className="flex flex-col gap-2 p-2">
            <Input
              value={customValue}
              onChange={(e) => setCustomValue(e.target.value)}
              placeholder={t("models.combobox.customPlaceholder")}
              className="h-8 font-mono text-sm"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") handleCustomConfirm()
                if (e.key === "Escape") {
                  setCustomMode(false)
                  setOpen(false)
                }
              }}
            />
            <div className="flex gap-1">
              <Button
                variant="ghost"
                size="sm"
                className="h-7 flex-1 text-xs"
                onClick={() => {
                  setCustomMode(false)
                  setOpen(false)
                }}
              >
                {t("common.cancel")}
              </Button>
              <Button
                size="sm"
                className="h-7 flex-1 text-xs"
                onClick={handleCustomConfirm}
                disabled={!customValue.trim()}
              >
                {t("common.confirm")}
              </Button>
            </div>
          </div>
        ) : (
          <Command>
            <CommandInput placeholder={t("models.combobox.searchProvider")} />
            <CommandList>
              <CommandEmpty>{t("models.combobox.noProvider")}</CommandEmpty>
              <CommandGroup>
                {sorted.map((provider) => (
                  <CommandItem
                    key={provider.key}
                    value={provider.key}
                    keywords={[
                      provider.label,
                      provider.labelZh || "",
                      ...(provider.aliases || []),
                    ]}
                    onSelect={handleSelect}
                  >
                    <span className="flex items-center gap-2">
                      <ProviderIcon
                        providerKey={provider.key}
                        providerLabel={provider.label}
                      />
                      <span>{provider.labelZh || provider.label}</span>
                      {provider.isLocal && (
                        <span className="text-muted-foreground text-xs">
                          {t("models.combobox.local")}
                        </span>
                      )}
                    </span>
                    <IconCheck
                      className={cn(
                        "ml-auto size-4",
                        value === provider.key ? "opacity-100" : "opacity-0",
                      )}
                    />
                  </CommandItem>
                ))}
                <CommandItem
                  value="__custom__"
                  keywords={["custom", "自定义"]}
                  onSelect={handleSelect}
                >
                  <span className="text-muted-foreground italic">
                    {t("models.combobox.custom")}
                  </span>
                  {isCustom && (
                    <IconCheck className="ml-auto size-4 opacity-100" />
                  )}
                </CommandItem>
              </CommandGroup>
            </CommandList>
          </Command>
        )}
      </PopoverContent>
    </Popover>
  )
}
