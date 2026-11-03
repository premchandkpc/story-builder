import { useState, useMemo, useEffect } from "react"
import { Outlet, useParams } from "react-router-dom"

import { useStories, useAllStoryStats, useCreateStory, useDeleteStory, setToastFns } from "../api/hooks"
import { inputStyle, btnStyle, skeletonStyle } from "../api/types"
import StoryListItem from "./StoryListItem"
import TopBar from "./TopBar"
import { useToast } from "./Toast"

const sidebarToggleSvg = (
  <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M6 3L3 8l3 5" />
    <path d="M13 3l-3 5 3 5" />
  </svg>
)

const plusSvg = (
  <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M8 3v10M3 8h10" />
  </svg>
)

const storyIconSvg = (
  <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" style={{ opacity: 0.3 }}>
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
    <path d="M14 2v6h6" />
    <path d="M12 18v-6" />
    <path d="M9 15h6" />
  </svg>
)

const searchIconSvg = (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" style={{ opacity: 0.3 }}>
    <circle cx="11" cy="11" r="8" />
    <path d="M21 21l-4.35-4.35" />
  </svg>
)

export default function Layout() {
  const toast = useToast()
  const { error: showError, success } = toast
  const { storyId } = useParams<{ storyId: string }>()

  /** Wire toast into hooks.ts for optimistic update notifications */
  useEffect(() => {
    setToastFns({ success, error: showError })
    return () => setToastFns(null)
  }, [success, showError])

  const { data: stories = [], isLoading, isError, refetch } = useStories()
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
  const deleteStoryMut = useDeleteStory()

  const handleCreateStory = () => {
    const t = newTitle.trim()
    if (!t) return
    createStoryMut.mutate(t)
    setNewTitle("")
  }

  const handleDeleteStory = (storyId: string) => {
    deleteStoryMut.mutate(storyId)
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
          background: "var(--bg-warm)",
          overflow: "hidden",
          transition: "width 0.2s var(--ease-out)",
        }}>
          <div style={{
            padding: "12px 16px 10px",
            borderBottom: "1px solid var(--border)",
            display: "flex",
            gap: 6,
            minWidth: 280,
          }}>
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="New story title..."
              style={{ ...inputStyle, fontSize: 12, flex: 1 }}
              onKeyDown={(e) => e.key === "Enter" && handleCreateStory()}
            />
            <button
              onClick={handleCreateStory}
              style={btnStyle("var(--accent)", !newTitle.trim())}
              disabled={!newTitle.trim()}
              title="Create story"
            >
              {createStoryMut.isPending ? (
                <div style={{
                  width: 14, height: 14,
                  border: "2px solid rgba(0,0,0,0.2)",
                  borderTopColor: "#1a1512",
                  borderRadius: "50%",
                  animation: "spin 0.6s linear infinite",
                }} />
              ) : plusSvg}
            </button>
          </div>

          <div style={{
            fontSize: 9,
            color: "var(--text-faint)",
            textTransform: "uppercase",
            letterSpacing: "0.1em",
            fontWeight: 600,
            padding: "10px 16px 7px",
            borderBottom: "1px solid var(--border)",
            minWidth: 280,
          }}>
            Stories {filteredStories.length > 0 && `(${filteredStories.length})`}
          </div>

          <div style={{
            flex: 1, overflowY: "auto", minWidth: 280,
            background: "var(--bg)",
          }}>
            {isLoading ? (
              <div className="stagger-fade-in" style={{ padding: 14, display: "flex", flexDirection: "column", gap: 12 }}>
                {[1, 2, 3, 4].map((i) => (
                  <div key={i} style={{
                    display: "flex", flexDirection: "column", gap: 4,
                    animation: `fadeIn 0.25s var(--ease-out) ${i * 0.06}s both`,
                  }}>
                    <div style={skeletonStyle("75%", 15)} />
                    <div style={skeletonStyle("45%", 10)} />
                  </div>
                ))}
              </div>
            ) : isError ? (
              <div style={{
                padding: 40, textAlign: "center", color: "var(--text-dim)", fontSize: 12,
                display: "flex", flexDirection: "column", alignItems: "center", gap: 12,
              }}>
                <span role="alert" style={{ color: "var(--error)", fontSize: 13 }}>
                  Failed to load stories
                </span>
                <button
                  onClick={() => refetch()}
                  style={{
                    ...btnStyle("var(--accent)", false),
                    fontSize: 11, padding: "6px 14px",
                  }}
                >
                  Retry
                </button>
              </div>
            ) : filteredStories.length === 0 && searchQuery ? (
              <div style={{
                padding: 40, textAlign: "center", color: "var(--text-faint)", fontSize: 12,
                display: "flex", flexDirection: "column", alignItems: "center", gap: 8,
              }}>
                {searchIconSvg}
                <span>No stories match "{searchQuery}"</span>
                <button
                  onClick={() => setSearchQuery("")}
                  style={{
                    background: "none", border: "none",
                    color: "var(--accent)", cursor: "pointer", fontSize: 11,
                    textDecoration: "underline", textUnderlineOffset: 3,
                  }}
                >
                  Clear search
                </button>
              </div>
            ) : filteredStories.length === 0 && !searchQuery && !isLoading ? (
              <div className="stagger-fade-in" style={{
                padding: 48, textAlign: "center", color: "var(--text-faint)", fontSize: 12,
                display: "flex", flexDirection: "column", alignItems: "center", gap: 8,
              }}>
                {storyIconSvg}
                <span style={{ fontFamily: "var(--font-heading)", fontSize: 15, marginTop: 4 }}>
                  No stories yet
                </span>
                <span style={{ fontSize: 11, color: "var(--text-dim)", maxWidth: 200, lineHeight: 1.5 }}>
                  Create one above to get started, or generate a full story from synopsis on the home page.
                </span>
              </div>
            ) : (
              filteredStories.map((s, i) => {
                const st = stats.data?.[s.id] ?? { total: 0, accepted: 0, generated: 0, stale: 0 }
                const isPlaceholder = s.id.startsWith("new-")
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
                    isOptimistic={isPlaceholder}
                    onDelete={handleDeleteStory}
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
            borderRadius: "0 5px 5px 0",
            padding: "10px 3px",
            cursor: "pointer",
            color: "var(--text-faint)",
            zIndex: 10,
            transition: "left 0.2s var(--ease-out), color 0.15s, background 0.15s",
            display: "flex",
            alignItems: "center",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = "var(--accent)"
            e.currentTarget.style.background = "var(--surface-hover)"
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = "var(--text-faint)"
            e.currentTarget.style.background = "var(--surface)"
          }}
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
