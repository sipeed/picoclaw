"use client"

import { useState } from "react"
import { IconBook, IconDatabase, IconCircleCheck, IconSparkles, IconShield, IconActivity, IconCpu, IconFileText, IconSettings } from "@tabler/icons-react"
import { ResearchAgents } from "./research-agents"
import { ResearchGraph } from "./research-graph"
import { ResearchConfig } from "./research-config"
import { ResearchReports } from "./research-reports"
import { Badge } from "@/components/ui/badge"

interface ResearchAgent {
  id: string
  name: string
  icon: React.ComponentType<{ className?: string }>
  active: boolean
  progress: number
  ram: string
}

interface ResearchReport {
  id: string
  title: string
  status: "in-progress" | "complete"
  timestamp: string
  pages?: number
  words?: number
}

interface ResearchNode {
  name: string
  abbr: string
  x: number
  y: number
}

const defaultAgents: ResearchAgent[] = [
  { id: "literature", name: "Literature Analyzer", icon: IconBook, active: true, progress: 94, ram: "2.4GB" },
  { id: "extractor", name: "Data Extractor", icon: IconDatabase, active: true, progress: 87, ram: "1.8GB" },
  { id: "validator", name: "Fact Validator", icon: IconCircleCheck, active: true, progress: 76, ram: "1.2GB" },
  { id: "synthesizer", name: "Synthesizer", icon: IconSparkles, active: false, progress: 65, ram: "0.9GB" },
]

const defaultReports: ResearchReport[] = [
  { id: "1", title: "AI trends 2026", status: "in-progress", timestamp: "2 min ago", pages: 24, words: 7200 },
  { id: "2", title: "Quantum computing", status: "complete", timestamp: "1 hour ago", pages: 45, words: 13500 },
]

const defaultNodes: ResearchNode[] = [
  { name: "Neural Networks", abbr: "NN", x: 150, y: 80 },
  { name: "Transformers", abbr: "TR", x: 150, y: 120 },
  { name: "LLM Optimization", abbr: "LO", x: 150, y: 160 },
  { name: "Edge Computing", abbr: "EC", x: 150, y: 210 },
  { name: "Multi-Agent Systems", abbr: "MA", x: 150, y: 260 },
  { name: "Vision Models", abbr: "VM", x: 650, y: 100 },
  { name: "RAG Systems", abbr: "RA", x: 650, y: 200 },
  { name: "Knowledge Graphs", abbr: "KG", x: 400, y: 150 },
  { name: "Agent Architecture", abbr: "AA", x: 400, y: 180 },
  { name: "Fine-tuning Methods", abbr: "FT", x: 400, y: 250 },
]

export function ResearchPage() {
  const [agents, setAgents] = useState<ResearchAgent[]>(defaultAgents)
  const [researchType, setResearchType] = useState<string>("1.5")
  const [depth, setDepth] = useState<string>("1.5")
  const [restrictToGraph, setRestrictToGraph] = useState(false)
  const [selectedNodes, setSelectedNodes] = useState<Set<string>>(new Set())

  const handleToggleAgent = (id: string) => {
    setAgents(agents.map(agent =>
      agent.id === id ? { ...agent, active: !agent.active } : agent
    ))
  }

  const activeAgents = agents.filter(a => a.active)
  const totalProgress = activeAgents.length > 0
    ? Math.round(activeAgents.reduce((sum, a) => sum + a.progress, 0) / activeAgents.length)
    : 0

  return (
    <div className="relative min-h-screen bg-[#050505] overflow-hidden">
      {/* Ghost Background Typography */}
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        <div className="text-[200px] font-black text-[#F27D26]/[0.03] tracking-[0.3em] leading-none select-none">
          RESEARCH
        </div>
      </div>

      {/* Header */}
      <header className="relative z-10 border-b border-white/10 bg-[#0A0A0A]/80 backdrop-blur-sm">
        <div className="flex items-center justify-between px-6 py-4">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-[#F27D26] to-[#e05a10] flex items-center justify-center">
                <IconShield className="w-5 h-5 text-white" />
              </div>
              <div>
                <h1 className="text-lg font-bold text-[#F2F2F2]">Research Mode</h1>
                <p className="text-[10px] text-white/40">AI-Powered Research Assistant</p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <IconActivity className="w-4 h-4 text-[#F27D26]" />
              <span className="text-xs text-white/60">Status:</span>
              <Badge className="bg-green-500/20 text-green-400 text-[10px] px-2">
                Active
              </Badge>
            </div>
            <div className="flex items-center gap-2">
              <IconCpu className="w-4 h-4 text-white/40" />
              <span className="text-xs text-white/60">Progress:</span>
              <span className="text-sm font-semibold text-[#F27D26]">{totalProgress}%</span>
            </div>
            <div className="flex items-center gap-2">
              <IconFileText className="w-4 h-4 text-white/40" />
              <span className="text-xs text-white/60">Reports:</span>
              <span className="text-sm font-semibold text-[#F2F2F2]">{defaultReports.filter(r => r.status === "complete").length}</span>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content - 3 Column Layout */}
      <main className="relative z-10 flex h-[calc(100vh-130px)]">
        {/* Left Column - Agents */}
        <div className="w-80 border-r border-white/10 bg-[#0A0A0A]/50 p-4 overflow-y-auto">
          <ResearchAgents
            agents={agents}
            onToggleAgent={handleToggleAgent}
          />
        </div>

        {/* Center Column - Graph */}
        <div className="flex-1 bg-[#050505] relative">
          <ResearchGraph
            nodes={defaultNodes}
            selectedNodes={selectedNodes}
            onNodeToggle={(name: string) => {
              setSelectedNodes(prev => {
                const next = new Set(prev)
                if (next.has(name)) {
                  next.delete(name)
                } else {
                  next.add(name)
                }
                return next
              })
            }}
          />
        </div>

        {/* Right Column - Config + Reports */}
        <div className="w-80 border-l border-white/10 bg-[#0A0A0A]/50 p-4 overflow-y-auto flex flex-col gap-6">
          <ResearchConfig
            researchType={researchType}
            setResearchType={setResearchType}
            depth={depth}
            setDepth={setDepth}
            restrictToGraph={restrictToGraph}
            setRestrictToGraph={setRestrictToGraph}
          />
          <ResearchReports
            reports={defaultReports}
          />
        </div>
      </main>

      {/* Footer */}
      <footer className="relative z-10 border-t border-white/10 bg-[#0A0A0A]/80 px-6 py-2">
        <div className="flex items-center justify-between text-[10px] text-white/40">
          <div className="flex items-center gap-4">
            <span>PicoClaw Research Engine v2.4.1</span>
            <span className="text-white/20">|</span>
            <span>Nodes: {defaultNodes.length}</span>
            <span className="text-white/20">|</span>
            <span>Agents: {activeAgents.length} active</span>
          </div>
          <div className="flex items-center gap-2">
            <IconSettings className="w-3 h-3" />
            <span>Last updated: {new Date().toLocaleTimeString()}</span>
          </div>
        </div>
      </footer>
    </div>
  )
}