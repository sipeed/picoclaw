import { IconEdit, IconTrash, IconRobot, IconSettings } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { Agent } from "@/api/agents"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

interface AgentCardProps {
  agent: Agent
  onEdit: () => void
  onDelete: () => void
  onToggle: (enabled: boolean) => void
}

export function AgentCard({ agent, onEdit, onDelete, onToggle }: AgentCardProps) {
  const { t } = useTranslation()

  const statusColor = agent.status === "enabled" ? "bg-[#F27D26] text-black" : "bg-white/10 text-white/40"

  return (
    <Card
      className="group relative overflow-hidden transition-all hover:border-[#F27D26]/50 hover:shadow-lg"
      size="sm"
    >
      <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-transparent via-[#F27D26]/30 to-transparent opacity-0 transition-opacity duration-500 group-hover:opacity-100" />

      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <CardTitle className="text-base font-semibold tracking-tight">
                {agent.name}
              </CardTitle>
              <Badge className={cn("text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5", statusColor)}>
                {agent.status}
              </Badge>
            </div>
            <CardDescription className="line-clamp-2 text-sm leading-relaxed">
              {agent.description}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Switch
              checked={agent.status === "enabled"}
              onCheckedChange={onToggle}
              className="data-[state=checked]:bg-[#F27D26]"
            />
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0 space-y-3">
        <div className="flex flex-wrap items-center gap-3 text-xs">
          <div className="flex items-center gap-1 text-white/60">
            <IconRobot className="size-4" />
            <span className="font-mono">{agent.model}</span>
          </div>
          {agent.tool_permissions.length > 0 && (
            <div className="flex items-center gap-1 text-white/60">
               <IconSettings className="size-4" />
              <span>{agent.tool_permissions.slice(0, 3).join(", ")}</span>
              {agent.tool_permissions.length > 3 && (
                <span className="text-white/40">+{agent.tool_permissions.length - 3}</span>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 pt-2 border-t border-white/5">
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={onEdit}
            className="text-white/60 hover:text-white hover:bg-white/10"
            title={t("common.edit")}
          >
            <IconEdit className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={onDelete}
            className="text-white/60 hover:text-destructive hover:bg-destructive/10"
            title={t("common.delete")}
          >
            <IconTrash className="size-3.5" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
