import { useState } from "react"
import type { Generation } from "../api/types"
import CompressionStats from "./CompressionStats"

interface GenerationCompareProps {
  generations: Generation[]
}

export default function GenerationCompare({ generations }: GenerationCompareProps) {
  const [leftIdx, setLeftIdx] = useState(0)
  const [rightIdx, setRightIdx] = useState(
    generations.length > 1 ? 1 : 0
  )

  const nonEmpty = generations.filter(g => g.output)
  if (nonEmpty.length < 2) {
    return (
      <div style={{ color: "var(--text-muted)", fontSize: 12, fontStyle: "italic", padding: 8 }}>
        Need at least 2 generations with output to compare.
      </div>
    )
  }

  const left = nonEmpty[leftIdx] || nonEmpty[0]
  const right = nonEmpty[rightIdx] || nonEmpty[nonEmpty.length > 1 ? 1 : 0]

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <div style={{ flex: 1 }}>
          <select
            value={leftIdx}
            onChange={(e) => setLeftIdx(Number(e.target.value))}
            style={{
              width: "100%", padding: "4px 8px", fontSize: 11,
              background: "var(--bg)", border: "1px solid var(--border)",
              borderRadius: 4, color: "var(--text)",
            }}
          >
            {nonEmpty.map((g, i) => (
              <option key={g.id} value={i}>
                {g.model || "?"} — {new Date(g.created_at).toLocaleString()}
                {g.accepted ? " (accepted)" : ""}
              </option>
            ))}
          </select>
        </div>
        <span style={{ color: "var(--text-muted)", fontSize: 11 }}>vs</span>
        <div style={{ flex: 1 }}>
          <select
            value={rightIdx}
            onChange={(e) => setRightIdx(Number(e.target.value))}
            style={{
              width: "100%", padding: "4px 8px", fontSize: 11,
              background: "var(--bg)", border: "1px solid var(--border)",
              borderRadius: 4, color: "var(--text)",
            }}
          >
            {nonEmpty.map((g, i) => (
              <option key={g.id} value={i}>
                {g.model || "?"} — {new Date(g.created_at).toLocaleString()}
                {g.accepted ? " (accepted)" : ""}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div style={{ display: "flex", gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontSize: 10, color: left.accepted ? "var(--success)" : "var(--text-muted)",
            fontWeight: 600, marginBottom: 4, textTransform: "uppercase", letterSpacing: "0.04em",
          }}>
            {left.model || "?"}
          </div>
          <div style={{
            padding: 8, background: "var(--bg)", borderRadius: 4,
            fontSize: 11, lineHeight: 1.6,
            maxHeight: 400, overflowY: "auto",
            whiteSpace: "pre-wrap", color: "var(--text)",
            border: left.accepted ? "1px solid var(--accent)" : "1px solid var(--border)",
          }}>
            {left.output}
          </div>
          <CompressionStats
            system={left.prompt_snapshot}
            userMessage={left.prompt_snapshot}
            model={left.model || "claude-sonnet"}
          />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontSize: 10, color: right.accepted ? "var(--success)" : "var(--text-muted)",
            fontWeight: 600, marginBottom: 4, textTransform: "uppercase", letterSpacing: "0.04em",
          }}>
            {right.model || "?"}
          </div>
          <div style={{
            padding: 8, background: "var(--bg)", borderRadius: 4,
            fontSize: 11, lineHeight: 1.6,
            maxHeight: 400, overflowY: "auto",
            whiteSpace: "pre-wrap", color: "var(--text)",
            border: right.accepted ? "1px solid var(--accent)" : "1px solid var(--border)",
          }}>
            {right.output}
          </div>
          <CompressionStats
            system={right.prompt_snapshot}
            userMessage={right.prompt_snapshot}
            model={right.model || "claude-sonnet"}
          />
        </div>
      </div>
    </div>
  )
}
