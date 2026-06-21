import { useState } from "react"
import type { Generation } from "../api/types"
import { spinnerStyle } from "../api/types"
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

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
        <div style={{ color: "var(--text-dim)", fontWeight: 500 }}>Generations</div>
        {generations.filter(g => g.output).length >= 2 && (
          <button
            onClick={() => setShowCompare(!showCompare)}
            style={{
              background: showCompare ? "var(--accent)" : "transparent",
              border: `1px solid ${showCompare ? "var(--accent)" : "var(--border)"}`,
              borderRadius: 4, padding: "3px 10px",
              color: showCompare ? "#1a1a24" : "var(--text-dim)",
              cursor: "pointer", fontSize: 10, fontWeight: 600,
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
              No generations yet. Click <strong style={{ color: "var(--accent)" }}>Generate</strong> to start.
            </div>
          )}
          {!gensLoading && generations.map((g) => {
            const isAccepted = g.accepted
            const isExpanded = expandedGen === g.id
            return (
              <div key={g.id} style={{
                border: `1px solid ${isAccepted ? "var(--accent)" : "var(--border)"}`,
                borderRadius: 6, padding: 10,
                background: isAccepted ? "rgba(212,168,83,0.06)" : "transparent",
                transition: "border-color 0.15s",
              }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
                  <span style={{ fontWeight: 600, fontSize: 11, color: isAccepted ? "var(--accent)" : "var(--text)" }}>
                    {g.model || "unknown"}
                  </span>
                  <span style={{ fontSize: 10, color: "var(--text-dim)" }}>
                    {new Date(g.created_at).toLocaleString()}
                  </span>
                </div>
                <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                  <span style={{
                    fontSize: 10, padding: "2px 7px", borderRadius: 3,
                    background: isAccepted ? "var(--accent)" : g.output ? "rgba(201,115,74,0.15)" : "rgba(140,126,112,0.15)",
                    color: isAccepted ? "#1a1a24" : g.output ? "#c9734a" : "var(--text-dim)",
                    fontWeight: 600,
                  }}>
                    {isAccepted ? "Accepted" : g.output ? "Generated" : "Pending"}
                  </span>
                  {g.output && (
                    <button
                      onClick={() => setExpandedGen(isExpanded ? null : g.id)}
                      style={{
                        background: "none", border: "none", color: "var(--accent)",
                        cursor: "pointer", fontSize: 10, padding: 0,
                      }}
                    >
                      {isExpanded ? "Collapse" : "Preview"}
                    </button>
                  )}
                </div>
                {isExpanded && g.output && (
                  <div style={{
                    marginTop: 8, padding: 10,
                    background: "var(--bg)", borderRadius: 4,
                    fontSize: 11, lineHeight: 1.6,
                    maxHeight: 250, overflowY: "auto",
                    whiteSpace: "pre-wrap", color: "var(--text)",
                    border: "1px solid var(--border)",
                  }}>
                    {g.output}
                  </div>
                )}
                {g.output && !isAccepted && (
                  <button
                    onClick={() => onAccept(selectedNodeId, g.id)}
                    style={{
                      marginTop: 8, padding: "5px 10px",
                      background: "var(--accent)", color: "#1a1a24",
                      border: "none", borderRadius: 4,
                      cursor: "pointer", fontWeight: 600, fontSize: 11,
                      width: "100%", transition: "background 0.15s",
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.background = "var(--accent-hover)"}
                    onMouseLeave={(e) => e.currentTarget.style.background = "var(--accent)"}
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
