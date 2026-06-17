// ---- Layout ----
// The main app layout/shell component.
// Renders:
//   - TopBar (search + app title)
//   - Sidebar (story list with search filter, create story input)
//   - Content area (renders child routes via <Outlet/>)

// useState: hook for component-level state (search query, new title input)
// useMemo: hook that memoizes/memoizes a computed value.
//   - Only recomputes when dependencies change (stories, searchQuery).
//   - Avoids filtering the array on every render if inputs haven't changed.
import { useState, useMemo } from "react"

// Outlet: React Router component that renders the matched child route.
//   - In our route tree (routes.tsx), Layout has children (HomeView, StoryView).
//   - <Outlet/> is where those children get rendered.
//
// useParams: extracts URL parameters (specifically :storyId to know if we're
// viewing a story, which highlights it in the sidebar).
import { Outlet, useParams } from "react-router-dom"

import { useStories, useAllStoryStats, useCreateStory } from "../api/hooks"
import { inputStyle, btnStyle } from "../api/types"
import StoryListItem from "./StoryListItem"
import TopBar from "./TopBar"

// ---- Component ----
export default function Layout() {
  // ---- URL params ----
  // storyId: if we're on a story page like /stories/abc, this is "abc".
  // Used to highlight the active story in the sidebar.
  const { storyId } = useParams<{ storyId: string }>()

  // ---- Data fetching (React Query hooks) ----
  // useStories: fetches list of all stories.
  // `data` defaults to empty array via `= []` for when data hasn't loaded yet.
  const { data: stories = [] } = useStories()

  // useAllStoryStats: fetches node counts for every story (parallel requests).
  // This is a query, not the data itself — we access stats.data later.
  const stats = useAllStoryStats(stories)

  // ---- Local state ----
  // searchQuery: the text typed in the search bar (controlled input)
  const [searchQuery, setSearchQuery] = useState("")
  // newTitle: the text typed in the "create story" input
  const [newTitle, setNewTitle] = useState("")

  // ---- Mutations ----
  const createStoryMut = useCreateStory()

  // ---- Derived data ----
  // filteredStories: stories whose title contains the search text (case-insensitive).
  // useMemo ensures this is only recalculated when stories or searchQuery changes.
  const filteredStories = useMemo(
    () => stories.filter((s) => s.title.toLowerCase().includes(searchQuery.toLowerCase())),
    [stories, searchQuery],
  )

  // ---- Render ----
  return (
    // Outer container: full viewport height, dark background, flex column
    <div style={{
      display: "flex",
      flexDirection: "column",
      height: "100vh",
      background: "#0f172a",
      color: "#e2e8f0",
      fontFamily: "system-ui, sans-serif",
    }}>
      {/*
        TopBar component:
          searchQuery      — current search text
          onSearchChange   — updates searchQuery when user types
          hasActiveStory   — true when we're viewing a story (used to show "Home" button)
      */}
      <TopBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        hasActiveStory={!!storyId}
      />

      {/* Main content area: sidebar + outlet side by side */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        {/* ---- Sidebar ---- */}
        <div style={{
          width: 280,
          borderRight: "1px solid #334155",
          display: "flex",
          flexDirection: "column",
          flexShrink: 0,       // prevent sidebar from shrinking
          background: "#0f172a",
        }}>
          {/* Create story input row */}
          <div style={{
            padding: "10px 12px",
            borderBottom: "1px solid #334155",
            display: "flex",
            gap: 6,
          }}>
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="New story title..."
              style={{ ...inputStyle, fontSize: 13, flex: 1 }}
              // Press Enter to create a story
              onKeyDown={(e) => e.key === "Enter" && newTitle.trim() && createStoryMut.mutate(newTitle.trim())}
            />
            <button
              onClick={() => createStoryMut.mutate(newTitle.trim())}
              style={btnStyle("#3b82f6", !newTitle.trim())}
              disabled={!newTitle.trim()}
            >
              +
            </button>
          </div>

          {/* Scrollable story list */}
          <div style={{ flex: 1, overflowY: "auto" }}>
            {/* Empty state: if no matching stories */}
            {filteredStories.length === 0 ? (
              <div style={{ padding: 24, textAlign: "center", color: "#64748b", fontSize: 13 }}>
                {searchQuery ? "No matching stories" : "No stories yet"}
              </div>
            ) : (
              // Map over filtered stories and render each as a StoryListItem
              filteredStories.map((s) => {
                // Get stats for this story, or default to zeros
                const st = stats.data?.[s.id] ?? { total: 0, accepted: 0, generated: 0, stale: 0 }
                return (
                  <StoryListItem
                    key={s.id}                        // React key for list reconciliation
                    story={s}
                    chapterCount={st.total}
                    sceneCount={st.total}
                    accepted={st.accepted}
                    generated={st.generated}
                    stale={st.stale}
                    isActive={storyId === s.id}       // highlight if currently viewed
                  />
                )
              })
            )}
          </div>
        </div>

        {/* ---- Content area ---- */}
        {/*
          <Outlet/> renders the matched child route:
            - At "/"       → HomeView
            - At "/stories/:id" → StoryView
          flex: 1 makes it fill remaining space.
        */}
        <div style={{ flex: 1, overflow: "auto" }}>
          <Outlet />
        </div>
      </div>
    </div>
  )
}
