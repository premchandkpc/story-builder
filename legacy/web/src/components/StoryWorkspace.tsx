import { useCallback } from "react"
import { useParams, useNavigate } from "react-router-dom"
import type { StoryViewMode } from "../api/types"
import { useStories } from "../api/hooks"
import StoryHeader from "./StoryHeader"
import StoryGraph from "./StoryGraph"
import StoryReadView from "./StoryReadView"
import StoryOutlineView from "./StoryOutlineView"
import StoryInspectView from "./StoryInspectView"

export default function StoryWorkspace() {
  const { storyId, viewMode: rawMode } = useParams<{ storyId: string; viewMode?: string }>()
  const navigate = useNavigate()
  const { data: stories } = useStories()
  const story = stories?.find((s) => s.id === storyId)

  const viewMode: StoryViewMode = rawMode === "read" ? "read"
    : rawMode === "outline" ? "outline"
    : rawMode === "inspect" ? "inspect"
    : "graph"

  const onViewModeChange = useCallback((mode: StoryViewMode) => {
    navigate(`/stories/${storyId}/${mode}`, { replace: true })
  }, [navigate, storyId])

  if (!storyId) return null

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <StoryHeader
        storyId={storyId}
        title={story?.title || "Loading..."}
        viewMode={viewMode}
        onViewModeChange={onViewModeChange}
      />
      <div style={{ flex: 1, minHeight: 0 }}>
        {viewMode === "read" && <StoryReadView storyId={storyId} />}
        {viewMode === "outline" && <StoryOutlineView storyId={storyId} />}
        {viewMode === "graph" && <StoryGraph storyId={storyId} />}
        {viewMode === "inspect" && <StoryInspectView storyId={storyId} />}
      </div>
    </div>
  )
}
