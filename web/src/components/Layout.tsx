import { useState, useMemo, useEffect } from "react"
import { Outlet, useParams } from "react-router-dom"

import { useStories, useAllStoryStats, useCreateStory } from "../api/hooks"
import { inputStyle, btnStyle, skeletonStyle } from "../api/types"
import StoryListItem from "./StoryListItem"
import TopBar from "./TopBar"
import { useToast } from "./Toast"

const sidebarToggleSvg = (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M6 3L3 8l3 5" />
    <path d="M13 3l-3 5 3 5" />
  </svg>
)

const plusSvg = (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M8 3v10M3 8h10" />
  </svg>
)

export default function Layout() {
  const { toast, error: showError } = useToast()
  const { storyId } = useParams<{ storyId: string }>()

  const { data: stories = [], isLoading } = useStories()
  const stats = useAllStoryStats(stories)

  const [searchQuery, setSearchQuery] = useState("")
  const [newTitle, setNewTitle] = useState("")
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    const saved = localStorage.getItem("sidebarOpen")
    return saved !== null ? saved === "true" : true
  })

  useEffect(() => {
    localStorage.setItem("sidebarOpen", String(sidebarOpen))
  }, [sidebarOpen])

  const createStoryMut = useCreateStory()

  const handleCreateStory = () => {
    const t = newTitle.trim()
    if (!t) return
    createStoryMut.mutate(t, {
      onError: () => showError("Failed to create story"),
    })
  }

  const filteredStories = useMemo(
    () => stories.filter((s) => s.title.toLowerCase().includes(searchQuery.toLowerCase())),
    [stories, searchQuery],
  )

  return (
    <div style={{
      display: "flex",
      flexDirection: "column",
      height: "100vh",
      background: "var(--bg)",
      color: "var(--text)",
      fontFamily: "var(--font-body)",
    }}>
      <TopBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        hasActiveStory={!!storyId}
        onToggleSidebar={() => setSidebarOpen((o) => !o)}
        sidebarOpen={sidebarOpen}
      />

      <div style={{ display: "flex", flex: 1, overflow: "hidden", position: "relative" }}>
        <div style={{
          width: sidebarOpen ? 280 : 0,
          borderRight: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          flexShrink: 0,
          background: "var(--bg)",
          overflow: "hidden",
          transition: "width 0.2s var(--ease-out)",
        }}>
          <div style={{
            padding: "12px 14px",
            borderBottom: "1px solid var(--border)",
            display: "flex",
            gap: 6,
            minWidth: 280,
          }}>
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="New story title..."
              style={{ ...inputStyle, fontSize: 13, flex: 1 }}
              onKeyDown={(e) => e.key === "Enter" && handleCreateStory()}
            />
            <button
              onClick={handleCreateStory}
              style={btnStyle("var(--accent)", !newTitle.trim())}
              disabled={!newTitle.trim()}
              title="Create story"
            >
              {plusSvg}
            </button>
          </div>

          <div style={{ flex: 1, overflowY: "auto", minWidth: 280 }}>
            {isLoading ? (
              <div style={{ padding: 14, display: "flex", flexDirection: "column", gap: 12 }}>
                {[1, 2, 3, 4].map((i) => (
                  <div key={i} style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    <div style={skeletonStyle("70%", 16)} />
                    <div style={skeletonStyle("40%", 11)} />
                  </div>
                ))}
              </div>
            ) : filteredStories.length === 0 ? (
              <div style={{
                padding: 32, textAlign: "center", color: "var(--text-muted)", fontSize: 13,
                display: "flex", flexDirection: "column", alignItems: "center", gap: 8,
              }}>
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" style={{ opacity: 0.4 }}>
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                  <path d="M14 2v6h6" />
                  <path d="M12 18v-6" />
                  <path d="M9 15h6" />
                </svg>
                <span>{searchQuery ? "No matching stories" : "No stories yet"}</span>
                <span style={{ fontSize: 11, opacity: 0.6 }}>Create one above</span>
              </div>
            ) : (
              filteredStories.map((s, i) => {
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
                    index={i}
                  />
                )
              })
            )}
          </div>
        </div>

        <button
          onClick={() => setSidebarOpen((o) => !o)}
          style={{
            position: "absolute",
            left: sidebarOpen ? 280 : 0,
            top: "50%",
            transform: "translateY(-50%)",
            background: "var(--surface)",
            border: "1px solid var(--border)",
            borderLeft: "none",
            borderRadius: "0 4px 4px 0",
            padding: "8px 4px",
            cursor: "pointer",
            color: "var(--text-muted)",
            zIndex: 10,
            transition: "left 0.2s var(--ease-out), color 0.15s",
            display: "flex",
            alignItems: "center",
          }}
          onMouseEnter={(e) => e.currentTarget.style.color = "var(--accent)"}
          onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-muted)"}
          title={sidebarOpen ? "Collapse sidebar" : "Expand sidebar"}
        >
          <div style={{ transform: sidebarOpen ? "none" : "rotate(180deg)", transition: "transform 0.2s" }}>
            {sidebarToggleSvg}
          </div>
        </button>

        <div style={{
          flex: 1,
          overflow: "auto",
          animation: "fadeIn 0.25s var(--ease-out)",
        }}>
          <Outlet />
        </div>
      </div>
    </div>
  )
}