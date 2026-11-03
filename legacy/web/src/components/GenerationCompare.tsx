import { useState } from "react"
import type { Generation, EventDiff } from "../api/types"
import { useGenDiff } from "../api/hooks"
import CompressionStats from "./CompressionStats"
import { cardStyle, labelStyle, skeletonStyle, badgeStyle } from "../api/types"

interface GenerationCompareProps {
  generations: Generation[]
  storyId?: string
  nodeId?: string
}

function EventDiffList({ diffs }: { diffs: EventDiff[] }) {
  if (diffs.length === 0) {
    return <div style={{ fontSize: 11, color: "var(--text-dim)", padding: "4px 0" }}>No event differences</div>
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {diffs.map((d, i) => (
        <div key={i} style={{
          fontSize: 11, padding: "6px 8px", borderRadius: "var(--radius-sm)",
          background: d.eventType === "added" ? "rgba(34,197,94,0.08)" : d.eventType === "removed" ? "rgba(239,68,68,0.08)" : "transparent",
          border: `1px solid ${d.eventType === "added" ? "rgba(34,197,94,0.2)" : d.eventType === "removed" ? "rgba(239,68,68,0.2)" : "var(--border)"}`,
        }}>
          <div style={{ display: "flex", gap: 6, alignItems: "center", marginBottom: 2 }}>
            <span style={badgeStyle(
              d.eventType === "added" ? "var(--success)" : d.eventType === "removed" ? "var(--error)" : "var(--text-dim)",
              d.eventType === "added" ? "rgba(34,197,94,0.12)" : d.eventType === "removed" ? "rgba(239,68,68,0.12)" : "var(--surface)",
            )}>
              {d.eventType}
            </span>
            {(d.a?.subjectType || d.b?.subjectType) && (
              <span style={{ color: "var(--text-dim)" }}>{d.a?.subjectType || d.b?.subjectType}</span>
            )}
          </div>
          <div style={{ color: "var(--text-dim)", fontSize: 10 }}>
            {d.a && <div>A: {JSON.stringify(d.a.payload).slice(0, 100)}</div>}
            {d.b && <div>B: {JSON.stringify(d.b.payload).slice(0, 100)}</div>}
          </div>
        </div>
      ))}
    </div>
  )
}

export default function GenerationCompare({ generations, storyId, nodeId }: GenerationCompareProps) {
  const [leftIdx, setLeftIdx] = useState(0)
  const [rightIdx, setRightIdx] = useState(
    generations.length > 1 ? 1 : 0
  )
  const [showDiff, setShowDiff] = useState(false)

  const nonEmpty = generations.filter(g => g.output)
  const left = nonEmpty[leftIdx] || nonEmpty[0]
  const right = nonEmpty[rightIdx] || (nonEmpty.length > 1 ? nonEmpty[1] : nonEmpty[0])

  const { data: diffData, isLoading: diffLoading } = useGenDiff(
    storyId || "",
    nodeId || "",
    showDiff && nonEmpty.length >= 2 ? left?.id ?? null : null,
    showDiff && nonEmpty.length >= 2 ? right?.id ?? null : null,
  )

  if (nonEmpty.length < 2) {
    return (
      <div style={{ color: "var(--text-dim)", fontSize: 12, fontStyle: "italic", padding: 8 }}>
        Need at least 2 generations with output to compare.
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <div style={{ flex: 1 }}>
          <select
            value={leftIdx}
            onChange={(e) => { setLeftIdx(Number(e.target.value)); setShowDiff(false) }}
            style={{
              width: "100%", padding: "6px 8px", fontSize: 11,
              background: "var(--bg)", border: "1px solid var(--border)",
              borderRadius: "var(--radius-sm)", color: "var(--text)",
              fontFamily: "var(--font-body)",
              boxShadow: "var(--shadow-inner)",
            }}
          >
            {nonEmpty.map((g, i) => (
              <option key={g.id} value={i}>
                {g.model || "?"} — {g.created_at ? new Date(g.created_at).toLocaleString() : ""}
                {g.accepted ? " (accepted)" : ""}
              </option>
            ))}
          </select>
        </div>
        <span style={{ color: "var(--text-faint)", fontSize: 11, fontWeight: 600 }}>vs</span>
        <div style={{ flex: 1 }}>
          <select
            value={rightIdx}
            onChange={(e) => { setRightIdx(Number(e.target.value)); setShowDiff(false) }}
            style={{
              width: "100%", padding: "6px 8px", fontSize: 11,
              background: "var(--bg)", border: "1px solid var(--border)",
              borderRadius: "var(--radius-sm)", color: "var(--text)",
              fontFamily: "var(--font-body)",
              boxShadow: "var(--shadow-inner)",
            }}
          >
            {nonEmpty.map((g, i) => (
              <option key={g.id} value={i}>
                {g.model || "?"} — {g.created_at ? new Date(g.created_at).toLocaleString() : ""}
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
            fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.04em",
          }}>
            {left.model || "?"}
          </div>
          <div style={{
            padding: 10, background: "var(--bg)", borderRadius: "var(--radius-sm)",
            fontSize: 11, lineHeight: 1.6,
            maxHeight: 400, overflowY: "auto",
            whiteSpace: "pre-wrap", color: "var(--text)",
            border: left.accepted ? "1px solid var(--accent)" : "1px solid var(--border)",
            boxShadow: "var(--shadow-inner)",
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
            fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.04em",
          }}>
            {right.model || "?"}
          </div>
          <div style={{
            padding: 10, background: "var(--bg)", borderRadius: "var(--radius-sm)",
            fontSize: 11, lineHeight: 1.6,
            maxHeight: 400, overflowY: "auto",
            whiteSpace: "pre-wrap", color: "var(--text)",
            border: right.accepted ? "1px solid var(--accent)" : "1px solid var(--border)",
            boxShadow: "var(--shadow-inner)",
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

      {storyId && nodeId && (
        <div>
          <button
            onClick={() => setShowDiff(!showDiff)}
            className="btn-press"
            style={{
              width: "100%", padding: "6px 12px", fontSize: 11,
              background: showDiff ? "var(--accent-dim)" : "var(--surface)",
              border: `1px solid ${showDiff ? "var(--accent)" : "var(--border)"}`,
              borderRadius: "var(--radius-sm)", color: showDiff ? "var(--accent)" : "var(--text-dim)",
              cursor: "pointer", fontWeight: 600,
              textTransform: "uppercase", letterSpacing: "0.04em",
              transition: "all 0.15s",
            }}
          >
            {showDiff ? "Hide" : "Show"} Server-side Diff
          </button>

          {showDiff && (
            <div style={{ marginTop: 8, ...cardStyle } as React.CSSProperties}>
              <div style={{ padding: "10px 12px", borderBottom: "1px solid var(--border)" }}>
                <span style={labelStyle}>Prose Diff</span>
                {diffLoading ? (
                  <div style={skeletonStyle("100%", 40)} />
                ) : diffData?.proseDiff ? (
                  <pre style={{
                    fontSize: 10, lineHeight: 1.5, color: "var(--text)",
                    whiteSpace: "pre-wrap", margin: "4px 0 0",
                    fontFamily: "var(--font-mono)",
                    background: "var(--bg)", padding: 8, borderRadius: "var(--radius-sm)",
                    maxHeight: 150, overflowY: "auto",
                  }}>
                    {diffData.proseDiff}
                  </pre>
                ) : (
                  <div style={{ fontSize: 11, color: "var(--text-dim)", marginTop: 4 }}>
                    {diffLoading ? "Loading…" : "Identical prose"}
                  </div>
                )}
              </div>

              <div style={{ padding: "10px 12px", borderBottom: "1px solid var(--border)" }}>
                <span style={labelStyle}>Token Diff</span>
                {diffData && (
                  <div style={{ display: "flex", gap: 12, marginTop: 4, fontSize: 11 }}>
                    <span>A: <strong>{diffData.tokenDiff.a}</strong></span>
                    <span>B: <strong>{diffData.tokenDiff.b}</strong></span>
                    <span style={{ color: "var(--text-dim)" }}>
                      Δ {diffData.tokenDiff.b - diffData.tokenDiff.a >= 0 ? "+" : ""}{diffData.tokenDiff.b - diffData.tokenDiff.a}
                    </span>
                  </div>
                )}
              </div>

              <div style={{ padding: "10px 12px" }}>
                <span style={labelStyle}>Event Diffs ({diffData?.eventDiffs?.length || 0})</span>
                <div style={{ marginTop: 4 }}>
                  {diffLoading ? (
                    <div style={skeletonStyle("100%", 60)} />
                  ) : diffData?.eventDiffs ? (
                    <EventDiffList diffs={diffData.eventDiffs} />
                  ) : null}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
