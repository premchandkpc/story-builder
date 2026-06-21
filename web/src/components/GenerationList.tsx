import { useState } from "react"
import type { Generation } from "../api/types"
import { spinnerStyle, badgeStyle } from "../api/types"
import GenerationCompare from "./GenerationCompare"

interface GenerationListProps {
  generations: Generation[]
  gensLoading: boolean
  selectedNodeId: string
  onAccept: (nodeId: string, genId: string) => void
}

export default function GenerationList({ generations, gensLoading, selectedNodeId, onAccept }: GenerationListProps) {
  const [expandedGen, setExpandedGen] = useState<string | null>(null)
  const [showCompare, setShowCompare] = useState(false)

  const statusBadge = (g: Generation) => {
    if (g.accepted) return badgeStyle("#1a1512", "var(--accent)")
    if (g.output) return badgeStyle("var(--warn)", "rgba(201,115,74,0.15)")
    return badgeStyle("var(--text-dim)", "rgba(140,126,112,0.15)")
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
        <span style={{ color: "var(--text-faint)", fontSize: 10, letterSpacing: "0.04em", textTransform: "uppercase", fontWeight: 500 }}>
          Generations
        </span>
        {generations.filter(g => g.output).length >= 2 && (
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
        <GenerationCompare generations={generations} />
      ) : (
        <>
          {gensLoading && (
            <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--text-dim)", padding: 8 }}>
              <div style={spinnerStyle} />
              <span style={{ fontSize: 12 }}>Loading generations...</span>
            </div>
          )}
          {!gensLoading && generations.length === 0 && (
            <div style={{
              color: "var(--text-dim)", fontStyle: "italic",
              padding: 12, textAlign: "center", fontSize: 11,
            }}>
              No generations yet.
            </div>
          )}
          {!gensLoading && generations.map((g) => {
            const isAccepted = g.accepted
            const isExpanded = expandedGen === g.id
            return (
              <div key={g.id} className={isAccepted ? undefined : "card-hover"} style={{
                border: `1px solid ${isAccepted ? "rgba(212,168,83,0.3)" : "var(--border)"}`,
                borderRadius: "var(--radius-md)", padding: 10,
                background: isAccepted ? "rgba(212,168,83,0.04)" : "var(--surface)",
                transition: "border-color var(--transition-base), box-shadow var(--transition-base)",
              }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 5 }}>
                  <span style={{ fontWeight: 600, fontSize: 11, color: isAccepted ? "var(--accent)" : "var(--text)", fontFamily: "var(--font-mono)" }}>
                    {g.model || "unknown"}
                  </span>
                  <span style={{ fontSize: 9, color: "var(--text-faint)" }}>
                    {new Date(g.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}
                  </span>
                </div>
                <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                  <span style={statusBadge(g)}>
                    {isAccepted ? "Accepted" : g.output ? "Generated" : "Pending"}
                  </span>
                  {g.output && (
                    <button
                      onClick={() => setExpandedGen(isExpanded ? null : g.id)}
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
                {g.output && !isAccepted && (
                  <button
                    onClick={() => onAccept(selectedNodeId, g.id)}
                    style={{
                      marginTop: 8, padding: "5px 10px",
                      background: "var(--accent)", color: "#1a1512",
                      border: "none", borderRadius: "var(--radius-sm)",
                      cursor: "pointer", fontWeight: 600, fontSize: 10,
                      letterSpacing: "0.03em",
                      width: "100%",
                      boxShadow: "0 1px 2px rgba(0,0,0,0.2)",
                      transition: "background 0.15s, box-shadow 0.15s",
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = "var(--accent-hover)"
                      e.currentTarget.style.boxShadow = "0 2px 6px rgba(212,168,83,0.3)"
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = "var(--accent)"
                      e.currentTarget.style.boxShadow = "0 1px 2px rgba(0,0,0,0.2)"
                    }}
                  >
                    Accept
                  </button>
                )}
              </div>
            )
          })}
        </>
      )}
    </div>
  )
}
