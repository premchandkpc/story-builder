import { useState } from "react"
import type { PromptSnapshot } from "../api/types"
import { cardStyle } from "../api/types"

interface PromptSectionViewerProps {
  snapshot: PromptSnapshot | null
}

export default function PromptSectionViewer({ snapshot }: PromptSectionViewerProps) {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set())

  const toggle = (name: string) => {
    const next = new Set(expandedSections)
    if (next.has(name)) next.delete(name)
    else next.add(name)
    setExpandedSections(next)
  }

  if (!snapshot) {
    return <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 11, padding: 8 }}>No prompt data.</div>
  }

  const allSections = [
    { name: "System", tokens: snapshot.tokenCount, snippet: snapshot.system },
    ...(snapshot.sections || []),
  ]

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {allSections.map((s) => {
        const open = expandedSections.has(s.name)
        return (
          <div key={s.name} style={{ ...cardStyle, padding: "8px 10px", fontSize: 11 }}>
            <div
              onClick={() => toggle(s.name)}
              style={{ cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "space-between" }}
            >
              <span style={{ fontWeight: 600 }}>{open ? "▾" : "▸"} {s.name}</span>
              <span style={{ color: "var(--text-faint)", fontSize: 10 }}>{s.tokens.toLocaleString()} tokens</span>
            </div>
            {open && (
              <div style={{ marginTop: 6, maxHeight: 200, overflowY: "auto" }}>
                <pre style={{ fontSize: 10, color: "var(--text-dim)", whiteSpace: "pre-wrap", margin: 0 }}>
                  {s.snippet || "No content"}
                </pre>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
