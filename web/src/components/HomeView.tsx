import { useState } from "react"
import { inputStyle, btnStyle, spinnerStyle, slideUpStyle } from "../api/types"
import { useCreateStory, useGenerateTitle, useGenerateStory } from "../api/hooks"

const wandSvg = (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M15 4V2M15 16v-2M8 9h-2M20 9h-2M17 6l-9 9" />
    <path d="M20 17l-9 9" transform="rotate(45 15.5 15.5)" />
  </svg>
)

const sparkleSvg = (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5z" />
  </svg>
)

export default function HomeView() {
  const [newTitle, setNewTitle] = useState("")
  const [synopsis, setSynopsis] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [successMsg, setSuccessMsg] = useState("")

  const createStoryMut = useCreateStory()
  const generateTitleMut = useGenerateTitle()
  const generateStoryMut = useGenerateStory()

  const handleGenerateTitle = async () => {
    if (!synopsis.trim()) return
    setError(null)
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
    setError(null)
    setSuccessMsg("")
    createStoryMut.mutate(t)
  }

  const handleGenerateStory = async () => {
    if (!synopsis.trim()) return
    setError(null)
    setSuccessMsg("")
    try {
      await generateStoryMut.mutateAsync(synopsis)
      setSynopsis("")
      setSuccessMsg("Story generation started! Redirecting...")
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
      gap: 32,
      padding: 60,
      animation: "fadeIn 0.3s var(--ease-out)",
    }}>
      {(error || successMsg) && (
        <div style={{
          ...slideUpStyle,
          background: error ? "rgba(212, 103, 103, 0.12)" : "rgba(107, 143, 94, 0.12)",
          color: error ? "var(--error)" : "var(--success)",
          border: `1px solid ${error ? "var(--error)" : "var(--success)"}`,
          padding: "10px 18px",
          borderRadius: 8,
          fontSize: 13,
          display: "flex",
          alignItems: "center",
          gap: 12,
        }}>
          <span style={{
            width: 6, height: 6, borderRadius: "50%",
            background: error ? "var(--error)" : "var(--success)",
            flexShrink: 0,
          }} />
          {error || successMsg}
          <button onClick={() => { setError(null); setSuccessMsg("") }} style={{
            background: "none", border: "none",
            color: error ? "var(--error)" : "var(--success)",
            cursor: "pointer", fontSize: 18, lineHeight: 1, padding: 0, marginLeft: "auto",
          }}>×</button>
        </div>
      )}

      <div style={{ textAlign: "center", animation: "slideUp 0.4s var(--ease-out)" }}>
        <div style={{
          fontFamily: "var(--font-heading)",
          fontSize: 36,
          fontWeight: 700,
          color: "var(--accent)",
          letterSpacing: "-0.02em",
          marginBottom: 8,
        }}>
          Story Builder
        </div>
        <div style={{
          width: 48,
          height: 3,
          background: "linear-gradient(90deg, transparent, var(--accent), transparent)",
          borderRadius: 2,
          margin: "0 auto",
          opacity: 0.6,
        }} />
        <p style={{
          color: "var(--text-dim)",
          fontSize: 14,
          marginTop: 16,
          marginBottom: 0,
          lineHeight: 1.5,
          maxWidth: 420,
        }}>
          Create branching narratives with AI-powered scene generation
        </p>
      </div>

      <div style={{
        ...slideUpStyle,
        background: "var(--surface)",
        border: "1px solid var(--border)",
        borderRadius: 12,
        padding: 32,
        width: 500,
        display: "flex",
        flexDirection: "column",
        gap: 16,
        transition: "border-color 0.2s, box-shadow 0.2s",
        boxShadow: "0 4px 24px rgba(0,0,0,0.2)",
      }}
        onMouseEnter={(e) => { e.currentTarget.style.borderColor = "var(--accent)"; e.currentTarget.style.boxShadow = "0 4px 32px rgba(0,0,0,0.3)" }}
        onMouseLeave={(e) => { e.currentTarget.style.borderColor = "var(--border)"; e.currentTarget.style.boxShadow = "0 4px 24px rgba(0,0,0,0.2)" }}
      >
        <div style={{
          fontSize: 11,
          color: "var(--text-dim)",
          fontWeight: 600,
          textTransform: "uppercase",
          letterSpacing: "0.08em",
        }}>
          New Story
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
            style={{
              ...btnStyle("var(--secondary)", !synopsis.trim() || generateTitleMut.isPending),
              width: 40, display: "flex", alignItems: "center", justifyContent: "center",
              flexShrink: 0,
            }}
            title="Generate title from synopsis"
          >
            {generateTitleMut.isPending ? <div style={spinnerStyle} /> : wandSvg}
          </button>
        </div>

        <textarea
          value={synopsis}
          onChange={(e) => setSynopsis(e.target.value)}
          placeholder="Describe the story you want to generate...&#10;&#10;A detective in a rain-soaked city uncovers a conspiracy that connects a series of seemingly unrelated murders."
          rows={5}
          style={{ ...inputStyle, resize: "vertical", lineHeight: 1.6, padding: "12px" }}
        />

        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={() => handleCreate()}
            style={{ ...btnStyle("var(--accent)", !newTitle.trim() && !synopsis.trim()), flex: 1, display: "flex", alignItems: "center", justifyContent: "center", gap: 6 }}
            disabled={!newTitle.trim() && !synopsis.trim()}
          >
            {createStoryMut.isPending ? <div style={spinnerStyle} /> : sparkleSvg}
            Create{!newTitle.trim() && synopsis.trim() ? " (auto-title)" : ""}
          </button>
          <button
            onClick={handleGenerateStory}
            disabled={generateStoryMut.isPending || !synopsis.trim()}
            style={{ ...btnStyle("var(--secondary)", generateStoryMut.isPending || !synopsis.trim()), flex: 1, display: "flex", alignItems: "center", justifyContent: "center", gap: 6 }}
          >
            {generateStoryMut.isPending ? <div style={spinnerStyle} /> : null}
            {generateStoryMut.isPending ? "Generating..." : "Full Generate"}
          </button>
        </div>
      </div>
    </div>
  )
}
