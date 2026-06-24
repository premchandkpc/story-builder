import type { RunStep } from "../api/types"

const stepColor: Record<string, string> = {
  done: "#00b894",
  failed: "#d46767",
  running: "#74b9ff",
  pending: "var(--text-faint)",
  skipped: "#b2bec3",
}

interface RunTimelineProps {
  steps: RunStep[]
  onStepClick?: (step: RunStep) => void
}

export default function RunTimeline({ steps, onStepClick }: RunTimelineProps) {
  const maxDuration = Math.max(...steps.map(s => (s.finished_at && s.started_at ? new Date(s.finished_at).getTime() - new Date(s.started_at).getTime() : 0)), 1000)

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
      {steps.map((step) => {
        const start = step.started_at ? new Date(step.started_at).getTime() : 0
        const end = step.finished_at ? new Date(step.finished_at).getTime() : Date.now()
        const dur = end - start
        const pct = Math.max((dur / maxDuration) * 100, 5)
        const color = stepColor[step.status] || "var(--text-faint)"
        return (
          <div
            key={step.id}
            onClick={() => onStepClick?.(step)}
            style={{
              display: "flex", alignItems: "center", gap: 8, cursor: onStepClick ? "pointer" : "default",
              fontSize: 11, padding: "2px 0",
            }}
          >
            <span style={{ width: 80, textTransform: "capitalize", color: "var(--text-dim)", flexShrink: 0 }}>
              {step.step_name.replace(/_/g, " ")}
            </span>
            <div
              style={{
                height: 16, width: `${pct}%`, borderRadius: 3, background: color,
                opacity: step.status === "pending" ? 0.3 : 0.8,
                transition: "width 0.3s var(--ease-out)",
                minWidth: 4,
              }}
              title={`${(dur / 1000).toFixed(1)}s`}
            />
            <span style={{ color: "var(--text-faint)", fontSize: 10, whiteSpace: "nowrap" }}>
              {(dur / 1000).toFixed(1)}s
              {step.model && ` · ${step.model}`}
            </span>
          </div>
        )
      })}
    </div>
  )
}
