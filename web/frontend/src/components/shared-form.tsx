import { IconChevronDown, IconEye, IconEyeOff } from "@tabler/icons-react"
import { type ReactNode, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  FieldDescription,
  FieldLabel,
  Field as UiField,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"

interface FieldProps {
  label: string
  hint?: string
  error?: string
  required?: boolean
  children: ReactNode
}

export function Field({ label, hint, error, required, children }: FieldProps) {
  return (
    <UiField className="gap-2.5">
      <div className="space-y-1">
        <FieldLabel>
          {label}
          {required && <span className="text-destructive ml-1">*</span>}
        </FieldLabel>
        {hint && (
          <FieldDescription className="text-xs leading-normal">
            {hint}
          </FieldDescription>
        )}
      </div>
      {children}
      {error && (
        <FieldDescription className="text-destructive text-xs leading-normal">
          {error}
        </FieldDescription>
      )}
    </UiField>
  )
}

interface KeyInputProps {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}

export function KeyInput({ value, onChange, placeholder }: KeyInputProps) {
  const [show, setShow] = useState(false)

  return (
    <div className="relative">
      <Input
        type={show ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="pr-10"
      />
      <button
        type="button"
        onClick={() => setShow((v) => !v)}
        tabIndex={-1}
        className="text-muted-foreground hover:text-foreground absolute top-1/2 right-3 -translate-y-1/2 transition-colors"
      >
        {show ? (
          <IconEyeOff className="size-4" />
        ) : (
          <IconEye className="size-4" />
        )}
      </button>
    </div>
  )
}

interface SwitchCardFieldProps {
  label: string
  hint?: string
  error?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  ariaLabel?: string
  disabled?: boolean
  children?: ReactNode
}

export function SwitchCardField({
  label,
  hint,
  error,
  checked,
  onCheckedChange,
  ariaLabel,
  disabled,
  children,
}: SwitchCardFieldProps) {
  return (
    <div className="border-border/60 bg-background rounded-lg border px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-medium">{label}</p>
          {hint && (
            <p className="text-muted-foreground mt-0.5 text-xs leading-normal">
              {hint}
            </p>
          )}
        </div>
        <Switch
          checked={checked}
          onCheckedChange={onCheckedChange}
          disabled={disabled}
          aria-label={ariaLabel ?? label}
        />
      </div>
      {children && <div className="mt-3">{children}</div>}
      {error && (
        <p className="text-destructive mt-2 text-xs leading-normal">{error}</p>
      )}
    </div>
  )
}

interface AdvancedSectionProps {
  children: ReactNode
}

export function AdvancedSection({ children }: AdvancedSectionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <div className="border-border/50 rounded-lg border">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="hover:bg-muted/40 flex w-full items-center justify-between rounded-lg px-4 py-3 transition-colors"
      >
        <span className="text-muted-foreground text-sm">
          {t("models.advanced.toggle")}
        </span>
        <IconChevronDown
          className={[
            "text-muted-foreground size-4 transition-transform duration-200",
            open ? "rotate-180" : "",
          ].join(" ")}
        />
      </button>
      {open && (
        <div className="border-border/30 space-y-5 border-t px-4 pt-4 pb-4">
          {children}
        </div>
      )}
    </div>
  )
}
