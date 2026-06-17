// ---- HomeView ----
// The home/landing page. Shows a form to create stories or generate them via LLM.
// Users can:
//   1. Type a title and click "Create" for a blank story
//   2. Write a synopsis and click "✨" to generate a title
//   3. Write a synopsis and click "Full Generate" to generate an entire story

// useState: a React hook that lets components remember values between renders.
//   - Returns [value, setValue] pair
//   - When setValue is called, the component re-renders with the new value
import { useState } from "react"
import { inputStyle, btnStyle } from "../api/types"
import { useCreateStory, useGenerateTitle, useGenerateStory } from "../api/hooks"

// ---- Component ----
export default function HomeView() {
  // ---- State variables ----
  // newTitle: the text typed into the "Story title" input field
  const [newTitle, setNewTitle] = useState("")
  // synopsis: the text typed into the textarea (story description for AI)
  const [synopsis, setSynopsis] = useState("")
  // error: nullable string — non-null means an error banner is shown
  const [error, setError] = useState<string | null>(null)

  // ---- Mutations (React Query) ----
  // These are "mutation" hooks that send data to the server.
  // Each returns { mutate, mutateAsync, isPending, isError, ... }
  const createStoryMut = useCreateStory()       // creates a story and navigates
  const generateTitleMut = useGenerateTitle()   // generates AI title from synopsis
  const generateStoryMut = useGenerateStory()   // generates full story via LLM

  // ---- Handlers ----

  // handleGenerateTitle: sends synopsis to LLM for title generation
  const handleGenerateTitle = async () => {
    if (!synopsis.trim()) return                        // skip if empty
    try {
      // mutateAsync returns a promise (vs mutate which is fire-and-forget)
      const res = await generateTitleMut.mutateAsync(synopsis)
      setNewTitle(res.title)                            // fill the title field with AI result
    } catch {
      setError("Failed to generate title")              // show error message
    }
  }

  // handleCreate: creates a new story with given or auto-generated title
  const handleCreate = (title?: string) => {
    // Priority: explicit title > newTitle state > first 50 chars of synopsis
    const t = (title || newTitle || synopsis.trim().slice(0, 50)).trim()
    if (!t) return               // still empty — do nothing
    createStoryMut.mutate(t)   // fire the mutation (React Query handles the API call)
  }

  // handleGenerateStory: kicks off full AI story generation
  const handleGenerateStory = async () => {
    if (!synopsis.trim()) return
    try {
      await generateStoryMut.mutateAsync(synopsis)
      setSynopsis("")                               // clear the textarea
      setError("Story generation started (async). Refresh in a moment.")
    } catch {
      setError("Failed to start generation")
    }
  }

  // ---- Render ----
  return (
    // Center everything vertically and horizontally
    <div style={{
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      minHeight: "100%",
      gap: 20,
      padding: 40,
    }}>
      {/* Error banner — only shown when error is not null */}
      {error && (
        <div style={{ background: "#fdd", color: "#c00", padding: "8px 16px", borderRadius: 4 }}>
          {error}
          {/*
            "×" button dismisses the error by setting it back to null.
            Inline styles for a minimal close button.
          */}
          <button onClick={() => setError(null)} style={{
            background: "none", border: "none", color: "#c00",
            cursor: "pointer", fontSize: 18, marginLeft: 8
          }}>×</button>
        </div>
      )}

      {/* Page heading */}
      <div style={{ fontSize: 22, fontWeight: 700 }}>Story Builder</div>

      {/* Card container for the form */}
      <div style={{
        background: "#1e293b",
        border: "1px solid #334155",
        borderRadius: 8,
        padding: 24,
        width: 460,
        display: "flex",
        flexDirection: "column",
        gap: 12,
      }}>
        {/* Section label */}
        <div style={{ fontSize: 14, color: "#94a3b8" }}>Create Story</div>

        {/* Row: title input + generate-title button */}
        <div style={{ display: "flex", gap: 8 }}>
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Story title (or generate from synopsis)"
            style={{ ...inputStyle, flex: 1 }}
            // Press Enter to create story
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          />
          <button
            onClick={handleGenerateTitle}
            disabled={!synopsis.trim() || generateTitleMut.isPending}
            style={btnStyle("#8b5cf6", !synopsis.trim() || generateTitleMut.isPending)}
            title="Generate title from synopsis"
          >
            {generateTitleMut.isPending ? "..." : "✨"}
            {/* Show "..." while generating, otherwise sparkles emoji */}
          </button>
        </div>

        {/* Synopsis textarea */}
        <textarea
          value={synopsis}
          onChange={(e) => setSynopsis(e.target.value)}
          placeholder="Describe the story you want to generate..."
          rows={4}
          style={{ ...inputStyle, resize: "vertical" }}
        />

        {/* Row: Create button + Full Generate button */}
        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={() => handleCreate()}
            style={{ ...btnStyle("#3b82f6", !newTitle.trim() && !synopsis.trim()), flex: 1 }}
            disabled={!newTitle.trim() && !synopsis.trim()}
          >
            {/*
              Button text changes to "(auto-title)" if no title was typed
              but a synopsis exists, hinting the first 50 chars will be used.
            */}
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
