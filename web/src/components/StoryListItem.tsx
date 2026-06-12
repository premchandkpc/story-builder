import { useNavigate } from "react-router-dom"
import type { Story } from "../api/types"

interface StoryListItemProps {
  story: Story
  nodeCount: number
  accepted: number
  generated: number
  stale: number
  isActive: boolean
}

export default function StoryListItem({ story, nodeCount, accepted, generated, stale, isActive }: StoryListItemProps) {
  const navigate = useNavigate()
  const nodeInfo =
    nodeCount === 0
      ? "No nodes"
      : `${nodeCount} nodes · ${accepted} accepted · ${generated} generated`
  const statusColor =
    stale > 0 ? "#ef4444" : nodeCount > 0 && accepted === nodeCount ? "#22c55e" : nodeCount > 0 ? "#eab308" : "#64748b"
  const date = new Date(story.created_at).toLocaleDateString()

  return (
    <button
      onClick={() => navigate(`/stories/${story.id}`)}
      style={{
        width: "100%",
        padding: "10px 12px",
        background: isActive ? "#1e293b" : "transparent",
        border: "none",
        borderBottom: "1px solid #1e293b",
        color: "#e2e8f0",
        cursor: "pointer",
        textAlign: "left",
        fontSize: 14,
        display: "flex",
        flexDirection: "column",
        gap: 2,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span style={{ width: 8, height: 8, borderRadius: "50%", background: statusColor, flexShrink: 0 }} />
        <span style={{ fontWeight: isActive ? 700 : 500 }}>{story.title}</span>
      </div>
      <div style={{ fontSize: 11, color: "#64748b", paddingLeft: 14 }}>{nodeInfo} · {date}</div>
    </button>
  )
}
