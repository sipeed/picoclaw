import { useState } from "react"

import type { ResearchNode } from "@/api/research"

interface ResearchGraphProps {
  nodes: ResearchNode[]
  selectedNodes: Set<string>
  onNodeToggle: (name: string) => void
}

const VIEWBOX_WIDTH = 800
const VIEWBOX_HEIGHT = 500

export function ResearchGraph({ nodes, selectedNodes, onNodeToggle }: ResearchGraphProps) {
  const [hoveredNode, setHoveredNode] = useState<string | null>(null)

  const connections = [
    { from: { x: 150, y: 80 }, to: { x: 400, y: 150 } },
    { from: { x: 150, y: 120 }, to: { x: 400, y: 180 } },
    { from: { x: 150, y: 160 }, to: { x: 400, y: 250 } },
    { from: { x: 150, y: 210 }, to: { x: 400, y: 300 } },
    { from: { x: 150, y: 260 }, to: { x: 400, y: 350 } },
    { from: { x: 400, y: 200 }, to: { x: 650, y: 100 } },
    { from: { x: 400, y: 250 }, to: { x: 650, y: 200 } },
  ]

  return (
    <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[#0A0A0A]">
      <svg
        viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
        className="h-[420px] w-full"
        role="img"
        aria-label="Research knowledge graph"
      >
        <defs>
          <filter id="researchGlow">
            <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
            <feMerge>
              <feMergeNode in="coloredBlur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
          </filter>
          <linearGradient id="researchGrid" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#0c1a14" />
            <stop offset="100%" stopColor="#050a08" />
          </linearGradient>
        </defs>
        
        <rect width={VIEWBOX_WIDTH} height={VIEWBOX_HEIGHT} fill="url(#researchGrid)" />
        
        {/* Grid lines */}
        {Array.from({ length: 8 }).map((_, index) => (
          <line
            key={`v-${index}`}
            x1={(VIEWBOX_WIDTH / 8) * index}
            y1="0"
            x2={(VIEWBOX_WIDTH / 8) * index}
            y2={VIEWBOX_HEIGHT}
            stroke="#0d2817"
            strokeWidth="1"
          />
        ))}
        {Array.from({ length: 6 }).map((_, index) => (
          <line
            key={`h-${index}`}
            x1="0"
            y1={(VIEWBOX_HEIGHT / 6) * index}
            x2={VIEWBOX_WIDTH}
            y2={(VIEWBOX_HEIGHT / 6) * index}
            stroke="#0d2817"
            strokeWidth="1"
          />
        ))}

        {/* Connections */}
        {connections.map((conn, i) => (
          <line
            key={`conn-${i}`}
            x1={conn.from.x}
            y1={conn.from.y}
            x2={conn.to.x}
            y2={conn.to.y}
            stroke="#1f5c34"
            strokeWidth="1.2"
            strokeOpacity="0.5"
          />
        ))}

        {/* Center knowledge base node */}
        <g>
          <circle
            cx="400"
            cy="200"
            r="30"
            fill="#10b981"
            opacity="0.1"
            filter="url(#researchGlow)"
          />
          <circle
            cx="400"
            cy="200"
            r="18"
            fill="#07110a"
            stroke="#10b981"
            strokeWidth="2.5"
          />
          <text
            x="400"
            y="203"
            textAnchor="middle"
            fill="#10b981"
            fontSize="11"
            fontWeight="bold"
            fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
          >
            KB
          </text>
        </g>

        {/* Knowledge nodes */}
        {nodes.map((node) => {
          const isSelected = selectedNodes.has(node.name)
          const isHovered = hoveredNode === node.name
          
          return (
            <g
              key={node.name}
              onClick={() => onNodeToggle(node.name)}
              onMouseEnter={() => setHoveredNode(node.name)}
              onMouseLeave={() => setHoveredNode(null)}
              className="cursor-pointer"
            >
              {/* Outer glow */}
              <circle
                cx={node.x}
                cy={node.y}
                r="22"
                fill={isSelected ? "#10b981" : "#F27D26"}
                opacity={isSelected || isHovered ? "0.15" : "0.08"}
                filter="url(#researchGlow)"
              />
              
              {/* Main node */}
              <circle
                cx={node.x}
                cy={node.y}
                r="14"
                fill="#07110a"
                stroke={isSelected ? "#10b981" : "#F27D26"}
                strokeWidth={isSelected || isHovered ? "2.5" : "1.5"}
              />
              
              {/* Inner glow */}
              <circle
                cx={node.x}
                cy={node.y}
                r="10"
                fill={isSelected ? "#10b981" : "#F27D26"}
                opacity="0.12"
              />
              
              {/* Text */}
              <text
                x={node.x}
                y={node.y + 3}
                textAnchor="middle"
                fill={isSelected ? "#10b981" : "#F27D26"}
                fontSize="9"
                fontWeight="bold"
                fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
              >
                {node.abbr}
              </text>
              
              {/* Tooltip on hover */}
              {(isHovered || isSelected) && (
                <g>
                  <rect
                    x={node.x - 40}
                    y={node.y - 38}
                    width="80"
                    height="16"
                    rx="3"
                    fill="#0A0A0A"
                    stroke="#1f5c34"
                  />
                  <text
                    x={node.x}
                    y={node.y - 27}
                    textAnchor="middle"
                    fill="#95d7a5"
                    fontSize="8"
                  >
                    {node.name.slice(0, 12)}
                  </text>
                </g>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}