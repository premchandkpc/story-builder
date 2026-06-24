import { useState } from "react"
import type { NarrativeEvent } from "../api/types"
import { cardStyle } from "../api/types"

interface EventListProps {
  events: NarrativeEvent[]
  loading?: boolean
}

export default function EventList({ events, loading }: EventListProps) {
  const [selected, setSelected] = useState<NarrativeEvent | null>(null)

  if (loading) {
    return <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 11, padding: 8 }}>Loading events...</div>
  }

  if (events.length === 0) {
    return <div style={{ color: "var(--text-dim)", fontStyle: "italic", fontSize: 11, padding: 8 }}>No events for this run.</div>
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {events.map((e) => (
        <div
          key={e.id}
          onClick={() => setSelected(selected?.id === e.id ? null : e)}
          style={{
            ...cardStyle, padding: "8px 10px", fontSize: 11, cursor: "pointer",
            borderLeft: `3px solid ${e.confidence > 0.8 ? "#00b894" : "#fdcb6e"}`,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <span style={{ fontWeight: 600 }}>{e.event_type}</span>
            <span style={{ color: "var(--text-faint)", fontSize: 10 }}>
              {e.subject_type}:{e.subject_id?.slice(0, 8)}
            </span>
          </div>
          <div style={{ display: "flex", gap: 8, color: "var(--text-faint)", fontSize: 10, marginTop: 2 }}>
            <span>v{e.version}</span>
            <span>{(e.confidence * 100).toFixed(0)}%</span>
            {e.source_agent && <span>{e.source_agent}</span>}
          </div>
          {selected?.id === e.id && e.payload && Object.keys(e.payload).length > 0 && (
            <pre style={{ fontSize: 10, color: "var(--text-dim)", marginTop: 4, maxHeight: 120, overflowY: "auto" }}>
              {JSON.stringify(e.payload, null, 2)}
            </pre>
          )}
        </div>
      ))}
    </div>
  )
}
