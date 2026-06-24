import { useState, useMemo } from "react"
import { useTimeline, useCrossStoryTimeline } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface TimelineViewProps {
  storyId: string
}

type FilterMode = "all" | "local" | "cross"

const eventTypeColor: Record<string, string> = {
  scene: "var(--accent)",
  choice: "#8b5cf6",
  branch: "#f59e0b",
  converge: "#22c55e",
  climax: "#ef4444",
}

const PAGE_SIZE = 20

export default function TimelineView({ storyId }: TimelineViewProps) {
  const { data: localEvents, isLoading: localLoading } = useTimeline(storyId)
  const { data: crossEvents, isLoading: crossLoading } = useCrossStoryTimeline(storyId)
  const [filter, setFilter] = useState<FilterMode>("all")
  const [page, setPage] = useState(1)

  const local = useMemo(() => localEvents || [], [localEvents])
  const cross = useMemo(() => crossEvents || [], [crossEvents])
  const crossIds = useMemo(() => new Set(cross.map((c) => c.id)), [cross])

  const filtered = useMemo(() => {
    const events = filter === "cross" ? cross :
                 filter === "local" ? local :
                 [...local, ...cross]
    return events.sort((a, b) => a.order - b.order || (a.created_at || "").localeCompare(b.created_at || ""))
  }, [local, cross, filter])

  if (localLoading || crossLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 32 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  const totalCount = filtered.length
  const hasMore = totalCount > page * PAGE_SIZE
  const visible = filtered.slice(0, page * PAGE_SIZE)

  if (local.length === 0 && cross.length === 0) {
    return (
      <div style={{
        display: "flex", flexDirection: "column", alignItems: "center",
        gap: 8, padding: "20px 16px", color: "var(--text-faint)", fontSize: 12,
        textAlign: "center",
      }}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"
          style={{ opacity: 0.25 }}>
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
        <span style={{ fontStyle: "italic" }}>
          No timeline events yet. Generate scenes to populate the timeline.
        </span>
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10, ...fadeInStyle }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h3 style={{
          margin: 0, fontSize: 14,
          fontFamily: "var(--font-heading)", color: "var(--accent)",
          fontWeight: 600, letterSpacing: "0.02em",
        }}>
          Timeline
        </h3>
        <span style={{
          fontSize: 10, color: "var(--text-dim)", fontFamily: "var(--font-mono)",
          background: "var(--surface)", padding: "2px 6px",
          borderRadius: "var(--radius-sm)",
        }}>
          {local.length} + {cross.length}
        </span>
      </div>

      {/* Filter tabs */}
      <div style={{ display: "flex", gap: 4 }}>
        {(["all", "local", "cross"] as FilterMode[]).map((f) => (
          <button
            key={f}
            onClick={() => { setFilter(f); setPage(1) }}
            style={{
              flex: 1, padding: "4px 0", fontSize: 10,
              fontWeight: filter === f ? 600 : 400,
              background: filter === f ? "var(--accent)" : "var(--surface)",
              color: filter === f ? "#1a1512" : "var(--text-dim)",
              border: "none", borderRadius: "var(--radius-sm)",
              cursor: "pointer", textTransform: "uppercase",
              letterSpacing: "0.04em",
              transition: "background var(--transition-fast), color var(--transition-fast)",
            }}
            onMouseEnter={(e) => {
              if (filter !== f) e.currentTarget.style.background = "var(--surface-hover)"
            }}
            onMouseLeave={(e) => {
              if (filter !== f) e.currentTarget.style.background = "var(--surface)"
            }}
          >
            {f === "all" ? `${local.length + cross.length} All` :
             f === "local" ? `${local.length} Local` :
             `${cross.length} Global`}
          </button>
        ))}
      </div>

      {/* Timeline list */}
      <div style={{ position: "relative", paddingLeft: 20 }}>
        <div style={{
          position: "absolute", left: 8, top: 0, bottom: 0,
          width: 1, background: "var(--border)",
        }} />
        {visible.map((evt, i) => {
          const isCrossStory = crossIds.has(evt.id)
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
                  background: `${eventTypeColor[evt.event_type || "scene"] || "var(--accent)"}20`,
                  color: eventTypeColor[evt.event_type || "scene"] || "var(--accent)",
                  fontWeight: 600, textTransform: "uppercase",
                  letterSpacing: "0.04em",
                }}>
                  {evt.event_type || "scene"}
                </span>
                {isCrossStory ? (
                  <span style={{
                    fontSize: 9, color: "var(--info)", fontStyle: "italic",
                    marginLeft: "auto",
                  }}>
                    from {evt.story_id.slice(-6)}
                  </span>
                ) : (
                  evt.related_story_ids && evt.related_story_ids.length > 0 && (
                    <span style={{
                      fontSize: 9, color: "var(--text-dim)", fontStyle: "italic",
                      marginLeft: "auto",
                    }}>
                      +{evt.related_story_ids.length} shared
                    </span>
                  )
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

      {hasMore && (
        <button
          onClick={() => setPage((p) => p + 1)}
          style={{
            padding: "6px 12px", fontSize: 10,
            background: "var(--surface)", color: "var(--text-dim)",
            border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
            cursor: "pointer", width: "100%",
            transition: "background var(--transition-fast)",
          }}
          onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-hover)"}
          onMouseLeave={(e) => e.currentTarget.style.background = "var(--surface)"}
        >
          Show {Math.min(PAGE_SIZE, totalCount - page * PAGE_SIZE)} more ({totalCount - page * PAGE_SIZE} remaining)
        </button>
      )}
    </div>
  )
}
