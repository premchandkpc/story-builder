import { useState } from "react"
import { useStoryRuns, useRunDetails, useRunSteps } from "../api/hooks"
import { fadeInStyle, cardStyle, badgeStyle } from "../api/types"
import type { StoryRun, RunStep } from "../api/types"

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

const stepStatusColor: Record<string, string> = {
  pending: "var(--text-faint)",
  running: "#74b9ff",
  done: "#00b894",
  failed: "#d46767",
  skipped: "#b2bec3",
}

function RunStepItem({ step }: { step: RunStep }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <div style={{
      ...cardStyle, padding: "10px 12px", fontSize: 11,
      borderLeft: `3px solid ${stepStatusColor[step.status] || "var(--text-faint)"}`,
    }}>
      <div
        onClick={() => setExpanded(!expanded)}
        style={{ cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "space-between" }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ fontWeight: 600, textTransform: "capitalize" }}>{step.step_name.replace(/_/g, " ")}</span>
          <span style={badgeStyle("#1a1512", stepStatusColor[step.status] || "var(--text-faint)")}>
            {step.status}
          </span>
        </div>
        <div style={{ display: "flex", gap: 8, color: "var(--text-faint)", fontSize: 10 }}>
          {step.model && <span>{step.model}</span>}
          {step.tokens_in > 0 && <span>⬆{step.tokens_in}</span>}
          {step.tokens_out > 0 && <span>⬇{step.tokens_out}</span>}
          <span>{expanded ? "▾" : "▸"}</span>
        </div>
      </div>
      {expanded && (
        <div style={{ marginTop: 8, display: "flex", flexDirection: "column", gap: 4 }}>
          {step.prompt_hash && (
            <div>
              <span style={{ color: "var(--text-faint)" }}>Prompt Hash: </span>
              <code style={{ fontSize: 10, color: "var(--text-dim)", wordBreak: "break-all" }}>{step.prompt_hash}</code>
            </div>
          )}
          {step.error && (
            <div style={{ color: "var(--error)", fontSize: 10, background: "rgba(212,103,103,0.08)", padding: "4px 8px", borderRadius: 3 }}>
              {step.error}
            </div>
          )}
          {step.artifacts && Object.keys(step.artifacts).length > 0 && (
            <details>
              <summary style={{ cursor: "pointer", color: "var(--text-dim)", fontSize: 10 }}>Artifacts</summary>
              <pre style={{ fontSize: 10, color: "var(--text-faint)", maxHeight: 200, overflowY: "auto", marginTop: 4 }}>
                {JSON.stringify(step.artifacts, null, 2)}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  )
}

function RunCard({ run }: { run: StoryRun }) {
  const [expanded, setExpanded] = useState(false)
  const { data: steps } = useRunSteps(expanded ? run.id : null)

  const phaseColors: Record<string, string> = {
    director: "#74b9ff",
    narrator: "#a29bfe",
    character: "#fdcb6e",
    editor: "#00b894",
    canon_guard: "#d46767",
    state_extract: "#e17055",
    generate: "#74b9ff",
    extract: "#e17055",
    memory: "#00b894",
    timeline: "#fdcb6e",
    summary: "#a29bfe",
    validate: "#d46767",
  }

  const stepName = run.current_step || ""
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
          {new Date(run.created_at).toLocaleString()}
        </div>
      </div>

      <div style={{ display: "flex", gap: 12, color: "var(--text-dim)", fontSize: 10, flexWrap: "wrap" }}>
        {run.current_step && (
          <span>
            Step: <span style={{ color: phaseColor, fontWeight: 600 }}>{run.current_step}</span>
          </span>
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

      <button
        onClick={() => setExpanded(!expanded)}
        style={{
          marginTop: 8, background: "none", border: "none", color: "var(--accent)",
          cursor: "pointer", fontSize: 10, fontWeight: 600, padding: 0,
          fontFamily: "var(--font-body)",
        }}
      >
        {expanded ? "▾ Hide Steps" : "▸ Show Steps"}
      </button>

      {expanded && (
        <div style={{ marginTop: 8, display: "flex", flexDirection: "column", gap: 4 }}>
          {!steps ? (
            <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 10 }}>Loading steps...</div>
          ) : steps.length === 0 ? (
            <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 10 }}>No steps recorded.</div>
          ) : (
            steps.map((s) => <RunStepItem key={s.id} step={s} />)
          )}
        </div>
      )}
    </div>
  )
}

export default function RunInspector({ storyId, nodeId }: RunInspectorProps) {
  const { data: runs, isLoading } = useStoryRuns(storyId, 50)
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
      {filtered.map((run) => (
        <RunCard key={run.id} run={run} />
      ))}
    </div>
  )
}
