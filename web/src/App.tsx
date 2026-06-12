import { useCallback, useEffect, useMemo, useState } from "react"
import StoryGraph from "./components/StoryGraph"
import { api } from "./api/client"
import type { Story } from "./api/types"

type StoryStats = Record<string, { total: number; generated: number; accepted: number; stale: number }>

const inputStyle: Record<string, string | number> = {
  width: "100%",
  padding: "10px 12px",
  background: "#1e293b",
  border: "1px solid #334155",
  borderRadius: 6,
  color: "#e2e8f0",
  fontSize: 14,
  boxSizing: "border-box",
  outline: "none",
}

const btnStyle = (bg: string, disabled = false): Record<string, string | number> => ({
  padding: "10px 16px",
  background: disabled ? "#64748b" : bg,
  color: "#fff",
  border: "none",
  borderRadius: 6,
  cursor: disabled ? "not-allowed" : "pointer",
  fontWeight: 600,
  fontSize: 14,
})

export default function App() {
  const [stories, setStories] = useState<Story[]>([])
  const [activeStoryId, setActiveStoryId] = useState<string | null>(null)
  const [newTitle, setNewTitle] = useState("")
  const [synopsis, setSynopsis] = useState("")
  const [generating, setGenerating] = useState(false)
  const [generatingTitle, setGeneratingTitle] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [stats, setStats] = useState<StoryStats>({})

  useEffect(() => {
    api.stories.list().then(loadStats).catch(console.error)
  }, [])

  const loadStats = async (storyList: Story[]) => {
    setStories(storyList)
    const entries = await Promise.all(
      storyList.map(async (s) => {
        try {
          const nodes = await api.nodes.list(s.id)
          const total = nodes.length
          const generated = nodes.filter((n) => n.status === "generated").length
          const accepted = nodes.filter((n) => n.status === "accepted").length
          const stale = nodes.filter((n) => n.status === "stale").length
          return [s.id, { total, generated, accepted, stale }] as const
        } catch {
          return [s.id, { total: 0, generated: 0, accepted: 0, stale: 0 }] as const
        }
      }),
    )
    setStats(Object.fromEntries(entries))
  }

  const filteredStories = useMemo(
    () => stories.filter((s) => s.title.toLowerCase().includes(searchQuery.toLowerCase())),
    [stories, searchQuery],
  )

  const createStory = useCallback(async (title?: string) => {
    const t = (title || newTitle || synopsis.trim().slice(0, 50)).trim()
    if (!t) return
    try {
      const story = await api.stories.create({ title: t })
      setStories((prev) => [...prev, story])
      setActiveStoryId(story.id)
      setNewTitle("")
    } catch (err) {
      console.error(err)
    }
  }, [newTitle, synopsis])

  const generateTitle = useCallback(async () => {
    if (!synopsis.trim() || generatingTitle) return
    setGeneratingTitle(true)
    try {
      const res = await api.stories.generateTitle({ synopsis })
      setNewTitle(res.title)
    } catch (err) {
      console.error(err)
    }
    setGeneratingTitle(false)
  }, [synopsis, generatingTitle])

  const generateStory = useCallback(async () => {
    if (!synopsis.trim()) return
    setGenerating(true)
    try {
      await api.stories.generate({ synopsis })
      setSynopsis("")
      alert("Story generation started (async). Refresh the list in a moment.")
      setTimeout(async () => {
        const list = await api.stories.list()
        await loadStats(list)
        setGenerating(false)
      }, 3000)
    } catch (err) {
      console.error(err)
      setGenerating(false)
    }
  }, [synopsis])

  const storyBar = (s: Story) => {
    const st = stats[s.id]
    const nodeCount = st?.total ?? 0
    const nodeInfo =
      nodeCount === 0
        ? "No nodes"
        : `${nodeCount} nodes · ${st!.accepted} accepted · ${st!.generated} generated`
    const statusColor =
      (st?.stale ?? 0) > 0 ? "#ef4444" : nodeCount > 0 && st!.accepted === nodeCount ? "#22c55e" : nodeCount > 0 ? "#eab308" : "#64748b"
    const date = new Date(s.created_at).toLocaleDateString()
    return (
      <button
        key={s.id}
        onClick={() => setActiveStoryId(s.id)}
        style={{
          width: "100%",
          padding: "10px 12px",
          background: activeStoryId === s.id ? "#1e293b" : "transparent",
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
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <span style={{ width: 8, height: 8, borderRadius: "50%", background: statusColor, flexShrink: 0 }} />
          <span style={{ fontWeight: activeStoryId === s.id ? 700 : 500 }}>{s.title}</span>
        </div>
        <div style={{ fontSize: 11, color: "#64748b", paddingLeft: 14 }}>{nodeInfo} · {date}</div>
      </button>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", background: "#0f172a", color: "#e2e8f0", fontFamily: "system-ui, sans-serif" }}>
      {/* ── Top Bar ── */}
      <div style={{ display: "flex", alignItems: "center", padding: "8px 16px", background: "#1e293b", borderBottom: "1px solid #334155", gap: 12, flexShrink: 0 }}>
        <span style={{ fontSize: 18, fontWeight: 700, color: "#f8fafc", marginRight: 8 }}>Story Builder</span>
        <div style={{ position: "relative", flex: "0 1 300px" }}>
          <input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search stories..."
            style={{ ...inputStyle, paddingLeft: 32, fontSize: 13 }}
          />
          <span style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "#64748b", fontSize: 13 }}>🔍</span>
        </div>
        <div style={{ flex: 1 }} />
        {activeStoryId && (
          <button onClick={() => setActiveStoryId(null)} style={btnStyle("#475569")}>Home</button>
        )}
      </div>

      {/* ── Body ── */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        {/* ── Sidebar ── */}
        <div style={{ width: 280, borderRight: "1px solid #334155", display: "flex", flexDirection: "column", flexShrink: 0, background: "#0f172a" }}>
          <div style={{ padding: "10px 12px", borderBottom: "1px solid #334155", display: "flex", gap: 6 }}>
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="New story title..."
              style={{ ...inputStyle, fontSize: 13, flex: 1 }}
              onKeyDown={(e) => e.key === "Enter" && createStory()}
            />
            <button onClick={createStory} style={btnStyle("#3b82f6", !newTitle.trim())} disabled={!newTitle.trim()}>+</button>
          </div>
          <div style={{ flex: 1, overflowY: "auto" }}>
            {filteredStories.length === 0 ? (
              <div style={{ padding: 24, textAlign: "center", color: "#64748b", fontSize: 13 }}>
                {searchQuery ? "No matching stories" : "No stories yet"}
              </div>
            ) : (
              filteredStories.map(storyBar)
            )}
          </div>
        </div>

        {/* ── Main Content ── */}
        <div style={{ flex: 1, overflow: "auto" }}>
          {activeStoryId ? (
            <StoryGraph storyId={activeStoryId} />
          ) : (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", minHeight: "100%", gap: 20, padding: 40 }}>
              <div style={{ fontSize: 22, fontWeight: 700 }}>Story Builder</div>
              <div style={{ background: "#1e293b", border: "1px solid #334155", borderRadius: 8, padding: 24, width: 460, display: "flex", flexDirection: "column", gap: 12 }}>
                <div style={{ fontSize: 14, color: "#94a3b8" }}>Create Story</div>
                <div style={{ display: "flex", gap: 8 }}>
                  <input
                    value={newTitle}
                    onChange={(e) => setNewTitle(e.target.value)}
                    placeholder="Story title (or generate from synopsis)"
                    style={{ ...inputStyle, flex: 1 }}
                    onKeyDown={(e) => e.key === "Enter" && createStory()}
                  />
                  <button
                    onClick={generateTitle}
                    disabled={!synopsis.trim() || generatingTitle}
                    style={btnStyle("#8b5cf6", !synopsis.trim() || generatingTitle)}
                    title="Generate title from synopsis"
                  >
                    {generatingTitle ? "..." : "✨"}
                  </button>
                </div>
                <textarea
                  value={synopsis}
                  onChange={(e) => setSynopsis(e.target.value)}
                  placeholder="Describe the story you want to generate..."
                  rows={4}
                  style={{ ...inputStyle, resize: "vertical" }}
                />
                <div style={{ display: "flex", gap: 8 }}>
                  <button
                    onClick={() => createStory()}
                    style={{ ...btnStyle("#3b82f6", !newTitle.trim() && !synopsis.trim()), flex: 1 }}
                    disabled={!newTitle.trim() && !synopsis.trim()}
                  >
                    Create{!newTitle.trim() && synopsis.trim() ? " (auto-title)" : ""}
                  </button>
                  <button onClick={generateStory} disabled={generating || !synopsis.trim()} style={{ ...btnStyle("#8b5cf6", generating || !synopsis.trim()), flex: 1 }}>
                    {generating ? "Generating..." : "Full Generate"}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
