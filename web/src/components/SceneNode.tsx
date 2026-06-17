// ---- SceneNode: A custom React Flow node ----
// This component renders a single node in the story graph (DAG).
// It displays the scene's label, status badge, beat intent, and metadata.

// memo: a React higher-order component that prevents re-rendering
// if the props haven't changed (performance optimization).
import { memo } from "react"

// Handle: the small connection circles on node edges (input/output ports).
// Position: enum for handle placement (LEFT for target, RIGHT for source).
// NodeProps: TypeScript type that describes the props React Flow passes to custom nodes.
// Node: TypeScript type for a React Flow node object.
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"

// ---- Local type definition ----
// Describes the custom data attached to this node type.
type SceneNodeData = {
  label: string                                         // display label
  status: "draft" | "generated" | "accepted" | "stale" // generation lifecycle status
  beatIntent: string                                    // what this scene accomplishes
  pov: string                                           // point of view
  tone: string                                          // emotional tone
  targetWords: number                                   // target word count
}

// ---- Lookup tables ----
// statusColors maps status strings to CSS color values (used for the border and badge).
const statusColors: Record<string, string> = {
  draft:     "#94a3b8", // gray
  generated: "#f59e0b", // amber/orange
  accepted:  "#22c55e", // green
  stale:     "#ef4444", // red
}

// statusLabels maps status to human-readable text for the badge.
const statusLabels: Record<string, string> = {
  draft:     "Draft",
  generated: "Generated",
  accepted:  "Accepted",
  stale:     "Stale",
}

// ---- Component ----
// NodeProps<Node<SceneNodeData>> is the type for props that React Flow passes.
// React Flow automatically injects `data`, `id`, `selected`, etc.
// We destructure only `data` since that's all we need.
function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  return (
    // The outer div is the node's visual container.
    <div
      style={{
        background: "#1e293b",                                        // dark card background
        border: `2px solid ${statusColors[data.status] || statusColors.draft}`, // color-coded border
        borderRadius: 8,
        padding: "12px 16px",
        minWidth: 200,                                                // minimum width for readability
        color: "#e2e8f0",
        fontFamily: "system-ui, sans-serif",
        fontSize: 13,
      }}
    >
      {/*
        Handle (target) — input port on the LEFT side.
        type="target" means edges connect INTO this handle.
        Position.Left places it on the left edge of the node.
      */}
      <Handle type="target" position={Position.Left} />

      {/* Header row: label on the left, status badge on the right */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 6,
        }}
      >
        <strong style={{ fontSize: 14 }}>{data.label}</strong>

        {/* Status badge — a colored pill showing draft/generated/accepted/stale */}
        <span
          style={{
            background: statusColors[data.status],
            color: "#000",
            padding: "1px 8px",
            borderRadius: 10,       // rounded pill shape
            fontSize: 11,
            fontWeight: 600,
          }}
        >
          {statusLabels[data.status]}
        </span>
      </div>

      {/* Beat intent — what this scene is supposed to accomplish */}
      <div style={{ fontSize: 12, color: "#94a3b8", marginBottom: 4 }}>
        {data.beatIntent}
      </div>

      {/* Metadata row: POV · Tone · Word count */}
      <div style={{ fontSize: 11, color: "#64748b" }}>
        {data.pov} · {data.tone} · {data.targetWords}w
      </div>

      {/*
        Handle (source) — output port on the RIGHT side.
        type="source" means edges START from this handle.
        Position.Right places it on the right edge of the node.
      */}
      <Handle type="source" position={Position.Right} />
    </div>
  )
}

// ---- Export ----
// memo() wraps the component so it only re-renders when its props change.
// This is important for React Flow performance when many nodes exist.
export default memo(SceneNode)
