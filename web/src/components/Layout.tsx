import { useState, useMemo } from "react"
import { Outlet, useParams } from "react-router-dom"
import { useStories, useAllStoryStats, useCreateStory } from "../api/hooks"
import { inputStyle, btnStyle } from "../api/types"
import StoryListItem from "./StoryListItem"
import TopBar from "./TopBar"

export default function Layout() {
  const { storyId } = useParams<{ storyId: string }>()
  const { data: stories = [] } = useStories()
  const stats = useAllStoryStats(stories)
  const [searchQuery, setSearchQuery] = useState("")
  const [newTitle, setNewTitle] = useState("")
  const createStoryMut = useCreateStory()

  const filteredStories = useMemo(
    () => stories.filter((s) => s.title.toLowerCase().includes(searchQuery.toLowerCase())),
    [stories, searchQuery],
  )

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", background: "#0f172a", color: "#e2e8f0", fontFamily: "system-ui, sans-serif" }}>
      <TopBar searchQuery={searchQuery} onSearchChange={setSearchQuery} hasActiveStory={!!storyId} />

      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        <div style={{ width: 280, borderRight: "1px solid #334155", display: "flex", flexDirection: "column", flexShrink: 0, background: "#0f172a" }}>
          <div style={{ padding: "10px 12px", borderBottom: "1px solid #334155", display: "flex", gap: 6 }}>
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="New story title..."
              style={{ ...inputStyle, fontSize: 13, flex: 1 }}
              onKeyDown={(e) => e.key === "Enter" && newTitle.trim() && createStoryMut.mutate(newTitle.trim())}
            />
            <button onClick={() => createStoryMut.mutate(newTitle.trim())} style={btnStyle("#3b82f6", !newTitle.trim())} disabled={!newTitle.trim()}>+</button>
          </div>

          <div style={{ flex: 1, overflowY: "auto" }}>
            {filteredStories.length === 0 ? (
              <div style={{ padding: 24, textAlign: "center", color: "#64748b", fontSize: 13 }}>
                {searchQuery ? "No matching stories" : "No stories yet"}
              </div>
            ) : (
              filteredStories.map((s) => {
                const st = stats.data?.[s.id] ?? { total: 0, accepted: 0, generated: 0, stale: 0 }
                return (
                  <StoryListItem
                    key={s.id}
                    story={s}
                    chapterCount={st.total}
                    sceneCount={st.total}
                    accepted={st.accepted}
                    generated={st.generated}
                    stale={st.stale}
                    isActive={storyId === s.id}
                  />
                )
              })
            )}
          </div>
        </div>

        <div style={{ flex: 1, overflow: "auto" }}>
          <Outlet />
        </div>
      </div>
    </div>
  )
}
