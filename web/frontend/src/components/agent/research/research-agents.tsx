import { IconBook, IconDatabase, IconCircleCheck, IconSparkles } from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import { ResearchAgent } from "@/api/research"

interface ResearchAgentsProps {
  agents: ResearchAgent[]
  onToggleAgent: (id: string) => void
}

const agentIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  literature: IconBook,
  extractor: IconDatabase,
  validator: IconCircleCheck,
  synthesizer: IconSparkles,
}

const agentLabels: Record<string, string> = {
  literature: "Literature Analyzer",
  extractor: "Data Extractor",
  validator: "Fact Validator",
  synthesizer: "Synthesizer",
}

const statusLabels: Record<string, string> = {
  literature: "Analyzing papers",
  extractor: "Extracting data",
  validator: "Validating facts",
  synthesizer: "Synthesizing",
}

export function ResearchAgents({ agents, onToggleAgent }: ResearchAgentsProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between border-b border-white/10 pb-2">
        <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">
          Research Agents
        </span>
        <span className="text-[10px] text-white/40 font-mono">
          {agents.filter(a => a.active).length}/{agents.length} active
        </span>
      </div>

      <div className="space-y-3">
        {agents.map((agent) => {
          const Icon = agentIcons[agent.id] || IconBook
          const isComplete = agent.progress > 90
          const isProcessing = agent.progress > 50
          
          return (
            <div
              key={agent.id}
              className={cn(
                "group relative rounded-xl border p-4 transition-all cursor-pointer",
                agent.active 
                  ? "border-white/20 bg-[#0A0A0A] hover:border-[#F27D26]/50" 
                  : "border-white/5 bg-[#050505] opacity-60"
              )}
              onClick={() => onToggleAgent(agent.id)}
            >
              {/* Active glow effect */}
              {agent.active && (
                <div className="absolute inset-0 rounded-xl bg-gradient-to-br from-[#F27D26]/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
              )}
              
              <div className="relative z-10">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      "w-10 h-10 rounded-xl flex items-center justify-center",
                      agent.active 
                        ? "bg-gradient-to-br from-[#F27D26] to-[#e05a10]" 
                        : "bg-white/10"
                    )}>
                      <Icon className="w-5 h-5 text-white" />
                    </div>
                    <div>
                      <div className="text-sm font-semibold text-[#F2F2F2]">
                        {agentLabels[agent.id] || agent.name}
                      </div>
                      <div className="text-[10px] text-white/40 flex items-center gap-1 mt-0.5">
                        <span className={cn(
                          "w-1.5 h-1.5 rounded-full",
                          agent.active ? "bg-green-500 animate-pulse" : "bg-white/20"
                        )} />
                        {agent.active ? (statusLabels[agent.id] || "Running") : "Stopped"}
                      </div>
                    </div>
                  </div>
                  <Switch
                    checked={agent.active}
                    disabled={false}
                    onClick={(e) => e.stopPropagation()}
                    onCheckedChange={() => onToggleAgent(agent.id)}
                  />
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between text-[10px]">
                    <span className="text-white/40">Progress</span>
                    <span className="text-[#F27D26] font-semibold">{agent.progress}%</span>
                  </div>
                  <div className="h-1.5 bg-white/10 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-gradient-to-r from-[#F27D26] to-[#fb923c] rounded-full transition-all"
                      style={{ width: `${agent.progress}%` }}
                    />
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <div className="text-[10px]">
                      <span className="text-white/40">Memory</span>
                      <span className="ml-1.5 text-[#F2F2F2] font-medium">{agent.ram}</span>
                    </div>
                    <Badge
                      className={cn(
                        "text-[9px] px-2 py-0.5 rounded-none font-bold uppercase",
                        isComplete 
                          ? "bg-green-500/20 text-green-400" 
                          : isProcessing 
                            ? "bg-[#F27D26]/20 text-[#F27D26]"
                            : "bg-white/10 text-white/40"
                      )}
                    >
                      {isComplete ? "Finalizing" : isProcessing ? "Processing" : "Starting"}
                    </Badge>
                  </div>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}