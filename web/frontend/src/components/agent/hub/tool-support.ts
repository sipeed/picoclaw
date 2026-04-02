import type { TFunction } from "i18next"

import type { ToolSupportItem } from "@/api/tools"

type MarketplaceTool = Pick<ToolSupportItem, "status" | "reason_code"> | undefined

export interface UnavailableToolMessage {
  key: "search" | "install"
  label: string
  message: string
}

export function buildUnavailableToolMessages({
  searchTool,
  installTool,
  t,
}: {
  searchTool: MarketplaceTool
  installTool: MarketplaceTool
  t: TFunction
}): UnavailableToolMessage[] {
  const searchMessage = getToolSupportMessage(searchTool, t)
  const installMessage = getToolSupportMessage(installTool, t)

  return [
    searchMessage
      ? {
          key: "search",
          label: t("pages.agent.skills.marketplace_search_status"),
          message: searchMessage,
        }
      : null,
    installMessage
      ? {
          key: "install",
          label: t("pages.agent.skills.marketplace_install_status"),
          message: installMessage,
        }
      : null,
  ].filter((item): item is UnavailableToolMessage => Boolean(item))
}

function getToolSupportMessage(
  tool: MarketplaceTool,
  t: TFunction,
): string | null {
  if (!tool || tool.status === "enabled") {
    return null
  }
  if (tool.reason_code) {
    return `${t(`pages.agent.tools.reasons.${tool.reason_code}`)} ${t("pages.agent.skills.marketplace_status_enable_hint")}`
  }
  return t("pages.agent.skills.marketplace_status_disabled")
}
