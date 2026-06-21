import { useTurns } from "../api/hooks"
import { fadeInStyle } from "../api/types"
import TurnItem from "./TurnItem"

interface TurnTimelineProps {
  storyId: string
  nodeId: string
  compact?: boolean
}

export default function TurnTimeline({ storyId, nodeId, compact }: TurnTimelineProps) {
  const { data: turns, isLoading } = useTurns(storyId, nodeId)

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, padding: 8, fontStyle: "italic" }}>Loading turns...</div>
  }

  if (!turns || turns.length === 0) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>No turns yet. Generate to populate.</div>
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 0, ...(compact ? {} : fadeInStyle) }}>
      {turns.map((turn, i) => (
        <TurnItem key={turn.id} turn={turn} index={i} compact={compact} />
      ))}
    </div>
  )
}
