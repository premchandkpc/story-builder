import { memo } from "react"
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"

type SceneNodeData = {
  label: string
  status: "draft" | "generated" | "accepted" | "stale"
  beatIntent: string
  pov: string
  tone: string
  targetWords: number
}

const statusColors: Record<string, string> = {
  draft: "#94a3b8",
  generated: "#f59e0b",
  accepted: "#22c55e",
  stale: "#ef4444",
}

const statusLabels: Record<string, string> = {
  draft: "Draft",
  generated: "Generated",
  accepted: "Accepted",
  stale: "Stale",
}

function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  return (
    <div
      style={{
        background: "#1e293b",
        border: `2px solid ${statusColors[data.status] || statusColors.draft}`,
        borderRadius: 8,
        padding: "12px 16px",
        minWidth: 200,
        color: "#e2e8f0",
        fontFamily: "system-ui, sans-serif",
        fontSize: 13,
      }}
    >
      <Handle type="target" position={Position.Left} />
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 6,
        }}
      >
        <strong style={{ fontSize: 14 }}>{data.label}</strong>
        <span
          style={{
            background: statusColors[data.status],
            color: "#000",
            padding: "1px 8px",
            borderRadius: 10,
            fontSize: 11,
            fontWeight: 600,
          }}
        >
          {statusLabels[data.status]}
        </span>
      </div>
      <div style={{ fontSize: 12, color: "#94a3b8", marginBottom: 4 }}>
        {data.beatIntent}
      </div>
      <div style={{ fontSize: 11, color: "#64748b" }}>
        {data.pov} · {data.tone} · {data.targetWords}w
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  )
}

export default memo(SceneNode)
