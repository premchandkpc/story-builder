import { useState, useEffect } from "react"
import StoryGraph from "./components/StoryGraph"
import { api } from "./api/client"
import type { Story } from "./api/types"

export default function App() {
  const [stories, setStories] = useState<Story[]>([])
  const [activeStoryId, setActiveStoryId] = useState<string | null>(null)
  const [newTitle, setNewTitle] = useState("")

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
      <div style={{ display: "flex", flexDirection: "column", gap: 12, width: 320 }}>
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
