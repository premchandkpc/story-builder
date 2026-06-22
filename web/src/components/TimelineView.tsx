import { useTimeline, useCrossStoryTimeline } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface TimelineViewProps {
  storyId: string
}

const eventTypeColor: Record<string, string> = {
  scene: "var(--accent)",
  choice: "#8b5cf6",
  branch: "#f59e0b",
  converge: "#22c55e",
  climax: "#ef4444",
}

export default function TimelineView({ storyId }: TimelineViewProps) {
  const { data: localEvents, isLoading: localLoading } = useTimeline(storyId)
  const { data: crossEvents, isLoading: crossLoading } = useCrossStoryTimeline(storyId)

  if (localLoading || crossLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 32 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  const allEvents = [...(localEvents || []), ...(crossEvents || [])]
    .sort((a, b) => a.order - b.order || a.created_at.localeCompare(b.created_at))

  if (allEvents.length === 0) {
    return (
      <div style={{ padding: 16, color: "var(--text-faint)", fontSize: 12, fontStyle: "italic" }}>
        No timeline events yet. Generate scenes to populate the timeline.
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, ...fadeInStyle }}>
      <h3 style={{
        margin: 0, fontSize: 14,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
        fontWeight: 600, letterSpacing: "0.02em",
      }}>
        Timeline
        {crossEvents && crossEvents.length > 0 && (
          <span style={{
            fontSize: 10, color: "var(--text-dim)", fontWeight: 400,
            marginLeft: 8, background: "var(--surface)", padding: "2px 6px",
            borderRadius: "var(--radius-sm)", verticalAlign: "middle",
          }}>
            +{crossEvents.length} cross-story
          </span>
        )}
      </h3>

      <div style={{ position: "relative", paddingLeft: 20 }}>
        <div style={{
          position: "absolute", left: 8, top: 0, bottom: 0,
          width: 1, background: "var(--border)",
        }} />
        {allEvents.map((evt, i) => {
          const isCrossStory = crossEvents?.some((c) => c.id === evt.id)
          return (
            <div key={evt.id} style={{
              position: "relative",
              padding: "6px 0 6px 12px",
              animation: `slideUp 0.2s var(--ease-out) ${i * 0.03}s both`,
            }}>
              <div style={{
                position: "absolute", left: -12, top: 10,
                width: 8, height: 8, borderRadius: "50%",
                background: eventTypeColor[evt.event_type || "scene"] || "var(--accent)",
                border: "2px solid var(--bg-warm)",
              }} />
              <div style={{
                display: "flex", alignItems: "center", gap: 6, marginBottom: 2,
              }}>
                <span style={{ fontSize: 10, color: "var(--text-faint)", fontFamily: "var(--font-mono)" }}>
                  #{evt.order}
                </span>
                <span style={{
                  fontSize: 9, padding: "1px 5px", borderRadius: "var(--radius-sm)",
                  background: `${eventTypeColor[evt.event_type || "scene"] || "var(--accent")}20`,
                  color: eventTypeColor[evt.event_type || "scene"] || "var(--accent)",
                  fontWeight: 600, textTransform: "uppercase",
                  letterSpacing: "0.04em",
                }}>
                  {evt.event_type || "scene"}
                </span>
                {isCrossStory && (
                  <span style={{
                    fontSize: 9, color: "var(--text-dim)", fontStyle: "italic",
                  }}>
                    shared
                  </span>
                )}
              </div>
              <div style={{ fontSize: 12, fontWeight: 600, color: "var(--text)" }}>
                {evt.title}
              </div>
              {evt.description && (
                <div style={{ fontSize: 11, color: "var(--text-dim)", marginTop: 2, lineHeight: 1.5 }}>
                  {evt.description}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
