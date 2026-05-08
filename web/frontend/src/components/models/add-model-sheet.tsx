import { IconLoader2 } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ModelProviderOption,
  addModel,
  setDefaultModel,
} from "@/api/models"
import { ConfigChangeNotice } from "@/components/config-change-notice"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import {
  AdvancedSection,
  Field,
  KeyInput,
  SwitchCardField,
} from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

import {
  findProviderOption,
  getProviderDefaultAPIBase,
  getProviderDefaultAuthMethod,
  getProviderLabel,
  getSortedProviderOptions,
  isProviderAuthMethodLocked,
} from "./provider-label"

interface AddForm {
  modelName: string
  provider: string
  model: string
  apiBase: string
  apiKey: string
  proxy: string
  authMethod: string
  connectMode: string
  workspace: string
  rpm: string
  maxTokensField: string
  requestTimeout: string
  thinkingLevel: string
  toolSchemaTransform: string
  extraBody: string
  customHeaders: string
}

const EMPTY_ADD_FORM: AddForm = {
  modelName: "",
  provider: "openai",
  model: "",
  apiBase: "",
  apiKey: "",
  proxy: "",
  authMethod: "",
  connectMode: "",
  workspace: "",
  rpm: "",
  maxTokensField: "",
  requestTimeout: "",
  thinkingLevel: "",
  toolSchemaTransform: "",
  extraBody: "",
  customHeaders: "",
}

interface AddModelSheetProps {
  open: boolean
  onClose: () => void
  onSaved: () => void
  existingModelNames: string[]
  providerOptions: ModelProviderOption[]
}

export function AddModelSheet({
  open,
  onClose,
  onSaved,
  existingModelNames,
  providerOptions,
}: AddModelSheetProps) {
  const { t } = useTranslation()
  const [form, setForm] = useState<AddForm>(EMPTY_ADD_FORM)
  const [saving, setSaving] = useState(false)
  const [setAsDefault, setSetAsDefault] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<
    Partial<Record<keyof AddForm, string>>
  >({})
  const [serverError, setServerError] = useState("")
  const apiKeyPlaceholder = maskedSecretPlaceholder(
    form.apiKey,
    t("models.field.apiKeyPlaceholder"),
  )
  const sortedProviderOptions = useMemo(
    () => getSortedProviderOptions(providerOptions),
    [providerOptions],
  )
  const creatableProviderOptions = useMemo(
    () => sortedProviderOptions.filter((option) => option.create_allowed),
    [sortedProviderOptions],
  )
  const selectedProviderOption = findProviderOption(
    form.provider,
    providerOptions,
  )
  const authMethodLocked = isProviderAuthMethodLocked(
    form.provider,
    providerOptions,
  )
  const defaultAuthMethod = getProviderDefaultAuthMethod(
    form.provider,
    providerOptions,
  )
  const effectiveAuthMethod = (
    authMethodLocked ? defaultAuthMethod : form.authMethod
  )
    .trim()
    .toLowerCase()
  const isOAuth = effectiveAuthMethod === "oauth"
  const defaultModelAllowed =
    selectedProviderOption?.default_model_allowed !== false
  const apiBasePlaceholder =
    getProviderDefaultAPIBase(form.provider, providerOptions) ||
    "https://api.example.com/v1"
  const isDirty =
    JSON.stringify(form) !== JSON.stringify(EMPTY_ADD_FORM) || setAsDefault

  useEffect(() => {
    if (open) {
      setForm(EMPTY_ADD_FORM)
      setSetAsDefault(false)
      setFieldErrors({})
      setServerError("")
    }
  }, [open])

  const validate = (): boolean => {
    const errors: Partial<Record<keyof AddForm, string>> = {}
    const modelName = form.modelName.trim()
    if (!modelName) {
      errors.modelName = t("models.add.errorRequired")
    } else if (existingModelNames.some((name) => name.trim() === modelName)) {
      errors.modelName = t("models.add.errorDuplicateModelName")
    }
    if (!selectedProviderOption) {
      errors.provider = t("models.field.providerInvalid")
    }
    if (!form.model.trim()) errors.model = t("models.add.errorRequired")
    setFieldErrors(errors)
    return Object.keys(errors).length === 0
  }

  const setField =
    (key: keyof AddForm) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setForm((f) => ({ ...f, [key]: e.target.value }))
      if (fieldErrors[key]) {
        setFieldErrors((prev) => ({ ...prev, [key]: undefined }))
      }
    }

  const setProvider = (value: string) => {
    setForm((f) => {
      const previousOption = findProviderOption(f.provider, providerOptions)
      const nextOption = findProviderOption(value, providerOptions)
      let authMethod = f.authMethod
      if (nextOption?.auth_method_locked) {
        authMethod = nextOption.default_auth_method ?? ""
      } else if (
        previousOption?.auth_method_locked &&
        f.authMethod === (previousOption.default_auth_method ?? "")
      ) {
        authMethod = ""
      }
      return { ...f, provider: value, authMethod }
    })
    const nextOption = findProviderOption(value, providerOptions)
    if (nextOption?.default_model_allowed === false) {
      setSetAsDefault(false)
    }
    if (fieldErrors.provider) {
      setFieldErrors((prev) => ({ ...prev, provider: undefined }))
    }
  }

  const handleSave = async () => {
    if (!validate()) return
    setSaving(true)
    setServerError("")
    try {
      const modelName = form.modelName.trim()
      const modelId = form.model.trim()
      await addModel({
        model_name: modelName,
        provider: form.provider.trim(),
        model: modelId,
        api_base: form.apiBase.trim() || undefined,
        api_key: form.apiKey.trim() || undefined,
        proxy: form.proxy.trim() || undefined,
        auth_method: authMethodLocked
          ? defaultAuthMethod || undefined
          : form.authMethod.trim() || undefined,
        connect_mode: form.connectMode.trim() || undefined,
        workspace: form.workspace.trim() || undefined,
        rpm: form.rpm ? Number(form.rpm) : undefined,
        max_tokens_field: form.maxTokensField.trim() || undefined,
        request_timeout: form.requestTimeout
          ? Number(form.requestTimeout)
          : undefined,
        thinking_level: form.thinkingLevel.trim() || undefined,
        tool_schema_transform: form.toolSchemaTransform.trim() || undefined,
        extra_body: form.extraBody.trim()
          ? JSON.parse(form.extraBody.trim())
          : undefined,
        custom_headers: form.customHeaders.trim()
          ? JSON.parse(form.customHeaders.trim())
          : undefined,
      })
      if (setAsDefault) {
        await setDefaultModel(modelName)
      }
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("models.add.saveSuccess"),
        modelName,
        gateway?.restartRequired === true,
      )
      onSaved()
      onClose()
    } catch (e) {
      setServerError(e instanceof Error ? e.message : t("models.add.saveError"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[560px] data-[side=right]:sm:!max-w-[560px]"
      >
        <SheetHeader className="border-b-muted border-b px-6 py-5">
          <SheetTitle className="text-base">{t("models.add.title")}</SheetTitle>
          <SheetDescription className="text-xs">
            {t("models.add.description")}
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field
              label={t("models.add.modelName")}
              hint={t("models.add.modelNameHint")}
            >
              <Input
                value={form.modelName}
                onChange={setField("modelName")}
                placeholder={t("models.add.modelNamePlaceholder")}
                aria-invalid={!!fieldErrors.modelName}
              />
              {fieldErrors.modelName && (
                <p className="text-destructive text-xs">
                  {fieldErrors.modelName}
                </p>
              )}
            </Field>

            <Field
              label={t("models.field.provider")}
              hint={t("models.field.providerHint")}
              error={fieldErrors.provider}
              required
            >
              <Select
                value={selectedProviderOption?.id}
                onValueChange={setProvider}
              >
                <SelectTrigger
                  className="w-full"
                  aria-invalid={!!fieldErrors.provider}
                >
                  <SelectValue
                    placeholder={t("models.field.providerPlaceholder")}
                  />
                </SelectTrigger>
                <SelectContent>
                  {creatableProviderOptions.map((option) => (
                    <SelectItem key={option.id} value={option.id}>
                      {getProviderLabel(option.id)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field
              label={t("models.add.modelId")}
              hint={t("models.add.modelIdHint")}
            >
              <Input
                value={form.model}
                onChange={setField("model")}
                placeholder={t("models.add.modelIdPlaceholder")}
                className="font-mono text-sm"
                aria-invalid={!!fieldErrors.model}
              />
              {fieldErrors.model && (
                <p className="text-destructive text-xs">{fieldErrors.model}</p>
              )}
            </Field>

            {!isOAuth && (
              <Field label={t("models.field.apiKey")}>
                <KeyInput
                  value={form.apiKey}
                  onChange={(v) => setForm((f) => ({ ...f, apiKey: v }))}
                  placeholder={apiKeyPlaceholder}
                />
              </Field>
            )}

            <Field
              label={t("models.field.apiBase")}
              hint={isOAuth ? t("models.edit.oauthNote") : undefined}
            >
              <Input
                value={form.apiBase}
                onChange={setField("apiBase")}
                placeholder={apiBasePlaceholder}
                disabled={isOAuth}
              />
            </Field>

            <SwitchCardField
              label={t("models.defaultOnSave.label")}
              hint={
                defaultModelAllowed
                  ? t("models.defaultOnSave.description")
                  : t("models.defaultOnSave.unsupportedProvider")
              }
              checked={setAsDefault}
              onCheckedChange={setSetAsDefault}
              disabled={!defaultModelAllowed}
            />

            <AdvancedSection>
              <Field
                label={t("models.field.proxy")}
                hint={t("models.field.proxyHint")}
              >
                <Input
                  value={form.proxy}
                  onChange={setField("proxy")}
                  placeholder="http://127.0.0.1:7890"
                />
              </Field>

              <Field
                label={t("models.field.authMethod")}
                hint={
                  authMethodLocked
                    ? t("models.field.authMethodManagedHint")
                    : t("models.field.authMethodHint")
                }
              >
                <Input
                  value={authMethodLocked ? defaultAuthMethod : form.authMethod}
                  onChange={setField("authMethod")}
                  placeholder="oauth"
                  disabled={authMethodLocked}
                />
              </Field>

              <Field
                label={t("models.field.connectMode")}
                hint={t("models.field.connectModeHint")}
              >
                <Input
                  value={form.connectMode}
                  onChange={setField("connectMode")}
                  placeholder="stdio"
                />
              </Field>

              <Field
                label={t("models.field.workspace")}
                hint={t("models.field.workspaceHint")}
              >
                <Input
                  value={form.workspace}
                  onChange={setField("workspace")}
                  placeholder="/path/to/workspace"
                />
              </Field>

              <Field
                label={t("models.field.requestTimeout")}
                hint={t("models.field.requestTimeoutHint")}
              >
                <Input
                  value={form.requestTimeout}
                  onChange={setField("requestTimeout")}
                  placeholder="60"
                  type="number"
                  min={0}
                />
              </Field>

              <Field
                label={t("models.field.rpm")}
                hint={t("models.field.rpmHint")}
              >
                <Input
                  value={form.rpm}
                  onChange={setField("rpm")}
                  placeholder="60"
                  type="number"
                  min={0}
                />
              </Field>

              <Field
                label={t("models.field.thinkingLevel")}
                hint={t("models.field.thinkingLevelHint")}
              >
                <Input
                  value={form.thinkingLevel}
                  onChange={setField("thinkingLevel")}
                  placeholder="off"
                />
              </Field>

              <Field
                label={t("models.field.maxTokensField")}
                hint={t("models.field.maxTokensFieldHint")}
              >
                <Input
                  value={form.maxTokensField}
                  onChange={setField("maxTokensField")}
                  placeholder="max_completion_tokens"
                />
              </Field>

              <Field
                label={t("models.field.toolSchemaTransform")}
                hint={t("models.field.toolSchemaTransformHint")}
              >
                <Input
                  value={form.toolSchemaTransform}
                  onChange={setField("toolSchemaTransform")}
                  placeholder="google"
                />
              </Field>

              <Field
                label={t("models.field.extraBody")}
                hint={t("models.field.extraBodyHint")}
              >
                <Textarea
                  value={form.extraBody}
                  onChange={setField("extraBody")}
                  placeholder='{"key": "value"}'
                  rows={3}
                />
              </Field>

              <Field
                label={t("models.field.customHeaders")}
                hint={t("models.field.customHeadersHint")}
              >
                <Textarea
                  value={form.customHeaders}
                  onChange={setField("customHeaders")}
                  placeholder='{"X-Source": "coding-plan"}'
                  rows={3}
                />
              </Field>
            </AdvancedSection>

            {serverError && (
              <p className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                {serverError}
              </p>
            )}
          </div>
        </div>

        <SheetFooter className="border-t-muted border-t px-6 py-4">
          {isDirty && (
            <ConfigChangeNotice
              kind="save"
              title={t("common.saveChangesTitle")}
              description={t("models.unsavedPrompt")}
            />
          )}
          <Button variant="ghost" onClick={onClose} disabled={saving}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSave} disabled={!isDirty || saving}>
            {saving && <IconLoader2 className="size-4 animate-spin" />}
            {t("models.add.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
