import { useMemo, useState } from "react"
import { motion } from "motion/react"

import type {
  PicoMemoryGraphEdge,
  PicoMemoryGraphNode,
} from "@/api/pico"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

const VIEWBOX_WIDTH = 1000
const VIEWBOX_HEIGHT = 560

type PositionedNode = PicoMemoryGraphNode & { x: number; y: number }

interface MemoryGraphProps {
  nodes: PicoMemoryGraphNode[]
  edges: PicoMemoryGraphEdge[]
}

const GROUP_COLUMNS: Record<string, number> = {
  memory: 170,
  "daily-1": 295,
  "daily-2": 295,
  "daily-3": 295,
  session: 700,
  tool: 840,
  media: 920,
}

function groupColor(group: string) {
  if (group === "memory") return "#00bcff"
  if (group.startsWith("daily-")) return "#22d3ee"
  if (group === "tool") return "#38bdf8"
  if (group === "media") return "#7dd3fc"
  return "#06b6d4"
}

function computeLayout(nodes: PicoMemoryGraphNode[]): PositionedNode[] {
  const groups = new Map<string, PicoMemoryGraphNode[]>()
  for (const node of nodes) {
    const items = groups.get(node.group) ?? []
    items.push(node)
    groups.set(node.group, items)
  }

  const layout: PositionedNode[] = []
  for (const [group, items] of groups.entries()) {
    const x = GROUP_COLUMNS[group] ?? 500
    const count = items.length
    const gap = VIEWBOX_HEIGHT / (count + 1)
    items.forEach((node, index) => {
      const offset =
        group === "session" && node.kind === "root"
          ? 0
          : Math.sin((index + 1) * 1.7) * 12
      layout.push({
        ...node,
        x,
        y: Math.max(52, Math.min(VIEWBOX_HEIGHT - 52, gap * (index + 1) + offset)),
      })
    })
  }

  return layout.sort((left, right) => left.x - right.x || left.y - right.y)
}

export function MemoryGraph({ nodes, edges }: MemoryGraphProps) {
  const [selectedId, setSelectedId] = useState<string | null>(nodes[0]?.id ?? null)

  const positionedNodes = useMemo(() => computeLayout(nodes), [nodes])
  const nodeMap = useMemo(
    () => new Map(positionedNodes.map((node) => [node.id, node])),
    [positionedNodes],
  )
  const selectedNode =
    positionedNodes.find((node) => node.id === selectedId) ?? positionedNodes[0]

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_240px]">
      <div className="overflow-hidden rounded-xl border border-[#00bcff]/20 bg-[#060e1a] relative">
        {/* Holographic scan line overlay */}
        <motion.div
          className="absolute inset-0 z-10 pointer-events-none bg-gradient-to-b from-transparent via-[#00bcff]/5 to-transparent"
          animate={{ y: ["-100%", "100%"] }}
          transition={{ duration: 4, repeat: Infinity, ease: "linear" }}
        />

        <svg
          viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
          className="h-[380px] w-full"
          role="img"
          aria-label="PicoClaw memory graph"
        >
          <defs>
            {/* 3D Grid Gradient */}
            <linearGradient id="jarvisGrid" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#0a1628" />
              <stop offset="50%" stopColor="#060e1a" />
              <stop offset="100%" stopColor="#0a1628" />
            </linearGradient>

            {/* Glow filter for edges */}
            <filter id="jarvisGlow">
              <feGaussianBlur stdDeviation="4" result="coloredBlur" />
              <feMerge>
                <feMergeNode in="coloredBlur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            {/* Stronger glow for active nodes */}
            <filter id="jarvisGlowStrong">
              <feGaussianBlur stdDeviation="8" result="coloredBlur" />
              <feMerge>
                <feMergeNode in="coloredBlur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            {/* Holographic radial gradient for nodes */}
            <radialGradient id="holoNodeGrad" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#00bcff" stopOpacity="0.3" />
              <stop offset="70%" stopColor="#00bcff" stopOpacity="0.1" />
              <stop offset="100%" stopColor="#00bcff" stopOpacity="0.02" />
            </radialGradient>
          </defs>

          {/* Background */}
          <rect width={VIEWBOX_WIDTH} height={VIEWBOX_HEIGHT} fill="url(#jarvisGrid)" />

          {/* 3D Perspective Grid Lines */}
          {Array.from({ length: 20 }).map((_, index) => (
            <line
              key={`v-${index}`}
              x1={(VIEWBOX_WIDTH / 20) * index}
              y1="0"
              x2={(VIEWBOX_WIDTH / 20) * index}
              y2={VIEWBOX_HEIGHT}
              stroke="rgba(0, 188, 255, 0.03)"
              strokeWidth="1"
            />
          ))}
          {Array.from({ length: 12 }).map((_, index) => (
            <line
              key={`h-${index}`}
              x1="0"
              y1={(VIEWBOX_HEIGHT / 12) * index}
              x2={VIEWBOX_WIDTH}
              y2={(VIEWBOX_HEIGHT / 12) * index}
              stroke="rgba(0, 188, 255, 0.03)"
              strokeWidth="1"
            />
          ))}

          {/* Edges with holographic glow */}
          {edges.map((edge) => {
            const source = nodeMap.get(edge.source)
            const target = nodeMap.get(edge.target)
            if (!source || !target) return null

            const active =
              selectedId != null &&
              (selectedId === edge.source || selectedId === edge.target)

            return (
              <g key={`${edge.source}-${edge.target}-${edge.kind}`}>
                {/* Glow layer */}
                {active && (
                  <line
                    x1={source.x}
                    y1={source.y}
                    x2={target.x}
                    y2={target.y}
                    stroke="#00bcff"
                    strokeOpacity={0.3}
                    strokeWidth={6}
                    filter="url(#jarvisGlowStrong)"
                  />
                )}
                {/* Main edge */}
                <line
                  x1={source.x}
                  y1={source.y}
                  x2={target.x}
                  y2={target.y}
                  stroke={active ? "#00bcff" : "rgba(0, 188, 255, 0.12)"}
                  strokeOpacity={active ? 0.9 : 0.5}
                  strokeWidth={active ? 2 : 1}
                  filter={active ? "url(#jarvisGlow)" : undefined}
                />
                {/* Animated pulse along edge when active */}
                {active && (
                  <circle r="3" fill="#00bcff" filter="url(#jarvisGlow)">
                    <animateMotion
                      dur="2s"
                      repeatCount="indefinite"
                      path={`M${source.x},${source.y} L${target.x},${target.y}`}
                    />
                  </circle>
                )}
              </g>
            )
          })}

          {/* Nodes with 3D holographic style */}
          {positionedNodes.map((node) => {
            const active = node.id === selectedNode?.id
            const color = groupColor(node.group)
            const radius =
              node.kind === "root" ? 24 : node.kind === "document" ? 18 : 14

            return (
              <g
                key={node.id}
                onMouseEnter={() => setSelectedId(node.id)}
                onClick={() => setSelectedId(node.id)}
                className="cursor-pointer"
              >
                {/* Outer holographic ring - pulsing */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={radius + 18}
                  fill="transparent"
                  stroke={active ? color : "transparent"}
                  strokeWidth="0.5"
                  strokeOpacity={active ? 0.2 : 0}
                  strokeDasharray="4 4"
                >
                  {active && (
                    <animate
                      attributeName="stroke-dashoffset"
                      from="0"
                      to="8"
                      dur="1s"
                      repeatCount="indefinite"
                    />
                  )}
                </circle>

                {/* Glow halo */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={radius + 10}
                  fill={active ? `${color}15` : "transparent"}
                />

                {/* Holographic field */}
                {active && (
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={radius + 6}
                    fill="url(#holoNodeGrad)"
                    filter="url(#jarvisGlow)"
                  />
                )}

                {/* Main node circle */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={radius}
                  fill="#0a1628"
                  stroke={color}
                  strokeWidth={active ? 2.5 : 1.5}
                  filter={active ? "url(#jarvisGlow)" : undefined}
                />

                {/* Inner glow ring */}
                {active && (
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={radius - 4}
                    fill="transparent"
                    stroke={color}
                    strokeWidth="0.5"
                    strokeOpacity="0.4"
                  />
                )}

                {/* Label */}
                <text
                  x={node.x}
                  y={node.y + 4}
                  textAnchor="middle"
                  fill={active ? "#e0f2fe" : color}
                  fontSize={node.kind === "root" ? "11" : "9"}
                  fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
                  fontWeight={active ? "600" : "400"}
                >
                  {node.kind === "root" ? node.label : node.label.slice(0, 16)}
                </text>
              </g>
            )
          })}
        </svg>
      </div>

      {/* Node Detail Panel */}
      <div className="space-y-3 rounded-xl border border-[#00bcff]/20 bg-black/40 backdrop-blur-sm p-4 shadow-[0_0_20px_#00bcff1a]">
        {selectedNode ? (
          <>
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="font-mono text-[9px] uppercase tracking-[0.25em] text-[#00bcff]">
                  Node Focus
                </p>
                <h3 className="mt-2 text-sm font-semibold text-cyan-100">
                  {selectedNode.label}
                </h3>
              </div>
              <Badge
                variant="outline"
                className={cn(
                  "border-[#00bcff]/30 text-[#00bcff] text-[9px]",
                  selectedNode.kind === "tool" && "border-[#38bdf8]/30 text-[#38bdf8]",
                )}
              >
                {selectedNode.kind}
              </Badge>
            </div>
            <p className="text-[11px] leading-relaxed text-cyan-100/50">
              {selectedNode.preview || "No extra preview for this node yet."}
            </p>
            <div className="grid gap-2 text-[11px] text-cyan-100/50">
              <div className="flex items-center justify-between rounded-lg border border-[#00bcff]/10 bg-[#00bcff]/3 px-3 py-2">
                <span>Cluster</span>
                <span className="font-mono uppercase text-cyan-100">
                  {selectedNode.group}
                </span>
              </div>
              <div className="flex items-center justify-between rounded-lg border border-[#00bcff]/10 bg-[#00bcff]/3 px-3 py-2">
                <span>Weight</span>
                <span className="font-mono text-cyan-100">
                  {selectedNode.weight ?? 1}
                </span>
              </div>
            </div>
          </>
        ) : (
          <p className="text-[11px] text-cyan-100/40">No graph data available.</p>
        )}
      </div>
    </div>
  )
}