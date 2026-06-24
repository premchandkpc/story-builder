import { useCallback, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useTopology, useSceneNarrativeEvents, useTimeline } from "../api/hooks"
import { api } from "../api/client"
import { useQuery } from "@tanstack/react-query"
import type { Generation } from "../api/types"
import SceneSlideDown from "./SceneSlideDown"
import SceneConnections from "./SceneConnections"

interface SceneDrawerProps {
  storyId: string
  sceneId: string | null
  onClose: () => void
}

export default function SceneDrawer({ storyId, sceneId, onClose }: SceneDrawerProps) {
  const navigate = useNavigate()
  const { data: topology } = useTopology(storyId)
  const { data: events } = useSceneNarrativeEvents(storyId, sceneId, 20)
  const { data: timelineEvents } = useTimeline(storyId)

  const { data: generations = [] } = useQuery<Generation[]>({
    queryKey: ["generations", storyId, sceneId],
    queryFn: () => api.generations.list(storyId, sceneId!),
    enabled: !!sceneId,
  })

  const node = topology?.nodes.find((n) => n.id === sceneId)
  const accepted = generations.find((g) => g.accepted)
  const lastGen = generations[generations.length - 1]
  const prose = accepted?.output || lastGen?.output || ""

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") onClose()
  }, [onClose])

  useEffect(() => {
    if (sceneId) {
      document.addEventListener("keydown", handleKeyDown)
      return () => document.removeEventListener("keydown", handleKeyDown)
    }
  }, [sceneId, handleKeyDown])

  if (!sceneId || !node) return null

  const sceneTimeline = timelineEvents?.filter((e) => e.node_id === sceneId) || []

  return (
    <>
      <div
        onClick={onClose}
        style={{
          position: "fixed", inset: 0, background: "rgba(0,0,0,0.3)",
          zIndex: 99,
        }}
      />
      <div style={{
        position: "fixed", top: 0, right: 0, bottom: 0, width: 420,
        background: "var(--bg-warm)", borderLeft: "1px solid var(--border)",
        boxShadow: "-4px 0 24px rgba(0,0,0,0.1)",
        zIndex: 100, display: "flex", flexDirection: "column",
        fontFamily: "var(--font-body)", fontSize: 13, color: "var(--text)",
      }}>
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "space-between",
          padding: "16px 20px", borderBottom: "1px solid var(--border)",
        }}>
          <h3 style={{
            margin: 0, fontSize: 15, fontWeight: 600,
            fontFamily: "var(--font-heading)", color: "var(--accent)",
          }}>
            {node.title || "Untitled Scene"}
          </h3>
          <div style={{ display: "flex", gap: 8 }}>
            <button
              onClick={() => navigate(`/stories/${storyId}/graph?scene=${sceneId}`, { replace: true })}
              className="btn-press"
              style={{
                padding: "4px 10px", fontSize: 10, color: "var(--accent)",
                border: "1px solid var(--accent-dim)", borderRadius: "var(--radius-sm)",
                background: "none", cursor: "pointer", textTransform: "uppercase",
                letterSpacing: "0.04em",
              }}
            >
              Open in Graph
            </button>
            <button
              onClick={onClose}
              className="btn-press"
              style={{
                background: "none", border: "none", color: "var(--text-faint)",
                cursor: "pointer", padding: 4, display: "flex", fontSize: 16,
              }}
            >
              ✕
            </button>
          </div>
        </div>

        <div style={{ flex: 1, overflowY: "auto", padding: "12px 20px 40px" }}>
          <div style={{ fontSize: 11, color: "var(--text-dim)", lineHeight: 1.6, marginBottom: 12 }}>
            <div style={{ display: "flex", gap: 12 }}>
              <span>Status: <strong>{node.status}</strong></span>
              <span>POV: <strong>{node.pov}</strong></span>
              <span>Tone: <strong>{node.tone}</strong></span>
              <span>Words: <strong>{node.target_words}</strong></span>
            </div>
            {node.beat_intent && (
              <div style={{ marginTop: 4, fontStyle: "italic", color: "var(--text-faint)" }}>
                {node.beat_intent}
              </div>
            )}
          </div>

          {prose && (
            <div style={{
              padding: "12px 16px", background: "var(--bg)",
              borderRadius: "var(--radius-md)", marginBottom: 12,
              fontSize: 13, lineHeight: 1.7, whiteSpace: "pre-wrap",
            }}>
              {prose}
            </div>
          )}

          <SceneSlideDown label="Connections">
            <SceneConnections sceneId={sceneId} nodes={topology?.nodes || []} edges={topology?.edges || []} />
          </SceneSlideDown>

          {generations.length > 0 && (
            <SceneSlideDown label="Generations" badge={generations.length}>
              {generations.map((g, i) => (
                <div key={g.id} style={{
                  display: "flex", alignItems: "center", gap: 6,
                  padding: "4px 0", borderBottom: i < generations.length - 1 ? "1px solid var(--border)" : "none",
                  fontSize: 11,
                }}>
                  <span style={{
                    width: 6, height: 6, borderRadius: "50%",
                    background: g.accepted ? "var(--success)" : g.status === "error" ? "var(--danger)" : "var(--text-faint)",
                    flexShrink: 0,
                  }} />
                  <span style={{ color: "var(--text-dim)", flex: 1 }}>
                    {g.model || "unknown"} · {g.status}
                    {g.critic_score != null && <span> · score: {g.critic_score}</span>}
                  </span>
                  {g.total_tokens && <span style={{ color: "var(--text-faint)" }}>{g.total_tokens}t</span>}
                </div>
              ))}
            </SceneSlideDown>
          )}

          {sceneTimeline.length > 0 && (
            <SceneSlideDown label="Timeline" badge={sceneTimeline.length}>
              {sceneTimeline.map((evt) => (
                <div key={evt.id} style={{ padding: "2px 0", fontSize: 11, color: "var(--text-dim)" }}>
                  <span style={{ color: "var(--text-faint)", marginRight: 6 }}>
                    {new Date(evt.created_at).toLocaleTimeString()}
                  </span>
                  {evt.event_type}: {evt.description || evt.event_data?.progression}
                </div>
              ))}
            </SceneSlideDown>
          )}

          {events && events.length > 0 && (
            <SceneSlideDown label="Narrative Events" badge={events.length}>
              {events.map((evt) => (
                <div key={evt.id} style={{ padding: "2px 0", fontSize: 11, color: "var(--text-dim)" }}>
                  [{evt.event_type}] {evt.description}
                </div>
              ))}
            </SceneSlideDown>
          )}
        </div>
      </div>
    </>
  )
}
