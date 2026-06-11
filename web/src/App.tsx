import { useState, useEffect } from "react"
import StoryGraph from "./components/StoryGraph"
import { api } from "./api/client"
import type { Story } from "./api/types"

export default function App() {
  const [stories, setStories] = useState<Story[]>([])
  const [activeStoryId, setActiveStoryId] = useState<string | null>(null)
  const [newTitle, setNewTitle] = useState("")
  const [synopsis, setSynopsis] = useState("")
  const [generating, setGenerating] = useState(false)

  useEffect(() => {
    api.stories.list().then(setStories).catch(console.error)
  }, [])

  const createStory = async () => {
    if (!newTitle.trim()) return
    try {
      const story = await api.stories.create({ title: newTitle })
      setStories([...stories, story])
      setActiveStoryId(story.id)
      setNewTitle("")
    } catch (err) {
      console.error(err)
    }
  }

  const generateStory = async () => {
    if (!synopsis.trim()) return
    setGenerating(true)
    try {
      await api.stories.generate({ synopsis })
      setSynopsis("")
      alert("Story generation started (async). Refresh the list in a moment.")
      setTimeout(async () => {
        const list = await api.stories.list()
        setStories(list)
        setGenerating(false)
      }, 3000)
    } catch (err) {
      console.error(err)
      setGenerating(false)
    }
  }

  if (activeStoryId) {
    return <StoryGraph storyId={activeStoryId} />
  }

  return (
    <div
      style={{
        minHeight: "100vh",
        background: "#0f172a",
        color: "#e2e8f0",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        fontFamily: "system-ui, sans-serif",
        gap: 24,
      }}
    >
      <h1 style={{ fontSize: 28, fontWeight: 700 }}>Story Builder</h1>
      <div style={{ display: "flex", flexDirection: "column", gap: 12, width: 360 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <h3 style={{ fontSize: 14, color: "#94a3b8", margin: 0 }}>Generate from Synopsis</h3>
          <textarea
            value={synopsis}
            onChange={(e) => setSynopsis(e.target.value)}
            placeholder="Describe the story you want to generate..."
            rows={4}
            style={{
              width: "100%",
              padding: "10px 12px",
              background: "#1e293b",
              border: "1px solid #334155",
              borderRadius: 6,
              color: "#e2e8f0",
              fontSize: 14,
              resize: "vertical",
              boxSizing: "border-box",
            }}
          />
          <button
            onClick={generateStory}
            disabled={generating}
            style={{
              padding: "10px 16px",
              background: generating ? "#64748b" : "#8b5cf6",
              color: "#fff",
              border: "none",
              borderRadius: 6,
              cursor: generating ? "not-allowed" : "pointer",
              fontWeight: 600,
            }}
          >
            {generating ? "Generating..." : "Generate Story"}
          </button>
        </div>
        <hr style={{ border: "none", borderTop: "1px solid #334155" }} />
        <div style={{ display: "flex", gap: 8 }}>
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Story title..."
            style={{
              flex: 1,
              padding: "10px 12px",
              background: "#1e293b",
              border: "1px solid #334155",
              borderRadius: 6,
              color: "#e2e8f0",
              fontSize: 14,
            }}
            onKeyDown={(e) => e.key === "Enter" && createStory()}
          />
          <button
            onClick={createStory}
            style={{
              padding: "10px 16px",
              background: "#3b82f6",
              color: "#fff",
              border: "none",
              borderRadius: 6,
              cursor: "pointer",
              fontWeight: 600,
            }}
          >
            Create
          </button>
        </div>
        {stories.length > 0 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            <h3 style={{ fontSize: 14, color: "#94a3b8", margin: 0 }}>Stories</h3>
            {stories.map((s) => (
              <button
                key={s.id}
                onClick={() => setActiveStoryId(s.id)}
                style={{
                  padding: "10px 12px",
                  background: "#1e293b",
                  border: "1px solid #334155",
                  borderRadius: 6,
                  color: "#e2e8f0",
                  cursor: "pointer",
                  textAlign: "left",
                  fontSize: 14,
                }}
              >
                {s.title}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
