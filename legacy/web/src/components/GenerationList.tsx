import { memo, useCallback, useState } from "react"
import type { Generation } from "../api/types"
import { spinnerStyle, badgeStyle } from "../api/types"
import GenerationCompare from "./GenerationCompare"

interface GenerationListProps {
  generations: Generation[]
  gensLoading: boolean
  selectedNodeId: string
  storyId?: string
  onAccept: (nodeId: string, genId: string) => void
  hasPending?: boolean
  onRetry?: () => void
  isError?: boolean
}

const statusInfo: Record<string, { label: string; color: string; bg: string }> = {
  pending:          { label: "Pending",    color: "var(--warn)",   bg: "rgba(201,115,74,0.12)" },
  running:          { label: "Running...", color: "var(--warn)",   bg: "rgba(201,115,74,0.12)" },
  queued:           { label: "Queued",     color: "var(--info)",  bg: "rgba(107,159,196,0.12)" },
  success:          { label: "Success",    color: "var(--success)",bg: "rgba(107,143,94,0.12)" },
  partial_success:  { label: "Partial",    color: "var(--warn)",   bg: "rgba(201,115,74,0.12)" },
  failed:           { label: "Failed",     color: "var(--error)",  bg: "rgba(176,92,80,0.12)" },
}

interface GenerationCardProps {
  g: Generation
  isExpanded: boolean
  selectedNodeId: string
  accepting: string | null
  onAccept: (nodeId: string, genId: string) => void
  onToggle: (id: string) => void
}

const GenerationCard = memo(function GenerationCard({
  g, isExpanded, selectedNodeId, accepting, onAccept, onToggle,
}: GenerationCardProps) {
  const isAccepted = g.accepted
  const isPending = g.status === "pending" || g.status === "running" || g.status === "queued"
  const isFailed = g.status === "failed"
  const si = statusInfo[g.status] || { label: g.status || "Unknown", color: "var(--text-dim)", bg: "transparent" }

  const handleAccept = async () => {
    await onAccept(selectedNodeId, g.id)
  }

  return (
    <div className={isAccepted ? undefined : "card-hover"} style={{
      border: `1px solid ${
        isPending ? "var(--warn)" :
        isAccepted ? "rgba(212,168,83,0.3)" :
        isFailed ? "var(--error)" :
        "var(--border)"
      }`,
      borderRadius: "var(--radius-md)", padding: 10,
      background: isAccepted ? "rgba(212,168,83,0.04)" :
                  isPending ? "rgba(201,115,74,0.04)" :
                  isFailed ? "rgba(176,92,80,0.04)" :
                  "var(--surface)",
      transition: "border-color var(--transition-base), box-shadow var(--transition-base)",
      animation: isPending ? "glowPulse 2s ease-in-out infinite" : undefined,
      position: "relative",
      opacity: isPending ? 0.9 : 1,
    }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 5 }}>
        <span style={{
          fontWeight: 600, fontSize: 11,
          color: isAccepted ? "var(--accent)" : isPending ? "var(--warn)" : "var(--text)",
          fontFamily: "var(--font-mono)",
        }}>
          {g.model || "unknown"}
          {isPending && (
            <span style={{ display: "inline-block", marginLeft: 4 }}>
              <div style={{ ...spinnerStyle, width: 10, height: 10, borderWidth: 1.5, verticalAlign: "middle" }} />
            </span>
          )}
        </span>
        <span style={{ fontSize: 9, color: "var(--text-faint)" }}>
           {g.created_at ? new Date(g.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }) : ""}
        </span>
      </div>

      <div style={{ display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
        <span style={{
          ...badgeStyle(si.color, si.bg),
          fontSize: 9,
          animation: isPending ? "pulse 1.5s ease-in-out infinite" : undefined,
        }}>
          {isAccepted ? "Accepted" : si.label}
        </span>

        {g.total_tokens !== undefined && g.total_tokens > 0 && (
          <span style={{ fontSize: 9, color: "var(--text-faint)", fontFamily: "var(--font-mono)" }}>
            {g.total_tokens}t · {g.duration_ms ? `${(g.duration_ms / 1000).toFixed(1)}s` : ""}
          </span>
        )}

        {g.output && (
          <button
            onClick={() => onToggle(g.id)}
            style={{
              background: "none", border: "none",
              color: isExpanded ? "var(--text-dim)" : "var(--accent)",
              cursor: "pointer", fontSize: 10, padding: 0,
              transition: "color 0.15s",
            }}
          >
            {isExpanded ? "Collapse" : "Preview"}
          </button>
        )}
      </div>

      {isFailed && g.error && (
        <div style={{
          marginTop: 6, padding: "6px 8px",
          background: "rgba(176,92,80,0.08)",
          borderRadius: "var(--radius-sm)",
          fontSize: 10, color: "var(--error)",
          lineHeight: 1.4, fontFamily: "var(--font-mono)",
          wordBreak: "break-word",
        }}>
          {g.error}
        </div>
      )}

      {isExpanded && g.output && (
        <div style={{
          marginTop: 8, padding: 10,
          background: "var(--bg)", borderRadius: "var(--radius-sm)",
          fontSize: 11, lineHeight: 1.7,
          maxHeight: 250, overflowY: "auto",
          whiteSpace: "pre-wrap", color: "var(--text-muted)",
          border: "1px solid var(--border)",
          boxShadow: "inset 0 1px 3px rgba(0,0,0,0.15)",
        }}>
          {g.output}
        </div>
      )}

      {isAccepted && (
        <div style={{
          marginTop: 6, fontSize: 9, color: "var(--success)",
          fontFamily: "var(--font-mono)", fontWeight: 600,
          display: "flex", alignItems: "center", gap: 4,
        }}>
          <span style={{ width: 4, height: 4, borderRadius: "50%", background: "var(--success)" }} />
          Active generation
        </div>
      )}

      {!isAccepted && g.output && !isPending && (
        <button
          onClick={handleAccept}
          disabled={accepting === g.id}
          style={{
            marginTop: 8, padding: "5px 10px",
            background: accepting === g.id ? "var(--text-dim)" : "var(--accent)",
            color: "#1a1512",
            border: "none", borderRadius: "var(--radius-sm)",
            cursor: accepting === g.id ? "default" : "pointer",
            fontWeight: 600, fontSize: 10,
            letterSpacing: "0.03em",
            width: "100%",
            transition: "background 0.15s, box-shadow 0.15s",
            opacity: accepting === g.id ? 0.6 : 1,
          }}
          onMouseEnter={(e) => {
            if (accepting !== g.id) {
              e.currentTarget.style.background = "var(--accent-hover)"
            }
          }}
          onMouseLeave={(e) => {
            if (accepting !== g.id) {
              e.currentTarget.style.background = "var(--accent)"
            }
          }}
        >
          {accepting === g.id ? "Accepting..." : "Accept"}
        </button>
      )}

      {!isAccepted && isPending && (
        <div style={{
          marginTop: 8,
          width: "100%", height: 3,
          background: "rgba(201,115,74,0.12)",
          borderRadius: 2,
          overflow: "hidden",
        }}>
          <div style={{
            width: "40%",
            height: "100%",
            background: "linear-gradient(90deg, var(--warn), transparent)",
            borderRadius: 2,
            animation: "shimmer 1.5s ease-in-out infinite",
          }} />
        </div>
      )}
    </div>
  )
})

export default function GenerationList({
  generations, gensLoading, selectedNodeId, storyId, onAccept,
  hasPending, onRetry, isError,
}: GenerationListProps) {
  const [expandedGen, setExpandedGen] = useState<string | null>(null)
  const [showCompare, setShowCompare] = useState(false)

  const [accepting, setAccepting] = useState<string | null>(null)

  const toggleExpand = useCallback((id: string) => setExpandedGen(id), [])

  const hasOutput = generations.some((g) => g.output)
  const hasMultipleOutputs = generations.filter((g) => g.output).length >= 2

  const handleAccept = useCallback(async (nodeId: string, genId: string) => {
    setAccepting(genId)
    try {
      await onAccept(nodeId, genId)
    } finally {
      setAccepting(null)
    }
  }, [onAccept])

  // Error state
  if (isError && generations.length === 0) {
    return (
      <div style={{
        display: "flex", flexDirection: "column", gap: 8, fontSize: 12,
      }}>
        <div style={{
          display: "flex", flexDirection: "column", alignItems: "center",
          gap: 10, padding: 24, color: "var(--text-dim)", textAlign: "center",
        }}>
          <span role="alert" style={{ fontSize: 12, color: "var(--error)" }}>
            Failed to load generations
          </span>
          {onRetry && (
            <button
              onClick={onRetry}
              style={{
                padding: "6px 14px",
                background: "var(--surface)",
                border: "1px solid var(--border)",
                borderRadius: "var(--radius-sm)",
                color: "var(--text)",
                cursor: "pointer", fontSize: 11,
                transition: "background var(--transition-fast)",
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-hover)"}
              onMouseLeave={(e) => e.currentTarget.style.background = "var(--surface)"}
            >
              Retry
            </button>
          )}
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
        <span style={{
          color: "var(--text-faint)", fontSize: 10, letterSpacing: "0.04em",
          textTransform: "uppercase", fontWeight: 500,
          display: "flex", alignItems: "center", gap: 6,
        }}>
          Generations
          {gensLoading && <div style={{ ...spinnerStyle, width: 12, height: 12 }} />}
          {hasPending && (
            <span style={{
              fontSize: 9, color: "var(--warn)", fontWeight: 600,
              animation: "pulse 1.5s ease-in-out infinite",
            }}>
              ● Generating
            </span>
          )}
        </span>
        {hasMultipleOutputs && (
          <button
            onClick={() => setShowCompare(!showCompare)}
            style={{
              background: showCompare ? "var(--accent)" : "transparent",
              border: `1px solid ${showCompare ? "var(--accent)" : "var(--border)"}`,
              borderRadius: "var(--radius-sm)", padding: "3px 10px",
              color: showCompare ? "#1a1512" : "var(--text-dim)",
              cursor: "pointer", fontSize: 10, fontWeight: 600,
              letterSpacing: "0.03em",
              transition: "all 0.15s",
            }}
          >
            {showCompare ? "List" : "Compare"}
          </button>
        )}
      </div>

      {showCompare ? (
        <GenerationCompare generations={generations} storyId={storyId} nodeId={selectedNodeId} />
      ) : (
        <>
          {/* Empty state */}
          {!gensLoading && generations.length === 0 && !hasPending && (
            <div className="stagger-fade-in" style={{
              display: "flex", flexDirection: "column", alignItems: "center",
              gap: 10, padding: 24, color: "var(--text-dim)",
              textAlign: "center",
            }}>
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                strokeWidth="1.3" strokeLinecap="round" style={{ opacity: 0.25 }}>
                <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
              </svg>
              <span style={{ fontSize: 12, color: "var(--text-faint)" }}>
                No generations yet
              </span>
              <span style={{ fontSize: 10, color: "var(--text-faint)", opacity: 0.7 }}>
                Configure the scene and click Generate to create prose
              </span>
            </div>
          )}

          {/* Pending indicator (when we have no output yet but something is being generated) */}
          {hasPending && !hasOutput && (
            <div style={{
              display: "flex", flexDirection: "column", gap: 10, padding: 12,
              border: "1px solid var(--warn)",
              borderRadius: "var(--radius-md)",
              background: "rgba(201,115,74,0.04)",
              animation: "glowPulse 2s ease-in-out infinite",
              alignItems: "center",
            }}>
              <div style={{ ...spinnerStyle, width: 20, height: 20, borderWidth: 2 }} />
              <div style={{ fontSize: 11, color: "var(--warn)", fontWeight: 500 }}>
                Generating scene...
              </div>
              <div style={{
                width: "100%", height: 3,
                background: "rgba(201,115,74,0.12)",
                borderRadius: 2,
                overflow: "hidden",
              }}>
                <div style={{
                  width: "30%",
                  height: "100%",
                  background: "var(--warn)",
                  borderRadius: 2,
                  animation: "shimmer 1.5s ease-in-out infinite",
                  boxShadow: "0 0 6px rgba(201,115,74,0.3)",
                }} />
              </div>
            </div>
          )}

          {/* Loading skeleton */}
          {gensLoading && generations.length === 0 && !hasPending && (
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              {[1, 2].map((i) => (
                <div key={i} style={{
                  padding: 12,
                  border: "1px solid var(--border)",
                  borderRadius: "var(--radius-md)",
                  display: "flex", flexDirection: "column", gap: 6,
                }}>
                  <div style={{
                    width: "40%", height: 10,
                    borderRadius: "var(--radius-sm)",
                    background: "linear-gradient(90deg, var(--surface) 25%, var(--surface-hover) 50%, var(--surface) 75%)",
                    backgroundSize: "200% 100%",
                    animation: "shimmer 1.5s infinite",
                  }} />
                  <div style={{
                    width: "60%", height: 8,
                    borderRadius: "var(--radius-sm)",
                    background: "linear-gradient(90deg, var(--surface) 25%, var(--surface-hover) 50%, var(--surface) 75%)",
                    backgroundSize: "200% 100%",
                    animation: "shimmer 1.5s infinite 0.2s",
                  }} />
                </div>
              ))}
            </div>
          )}

          {/* Generation items */}
          {!gensLoading && generations.map((g) => (
            <GenerationCard
              key={g.id}
              g={g}
              isExpanded={expandedGen === g.id}
              selectedNodeId={selectedNodeId}
              accepting={accepting}
              onAccept={handleAccept}
              onToggle={toggleExpand}
            />
          ))}
        </>
      )}
    </div>
  )
}
