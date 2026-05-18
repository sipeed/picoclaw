import {
  IconDownload,
  IconLoader2,
  IconPlugConnected,
} from "@tabler/icons-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ModelInfo,
  type ModelProviderOption,
  getCatalogs,
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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

import { FetchModelsDialog } from "./fetch-models-dialog"
import { type FieldValidation, validateModelField } from "./model-validation"
import { ProviderCombobox } from "./provider-combobox"
import { getProviderKey } from "./provider-label"
import { FETCHABLE_PROVIDER_KEYS, PROVIDER_API_BASES, PROVIDER_MAP } from "./provider-registry"
import { TestModelDialog } from "./test-model-dialog"

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
  open: boolean
  onClose: () => void
  onSaved: () => void
  providerOptions?: ModelProviderOption[]
}

function normalizeApiBase(value: string): string {
  return value.trim().replace(/\/+$/, "")
}

function getNextApiBaseForProviderChange(
  currentApiBase: string,
  currentProvider: string,
  nextProvider: string,
): string {
  const normalizedCurrentApiBase = normalizeApiBase(currentApiBase)
  const currentDefaultApiBase = normalizeApiBase(
    PROVIDER_API_BASES[currentProvider] || "",
  )
  const nextDefaultApiBase = PROVIDER_API_BASES[nextProvider] || ""

  if (!normalizedCurrentApiBase) {
    return nextDefaultApiBase
  }

  if (
    normalizedCurrentApiBase &&
    currentDefaultApiBase &&
    normalizedCurrentApiBase === currentDefaultApiBase
  ) {
    return nextDefaultApiBase
  }

  return currentApiBase
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
  open,
  onClose,
  onSaved,
  providerOptions,
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
  const [modelValidation, setModelValidation] =
    useState<FieldValidation | null>(null)
  const [testOpen, setTestOpen] = useState(false)
  const [fetchOpen, setFetchOpen] = useState(false)
  const [fetchedModels, setFetchedModels] = useState<string[]>([])
  const [catalogModels, setCatalogModels] = useState<string[]>([])
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const scrollContainerRef = useRef<HTMLDivElement>(null)

  const initialForm = model ? buildInitialEditForm(model) : null
  const isDirty =
    model != null &&
    (JSON.stringify(form) !== JSON.stringify(initialForm) ||
      setAsDefault !== model.is_default)

  useEffect(() => {
    if (model) {
      setForm(buildInitialEditForm(model))
      setSetAsDefault(model.is_default)
      setError("")
      setModelValidation(null)
      setFetchedModels([])
      setCatalogModels([])
      // Load matching catalog models
      const providerKey = getProviderKey(model.provider || undefined)
      const apiBase = (model.api_base ?? "").trim().replace(/\/+$/, "")
      getCatalogs()
        .then((res) => {
          const matched = (res.entries || []).filter((e) => {
            const ep = getProviderKey(e.provider || undefined)
            const eb = (e.api_base ?? "").trim().replace(/\/+$/, "")
            return ep === providerKey && eb === apiBase
          })
          const ids = matched.flatMap((e) => e.models.map((m) => m.id))
          const unique = [...new Set(ids)]
          if (unique.length > 0) setCatalogModels(unique)
        })
        .catch(() => {})
    }
  }, [model])

  const setField =
    (key: keyof EditForm) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [key]: e.target.value }))

  const debouncedValidateModel = useCallback(
    (value: string, provider: string) => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      debounceRef.current = setTimeout(() => {
        const result = validateModelField(value, provider || undefined)
        setModelValidation(result)
      }, 300)
    },
    [],
  )

  const handleModelChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    setForm((f) => ({ ...f, modelId: value }))
    debouncedValidateModel(value, form.provider)
  }

  const handleProviderChange = (provider: string) => {
    setForm((f) => ({
      ...f,
      provider,
      apiBase: getNextApiBaseForProviderChange(f.apiBase, f.provider, provider),
    }))
    if (form.modelId) {
      debouncedValidateModel(form.modelId, provider)
    }
    const allowed = providerOptions?.find((o) => o.id === provider)?.default_model_allowed ?? false
    if (!allowed) {
      setSetAsDefault(false)
    }
  }

  const applyFix = () => {
    if (modelValidation?.fix) {
      setForm((f) => ({ ...f, modelId: modelValidation.fix! }))
      setModelValidation(null)
    }
  }

  const handleCommonModel = (modelId: string) => {
    setForm((f) => ({ ...f, modelId }))
    setModelValidation(null)
  }

  const handleFetchFill = (models: string[]) => {
    setFetchedModels(models)
    if (models.length >= 1) {
      setForm((f) => ({ ...f, modelId: models[0] }))
      setModelValidation(null)
    }
  }

  const providerDef = PROVIDER_MAP.get(form.provider)
  const commonModels = providerDef?.commonModels || []
  const defaultModelAllowed = form.provider
    ? (providerOptions?.find((o) => o.id === form.provider)?.default_model_allowed ?? false)
    : false

  const handleSave = async () => {
    if (!model) return
    if (!form.modelId.trim()) {
      setError(t("models.add.errorRequired"))
      return
    }
    if (modelValidation?.level === "error") return

    let extraBody: Record<string, unknown> | undefined
    let customHeaders: Record<string, string> | undefined
    try {
      if (form.extraBody.trim()) {
        extraBody = JSON.parse(form.extraBody.trim())
      } else {
        extraBody = {}
      }
    } catch {
      setError(
        t("models.field.extraBody") + ": " + t("models.field.invalidJson"),
      )
      return
    }
    try {
      if (form.customHeaders.trim()) {
        customHeaders = JSON.parse(form.customHeaders.trim())
      } else {
        customHeaders = {}
      }
    } catch {
      setError(
        t("models.field.customHeaders") + ": " + t("models.field.invalidJson"),
      )
      return
    }

    setSaving(true)
    setError("")
    try {
      const modelId = form.modelId.trim()
      const provider = form.provider.trim()
      await updateModel(model.index, {
        model_name: model.model_name,
        provider: provider,
        model: modelId,
        api_base: form.apiBase || undefined,
        api_key: form.apiKey || undefined,
        proxy: form.proxy || undefined,
        auth_method: form.authMethod || undefined,
        connect_mode: form.connectMode || undefined,
        workspace: form.workspace || undefined,
        rpm: form.rpm ? Number(form.rpm) : undefined,
        max_tokens_field: form.maxTokensField || undefined,
        request_timeout: form.requestTimeout
          ? Number(form.requestTimeout)
          : undefined,
        thinking_level: form.thinkingLevel || undefined,
        tool_schema_transform: form.toolSchemaTransform.trim() || undefined,
        extra_body: extraBody,
        custom_headers: customHeaders,
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

  const isOAuth = model?.auth_method === "oauth"
  const hasSavedAPIKey = Boolean(model?.api_key)
  const apiKeyPlaceholder = hasSavedAPIKey
    ? maskedSecretPlaceholder(
        model?.api_key ?? "",
        t("models.field.apiKeyPlaceholderSet"),
      )
    : t("models.field.apiKeyPlaceholder")

  return (
    <>
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

          <div className="min-h-0 flex-1 overflow-y-auto" ref={scrollContainerRef}>
            <div className="space-y-5 px-6 py-5">
              <Field
                label={t("models.field.provider")}
                hint={t("models.field.providerHint")}
              >
                <ProviderCombobox
                  value={form.provider}
                  onChange={handleProviderChange}
                  placeholder={t("models.field.providerPlaceholder")}
                  backendOptions={providerOptions}
                  containerRef={scrollContainerRef}
                />
              </Field>

              <Field
                label={t("models.add.modelId")}
                hint={t("models.add.modelIdHint")}
              >
                <Input
                  value={form.modelId}
                  onChange={handleModelChange}
                  placeholder={
                    providerDef
                      ? `${commonModels[0] || "model-name"}`
                      : t("models.add.modelIdPlaceholder")
                  }
                  className="font-mono text-sm"
                  aria-invalid={!!error || modelValidation?.level === "error"}
                />
                {modelValidation && modelValidation.messageKey && (
                  <div
                    className={`flex items-center gap-2 text-xs ${
                      modelValidation.level === "error"
                        ? "text-destructive"
                        : modelValidation.level === "warning"
                          ? "text-yellow-600 dark:text-yellow-500"
                          : "text-green-600 dark:text-green-500"
                    }`}
                  >
                    <span>
                      {t(
                        modelValidation.messageKey,
                        modelValidation.messageParams,
                      )}
                    </span>
                    {modelValidation.fix && (
                      <button
                        type="button"
                        onClick={applyFix}
                        className="text-primary underline hover:no-underline"
                      >
                        {t("common.fix")}
                      </button>
                    )}
                  </div>
                )}
                {commonModels.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {commonModels.map((m) => (
                      <Badge
                        key={m}
                        variant="secondary"
                        className="hover:bg-secondary/80 cursor-pointer font-mono text-xs"
                        onClick={() => handleCommonModel(m)}
                      >
                        {m}
                      </Badge>
                    ))}
                  </div>
                )}
                {catalogModels.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {catalogModels.map((m) => (
                      <Badge
                        key={m}
                        variant={form.modelId === m ? "default" : "outline"}
                        className="cursor-pointer font-mono text-xs"
                        onClick={() => handleCommonModel(m)}
                      >
                        {m}
                      </Badge>
                    ))}
                  </div>
                )}
                {fetchedModels.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {fetchedModels.map((m) => (
                      <Badge
                        key={m}
                        variant={form.modelId === m ? "default" : "outline"}
                        className="cursor-pointer font-mono text-xs"
                        onClick={() => handleCommonModel(m)}
                      >
                        {m}
                      </Badge>
                    ))}
                  </div>
                )}
                <div className="flex items-center gap-2">
                  {form.provider && FETCHABLE_PROVIDER_KEYS.has(form.provider) && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setFetchOpen(true)}
                    >
                      <IconDownload className="size-3" />
                      {t("models.fetch.title")}
                    </Button>
                  )}
                </div>
              </Field>

              {!isOAuth && (
                <Field
                  label={t("models.field.apiKey")}
                  hint={
                    hasSavedAPIKey ? t("models.edit.apiKeyHint") : undefined
                  }
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
                  placeholder="https://api.example.com/v1"
                  disabled={isOAuth}
                />
              </Field>

              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setTestOpen(true)}
                  disabled={!model}
                >
                  <IconPlugConnected className="size-4" />
                  {t("models.test.testConnection")}
                </Button>
              </div>

              <SwitchCardField
                label={t("models.defaultOnSave.label")}
                hint={
                  !defaultModelAllowed
                    ? t("models.defaultOnSave.unsupportedProvider")
                    : t("models.defaultOnSave.description")
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
                  hint={t("models.field.authMethodHint")}
                >
                  <Input
                    value={form.authMethod}
                    onChange={setField("authMethod")}
                    placeholder="oauth"
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
            <Button
              onClick={handleSave}
              disabled={
                !isDirty || saving || modelValidation?.level === "error"
              }
            >
              {saving && <IconLoader2 className="size-4 animate-spin" />}
              {t("common.save")}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <TestModelDialog
        model={model}
        open={testOpen}
        onClose={() => setTestOpen(false)}
        inlineParams={{
          provider: form.provider,
          model: form.modelId,
          apiBase: form.apiBase,
          apiKey: form.apiKey,
          authMethod: form.authMethod,
          modelIndex: model?.index,
        }}
      />

      <FetchModelsDialog
        open={fetchOpen}
        onClose={() => setFetchOpen(false)}
        onFill={handleFetchFill}
        provider={form.provider}
        apiKey={form.apiKey}
        apiBase={form.apiBase}
      />
    </>
  )
}
