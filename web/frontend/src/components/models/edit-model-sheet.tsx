import { IconLoader2 } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ModelInfo,
  type ModelProviderOption,
  setDefaultModel,
  updateModel,
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

interface EditForm {
  provider: string
  modelId: string
  apiKey: string
  apiBase: string
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

interface EditModelSheetProps {
  model: ModelInfo | null
  providerOptions: ModelProviderOption[]
  open: boolean
  onClose: () => void
  onSaved: () => void
}

function buildInitialEditForm(model: ModelInfo): EditForm {
  return {
    provider: model.provider ?? "",
    modelId: model.model,
    apiKey: "",
    apiBase: model.api_base ?? "",
    proxy: model.proxy ?? "",
    authMethod: model.auth_method ?? "",
    connectMode: model.connect_mode ?? "",
    workspace: model.workspace ?? "",
    rpm: model.rpm ? String(model.rpm) : "",
    maxTokensField: model.max_tokens_field ?? "",
    requestTimeout: model.request_timeout ? String(model.request_timeout) : "",
    thinkingLevel: model.thinking_level ?? "",
    toolSchemaTransform: model.tool_schema_transform ?? "", // <-- AGGIUNGI QUESTA RIGA
    extraBody: model.extra_body
      ? JSON.stringify(model.extra_body, null, 2)
      : "",
    customHeaders: model.custom_headers
      ? JSON.stringify(model.custom_headers, null, 2)
      : "",
  }
}

export function EditModelSheet({
  model,
  providerOptions,
  open,
  onClose,
  onSaved,
}: EditModelSheetProps) {
  const { t } = useTranslation()
  const [form, setForm] = useState<EditForm>({
    provider: "",
    modelId: "",
    apiKey: "",
    apiBase: "",
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
  })
  const [saving, setSaving] = useState(false)
  const [setAsDefault, setSetAsDefault] = useState(false)
  const [error, setError] = useState("")
  const initialForm = model ? buildInitialEditForm(model) : null
  const sortedProviderOptions = useMemo(
    () => getSortedProviderOptions(providerOptions),
    [providerOptions],
  )
  const currentProviderID = model
    ? (findProviderOption(model.provider, providerOptions)?.id ??
      model.provider?.trim().toLowerCase() ??
      "")
    : ""
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
  const providerError = selectedProviderOption
    ? ""
    : t("models.field.providerInvalid")
  const defaultModelAllowed =
    selectedProviderOption?.default_model_allowed !== false
  const willClearDefaultOnSave =
    model?.is_default === true && defaultModelAllowed === false
  const apiBasePlaceholder =
    getProviderDefaultAPIBase(form.provider, providerOptions) ||
    "https://api.example.com/v1"
  const isDirty =
    model != null &&
    (JSON.stringify(form) !== JSON.stringify(initialForm) ||
      setAsDefault !== model.is_default)

  useEffect(() => {
    if (model) {
      const initialForm = buildInitialEditForm(model)
      const option = findProviderOption(initialForm.provider, providerOptions)
      if (option?.auth_method_locked && !initialForm.authMethod) {
        initialForm.authMethod = option.default_auth_method ?? ""
      }
      setForm(initialForm)
      setSetAsDefault(model.is_default && model.default_model_allowed !== false)
      setError("")
    }
  }, [model, providerOptions])

  const setField =
    (key: keyof EditForm) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      if (error) {
        setError("")
      }
      setForm((f) => ({ ...f, [key]: e.target.value }))
    }

  const setProvider = (value: string) => {
    if (error) {
      setError("")
    }
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
  }

  const handleSave = async () => {
    if (!model) return
    if (!selectedProviderOption) {
      setError(providerError)
      return
    }
    if (!form.modelId.trim()) {
      setError(t("models.add.errorRequired"))
      return
    }
    setSaving(true)
    setError("")
    try {
      await updateModel(model.index, {
        model_name: model.model_name,
        provider: form.provider.trim(),
        model: form.modelId.trim(),
        api_base: form.apiBase || undefined,
        api_key: form.apiKey || undefined,
        proxy: form.proxy || undefined,
        auth_method: authMethodLocked
          ? defaultAuthMethod || undefined
          : form.authMethod || undefined,
        connect_mode: form.connectMode || undefined,
        workspace: form.workspace || undefined,
        rpm: form.rpm ? Number(form.rpm) : undefined,
        max_tokens_field: form.maxTokensField || undefined,
        request_timeout: form.requestTimeout
          ? Number(form.requestTimeout)
          : undefined,
        thinking_level: form.thinkingLevel || undefined,
        tool_schema_transform: form.toolSchemaTransform.trim() || undefined,
        extra_body: form.extraBody.trim()
          ? JSON.parse(form.extraBody.trim())
          : {},
        custom_headers: form.customHeaders.trim()
          ? JSON.parse(form.customHeaders.trim())
          : {},
      })
      if (setAsDefault && !model.is_default) {
        await setDefaultModel(model.model_name)
      }
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("models.edit.saveSuccess"),
        model.model_name,
        gateway?.restartRequired === true,
      )
      onSaved()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : t("models.edit.saveError"))
    } finally {
      setSaving(false)
    }
  }

  const isOAuth = effectiveAuthMethod === "oauth"
  const hasSavedAPIKey = Boolean(model?.api_key)
  const apiKeyPlaceholder = hasSavedAPIKey
    ? maskedSecretPlaceholder(
        model?.api_key ?? "",
        t("models.field.apiKeyPlaceholderSet"),
      )
    : t("models.field.apiKeyPlaceholder")

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[560px] data-[side=right]:sm:!max-w-[560px]"
      >
        <SheetHeader className="border-b-muted border-b px-6 py-5">
          <SheetTitle className="text-base">
            {t("models.edit.title", { name: model?.model_name })}
          </SheetTitle>
          <SheetDescription className="font-mono text-xs">
            {model?.model}
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field
              label={t("models.field.provider")}
              hint={t("models.field.providerHint")}
              error={providerError}
              required
            >
              <Select
                value={selectedProviderOption?.id}
                onValueChange={setProvider}
              >
                <SelectTrigger
                  className="w-full"
                  aria-invalid={!!providerError}
                >
                  <SelectValue
                    placeholder={t("models.field.providerPlaceholder")}
                  />
                </SelectTrigger>
                <SelectContent>
                  {sortedProviderOptions.map((option) => (
                    <SelectItem
                      key={option.id}
                      value={option.id}
                      disabled={
                        !option.create_allowed &&
                        option.id !== currentProviderID
                      }
                    >
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
                value={form.modelId}
                onChange={setField("modelId")}
                placeholder={t("models.add.modelIdPlaceholder")}
                className="font-mono text-sm"
              />
            </Field>

            {!isOAuth && (
              <Field
                label={t("models.field.apiKey")}
                hint={hasSavedAPIKey ? t("models.edit.apiKeyHint") : undefined}
              >
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
                willClearDefaultOnSave
                  ? t("models.defaultOnSave.clearOnSave")
                  : defaultModelAllowed
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

            {error && (
              <p className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                {error}
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
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
