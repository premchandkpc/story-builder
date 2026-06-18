// ---- StoryListItem ----
// A single story entry in the sidebar list.
// Shows title, status indicator dot, stats, and creation date.
// When clicked, navigates to that story's detail page.

// useNavigate: React Router hook that returns a navigation function.
import { useNavigate } from "react-router-dom"
import type { Story } from "../api/types"

// ---- Props interface ----
// TypeScript interface describing all props this component receives.
interface StoryListItemProps {
  story: Story           // the story object from the API
  chapterCount: number   // number of chapters (or nodes) — might be same as sceneCount
  sceneCount: number     // total scenes/nodes count
  accepted: number       // how many scenes are in accepted status
  generated: number      // how many scenes have been generated
  stale: number          // how many scenes are stale (need regeneration)
  isActive: boolean      // is this story currently selected/active in the sidebar?
}

// ---- Component ----
export default function StoryListItem({
  story, chapterCount, sceneCount, accepted, generated, stale, isActive
}: StoryListItemProps) {
  const navigate = useNavigate() // get the navigation function

  // ---- Derived data ----
  // info: human-readable summary of stats (or "No scenes" if empty)
  const info =
    sceneCount === 0
      ? "No scenes"
      : `${chapterCount} chapters · ${sceneCount} scenes · ${accepted} accepted · ${generated} generated`

  // statusColor: colored dot indicator based on overall health
  //   - red if any scenes are stale
  //   - green if all scenes are accepted
  //   - yellow if there are scenes but not all accepted
  //   - gray if no scenes at all
  const statusColor =
    stale > 0                                ? "#ef4444" :  // red
    sceneCount > 0 && accepted === sceneCount ? "#22c55e" :  // green
    sceneCount > 0                           ? "#eab308" :  // yellow
                                               "#64748b"    // gray

  // Format the creation date as a locale-aware string (e.g. "6/17/2026")
  const date = story.createdAt ? new Date(story.createdAt).toLocaleDateString() : ""

  return (
    // The whole item is a <button> so it's keyboard-accessible.
    // Clicking navigates to the story's page.
    <button
      onClick={() => navigate(`/stories/${story.id}`)}
      style={{
        width: "100%",
        padding: "10px 12px",
        background: isActive ? "#1e293b" : "transparent", // highlight active story
        border: "none",
        borderBottom: "1px solid #1e293b",
        color: "#e2e8f0",
        cursor: "pointer",
        textAlign: "left",
        fontSize: 14,
        display: "flex",
        flexDirection: "column",
        gap: 2,
      }}
    >
      {/* Row 1: status dot + title */}
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        {/*
          Status dot: a small circle colored by overall health.
          flexShrink: 0 prevents the dot from being compressed.
        */}
        <span style={{ width: 8, height: 8, borderRadius: "50%", background: statusColor, flexShrink: 0 }} />
        {/* Title: bolder if this is the active story */}
        <span style={{ fontWeight: isActive ? 700 : 500 }}>{story.title}</span>
      </div>
      {/* Row 2: stats and date in smaller, dimmer text */}
      <div style={{ fontSize: 11, color: "#64748b", paddingLeft: 14 }}>{info} · {date}</div>
    </button>
  )
}
