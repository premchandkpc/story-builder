import { useState } from "react"
import { useCharacters, useMigrateCharacter, useStories } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface CharacterListPanelProps {
  storyId: string
}

export default function CharacterListPanel({ storyId }: CharacterListPanelProps) {
  const { data: characters, isLoading } = useCharacters(storyId)
  const { data: stories } = useStories()
  const migrateMutation = useMigrateCharacter(storyId)
  const [targetStoryId, setTargetStoryId] = useState("")
  const [migratingCharId, setMigratingCharId] = useState<string | null>(null)

  const otherStories = (stories || []).filter((s) => s.id !== storyId)

  if (isLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 16 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  if (!characters || characters.length === 0) {
    return (
      <div style={{ padding: "12px 0", color: "var(--text-faint)", fontSize: 12, fontStyle: "italic" }}>
        No characters in this story yet.
      </div>
    )
  }

  const handleMigrate = async (charId: string) => {
    if (!targetStoryId) return
    setMigratingCharId(charId)
    try {
      await migrateMutation.mutateAsync(charId)
      setTargetStoryId("")
    } finally {
      setMigratingCharId(null)
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, ...fadeInStyle }}>
      <h3 style={{
        margin: 0, fontSize: 14,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
        fontWeight: 600, letterSpacing: "0.02em",
      }}>
        Characters
      </h3>

      {characters.map((c) => (
        <div key={c.id} style={{
          padding: 8, background: "var(--surface)",
          border: "1px solid var(--border)", borderRadius: "var(--radius-md)",
          fontSize: 12,
        }}>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>{c.name}</div>
          {c.persona && (
            <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 6 }}>
              {c.persona}
            </div>
          )}
          {otherStories.length > 0 && (
            <div style={{ display: "flex", gap: 4, alignItems: "center", marginTop: 4 }}>
              <select
                value={migratingCharId === c.id ? targetStoryId : ""}
                onChange={(e) => {
                  if (migratingCharId === c.id) setTargetStoryId(e.target.value)
                  else setTargetStoryId(e.target.value)
                }}
                onFocus={() => {
                  if (migratingCharId !== c.id) setMigratingCharId(c.id)
                }}
                style={{
                  flex: 1, padding: "3px 6px", fontSize: 10,
                  background: "var(--bg)", color: "var(--text)",
                  border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
                }}
              >
                <option value="">Migrate to...</option>
                {otherStories.map((s) => (
                  <option key={s.id} value={s.id}>{s.title}</option>
                ))}
              </select>
              {migratingCharId === c.id && targetStoryId && (
                <button
                  onClick={() => handleMigrate(c.id)}
                  disabled={migrateMutation.isPending}
                  style={{
                    padding: "3px 8px", fontSize: 10,
                    background: migrateMutation.isPending ? "var(--text-dim)" : "var(--accent)",
                    color: "#1a1512",
                    border: "none", borderRadius: "var(--radius-sm)",
                    cursor: migrateMutation.isPending ? "default" : "pointer",
                    fontWeight: 600, whiteSpace: "nowrap",
                  }}
                >
                  {migrateMutation.isPending ? "..." : "Go"}
                </button>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
