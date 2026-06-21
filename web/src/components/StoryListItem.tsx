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
      ? "No scenes"
      : `${chapterCount} chapters · ${sceneCount} scenes · ${accepted} accepted · ${generated} generated`

  const statusColor =
    stale > 0                                ? "var(--error)"
    : sceneCount > 0 && accepted === sceneCount ? "var(--success)"
    : sceneCount > 0                           ? "var(--accent)"
                                                 : "var(--text-dim)"

  const hasIssues = stale > 0
  const isComplete = sceneCount > 0 && accepted === sceneCount
  const hasProgress = sceneCount > 0

  const date = story.createdAt ? new Date(story.createdAt).toLocaleDateString() : ""

  return (
    <button
      onClick={() => navigate(`/stories/${story.id}`)}
      style={{
        width: "100%",
        padding: "12px 16px",
        background: isActive ? "var(--surface)" : "transparent",
        border: "none",
        borderBottom: "1px solid var(--border)",
        borderLeft: isActive ? "2px solid var(--accent)" : "2px solid transparent",
        color: "var(--text)",
        cursor: "pointer",
        textAlign: "left",
        fontSize: 14,
        display: "flex",
        flexDirection: "column",
        gap: 4,
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
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{
          width: 7,
          height: 7,
          borderRadius: "50%",
          background: statusColor,
          flexShrink: 0,
          boxShadow: hasIssues ? "0 0 8px var(--error)" : isComplete ? "0 0 8px var(--success)" : hasProgress ? "0 0 6px var(--accent)" : "none",
        }} />
        <span style={{
          fontWeight: isActive ? 700 : 500,
          fontFamily: isActive ? "var(--font-heading)" : "var(--font-body)",
          color: isActive ? "var(--accent)" : "var(--text)",
          fontSize: isActive ? 15 : 14,
          whiteSpace: "nowrap",
          overflow: "hidden",
          textOverflow: "ellipsis",
        }}>
          {story.title}
        </span>
      </div>
      <div style={{
        fontSize: 11,
        color: "var(--text-dim)",
        paddingLeft: 16,
        display: "flex",
        gap: 8,
        alignItems: "center",
      }}>
        <span>{info}</span>
        {date && (
          <>
            <span style={{ opacity: 0.4 }}>·</span>
            <span>{date}</span>
          </>
        )}
      </div>
    </button>
  )
}
