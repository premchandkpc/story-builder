import { memo, useState, useCallback } from "react"
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
  isOptimistic?: boolean
  onDelete?: (storyId: string) => void
}

const StoryListItem = memo(function StoryListItem({
  story, chapterCount, sceneCount, accepted, generated, stale,
  isActive, index = 0, isOptimistic, onDelete,
}: StoryListItemProps) {
  const navigate = useNavigate()
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [deleteTimeout, setDeleteTimeout] = useState<ReturnType<typeof setTimeout> | null>(null)

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

  const handleClick = useCallback(() => {
    if (isOptimistic) return
    navigate(`/stories/${story.id}`)
  }, [story.id, navigate, isOptimistic])

  const handleDeleteClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation()
    if (!onDelete) return
    if (!confirmingDelete) {
      setConfirmingDelete(true)
      const t = setTimeout(() => {
        setConfirmingDelete(false)
      }, 3000)
      setDeleteTimeout(t)
      return
    }
    if (deleteTimeout) clearTimeout(deleteTimeout)
    setConfirmingDelete(false)
    onDelete(story.id)
  }, [confirmingDelete, deleteTimeout, onDelete, story.id])

  return (
    <div
      onClick={handleClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && handleClick()}
      style={{
        width: "100%",
        padding: "11px 16px",
        background: isOptimistic ? "var(--surface)" : isActive ? "var(--surface)" : "transparent",
        border: "none",
        borderBottom: "1px solid var(--border)",
        borderLeft: isActive ? "2px solid var(--accent)" : "2px solid transparent",
        color: "var(--text)",
        cursor: isOptimistic ? "default" : "pointer",
        textAlign: "left",
        fontSize: 13,
        display: "flex",
        flexDirection: "column",
        gap: 3,
        transition: "background var(--transition-base), border-left-color var(--transition-base)",
        animation: isOptimistic
          ? "expandIn 0.25s var(--ease-out)"
          : `slideInLeft 0.25s var(--ease-out) ${index * 0.03}s both`,
        opacity: isOptimistic ? 0.8 : 1,
        position: "relative",
      }}
      onMouseEnter={(e) => {
        if (!isActive && !isOptimistic) e.currentTarget.style.background = "var(--surface-hover)"
      }}
      onMouseLeave={(e) => {
        if (!isActive && !isOptimistic) e.currentTarget.style.background = "transparent"
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 7 }}>
        <span style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: isOptimistic ? "var(--text-faint)" : statusColor,
          flexShrink: 0,
          boxShadow: isOptimistic ? "none" : hasIssues ? "0 0 6px var(--error)" : isComplete ? "0 0 6px var(--success)" : hasProgress ? "0 0 5px var(--accent)" : "none",
          animation: isOptimistic ? "pulse 1.5s ease-in-out infinite" : undefined,
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
        <span>{isOptimistic ? "Creating..." : info}</span>
        {date && !isOptimistic && (
          <>
            <span style={{ opacity: 0.3 }}>·</span>
            <span>{date}</span>
          </>
        )}
      </div>
      {!isOptimistic && onDelete && (
        <button
          onClick={handleDeleteClick}
          style={{
            position: "absolute",
            right: 8,
            top: "50%",
            transform: "translateY(-50%)",
            background: confirmingDelete ? "var(--error)" : "transparent",
            border: confirmingDelete ? "none" : "none",
            borderRadius: "var(--radius-sm)",
            padding: "4px 8px",
            color: confirmingDelete ? "#fff" : "var(--text-faint)",
            cursor: "pointer",
            fontSize: 10,
            fontWeight: 600,
            opacity: isActive ? 1 : 0,
            transition: "opacity var(--transition-base), background var(--transition-base), color var(--transition-base)",
          }}
          onMouseEnter={(e) => {
            if (!isActive) e.currentTarget.style.opacity = "1"
          }}
          onMouseLeave={(e) => {
            if (!isActive && !confirmingDelete) e.currentTarget.style.opacity = "0"
          }}
          aria-label={confirmingDelete ? "Confirm delete" : "Delete story"}
        >
          {confirmingDelete ? "Confirm Delete?" : "×"}
        </button>
      )}
    </div>
  )
})

export default StoryListItem
