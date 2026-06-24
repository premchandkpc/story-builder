import { useScenePlan } from "../api/hooks"
import { skeletonStyle, cardStyle, labelStyle, badgeStyle } from "../api/types"

interface ScenePlanPanelProps {
  storyId: string
  nodeId: string
}

export default function ScenePlanPanel({ storyId, nodeId }: ScenePlanPanelProps) {
  const { data: plan, isLoading, error } = useScenePlan(storyId, nodeId)

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        <div style={skeletonStyle("60%", 14)} />
        <div style={skeletonStyle("40%", 14)} />
        <div style={skeletonStyle("80%", 14)} />
      </div>
    )
  }

  if (error || !plan) {
    return (
      <div style={{ color: "var(--text-dim)", fontSize: 12, padding: 8 }}>
        {error instanceof Error ? error.message : "No plan available"}
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      <div style={cardStyle as React.CSSProperties}>
        <div style={{ padding: "10px 12px", borderBottom: "1px solid var(--border)" }}>
          <span style={labelStyle}>Purpose</span>
          <div style={{ fontSize: 12, color: "var(--text)", marginTop: 4 }}>
            {plan.purpose.conflictType && (
              <span style={badgeStyle("var(--accent)", "var(--accent-dim)")}>
                {plan.purpose.conflictType}
              </span>
            )}
            {plan.purpose.advancingArcs.length > 0 && (
              <span style={{ ...badgeStyle("var(--primary)", "var(--primary-dim)"), marginLeft: 6 }}>
                {plan.purpose.advancingArcs.length} arc{plan.purpose.advancingArcs.length > 1 ? "s" : ""}
              </span>
            )}
          </div>
        </div>

        {plan.purpose.requiredBeats.length > 0 && (
          <div style={{ padding: "10px 12px", borderBottom: "1px solid var(--border)" }}>
            <span style={labelStyle}>Required Beats</span>
            <div style={{ display: "flex", flexDirection: "column", gap: 4, marginTop: 4 }}>
              {plan.purpose.requiredBeats.map((beat, i) => (
                <div key={i} style={{ fontSize: 11, color: "var(--text)", display: "flex", gap: 6 }}>
                  <span style={{ color: "var(--text-dim)", minWidth: 60, fontWeight: 600 }}>
                    {beat.type}
                  </span>
                  <span>{beat.description}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        <div style={{ padding: "10px 12px", borderBottom: "1px solid var(--border)" }}>
          <span style={labelStyle}>Config</span>
          <div style={{ display: "flex", gap: 12, marginTop: 4, fontSize: 12 }}>
            <span><span style={{ color: "var(--text-dim)" }}>Tone:</span> {plan.suggestedTone}</span>
            <span><span style={{ color: "var(--text-dim)" }}>POV:</span> {plan.suggestedPOV}</span>
            <span><span style={{ color: "var(--text-dim)" }}>Words:</span> ~{plan.suggestedWords}</span>
          </div>
        </div>

        {Object.keys(plan.participantIntent).length > 0 && (
          <div style={{ padding: "10px 12px" }}>
            <span style={labelStyle}>Participants</span>
            <div style={{ display: "flex", flexDirection: "column", gap: 4, marginTop: 4 }}>
              {Object.entries(plan.participantIntent).map(([charId, intent]) => (
                <div key={charId} style={{ fontSize: 11, color: "var(--text)" }}>
                  <span style={{ fontWeight: 600, color: "var(--accent)" }}>{charId.slice(0, 8)}…</span>
                  : {intent.slice(0, 120)}{intent.length > 120 ? "…" : ""}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
