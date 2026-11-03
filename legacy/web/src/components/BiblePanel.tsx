import { useState } from "react"
import { useBible, useGenerateBible, useReferencingBibles, useLinkBible, useUnlinkBible } from "../api/hooks"
import type { StoryBible } from "../api/types"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface BiblePanelProps {
  storyId: string
}

export default function BiblePanel({ storyId }: BiblePanelProps) {
  const { data: bible, isLoading, error } = useBible(storyId)
  const { data: sharedBibles } = useReferencingBibles(storyId)
  const generateBible = useGenerateBible(storyId)

  if (isLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 32 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  const sourceBible = bible || (sharedBibles && sharedBibles[0])
  const isShared = !bible && !!sourceBible

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, ...fadeInStyle }}>
      <h3 style={{
        margin: 0, fontSize: 14,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
        fontWeight: 600, letterSpacing: "0.02em",
      }}>
        Story Bible
        {isShared && (
          <span style={{
            fontSize: 10, color: "var(--text-dim)", fontWeight: 400,
            marginLeft: 8, background: "var(--surface)", padding: "2px 6px",
            borderRadius: "var(--radius-sm)", verticalAlign: "middle",
          }}>
            Shared
          </span>
        )}
      </h3>

      {error && (
        <div style={{ color: "#ef4444", fontSize: 12, fontStyle: "italic" }}>
          Failed to load bible
        </div>
      )}

      {!sourceBible && !generateBible.isPending && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12, alignItems: "center", padding: 24 }}>
          <p style={{ fontSize: 12, color: "var(--text-dim)", textAlign: "center", margin: 0, lineHeight: 1.6 }}>
            No bible yet. Generate a world bible from the story synopsis and characters.
          </p>
          <button
            onClick={() => generateBible.mutate()}
            className="btn-press"
            style={{
              padding: "8px 20px",
              background: "var(--accent)",
              color: "#1a1512",
              border: "none",
              borderRadius: "var(--radius-md)",
              cursor: "pointer",
              fontWeight: 600,
              fontSize: 13,
            }}
          >
            Generate Bible
          </button>
          {generateBible.isError && (
            <div style={{ color: "#ef4444", fontSize: 11 }}>
              {generateBible.error?.message || "Generation failed"}
            </div>
          )}
        </div>
      )}

      {generateBible.isPending && (
        <div style={{ display: "flex", alignItems: "center", gap: 8, padding: 16 }}>
          <div style={spinnerStyle} />
          <span style={{ fontSize: 12, color: "var(--text-dim)" }}>Generating bible...</span>
        </div>
      )}

      {sourceBible && (
        <>
          <div style={{ fontSize: 13, fontWeight: 600, color: "var(--text)" }}>
            {sourceBible.title || "Untitled World"}
          </div>

          {sourceBible.world && (
            <Section label="World">
              {sourceBible.world}
            </Section>
          )}

          {sourceBible.central_theme && (
            <Section label="Theme">
              {sourceBible.central_theme}
            </Section>
          )}

          {sourceBible.tone && (
            <Section label="Tone">
              {sourceBible.tone}
            </Section>
          )}

          {sourceBible.world_rules && sourceBible.world_rules.length > 0 && (
            <Section label={`Rules (${sourceBible.world_rules.length})`}>
              {sourceBible.world_rules.map((r, i) => (
                <div key={i} style={{
                  fontSize: 11, padding: "4px 8px", background: "var(--surface)",
                  borderRadius: "var(--radius-sm)", marginBottom: 4,
                  border: "1px solid var(--border)",
                }}>
                  <span style={{ color: "var(--accent)", fontWeight: 600 }}>{r.category}: </span>
                  <span style={{ color: "var(--text-dim)" }}>{r.description}</span>
                </div>
              ))}
            </Section>
          )}

          {sourceBible.factions && sourceBible.factions.length > 0 && (
            <Section label={`Factions (${sourceBible.factions.length})`}>
              {sourceBible.factions.map((f, i) => (
                <div key={i} style={{
                  fontSize: 11, color: "var(--text-dim)", padding: "4px 8px",
                }}>
                  <span style={{ color: "var(--text)", fontWeight: 600 }}>{f.name}</span>
                  {" — "}{f.goal}
                </div>
              ))}
            </Section>
          )}

          <ShareSection storyId={storyId} bible={sourceBible} isShared={isShared} />
        </>
      )}
    </div>
  )
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div style={{
        fontSize: 10, color: "var(--text-faint)", textTransform: "uppercase",
        letterSpacing: "0.05em", marginBottom: 4, fontWeight: 600,
      }}>
        {label}
      </div>
      <div style={{ fontSize: 12, color: "var(--text-dim)", lineHeight: 1.6 }}>
        {children}
      </div>
    </div>
  )
}

function ShareSection({ storyId, bible, isShared }: { storyId: string; bible: StoryBible; isShared: boolean }) {
  const [showShare, setShowShare] = useState(false)
  const generateBible = useGenerateBible(storyId)
  const linkBible = useLinkBible(storyId)
  const unlinkBible = useUnlinkBible(storyId)
  const [targetId, setTargetId] = useState("")

  const isLinked = (bible.reference_stories || []).includes(storyId)

  return (
    <div style={{ borderTop: "1px solid var(--border)", paddingTop: 8 }}>
      <button
        onClick={() => setShowShare(!showShare)}
        className="btn-press"
        style={{
          background: "none", border: "none",
          color: "var(--text-dim)", cursor: "pointer",
          fontSize: 11, padding: 0,
        }}
      >
        {showShare ? "Hide" : "Sharing"}
      </button>
      {showShare && (
        <div style={{ marginTop: 8, display: "flex", flexDirection: "column", gap: 8 }}>
          {isLinked || isShared ? (
            <>
              <div style={{ fontSize: 11, color: "var(--text-dim)" }}>
                This bible is shared with this story.
              </div>
              <button
                onClick={() => unlinkBible.mutate(storyId)}
                className="btn-press"
                disabled={unlinkBible.isPending}
                style={{
                  padding: "4px 10px",
                  background: "none",
                  border: "1px solid var(--border)",
                  borderRadius: "var(--radius-sm)",
                  color: "#ef4444",
                  cursor: "pointer",
                  fontSize: 10,
                  alignSelf: "flex-start",
                }}
              >
                {unlinkBible.isPending ? "..." : "Unlink"}
              </button>
              <button
                onClick={() => generateBible.mutate()}
                className="btn-press"
                disabled={generateBible.isPending}
                style={{
                  padding: "4px 10px",
                  background: "none",
                  border: "1px solid var(--border)",
                  borderRadius: "var(--radius-sm)",
                  color: "var(--accent)",
                  cursor: "pointer",
                  fontSize: 10,
                  alignSelf: "flex-start",
                }}
              >
                {generateBible.isPending ? "..." : "Generate Own Bible"}
              </button>
            </>
          ) : (
            <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
              <input
                value={targetId}
                onChange={(e) => setTargetId(e.target.value)}
                placeholder="Target story ID"
                style={{
                  flex: 1, padding: "4px 8px", fontSize: 11,
                  background: "var(--surface)", border: "1px solid var(--border)",
                  borderRadius: "var(--radius-sm)", color: "var(--text)",
                  outline: "none",
                }}
              />
              <button
                onClick={() => {
                  if (targetId) {
                    linkBible.mutate(targetId)
                    setTargetId("")
                  }
                }}
                disabled={!targetId || linkBible.isPending}
                className="btn-press"
                style={{
                  padding: "4px 10px",
                  background: "var(--accent)",
                  color: "#1a1512",
                  border: "none",
                  borderRadius: "var(--radius-sm)",
                  cursor: "pointer",
                  fontWeight: 600,
                  fontSize: 10,
                }}
              >
                {linkBible.isPending ? "..." : "Share"}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
