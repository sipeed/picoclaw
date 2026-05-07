import dayjs from "dayjs"
import { useMemo } from "react"
import { IconArrowRight, IconLayoutDashboard, IconUsers, IconBrain, IconFlask } from "@tabler/icons-react"

import { usePicoChat } from "@/hooks/use-pico-chat"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"

import { MemoryGraph } from "./memory-graph"
import { useAgentCockpit } from "./use-agent-cockpit"
import { AgentsPage } from "../agents"
import { SkillsPage } from "../skills"
import { ResearchPage } from "../research/research-page"

function reasonLabel(reasonCode?: string) {
  switch (reasonCode) {
    case "requires_subagent":
      return "Requires subagent runtime"
    case "requires_skills":
      return "Requires skills support"
    case "requires_mcp_discovery":
      return "Requires MCP discovery"
    case "requires_linux":
      return "Linux only"
    case "requires_serial_platform":
      return "Unsupported serial platform"
    default:
      return reasonCode ?? ""
  }
}

export function CockpitPage() {
  const { activeSessionId } = usePicoChat()
  const {
    groupedTools,
    pendingToolName,
    sessionSubagents,
    sessionMemoryGraph,
    toggleTool,
    activeTab,
    setActiveTab,
  } = useAgentCockpit(activeSessionId)

  const filteredToolCount = useMemo(
    () => groupedTools.reduce((total, [, items]) => total + items.length, 0),
    [groupedTools],
  )

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#050505] text-[#F2F2F2] selection:bg-[#F27D26] selection:text-black font-sans relative">
      {/* Ghost Background Typography */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none select-none opacity-[0.05]">
        <span className="absolute -left-20 -top-10 text-[35vw] font-black leading-none uppercase">COCKPIT</span>
        <span className="absolute -right-20 -bottom-20 text-[25vw] font-black leading-none uppercase">SYSTEM</span>
      </div>

      {/* Header */}
      <header className="flex justify-between items-start border-b border-white/10 p-6 md:px-12 md:py-8 z-10">
        <div className="flex flex-col gap-1">
          <span className="text-[10px] uppercase tracking-[0.3em] font-bold text-[#F27D26]">Terminal Status</span>
          <span className="text-xs opacity-60 font-mono tracking-tighter">CONNECTED / {dayjs().format('DD.MM.YYYY')}</span>
        </div>
        <div className="flex flex-col gap-1 text-right">
          <span className="text-[10px] uppercase tracking-[0.3em] font-bold text-[#F27D26]">Runtime Epoch</span>
          <span className="text-xs opacity-60 font-mono tracking-tighter">{dayjs().format('HH:mm')} GMT+1</span>
        </div>
      </header>

      <div className="flex-1 overflow-auto px-6 py-6 md:px-12 md:py-10 z-10">
        <div className="mx-auto grid w-full max-w-[1600px] gap-12 xl:grid-cols-[1fr_380px]">

          {/* Main Workspace */}
          <div className="space-y-16">
            {/* Hero Section */}
            <div className="relative">
              <h1 className="text-[12vw] leading-[0.85] font-black tracking-[-0.07em] uppercase m-0 p-0 text-[#F2F2F2]">
                AGENT<br/>INTERFACE
              </h1>
              <p className="mt-8 text-xl font-light max-w-xl opacity-60 leading-relaxed border-l-2 border-[#F27D26] pl-6">
                Active control node for autonomous agents. Managing tool surfaces, memory networks.
              </p>
            </div>

            <div className="flex items-center gap-4 border-b border-white/10 pb-4">
              <button
                onClick={() => setActiveTab("tools")}
                className={cn(
                  "flex items-center gap-2 text-xs uppercase tracking-widest font-bold transition-colors",
                  activeTab === "tools" ? "text-[#F27D26] border-b-2 border-[#F27D26] pb-4 -mb-4.5" : "text-white/40 hover:text-white/60"
                )}
              >
                <IconLayoutDashboard className="size-4" />
                Tools
              </button>
              <button
                onClick={() => setActiveTab("skills")}
                className={cn(
                  "flex items-center gap-2 text-xs uppercase tracking-widest font-bold transition-colors",
                  activeTab === "skills" ? "text-[#F27D26] border-b-2 border-[#F27D26] pb-4 -mb-4.5" : "text-white/40 hover:text-white/60"
                )}
              >
                <IconBrain className="size-4" />
                Skills
              </button>
              <button
                onClick={() => setActiveTab("agents")}
                className={cn(
                  "flex items-center gap-2 text-xs uppercase tracking-widest font-bold transition-colors",
                  activeTab === "agents" ? "text-[#F27D26] border-b-2 border-[#F27D26] pb-4 -mb-4.5" : "text-white/40 hover:text-white/60"
                )}
              >
                <IconUsers className="size-4" />
                Agents
              </button>
              <button
                onClick={() => setActiveTab("research")}
                className={cn(
                  "flex items-center gap-2 text-xs uppercase tracking-widest font-bold transition-colors",
                  activeTab === "research" ? "text-[#F27D26] border-b-2 border-[#F27D26] pb-4 -mb-4.5" : "text-white/40 hover:text-white/60"
                )}
              >
                <IconFlask className="size-4" />
                Research
              </button>
            </div>

            {activeTab === "skills" && <SkillsPage embedded />}

            {activeTab === "agents" && <AgentsPage embedded />}

            {activeTab === "research" && <ResearchPage />}

            {activeTab === "tools" && (
            <section className="grid gap-12">
              {/* Tool Grid */}
              <div className="space-y-8">
                <div className="flex items-end justify-between border-b border-white/10 pb-4">
                  <div className="flex flex-col gap-1">
                    <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">Active Modules</span>
                    <h2 className="text-4xl font-black uppercase tracking-tight">Tool Grid</h2>
                  </div>
                  <div className="flex items-center gap-6">
                    <span className="text-[10px] uppercase tracking-[0.2em] opacity-40 font-bold">{filteredToolCount} Visible</span>
                    <a href="/agent/tools" className="text-xs font-bold uppercase tracking-widest hover:text-[#F27D26] transition-colors border-b border-white/20 pb-1">
                      Configuration
                    </a>
                  </div>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  {groupedTools.flatMap(([, items]) =>
                    items.map((tool) => (
                      <div
                        key={tool.name}
                        className="group relative border border-white/10 bg-[#0A0A0A] p-4 hover:border-[#F27D26]/50 transition-all duration-300"
                      >
                        <div className="flex items-start justify-between gap-3 mb-3">
                          <div className="space-y-1 flex-1 min-w-0">
                            <h4 className="font-bold text-sm tracking-tight uppercase truncate group-hover:text-[#F27D26] transition-colors">
                              {tool.name}
                            </h4>
                            <Badge
                              className={cn(
                                "rounded-none text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5",
                                tool.status === "enabled" ? "bg-[#F27D26] text-black" : "bg-white/10 text-white/40"
                              )}
                            >
                              {tool.status}
                            </Badge>
                          </div>
                          <Switch
                            checked={tool.status !== "disabled"}
                            disabled={pendingToolName === tool.name}
                            onCheckedChange={(checked) => toggleTool(tool.name, checked)}
                          />
                        </div>
                        <p className="text-xs text-[#F2F2F2]/60 leading-relaxed font-light mb-4 line-clamp-2">
                          {tool.description}
                        </p>
                        <div className="flex items-center justify-between text-[8px] font-mono uppercase tracking-widest text-white/30 pt-3 border-t border-white/5">
                          <span className="truncate">{tool.config_key}</span>
                          {(tool as any).reason_code && (
                            <span className="text-[#F27D26]/60">{reasonLabel((tool as any).reason_code)}</span>
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Memory Network */}
              <div className="space-y-8">
                <div className="flex items-end justify-between border-b border-white/10 pb-4">
                  <div className="flex flex-col gap-1">
                    <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">Relational Map</span>
                    <h2 className="text-4xl font-black uppercase tracking-tight">Memory Network</h2>
                  </div>
                </div>
                <div className="p-8 border border-white/10 bg-[#0A0A0A] relative overflow-hidden">
                  <MemoryGraph
                    nodes={sessionMemoryGraph?.nodes ?? []}
                    edges={sessionMemoryGraph?.edges ?? []}
                  />
                </div>
              </div>
            </section>
            )}
          </div>

          {/* Right Sidebar */}
          <aside className="space-y-12">
             {/* Subagents */}
             <div className="space-y-8">
               <div className="border-b border-white/10 pb-2">
                 <span className="text-[10px] uppercase tracking-[0.3em] font-bold text-[#F27D26]">Subagent Manifest</span>
               </div>
               <div className="space-y-4">
                 {sessionSubagents === null ? (
                   <div className="border border-white/10 p-5 bg-[#0A0A0A] text-sm text-white/40">
                     Loading subagents...
                   </div>
                 ) : sessionSubagents.length === 0 ? (
                   <div className="border border-white/10 p-5 bg-[#0A0A0A] text-sm text-white/40">
                     No subagents have been created in this session yet.
                   </div>
                 ) : (
                   sessionSubagents.map((task) => (
                     <div key={task.id} className="group border border-white/10 p-5 bg-[#0A0A0A] hover:border-white/30 transition-all">
                       <div className="flex justify-between items-start mb-2">
                         <span className="font-bold text-sm uppercase tracking-tight">{task.label || task.id}</span>
                         <span className={cn(
                           "text-[9px] uppercase font-mono px-1.5 py-0.5",
                           task.status === "completed" ? "bg-green-500/20 text-green-400" : "bg-[#F27D26]/20 text-[#F27D26]"
                         )}>
                           {task.status}
                         </span>
                       </div>
                       <div className="text-[9px] font-mono text-white/30 truncate">
                         {dayjs(task.created).format("HH:mm:ss [UTC]")}
                       </div>
                     </div>
                   ))
                 )}
               </div>
             </div>
          </aside>
        </div>
      </div>

      {/* Footer */}
      <footer className="h-20 border-t border-white/10 flex items-center justify-between px-6 md:px-12 z-10 bg-[#050505]">
        <nav className="flex gap-10 text-[10px] font-bold uppercase tracking-[0.3em]">
          <a href="#" className="hover:text-[#F27D26] transition-colors">Portfolio</a>
          <a href="#" className="hover:text-[#F27D26] transition-colors">Documentation</a>
          <a href="#" className="hover:text-[#F27D26] transition-colors">Gateway</a>
        </nav>
        <div className="flex items-center gap-6">
          <span className="text-[10px] uppercase tracking-[0.2em] opacity-40 font-bold">System Ref: BOLD-UX-01</span>
          <div className="h-10 w-10 rounded-full border border-white/10 flex items-center justify-center hover:bg-white hover:text-black cursor-pointer transition-all">
            <IconArrowRight size={16} />
          </div>
        </div>
      </footer>
    </div>
  )
}
