import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type AppConfig, getAppConfig, patchAppConfig } from "@/api/channels"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { refreshGatewayState } from "@/store/gateway"

const LOG_LEVEL_OPTIONS = ["debug", "info", "warn", "error", "fatal"] as const
type GatewayLogLevel = (typeof LOG_LEVEL_OPTIONS)[number]

const LOG_LEVEL_LABELS: Record<GatewayLogLevel, string> = {
  debug: "Debug",
  info: "Info",
  warn: "Warn",
  error: "Error",
  fatal: "Fatal",
}

function getGatewayLogLevel(config: AppConfig | undefined): GatewayLogLevel {
  const gateway = config?.gateway
  if (typeof gateway === "object" && gateway !== null) {
    const logLevel = (gateway as Record<string, unknown>).log_level
    if (
      typeof logLevel === "string" &&
      LOG_LEVEL_OPTIONS.includes(logLevel as GatewayLogLevel)
    ) {
      return logLevel as GatewayLogLevel
    }
  }
  return "warn"
}

export function LogLevelSelect() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [logLevel, setLogLevel] = useState<GatewayLogLevel>("warn")
  const [savingLogLevel, setSavingLogLevel] = useState(false)

  const { data: configData } = useQuery({
    queryKey: ["config"],
    queryFn: getAppConfig,
  })

  useEffect(() => {
    setLogLevel(getGatewayLogLevel(configData))
  }, [configData])

  const handleLogLevelChange = async (nextValue: string) => {
    const nextLevel = nextValue as GatewayLogLevel
    const previousLevel = logLevel
    setLogLevel(nextLevel)
    setSavingLogLevel(true)

    try {
      await patchAppConfig({
        gateway: {
          log_level: nextLevel,
        },
      })
      await queryClient.invalidateQueries({ queryKey: ["config"] })
      await refreshGatewayState({ force: true })
    } catch (error) {
      setLogLevel(previousLevel)
      toast.error(
        error instanceof Error
          ? error.message
          : t("pages.logs.log_level_error"),
      )
    } finally {
      setSavingLogLevel(false)
    }
  }

  return (
    <div className="flex items-center gap-2">
      <Select
        value={logLevel}
        onValueChange={handleLogLevelChange}
        disabled={savingLogLevel}
      >
        <SelectTrigger size="sm" className="w-28">
          <SelectValue />
        </SelectTrigger>
        <SelectContent align="end">
          {LOG_LEVEL_OPTIONS.map((level) => (
            <SelectItem key={level} value={level}>
              {LOG_LEVEL_LABELS[level]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
