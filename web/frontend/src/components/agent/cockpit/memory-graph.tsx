import { useMemo, useState } from "react"

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
  if (group === "memory") return "#72f0a0"
  if (group.startsWith("daily-")) return "#4dd0e1"
  if (group === "tool") return "#f8d66d"
  if (group === "media") return "#f395d6"
  return "#90e89f"
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
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
      <div className="overflow-hidden rounded-xl border border-[#173621] bg-[#041008]">
        <svg
          viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
          className="h-[420px] w-full"
          role="img"
          aria-label="PicoClaw memory graph"
        >
          <defs>
            <linearGradient id="memoryGrid" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#0c2014" />
              <stop offset="100%" stopColor="#050d08" />
            </linearGradient>
          </defs>
          <rect width={VIEWBOX_WIDTH} height={VIEWBOX_HEIGHT} fill="url(#memoryGrid)" />
          {Array.from({ length: 10 }).map((_, index) => (
            <line
              key={`v-${index}`}
              x1={(VIEWBOX_WIDTH / 10) * index}
              y1="0"
              x2={(VIEWBOX_WIDTH / 10) * index}
              y2={VIEWBOX_HEIGHT}
              stroke="#0d2817"
              strokeWidth="1"
            />
          ))}
          {Array.from({ length: 8 }).map((_, index) => (
            <line
              key={`h-${index}`}
              x1="0"
              y1={(VIEWBOX_HEIGHT / 8) * index}
              x2={VIEWBOX_WIDTH}
              y2={(VIEWBOX_HEIGHT / 8) * index}
              stroke="#0d2817"
              strokeWidth="1"
            />
          ))}

          {edges.map((edge) => {
            const source = nodeMap.get(edge.source)
            const target = nodeMap.get(edge.target)
            if (!source || !target) {
              return null
            }

            const active =
              selectedId != null &&
              (selectedId === edge.source || selectedId === edge.target)

            return (
              <line
                key={`${edge.source}-${edge.target}-${edge.kind}`}
                x1={source.x}
                y1={source.y}
                x2={target.x}
                y2={target.y}
                stroke={active ? "#9dfdbb" : "#1f5c34"}
                strokeOpacity={active ? 0.9 : 0.45}
                strokeWidth={active ? 2.5 : 1.4}
              />
            )
          })}

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
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={radius + 10}
                  fill={active ? `${color}22` : "transparent"}
                />
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={radius}
                  fill="#07110a"
                  stroke={color}
                  strokeWidth={active ? 3 : 2}
                />
                <text
                  x={node.x}
                  y={node.y + 4}
                  textAnchor="middle"
                  fill={color}
                  fontSize={node.kind === "root" ? "12" : "10"}
                  fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
                >
                  {node.kind === "root" ? node.label : node.label.slice(0, 18)}
                </text>
              </g>
            )
          })}
        </svg>
      </div>

      <div className="space-y-3 rounded-xl border border-[#173621] bg-[#050d08] p-4">
        {selectedNode ? (
          <>
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="font-mono text-sm uppercase tracking-[0.22em] text-[#95d7a5]">
                  Node Focus
                </p>
                <h3 className="mt-2 text-sm font-semibold text-[#effff3]">
                  {selectedNode.label}
                </h3>
              </div>
              <Badge
                variant="outline"
                className={cn(
                  "border-[#2a6b3d] text-[#9fd8ae]",
                  selectedNode.kind === "tool" && "border-[#8c7931] text-[#f8d66d]",
                )}
              >
                {selectedNode.kind}
              </Badge>
            </div>
            <p className="text-sm leading-relaxed text-[#9cc8a8]">
              {selectedNode.preview || "No extra preview for this node yet."}
            </p>
            <div className="grid gap-2 text-xs text-[#74a282]">
              <div className="flex items-center justify-between rounded-lg border border-[#13301d] px-3 py-2">
                <span>Cluster</span>
                <span className="font-mono uppercase text-[#d7f9df]">
                  {selectedNode.group}
                </span>
              </div>
              <div className="flex items-center justify-between rounded-lg border border-[#13301d] px-3 py-2">
                <span>Weight</span>
                <span className="font-mono text-[#d7f9df]">
                  {selectedNode.weight ?? 1}
                </span>
              </div>
            </div>
          </>
        ) : (
          <p className="text-sm text-[#8bb39a]">No graph data available.</p>
        )}
      </div>
    </div>
  )
}
