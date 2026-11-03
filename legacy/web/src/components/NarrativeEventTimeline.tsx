import { useNarrativeEvents, useSceneNarrativeEvents } from "../api/hooks"
import { fadeInStyle, cardStyle, badgeStyle } from "../api/types"
import type { NarrativeEvent } from "../api/types"

interface NarrativeEventTimelineProps {
  storyId: string
  nodeId?: string | null
}

const eventTypeLabels: Record<string, string> = {
  "character.location.changed": "Location Change",
  "character.goal.updated": "Goal Updated",
  "character.emotion.changed": "Emotion Shift",
  "character.knowledge.added": "Knowledge Gained",
  "relationship.trust.changed": "Trust Shift",
  "relationship.status.changed": "Relationship Change",
  "timeline.event.recorded": "Timeline Event",
  "plot_thread.opened": "Plot Thread Opened",
  "plot_thread.advanced": "Plot Advanced",
  "plot_thread.resolved": "Plot Resolved",
  "canon.fact.asserted": "Canon Fact",
  "canon.fact.retracted": "Canon Retracted",
  "world.state.changed": "World Change",
}

const eventColors: Record<string, string> = {
  character: "#74b9ff",
  relationship: "#fdcb6e",
  timeline: "#a29bfe",
  plot_thread: "#e17055",
  canon: "#d46767",
  world: "#00b894",
}

function subjectColor(subjectType: string): string {
  return eventColors[subjectType] || "var(--text-faint)"
}

function EventCard({ event }: { event: NarrativeEvent }) {
  return (
    <div style={{
      ...cardStyle, padding: "10px 12px", fontSize: 11,
      borderLeft: `3px solid ${subjectColor(event.subject_type)}`,
    }}>
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 4 }}>
        <span style={badgeStyle("#1a1512", subjectColor(event.subject_type))}>
          {event.event_type.split(".").slice(0, 2).join(" ")}
        </span>
        <span style={{ fontWeight: 600, color: "var(--text)", fontSize: 11 }}>
          {eventTypeLabels[event.event_type] || event.event_type.replace(/_/g, " ")}
        </span>
      </div>

      <div style={{ display: "flex", gap: 8, color: "var(--text-dim)", fontSize: 10, flexWrap: "wrap" }}>
        <span>Subject: <strong>{event.subject_id}</strong></span>
        <span>Confidence: {Math.round(event.confidence * 100)}%</span>
        <span>v{event.version}</span>
      </div>

      {event.source_agent && (
        <div style={{ color: "var(--text-faint)", fontSize: 9, marginTop: 2 }}>
          Source: {event.source_agent}
        </div>
      )}

      {event.payload && Object.keys(event.payload).length > 0 && (
        <details style={{ marginTop: 4 }}>
          <summary style={{ cursor: "pointer", color: "var(--text-dim)", fontSize: 10 }}>Details</summary>
          <pre style={{ fontSize: 9, color: "var(--text-faint)", maxHeight: 150, overflowY: "auto", marginTop: 4 }}>
            {JSON.stringify(event.payload, null, 2)}
          </pre>
        </details>
      )}
    </div>
  )
}

export default function NarrativeEventTimeline({ storyId, nodeId }: NarrativeEventTimelineProps) {
  const listAll = useNarrativeEvents(storyId, 100)
  const listByScene = useSceneNarrativeEvents(storyId, nodeId || null, 100)
  const data = nodeId ? listByScene : listAll
  const isLoading = data.isLoading
  const events = data.data || []

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>Loading events...</div>
  }

  if (events.length === 0) {
    return (
      <div style={{ color: "var(--text-dim)", fontSize: 11, fontStyle: "italic", padding: 8 }}>
        No narrative events recorded yet.
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...fadeInStyle }}>
      <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 2 }}>
        {events.length} events · newest first
      </div>
      {events.map((ev) => (
        <EventCard key={ev.id} event={ev} />
      ))}
    </div>
  )
}
