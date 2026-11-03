import { useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useTopology } from "../api/hooks"
import SceneCard from "./SceneCard"

interface StoryReadViewProps {
  storyId: string
}

export default function StoryReadView({ storyId }: StoryReadViewProps) {
  const navigate = useNavigate()
  const { data: topology, isLoading } = useTopology(storyId)

  const onOpenInGraph = useCallback((sceneId: string) => {
    navigate(`/stories/${storyId}/graph?scene=${sceneId}`, { replace: true })
  }, [navigate, storyId])

  if (isLoading) {
    return (
      <div style={{
        display: "flex", alignItems: "center", justifyContent: "center",
        height: "100%", color: "var(--text-dim)", fontSize: 13,
      }}>
        Loading scenes...
      </div>
    )
  }

  if (!topology || topology.nodes.length === 0) {
    return (
      <div style={{
        display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
        height: "100%", gap: 8, color: "var(--text-dim)",
      }}>
        <span style={{ fontSize: 14, fontStyle: "italic", fontFamily: "var(--font-heading)" }}>
          No scenes yet
        </span>
        <span style={{ fontSize: 11, color: "var(--text-faint)" }}>
          Switch to Graph mode to add scenes
        </span>
      </div>
    )
  }

  const ordered = topology.topological_order
    .map((id) => topology.nodes.find((n) => n.id === id))
    .filter((n): n is NonNullable<typeof n> => n != null)

  return (
    <div style={{
      height: "100%", overflowY: "auto",
      display: "flex", flexDirection: "column", alignItems: "center",
    }}>
      <div style={{
        maxWidth: 720, width: "100%", padding: "24px 32px 48px",
        display: "flex", flexDirection: "column", gap: 12,
      }}>
        <div style={{
          fontSize: 10, color: "var(--text-faint)", letterSpacing: "0.05em",
          textTransform: "uppercase", marginBottom: 8,
        }}>
          {topology.nodes.length} scenes · {topology.edges.length} connections
        </div>

        {ordered.map((node) => (
          <SceneCard
            key={node.id}
            node={node}
            nodes={topology.nodes}
            edges={topology.edges}
            onOpenInGraph={onOpenInGraph}
          />
        ))}
      </div>
    </div>
  )
}
