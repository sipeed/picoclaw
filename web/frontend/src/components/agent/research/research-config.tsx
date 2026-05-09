import { useMemo } from "react"
import { IconShieldCheck } from "@tabler/icons-react"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"

interface ResearchConfigProps {
  researchType: string
  setResearchType: (value: string) => void
  depth: string
  setDepth: (value: string) => void
  restrictToGraph: boolean
  setRestrictToGraph: (value: boolean) => void
  onSave?: () => void
  isSaving?: boolean
}

export function ResearchConfig({
  researchType,
  setResearchType,
  depth,
  setDepth,
  restrictToGraph,
  setRestrictToGraph,
  onSave,
  isSaving = false,
}: ResearchConfigProps) {
  const scope = useMemo(() => {
    const type = parseFloat(researchType)
    const depthVal = parseFloat(depth)
    const basePages = 12
    const pages = Math.round(basePages * type * depthVal)
    const words = pages * 300

    let complexity = "Low"
    const score = type * depthVal
    if (score > 2.5) complexity = "High"
    else if (score > 1.5) complexity = "Moderate"

    const time = Math.round(pages * 1.2)

    return { pages, words, complexity, time }
  }, [researchType, depth])

  return (
    <div className="space-y-6">
      {/* Configuration Panel */}
      <div className="space-y-4">
        <div className="border-b border-white/10 pb-2">
          <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">
            Configuration
          </span>
        </div>

        <div className="rounded-xl border border-white/10 bg-[#0A0A0A] p-4 space-y-4">
          <div>
            <label className="text-[10px] font-medium text-white/40 uppercase tracking-wide block mb-2">
              Research Type
            </label>
            <select
              className="w-full px-3 py-2.5 rounded-lg bg-[#050505] border border-white/10 text-[#F2F2F2] text-xs focus:outline-none focus:border-[#F27D26] transition-colors cursor-pointer"
              value={researchType}
              onChange={(e) => setResearchType(e.target.value)}
            >
              <option value="1.0">Literature Review</option>
              <option value="1.5">Systematic</option>
              <option value="2.0">Meta-analysis</option>
              <option value="0.8">Exploratory</option>
            </select>
          </div>

          <div>
            <label className="text-[10px] font-medium text-white/40 uppercase tracking-wide block mb-2">
              Depth Level
            </label>
            <select
              className="w-full px-3 py-2.5 rounded-lg bg-[#050505] border border-white/10 text-[#F2F2F2] text-xs focus:outline-none focus:border-[#F27D26] transition-colors cursor-pointer"
              value={depth}
              onChange={(e) => setDepth(e.target.value)}
            >
              <option value="0.8">Shallow</option>
              <option value="1.5">Deep</option>
              <option value="2.2">Ultra</option>
            </select>
          </div>

          <div className="flex items-center justify-between py-3 px-4 rounded-lg bg-[#F27D26]/5 border border-[#F27D26]/20">
            <div className="flex items-center gap-2 text-xs text-[#F27D26] font-semibold">
              <IconShieldCheck className="w-4 h-4" />
              Restrict to Graph
            </div>
            <Switch
              checked={restrictToGraph}
              onCheckedChange={setRestrictToGraph}
            />
          </div>
        </div>
      </div>

      {/* Scope Calculator */}
      <div className="space-y-4">
        <div className="border-b border-white/10 pb-2">
          <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">
            Report Scope
          </span>
        </div>

        <div className="rounded-xl border border-white/10 bg-[#0A0A0A] p-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-[#F27D26] mb-1">{scope.pages}</div>
              <div className="text-[10px] text-white/40 uppercase tracking-wide">Pages</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-[#F2F2F2] mb-1">
                {(scope.words / 1000).toFixed(1)}k
              </div>
              <div className="text-[10px] text-white/40 uppercase tracking-wide">Words</div>
            </div>
          </div>

          <div className="h-px bg-white/10 my-4" />

          <div className="grid grid-cols-2 gap-4">
            <div className="text-center">
              <div className={cn(
                "text-xs font-semibold mb-1",
                scope.complexity === "High" ? "text-[#f59e0b]" :
                scope.complexity === "Moderate" ? "text-[#F27D26]" : "text-green-400"
              )}>
                {scope.complexity}
              </div>
              <div className="text-[10px] text-white/40 uppercase tracking-wide">Complexity</div>
            </div>
            <div className="text-center">
              <div className="text-xs font-semibold text-[#F2F2F2] mb-1">{scope.time} min</div>
              <div className="text-[10px] text-white/40 uppercase tracking-wide">Est. Time</div>
            </div>
          </div>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="space-y-2">
        <button
          onClick={onSave}
          disabled={isSaving}
          className="w-full px-4 py-3 rounded-xl bg-gradient-to-r from-[#F27D26] to-[#fb923c] text-black text-xs font-bold hover:from-[#ff8f4a] hover:to-[#fca55a] transition-all shadow-lg shadow-[#F27D26]/20 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSaving ? "Saving..." : "Start Research"}
        </button>
        <button className="w-full px-4 py-2.5 rounded-lg bg-[#050505] border border-white/10 text-white/60 text-xs font-medium hover:border-white/30 hover:text-white transition-all">
          Advanced Settings
        </button>
      </div>
    </div>
  )
}