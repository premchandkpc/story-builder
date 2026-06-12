import { useState } from "react"
import { inputStyle, btnStyle } from "../api/types"
import { useCreateStory, useGenerateTitle, useGenerateStory } from "../api/hooks"

export default function HomeView() {
  const [newTitle, setNewTitle] = useState("")
  const [synopsis, setSynopsis] = useState("")
  const [error, setError] = useState<string | null>(null)

  const createStoryMut = useCreateStory()
  const generateTitleMut = useGenerateTitle()
  const generateStoryMut = useGenerateStory()

  const handleGenerateTitle = async () => {
    if (!synopsis.trim()) return
    try {
      const res = await generateTitleMut.mutateAsync(synopsis)
      setNewTitle(res.title)
    } catch {
      setError("Failed to generate title")
    }
  }

  const handleCreate = (title?: string) => {
    const t = (title || newTitle || synopsis.trim().slice(0, 50)).trim()
    if (!t) return
    createStoryMut.mutate(t)
  }

  const handleGenerateStory = async () => {
    if (!synopsis.trim()) return
    try {
      await generateStoryMut.mutateAsync(synopsis)
      setSynopsis("")
      setError("Story generation started (async). Refresh in a moment.")
    } catch {
      setError("Failed to start generation")
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", minHeight: "100%", gap: 20, padding: 40 }}>
      {error && (
        <div style={{ background: "#fdd", color: "#c00", padding: "8px 16px", borderRadius: 4 }}>
          {error}
          <button onClick={() => setError(null)} style={{ background: "none", border: "none", color: "#c00", cursor: "pointer", fontSize: 18, marginLeft: 8 }}>×</button>
        </div>
      )}
      <div style={{ fontSize: 22, fontWeight: 700 }}>Story Builder</div>
      <div style={{ background: "#1e293b", border: "1px solid #334155", borderRadius: 8, padding: 24, width: 460, display: "flex", flexDirection: "column", gap: 12 }}>
        <div style={{ fontSize: 14, color: "#94a3b8" }}>Create Story</div>
        <div style={{ display: "flex", gap: 8 }}>
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Story title (or generate from synopsis)"
            style={{ ...inputStyle, flex: 1 }}
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          />
          <button
            onClick={handleGenerateTitle}
            disabled={!synopsis.trim() || generateTitleMut.isPending}
            style={btnStyle("#8b5cf6", !synopsis.trim() || generateTitleMut.isPending)}
            title="Generate title from synopsis"
          >
            {generateTitleMut.isPending ? "..." : "✨"}
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
            onClick={() => handleCreate()}
            style={{ ...btnStyle("#3b82f6", !newTitle.trim() && !synopsis.trim()), flex: 1 }}
            disabled={!newTitle.trim() && !synopsis.trim()}
          >
            Create{!newTitle.trim() && synopsis.trim() ? " (auto-title)" : ""}
          </button>
          <button
            onClick={handleGenerateStory}
            disabled={generateStoryMut.isPending || !synopsis.trim()}
            style={{ ...btnStyle("#8b5cf6", generateStoryMut.isPending || !synopsis.trim()), flex: 1 }}
          >
            {generateStoryMut.isPending ? "Generating..." : "Full Generate"}
          </button>
        </div>
      </div>
    </div>
  )
}
