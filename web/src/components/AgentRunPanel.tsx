import { useAgentRuns } from "../api/hooks"
import { fadeInStyle } from "../api/types"
import AgentRunItem from "./AgentRunItem"

interface AgentRunPanelProps {
  storyId: string
  nodeId: string
}

export default function AgentRunPanel({ storyId, nodeId }: AgentRunPanelProps) {
  const { data: runs, isLoading } = useAgentRuns(storyId, nodeId)

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, padding: 8, fontStyle: "italic" }}>Loading agent runs...</div>
  }

  if (!runs || runs.length === 0) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>No agent runs recorded.</div>
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...fadeInStyle }}>
      {runs.map((run, i) => (
        <AgentRunItem key={run.id} run={run} index={i} />
      ))}
    </div>
  )
}
