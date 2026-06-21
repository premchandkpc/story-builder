import { useNavigate } from "react-router-dom"
import type { Story } from "../api/types"

interface StoryListItemProps {
  story: Story
  chapterCount: number
  sceneCount: number
  accepted: number
  generated: number
  stale: number
  isActive: boolean
  index?: number
}

export default function StoryListItem({
  story, chapterCount, sceneCount, accepted, generated, stale, isActive, index = 0
}: StoryListItemProps) {
  const navigate = useNavigate()

  const info =
    sceneCount === 0
      ? "Empty"
      : `${chapterCount} ch · ${sceneCount} sc · ${accepted}✓ ${generated}○`

  const statusColor =
    stale > 0                                ? "var(--error)"
    : sceneCount > 0 && accepted === sceneCount ? "var(--success)"
    : sceneCount > 0                           ? "var(--accent)"
                                                 : "var(--text-faint)"

  const hasIssues = stale > 0
  const isComplete = sceneCount > 0 && accepted === sceneCount
  const hasProgress = sceneCount > 0

  const date = story.createdAt ? new Date(story.createdAt).toLocaleDateString(undefined, { month: "short", day: "numeric" }) : ""

  return (
    <button
      onClick={() => navigate(`/stories/${story.id}`)}
      style={{
        width: "100%",
        padding: "11px 16px",
        background: isActive ? "var(--surface)" : "transparent",
        border: "none",
        borderBottom: "1px solid var(--border)",
        borderLeft: isActive ? "2px solid var(--accent)" : "2px solid transparent",
        color: "var(--text)",
        cursor: "pointer",
        textAlign: "left",
        fontSize: 13,
        display: "flex",
        flexDirection: "column",
        gap: 3,
        transition: "background 0.15s, border-left-color 0.15s",
        animation: `slideInLeft 0.25s var(--ease-out) ${index * 0.03}s both`,
      }}
      onMouseEnter={(e) => {
        if (!isActive) e.currentTarget.style.background = "var(--surface-hover)"
      }}
      onMouseLeave={(e) => {
        if (!isActive) e.currentTarget.style.background = "transparent"
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 7 }}>
        <span style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: statusColor,
          flexShrink: 0,
          boxShadow: hasIssues ? "0 0 6px var(--error)" : isComplete ? "0 0 6px var(--success)" : hasProgress ? "0 0 5px var(--accent)" : "none",
        }} />
        <span style={{
          fontWeight: isActive ? 700 : 500,
          fontFamily: isActive ? "var(--font-heading)" : "var(--font-body)",
          color: isActive ? "var(--accent)" : "var(--text)",
          fontSize: isActive ? 14 : 13,
          whiteSpace: "nowrap",
          overflow: "hidden",
          textOverflow: "ellipsis",
        }}>
          {story.title}
        </span>
      </div>
      <div style={{
        fontSize: 10,
        color: "var(--text-faint)",
        paddingLeft: 14,
        display: "flex",
        gap: 6,
        alignItems: "center",
      }}>
        <span>{info}</span>
        {date && (
          <>
            <span style={{ opacity: 0.3 }}>·</span>
            <span>{date}</span>
          </>
        )}
      </div>
    </button>
  )
}
