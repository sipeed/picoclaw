import dayjs from "dayjs"
import { useEffect, useMemo, useState } from "react"
import { motion } from "motion/react"
import {
  IconLayoutDashboard, IconUsers, IconBrain, IconFlask,
  IconCircleDot, IconActivity, IconCpu, IconDatabase, IconBolt,
  IconShield, IconTerminal, IconWifi, IconLock, IconTrendingUp,
} from "@tabler/icons-react"

import { usePicoChat } from "@/hooks/use-pico-chat"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"

import { MemoryGraph } from "./memory-graph"
import { useAgentCockpit } from "./use-agent-cockpit"
import { AgentsPage } from "../agents"
import { SkillsPage } from "../skills"
import { ResearchPage } from "../research/research-page"

const TAB_CONFIG = [
  { key: "tools" as const, icon: IconLayoutDashboard, label: "Tools" },
  { key: "skills" as const, icon: IconBrain, label: "Skills" },
  { key: "agents" as const, icon: IconUsers, label: "Agents" },
  { key: "research" as const, icon: IconFlask, label: "Research" },
]

/* ─── 3D Holographic Circle Component ─── */
function HolographicCircle() {
  const [rotation, setRotation] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => {
      setRotation((prev) => (prev + 0.5) % 360)
    }, 50)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="relative w-full h-full flex items-center justify-center">
      {/* Outer Glow */}
      <motion.div
        animate={{ opacity: [0.3, 0.6, 0.3], scale: [1, 1.05, 1] }}
        transition={{ duration: 3, repeat: Infinity }}
        className="absolute inset-0 rounded-full bg-[#00bcff]/20 blur-3xl"
      />

      {/* Rotating Rings - 3D Perspective */}
      <div
        className="absolute inset-0"
        style={{ transform: "perspective(1000px) rotateX(65deg)" }}
      >
        {[0, 1, 2, 3, 4].map((index) => (
          <div
            key={index}
            className="absolute inset-0 rounded-full border-2 border-[#00bcff]/40 shadow-[0_0_20px_#00bcff4d]"
            style={{
              transform: `rotateZ(${rotation + index * 20}deg) scale(${1 - index * 0.15})`,
            }}
          />
        ))}

        {/* Inner Core */}
        <div className="absolute inset-[35%] rounded-full bg-gradient-to-br from-[#00bcff]/40 to-blue-500/40 backdrop-blur-sm shadow-[0_0_50px_#00bcff80,inset_0_0_30px_#00bcff4d]" />

        {/* Scanning Line */}
        <motion.div
          className="absolute inset-0"
          style={{
            background: `conic-gradient(from ${rotation}deg, transparent 0deg, #00bcff99 10deg, transparent 20deg)`,
          }}
        />
      </div>

      {/* Center Display */}
      <div className="relative z-10 flex flex-col items-center justify-center" style={{ transform: "perspective(1000px) rotateX(0deg)" }}>
        <motion.div
          animate={{ scale: [1, 1.1, 1], opacity: [0.8, 1, 0.8] }}
          transition={{ duration: 2, repeat: Infinity }}
          className="text-5xl font-bold text-transparent bg-clip-text bg-gradient-to-b from-cyan-300 to-blue-500"
        >
          AI
        </motion.div>
        <div className="text-[#00bcff] text-[10px] tracking-[0.3em] mt-2 uppercase">Core Online</div>

        {/* Data Points around circle */}
        {[0, 45, 90, 135, 180, 225, 270, 315].map((angle) => (
          <motion.div
            key={angle}
            className="absolute w-1.5 h-1.5 rounded-full bg-[#00bcff]"
            style={{
              left: "50%",
              top: "50%",
              transform: `translate(-50%, -50%) rotate(${angle}deg) translateY(-100px)`,
            }}
            animate={{ opacity: [0.3, 1, 0.3], scale: [0.8, 1.2, 0.8] }}
            transition={{ duration: 2, repeat: Infinity, delay: angle / 360 }}
          />
        ))}
      </div>

      {/* Holographic Scan Lines */}
      <div className="absolute inset-0 overflow-hidden rounded-full pointer-events-none">
        <motion.div
          className="absolute inset-0 bg-gradient-to-b from-transparent via-[#00bcff]/10 to-transparent"
          animate={{ y: ["-100%", "100%"] }}
          transition={{ duration: 3, repeat: Infinity, ease: "linear" }}
        />
      </div>
    </div>
  )
}

/* ─── Particle Field Component ─── */
function ParticleField() {
  const particles = useMemo(
    () =>
      Array.from({ length: 40 }, (_, i) => ({
        id: i,
        x: Math.random() * 100,
        y: Math.random() * 100,
        size: Math.random() * 3 + 1,
        duration: Math.random() * 10 + 10,
        delay: Math.random() * 5,
      })),
    []
  )

  return (
    <div className="absolute inset-0 overflow-hidden pointer-events-none">
      {particles.map((p) => (
        <motion.div
          key={p.id}
          className="absolute rounded-full bg-[#00bcff]/40"
          style={{
            left: `${p.x}%`,
            top: `${p.y}%`,
            width: p.size,
            height: p.size,
          }}
          animate={{
            y: [0, -100, 0],
            opacity: [0, 1, 0],
            scale: [0, 1, 0],
          }}
          transition={{
            duration: p.duration,
            repeat: Infinity,
            delay: p.delay,
            ease: "easeInOut",
          }}
        />
      ))}

      {/* Connection Lines */}
      <svg className="absolute inset-0 w-full h-full">
        {Array.from({ length: 15 }, (_, i) => (
          <motion.line
            key={i}
            x1={`${Math.random() * 100}%`}
            y1={`${Math.random() * 100}%`}
            x2={`${Math.random() * 100}%`}
            y2={`${Math.random() * 100}%`}
            stroke="#00bcff1a"
            strokeWidth="1"
            initial={{ pathLength: 0, opacity: 0 }}
            animate={{ pathLength: [0, 1, 0], opacity: [0, 0.5, 0] }}
            transition={{ duration: 3, repeat: Infinity, delay: i * 0.2 }}
          />
        ))}
      </svg>
    </div>
  )
}

/* ─── Metric Card Component ─── */
function MetricCard({ icon, label, value, trend }: { icon: React.ReactNode; label: string; value: string; trend: string }) {
  const isPositive = trend.startsWith("+")

  return (
    <motion.div
      whileHover={{ scale: 1.05 }}
      className="relative rounded-xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-4 overflow-hidden shadow-[0_0_20px_#00bcff1a]"
    >
      {/* Animated Background */}
      <motion.div
        className="absolute inset-0 bg-gradient-to-br from-[#00bcff]/5 to-blue-500/5"
        animate={{ opacity: [0.3, 0.5, 0.3] }}
        transition={{ duration: 2, repeat: Infinity }}
      />

      {/* Content */}
      <div className="relative z-10">
        <div className="flex items-center justify-between mb-3">
          <div className="p-2 rounded-lg bg-[#00bcff]/10 text-[#00bcff]">{icon}</div>
          <motion.div
            animate={{ opacity: [0.5, 1, 0.5] }}
            transition={{ duration: 1.5, repeat: Infinity }}
            className={`text-xs ${isPositive ? "text-green-400" : "text-red-400"}`}
          >
            {trend}
          </motion.div>
        </div>
        <div className="text-[10px] text-cyan-300/60 tracking-wider mb-1 uppercase">{label}</div>
        <div className="text-2xl font-bold text-cyan-100">{value}</div>
      </div>

      {/* Scan Line */}
      <motion.div
        className="absolute inset-x-0 h-px bg-gradient-to-r from-transparent via-[#00bcff]/50 to-transparent"
        animate={{ y: [0, 100] }}
        transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
      />

      {/* Corner Accent */}
      <div className="absolute bottom-0 right-0 w-16 h-16 border-r border-b border-[#00bcff]/20" />
    </motion.div>
  )
}

/* ─── Data Panel Component ─── */
function DataPanel({ title, items }: {
  title: string
  items: Array<{ label: string; status: string; progress: number }>
}) {
  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      className="rounded-2xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-5 overflow-hidden relative shadow-[0_0_30px_#00bcff1a,inset_0_0_20px_#00bcff0d]"
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-cyan-300 text-sm tracking-widest font-semibold">{title}</h3>
        <div className="flex gap-1">
          {[0, 1, 2].map((i) => (
            <motion.div
              key={i}
              className="w-1 h-3 bg-[#00bcff]/60 rounded-full"
              animate={{ height: ["12px", "20px", "12px"] }}
              transition={{ duration: 1, repeat: Infinity, delay: i * 0.2 }}
            />
          ))}
        </div>
      </div>

      {/* Items */}
      <div className="space-y-3">
        {items.map((item, index) => (
          <motion.div
            key={item.label}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.1 }}
            className="group"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="text-cyan-100/80 text-xs">{item.label}</span>
              <span
                className={`text-xs px-2 py-0.5 rounded-full ${
                  item.status === "ACTIVE" || item.status === "OPTIMAL"
                    ? "bg-green-400/20 text-green-400"
                    : item.status === "PROCESSING"
                    ? "bg-yellow-400/20 text-yellow-400"
                    : "bg-[#00bcff]/20 text-[#00bcff]"
                }`}
              >
                {item.status}
              </span>
            </div>

            {/* Progress Bar */}
            <div className="relative h-1.5 bg-cyan-950/50 rounded-full overflow-hidden">
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${item.progress}%` }}
                transition={{ duration: 1, delay: index * 0.1 + 0.3 }}
                className="absolute inset-y-0 left-0 bg-gradient-to-r from-cyan-500 to-blue-500 rounded-full shadow-[0_0_10px_#00bcff99]"
              />
              <motion.div
                className="absolute inset-y-0 left-0 right-0 bg-gradient-to-r from-transparent via-white/30 to-transparent"
                animate={{ x: ["-100%", "200%"] }}
                transition={{ duration: 2, repeat: Infinity, delay: index * 0.3 }}
              />
            </div>
          </motion.div>
        ))}
      </div>

      {/* Decorative Corner */}
      <div className="absolute top-0 right-0 w-20 h-20 border-r border-t border-[#00bcff]/20 pointer-events-none" />
    </motion.div>
  )
}

/* ─── Circular Progress Component ─── */
function CircularProgress({ icon, label, percentage, color = "cyan" }: {
  icon: React.ReactNode; label: string; percentage: number; color?: "cyan" | "blue" | "green"
}) {
  const radius = 70
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (percentage / 100) * circumference

  const colorMap = {
    cyan: { stroke: "#00bcff", glow: "#00bcff99", bg: "#00bcff1a" },
    blue: { stroke: "#0080ff", glow: "#0080ff99", bg: "#0080ff1a" },
    green: { stroke: "#00ff88", glow: "#00ff8899", bg: "#00ff881a" },
  }
  const colors = colorMap[color]

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      whileHover={{ scale: 1.05 }}
      className="relative rounded-2xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-5 flex flex-col items-center justify-center shadow-[0_0_30px_#00bcff1a]"
    >
      <div className="relative">
        <svg width="140" height="140" className="transform -rotate-90">
          <circle cx="70" cy="70" r={radius} stroke={colors.bg} strokeWidth="8" fill="none" />
          <motion.circle
            cx="70" cy="70" r={radius} stroke={colors.stroke} strokeWidth="8" fill="none"
            strokeLinecap="round" strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }} animate={{ strokeDashoffset: offset }}
            transition={{ duration: 1.5, ease: "easeOut" }}
            style={{ filter: `drop-shadow(0 0 8px ${colors.glow})` }}
          />
          <motion.circle
            cx="70" cy="70" r={radius} stroke={colors.stroke} strokeWidth="12" fill="none"
            strokeLinecap="round" strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }} animate={{ strokeDashoffset: offset }}
            transition={{ duration: 1.5, ease: "easeOut" }} opacity="0.3"
            style={{ filter: "blur(8px)" }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <motion.div
            animate={{ scale: [1, 1.1, 1] }}
            transition={{ duration: 2, repeat: Infinity }}
            className="text-[#00bcff] mb-1"
          >
            {icon}
          </motion.div>
          <motion.div
            initial={{ scale: 0 }} animate={{ scale: 1 }}
            transition={{ delay: 0.5, type: "spring" }}
            className="text-2xl font-bold text-cyan-100"
          >
            {percentage}%
          </motion.div>
        </div>
      </div>
      <div className="mt-3 text-cyan-300 text-xs tracking-[0.2em] uppercase">{label}</div>
      <div className="absolute top-3 left-3 right-3 h-px bg-gradient-to-r from-transparent via-[#00bcff]/30 to-transparent" />
      <div className="absolute bottom-3 left-3 right-3 h-px bg-gradient-to-r from-transparent via-[#00bcff]/30 to-transparent" />
    </motion.div>
  )
}

/* ─── System Status Grid Component ─── */
function SystemStatusGrid() {
  const statusItems = [
    { icon: <IconWifi className="size-4" />, label: "CONNECTION", value: "STABLE", status: "good" },
    { icon: <IconDatabase className="size-4" />, label: "STORAGE", value: "67% FREE", status: "good" },
    { icon: <IconLock className="size-4" />, label: "SECURITY", value: "ENCRYPTED", status: "good" },
    { icon: <IconTrendingUp className="size-4" />, label: "PERFORMANCE", value: "OPTIMAL", status: "good" },
  ]

  return (
    <div className="grid grid-cols-2 gap-3">
      {statusItems.map((item, index) => (
        <motion.div
          key={item.label}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: index * 0.1 }}
          whileHover={{ scale: 1.05 }}
          className="relative rounded-xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-3 overflow-hidden group shadow-[0_0_20px_#00bcff1a]"
        >
          {/* Hover Glow */}
          <div className="absolute inset-0 bg-gradient-to-br from-[#00bcff]/0 to-blue-500/0 group-hover:from-[#00bcff]/10 group-hover:to-blue-500/10 transition-all duration-300" />

          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-2">
              <div className="p-1.5 rounded bg-[#00bcff]/10 text-[#00bcff]">{item.icon}</div>
              <motion.div
                className="w-2 h-2 rounded-full bg-green-400 shadow-[0_0_8px_#00ff88cc]"
                animate={{ opacity: [0.5, 1, 0.5], scale: [1, 1.2, 1] }}
                transition={{ duration: 2, repeat: Infinity }}
              />
            </div>
            <div className="text-[9px] text-cyan-300/60 tracking-wider mb-1 uppercase">{item.label}</div>
            <div className="text-xs font-semibold text-cyan-100">{item.value}</div>
          </div>

          {/* Scan Effect */}
          <motion.div
            className="absolute inset-x-0 h-px bg-gradient-to-r from-transparent via-[#00bcff]/50 to-transparent"
            animate={{ y: [0, 60] }}
            transition={{ duration: 2, repeat: Infinity, delay: index * 0.3 }}
          />
        </motion.div>
      ))}
    </div>
  )
}

/* ─── Terminal Output Component ─── */
function TerminalOutput() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ delay: 0.6 }}
      className="flex-1 rounded-2xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-4 overflow-hidden shadow-[0_0_30px_#00bcff1a,inset_0_0_20px_#00bcff0d]"
    >
      <div className="flex items-center gap-2 mb-3">
        <IconTerminal className="size-4 text-[#00bcff]" />
        <span className="text-cyan-300 text-xs tracking-wider">TERMINAL</span>
      </div>
      <div className="space-y-1 font-mono text-xs">
        <motion.div
          initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.7 }}
          className="text-[#00bcff]/60"
        >
          &gt; Initializing core systems...
        </motion.div>
        <motion.div
          initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.8 }}
          className="text-green-400/60"
        >
          &gt; All systems operational
        </motion.div>
        <motion.div
          initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.9 }}
          className="text-[#00bcff]/60"
        >
          &gt; Awaiting commands...
        </motion.div>
        <motion.div
          animate={{ opacity: [0, 1, 0] }}
          transition={{ duration: 1.5, repeat: Infinity }}
          className="text-[#00bcff] inline-block"
        >
          |
        </motion.div>
      </div>
    </motion.div>
  )
}

/* ═══════════════════════════════════════════════
    MAIN COCKPIT PAGE
    ═══════════════════════════════════════════════ */
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
    <div className="relative size-full bg-[#0a0e27] overflow-hidden text-[#e0f2fe] font-sans">
      {/* ── 3D Perspective Grid Background ── */}
      <div className="absolute inset-0 bg-[linear-gradient(#00bcff08_1px,transparent_1px),linear-gradient(90deg,#00bcff08_1px,transparent_1px)] bg-[size:50px_50px] [transform:perspective(1000px)_rotateX(60deg)] origin-center opacity-40" />

      {/* ── Particle Field ── */}
      <ParticleField />

      {/* ── Scanline Effect ── */}
      <div className="absolute inset-0 bg-[linear-gradient(transparent_50%,#00bcff08_50%)] bg-[size:100%_4px] pointer-events-none animate-[scan_8s_linear_infinite]" />

      {/* ── Corner Accents ── */}
      <div className="absolute top-0 left-0 w-32 h-32 border-l-2 border-t-2 border-[#00bcff]/30 pointer-events-none" />
      <div className="absolute top-0 right-0 w-32 h-32 border-r-2 border-t-2 border-[#00bcff]/30 pointer-events-none" />
      <div className="absolute bottom-0 left-0 w-32 h-32 border-l-2 border-b-2 border-[#00bcff]/30 pointer-events-none" />
      <div className="absolute bottom-0 right-0 w-32 h-32 border-r-2 border-b-2 border-[#00bcff]/30 pointer-events-none" />

      {/* ── Main Content ── */}
      <div className="relative z-10 size-full p-6 flex flex-col">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="flex items-center justify-between mb-6"
        >
          <div>
            <h1 className="text-4xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 via-blue-400 to-cyan-300 tracking-wider">
              AFRICA
            </h1>
            <p className="text-cyan-300/60 text-xs mt-1 tracking-[0.3em]">
              ADVANCED FRAMEWORK FOR INTELLIGENT SYSTEM
            </p>
          </div>

          <div className="flex items-center gap-4">
            <div className="px-5 py-2.5 rounded-full border border-[#00bcff]/30 bg-[#00bcff]/5 backdrop-blur-sm shadow-[0_0_20px_#00bcff4d]">
              <span className="text-cyan-300 text-xs tracking-wider">SYSTEMS ONLINE</span>
            </div>
            <div className="text-[10px] font-mono text-cyan-300/40">
              {dayjs().format("DD.MM.YYYY")} / {dayjs().format("HH:mm")} UTC
            </div>
          </div>
        </motion.div>

        {/* Tab Navigation */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.1 }}
          className="flex items-center gap-1 p-1 rounded-xl border border-[#00bcff]/20 bg-black/30 backdrop-blur-sm shadow-[0_0_15px_#00bcff1a] mb-6"
        >
          {TAB_CONFIG.map(({ key, icon: Icon, label }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={cn(
                "flex items-center gap-2 px-5 py-2.5 rounded-lg text-[11px] uppercase tracking-widest font-bold transition-all duration-300",
                activeTab === key
                  ? "bg-[#00bcff]/15 text-[#00bcff] shadow-[0_0_15px_rgba(0,188,255,0.15)] border border-[#00bcff]/30"
                  : "text-[#94a3b8] hover:text-[#e0f2fe] hover:bg-[#00bcff]/5 border border-transparent"
              )}
            >
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </motion.div>

        {/* ── Main Grid Layout ── */}
        <div className="flex-1 grid grid-cols-12 gap-5 overflow-hidden">
          {/* Left Column - Holographic Display + System Status */}
          {activeTab === "tools" && (
            <motion.div
              initial={{ opacity: 0, x: -50 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 }}
              className="col-span-4 flex flex-col gap-5 overflow-y-auto"
            >
              {/* Main Holographic Circle */}
              <div className="relative aspect-square max-h-[320px]">
                <HolographicCircle />
              </div>

              {/* System Status Cards */}
              <SystemStatusGrid />
            </motion.div>
          )}

          {/* Center Column - Main Content Area */}
          <motion.div
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className={cn(
              "flex flex-col gap-5 overflow-y-auto",
              activeTab === "tools" ? "col-span-5" : "col-span-12"
            )}
          >
            {/* Tools Tab Content */}
            {activeTab === "tools" && (
              <>
                {/* Top Metrics */}
                <div className="grid grid-cols-2 gap-4">
                  <MetricCard
                    icon={<IconCpu className="size-5" />}
                    label="CPU LOAD"
                    value="47%"
                    trend="+2.3%"
                  />
                  <MetricCard
                    icon={<IconDatabase className="size-5" />}
                    label="MEMORY"
                    value="8.2 GB"
                    trend="-1.1%"
                  />
                  <MetricCard
                    icon={<IconBolt className="size-5" />}
                    label="POWER"
                    value="89%"
                    trend="+5.2%"
                  />
                  <MetricCard
                    icon={<IconActivity className="size-5" />}
                    label="NETWORK"
                    value="12.4 MB/s"
                    trend="+8.7%"
                  />
                </div>

                {/* Active Protocols / Tool Status */}
                <DataPanel
                  title="ACTIVE PROTOCOLS"
                  items={groupedTools.flatMap(([, items]) =>
                    items.slice(0, 4).map((tool) => ({
                      label: tool.name,
                      status: tool.status === "enabled" ? "ACTIVE" : "STANDBY",
                      progress: tool.status === "enabled" ? 94 : 30,
                    }))
                  )}
                />

                {/* Memory Network */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <IconCircleDot className="w-3.5 h-3.5 text-[#00bcff]" />
                      <span className="text-cyan-300 text-sm tracking-widest font-semibold">MEMORY NETWORK</span>
                    </div>
                  </div>
                  <div className="rounded-2xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-5 overflow-hidden shadow-[0_0_30px_#00bcff1a,inset_0_0_20px_#00bcff0d]">
                    <MemoryGraph
                      nodes={sessionMemoryGraph?.nodes ?? []}
                      edges={sessionMemoryGraph?.edges ?? []}
                    />
                  </div>
                </div>

                {/* Tool Grid - Compact */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-cyan-300 text-sm tracking-widest font-semibold">TOOL REGISTRY</span>
                    <span className="text-[9px] uppercase tracking-widest text-[#94a3b8] font-bold">{filteredToolCount} Active</span>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    {groupedTools.flatMap(([, items]) =>
                      items.map((tool) => (
                        <div
                          key={tool.name}
                          className="group relative rounded-xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-3 overflow-hidden shadow-[0_0_20px_#00bcff1a] hover:border-[#00bcff]/40 transition-all duration-300"
                        >
                          <div className="flex items-start justify-between gap-2 mb-2">
                            <div className="flex-1 min-w-0">
                              <h4 className="font-bold text-[11px] tracking-tight uppercase text-cyan-100 group-hover:text-[#00bcff] transition-colors truncate">
                                {tool.name}
                              </h4>
                            </div>
                            <Switch
                              checked={tool.status !== "disabled"}
                              disabled={pendingToolName === tool.name}
                              onCheckedChange={(checked) => toggleTool(tool.name, checked)}
                              className="data-[state=checked]:bg-[#00bcff] scale-75"
                            />
                          </div>
                          <p className="text-[10px] text-cyan-100/50 leading-relaxed line-clamp-2">
                            {tool.description}
                          </p>
                          <Badge
                            className={cn(
                              "rounded-md text-[7px] uppercase tracking-widest font-black px-1.5 py-0.5 mt-2",
                              tool.status === "enabled"
                                ? "bg-[#00bcff]/20 text-[#00bcff] border border-[#00bcff]/30"
                                : "bg-white/5 text-[#94a3b8] border border-white/10"
                            )}
                          >
                            {tool.status}
                          </Badge>

                          {/* Scan Line */}
                          <motion.div
                            className="absolute inset-x-0 h-px bg-gradient-to-r from-transparent via-[#00bcff]/30 to-transparent"
                            animate={{ y: [0, 80] }}
                            transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                          />
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </>
            )}

            {/* Skills Tab */}
            {activeTab === "skills" && <SkillsPage embedded />}

            {/* Agents Tab */}
            {activeTab === "agents" && <AgentsPage embedded />}

            {/* Research Tab */}
            {activeTab === "research" && <ResearchPage />}
          </motion.div>

          {/* Right Column - Circular Stats + Terminal */}
          {activeTab === "tools" && (
            <motion.div
              initial={{ opacity: 0, x: 50 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.4 }}
              className="col-span-3 flex flex-col gap-5 overflow-y-auto"
            >
              <CircularProgress
                icon={<IconShield className="size-7" />}
                label="DEFENSE"
                percentage={92}
                color="cyan"
              />
              <CircularProgress
                icon={<IconCpu className="size-7" />}
                label="UPTIME"
                percentage={99}
                color="blue"
              />
              <CircularProgress
                icon={<IconActivity className="size-7" />}
                label="CONNECTIVITY"
                percentage={87}
                color="cyan"
              />

              {/* Subagent Manifest */}
              <div className="rounded-2xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-4 shadow-[0_0_30px_#00bcff1a]">
                <div className="flex items-center gap-2 mb-3">
                  <IconCircleDot className="size-3.5 text-[#00bcff]" />
                  <span className="text-cyan-300 text-[10px] tracking-[0.2em] uppercase font-semibold">Subagents</span>
                </div>
                <div className="space-y-2">
                  {sessionSubagents === null ? (
                    <div className="text-[10px] text-cyan-100/50 flex items-center gap-2">
                      <motion.div
                        className="w-2 h-2 rounded-full bg-[#00bcff]"
                        animate={{ opacity: [0.3, 1, 0.3] }}
                        transition={{ duration: 1.5, repeat: Infinity }}
                      />
                      Loading...
                    </div>
                  ) : sessionSubagents.length === 0 ? (
                    <div className="text-[10px] text-cyan-100/40">No subagents active</div>
                  ) : (
                    sessionSubagents.map((task) => (
                      <div key={task.id} className="group rounded-lg border border-[#00bcff]/15 bg-black/30 p-2.5 hover:border-[#00bcff]/30 transition-all cursor-pointer">
                        <div className="flex justify-between items-center mb-1">
                          <span className="font-bold text-[10px] uppercase tracking-tight text-cyan-100 group-hover:text-[#00bcff] transition-colors truncate">
                            {task.label || task.id}
                          </span>
                          <span className={cn(
                            "text-[7px] uppercase font-mono px-1.5 py-0.5 rounded-md",
                            task.status === "completed"
                              ? "bg-green-400/15 text-green-400"
                              : task.status === "failed"
                              ? "bg-red-400/15 text-red-400"
                              : "bg-[#00bcff]/15 text-[#00bcff]"
                          )}>
                            {task.status}
                          </span>
                        </div>
                        <div className="text-[8px] font-mono text-cyan-100/30">
                          {dayjs(task.created).format("HH:mm:ss")}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Terminal Output */}
              <TerminalOutput />
            </motion.div>
          )}
        </div>
      </div>
    </div>
  )
}