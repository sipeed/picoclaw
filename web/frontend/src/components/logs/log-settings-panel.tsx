import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  archiveNow,
  getRequestLogConfig,
  updateRequestLogConfig,
  type RequestLogConfig,
} from "@/api/stats"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Field } from "@/components/shared-form"

export function LogSettingsPanel() {
  const { t } = useTranslation()
  const [config, setConfig] = useState<RequestLogConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    loadConfig()
  }, [])

  async function loadConfig() {
    try {
      const cfg = await getRequestLogConfig()
      setConfig(cfg)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load config")
    } finally {
      setLoading(false)
    }
  }

  async function handleSave() {
    if (!config) return
    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      await updateRequestLogConfig(config)
      setSuccess(t("pages.config.save_success"))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save config")
    } finally {
      setSaving(false)
    }
  }

  async function handleArchiveNow() {
    try {
      await archiveNow()
      setSuccess(t("pages.logs.archive_success"))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Archive failed")
    }
  }

  function updateField<K extends keyof RequestLogConfig>(
    key: K,
    value: RequestLogConfig[K]
  ) {
    if (!config) return
    setConfig({ ...config, [key]: value })
  }

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <PageHeader title={t("pages.logs.settings_title")} />
        <div className="mt-6 text-muted-foreground">{t("labels.loading")}</div>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="container mx-auto p-6">
        <PageHeader title={t("pages.logs.settings_title")} />
        <div className="mt-6 text-red-500">{error || t("pages.logs.config_unavailable")}</div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("pages.logs.settings_title")} />

      <div className="flex-1 overflow-auto p-4 sm:p-8">
        <div className="mx-auto max-w-2xl space-y-6">
          {error && (
            <div className="rounded-md bg-red-500/10 p-4 text-red-500">{error}</div>
          )}
          {success && (
            <div className="rounded-md bg-green-500/10 p-4 text-green-600">{success}</div>
          )}

          <div className="space-y-4">
            <Field
              label={t("pages.logs.enabled")}
              hint={t("pages.logs.enabled_hint")}
            >
              <Switch
                checked={config.enabled}
                onCheckedChange={(checked) => updateField("enabled", checked)}
              />
            </Field>

            <Field
              label={t("pages.logs.max_file_size")}
              hint={t("pages.logs.max_file_size_hint")}
            >
              <Input
                type="number"
                value={config.max_file_size_mb}
                onChange={(e) => updateField("max_file_size_mb", parseInt(e.target.value) || 0)}
              />
            </Field>

            <Field
              label={t("pages.logs.max_files")}
              hint={t("pages.logs.max_files_hint")}
            >
              <Input
                type="number"
                value={config.max_files}
                onChange={(e) => updateField("max_files", parseInt(e.target.value) || 0)}
              />
            </Field>

            <Field
              label={t("pages.logs.retention_days")}
              hint={t("pages.logs.retention_days_hint")}
            >
              <Input
                type="number"
                value={config.retention_days}
                onChange={(e) => updateField("retention_days", parseInt(e.target.value) || 0)}
              />
            </Field>

            <Field
              label={t("pages.logs.archive_interval")}
              hint={t("pages.logs.archive_interval_hint")}
            >
              <Input
                value={config.archive_interval}
                onChange={(e) => updateField("archive_interval", e.target.value)}
                placeholder="24h"
              />
            </Field>

            <Field
              label={t("pages.logs.compress_archive")}
              hint={t("pages.logs.compress_archive_hint")}
            >
              <Switch
                checked={config.compress_archive}
                onCheckedChange={(checked) => updateField("compress_archive", checked)}
              />
            </Field>

            <Field
              label={t("pages.logs.content_max_length")}
              hint={t("pages.logs.content_max_length_hint")}
            >
              <Input
                type="number"
                value={config.log_content_max_length}
                onChange={(e) => updateField("log_content_max_length", parseInt(e.target.value) || 0)}
              />
            </Field>
          </div>

          <div className="flex items-center gap-4 pt-4">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? t("common.saving") : t("common.save")}
            </Button>
            <Button variant="outline" onClick={handleArchiveNow}>
              {t("pages.logs.archive_now")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
