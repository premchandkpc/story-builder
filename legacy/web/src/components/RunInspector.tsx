import { useState } from "react"
import { useStoryRuns, useRunSteps, useRunPromptSections, useRunEvents, useRunCost, useRunStats } from "../api/hooks"
import { fadeInStyle, cardStyle, badgeStyle } from "../api/types"
import type { StoryRun } from "../api/types"
import RunTimeline from "./RunTimeline"
import PromptSectionViewer from "./PromptSectionViewer"
import EventList from "./EventList"
import CostCard from "./CostCard"

interface RunInspectorProps {
  storyId: string
  nodeId?: string | null
}

const statusColor: Record<string, string> = {
  queued: "var(--text-faint)",
  running: "#74b9ff",
  partial: "#fdcb6e",
  completed: "#00b894",
  failed: "#d46767",
  cancelled: "#b2bec3",
}

function TabBar({ tabs, active, onChange }: { tabs: string[]; active: string; onChange: (t: string) => void }) {
  return (
    <div style={{ display: "flex", gap: 0, borderBottom: "1px solid var(--border)", marginBottom: 8 }}>
      {tabs.map((t) => (
        <button
          key={t}
          onClick={() => onChange(t)}
          style={{
            padding: "6px 12px", fontSize: 11, fontWeight: active === t ? 600 : 400,
            border: "none", borderBottom: active === t ? "2px solid var(--accent)" : "2px solid transparent",
            background: "none", color: active === t ? "var(--text)" : "var(--text-dim)",
            cursor: "pointer", fontFamily: "var(--font-body)",
          }}
        >
          {t}
        </button>
      ))}
    </div>
  )
}

function RunDetail({ run }: { run: StoryRun }) {
  const [tab, setTab] = useState("Overview")
  const { data: steps } = useRunSteps(run.id)
  const { data: promptSnapshot } = useRunPromptSections(tab === "Prompt" ? run.id : null)
  const { data: events, isLoading: eventsLoading } = useRunEvents(tab === "Events" ? run.id : null)
  const { data: cost } = useRunCost(tab === "Cost" ? run.id : null)

  const tabs = ["Overview", "Prompt", "Timeline", "Events", "Cost"]

  return (
    <div style={{ ...cardStyle, padding: "12px 14px", fontSize: 11, ...fadeInStyle, marginTop: 8 }}>
      <TabBar tabs={tabs} active={tab} onChange={setTab} />

      {tab === "Overview" && (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <span style={{ color: "var(--text-dim)" }}>Run Type</span>
            <span style={{ fontWeight: 600, textTransform: "capitalize" }}>{run.run_type.replace(/_/g, " ")}</span>
          </div>
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <span style={{ color: "var(--text-dim)" }}>Status</span>
            <span style={badgeStyle("#1a1512", statusColor[run.status] || "var(--text-faint)")}>{run.status}</span>
          </div>
          {run.started_at && (
            <div style={{ display: "flex", justifyContent: "space-between" }}>
              <span style={{ color: "var(--text-dim)" }}>Started</span>
              <span>{new Date(run.started_at).toLocaleString()}</span>
            </div>
          )}
          {run.finished_at && (
            <div style={{ display: "flex", justifyContent: "space-between" }}>
              <span style={{ color: "var(--text-dim)" }}>Finished</span>
              <span>{new Date(run.finished_at).toLocaleString()}</span>
            </div>
          )}
          {run.input_context_hash && (
            <div style={{ display: "flex", justifyContent: "space-between" }}>
              <span style={{ color: "var(--text-dim)" }}>Context Hash</span>
              <code style={{ fontSize: 9 }}>{run.input_context_hash.slice(0, 16)}...</code>
            </div>
          )}
          {run.error_summary && (
            <div style={{ color: "var(--error)", background: "rgba(212,103,103,0.08)", padding: "6px 8px", borderRadius: 3, marginTop: 4 }}>
              {run.error_summary}
            </div>
          )}
        </div>
      )}

      {tab === "Prompt" && <PromptSectionViewer snapshot={promptSnapshot || null} />}

      {tab === "Timeline" && (
        !steps ? (
          <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 11 }}>Loading steps...</div>
        ) : steps.length === 0 ? (
          <div style={{ color: "var(--text-dim)", fontStyle: "italic", fontSize: 11 }}>No steps recorded.</div>
        ) : (
          <RunTimeline steps={steps} />
        )
      )}

      {tab === "Events" && <EventList events={events || []} loading={eventsLoading} />}

      {tab === "Cost" && <CostCard cost={cost || null} />}
    </div>
  )
}

function RunCard({ run, isSelected, onSelect }: { run: StoryRun; isSelected: boolean; onSelect: () => void }) {
  const { data: steps } = useRunSteps(isSelected ? run.id : null)

  const stepName = run.current_step || ""
  const phaseColors: Record<string, string> = {
    generate: "#74b9ff", extract: "#e17055", memory: "#00b894",
    timeline: "#fdcb6e", summary: "#a29bfe", validate: "#d46767",
  }
  const phaseColor = phaseColors[stepName] || "var(--text-faint)"

  return (
    <div style={{
      ...cardStyle, padding: "12px 14px", fontSize: 11,
      borderLeft: `3px solid ${statusColor[run.status] || "var(--text-faint)"}`,
      ...fadeInStyle,
    }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 6 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ fontWeight: 600, color: "var(--text)" }}>{run.run_type.replace(/_/g, " ")}</span>
          <span style={badgeStyle("#1a1512", statusColor[run.status] || "var(--text-faint)")}>{run.status}</span>
        </div>
        <div style={{ fontSize: 10, color: "var(--text-faint)" }}>
          {run.created_at ? new Date(run.created_at).toLocaleString() : ""}
        </div>
      </div>

      <div style={{ display: "flex", gap: 12, color: "var(--text-dim)", fontSize: 10, flexWrap: "wrap" }}>
        {run.current_step && (
          <span>Step: <span style={{ color: phaseColor, fontWeight: 600 }}>{run.current_step}</span></span>
        )}
        {run.started_at && <span>Started: {new Date(run.started_at).toLocaleTimeString()}</span>}
        {run.finished_at && <span>Finished: {new Date(run.finished_at).toLocaleTimeString()}</span>}
        {run.input_context_hash && (
          <span title={run.input_context_hash}>
            Context: <code style={{ fontSize: 9 }}>{run.input_context_hash.slice(0, 12)}...</code>
          </span>
        )}
      </div>

      {run.error_summary && (
        <div style={{ marginTop: 6, color: "var(--error)", fontSize: 10, background: "rgba(212,103,103,0.08)", padding: "4px 8px", borderRadius: 3 }}>
          {run.error_summary}
        </div>
      )}

      {steps && steps.length > 0 && (
        <div style={{ marginTop: 8 }}>
          <RunTimeline steps={steps} />
        </div>
      )}

      <button
        onClick={onSelect}
        style={{
          marginTop: 8, background: "none", border: "none", color: "var(--accent)",
          cursor: "pointer", fontSize: 10, fontWeight: 600, padding: 0,
          fontFamily: "var(--font-body)",
        }}
      >
        {isSelected ? "▾ Hide Details" : "▸ Show Details"}
      </button>

      {isSelected && <RunDetail run={run} />}
    </div>
  )
}

export default function RunInspector({ storyId, nodeId }: RunInspectorProps) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)
  const { data: runs, isLoading } = useStoryRuns(storyId, 50)
  const { data: stats } = useRunStats(storyId)
  const filtered = nodeId ? (runs || []).filter((r) => r.scene_id === nodeId) : (runs || [])

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>Loading runs...</div>
  }

  if (filtered.length === 0) {
    return (
      <div style={{ color: "var(--text-dim)", fontSize: 11, fontStyle: "italic", padding: 8 }}>
        {nodeId ? "No runs for this node yet." : "No runs yet for this story."}
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      {stats && (
        <div style={{ ...cardStyle, padding: "8px 12px", fontSize: 10, display: "flex", gap: 12, color: "var(--text-dim)" }}>
          <span>Total: <strong>{stats.total}</strong></span>
          <span style={{ color: "#00b894" }}>✓ {stats.completed}</span>
          <span style={{ color: "#d46767" }}>✗ {stats.failed}</span>
          <span style={{ color: "#b2bec3" }}>⊘ {stats.cancelled}</span>
          <span>{(stats.failureRate * 100).toFixed(1)}% failure</span>
        </div>
      )}
      {filtered.map((run) => (
        <RunCard
          key={run.id}
          run={run}
          isSelected={selectedRunId === run.id}
          onSelect={() => setSelectedRunId(selectedRunId === run.id ? null : run.id)}
        />
      ))}
    </div>
  )
}
