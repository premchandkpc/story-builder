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
    <div style={{
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      minHeight: "100%",
      gap: 24,
      padding: 40,
    }}>
      {error && (
        <div style={{
          background: "rgba(212, 103, 103, 0.15)",
          color: "var(--error)",
          border: "1px solid var(--error)",
          padding: "10px 18px",
          borderRadius: 8,
          fontSize: 13,
          display: "flex",
          alignItems: "center",
          gap: 12,
        }}>
          {error}
          <button onClick={() => setError(null)} style={{
            background: "none", border: "none", color: "var(--error)",
            cursor: "pointer", fontSize: 18, lineHeight: 1,
          }}>×</button>
        </div>
      )}

      <div style={{
        fontFamily: "var(--font-heading)",
        fontSize: 28,
        fontWeight: 700,
        color: "var(--accent)",
        letterSpacing: "-0.02em",
      }}>
        Story Builder
      </div>

      <div style={{
        background: "var(--surface)",
        border: "1px solid var(--border)",
        borderRadius: 12,
        padding: 28,
        width: 480,
        display: "flex",
        flexDirection: "column",
        gap: 14,
      }}>
        <div style={{ fontSize: 13, color: "var(--text-muted)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.08em" }}>
          Create Story
        </div>

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
            style={btnStyle("var(--secondary)", !synopsis.trim() || generateTitleMut.isPending)}
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
            style={{ ...btnStyle("var(--accent)", !newTitle.trim() && !synopsis.trim()), flex: 1 }}
            disabled={!newTitle.trim() && !synopsis.trim()}
          >
            Create{!newTitle.trim() && synopsis.trim() ? " (auto-title)" : ""}
          </button>
          <button
            onClick={handleGenerateStory}
            disabled={generateStoryMut.isPending || !synopsis.trim()}
            style={{ ...btnStyle("var(--secondary)", generateStoryMut.isPending || !synopsis.trim()), flex: 1 }}
          >
            {generateStoryMut.isPending ? "Generating..." : "Full Generate"}
          </button>
        </div>
      </div>
    </div>
  )
}