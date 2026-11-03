import { labelStyle, ghostBtnStyle, destructiveBtnStyle } from "../api/types"

interface SceneEditorForm {
  beat_intent: string
  pov: string
  tone: string
  target_words: number
}

interface SceneEditorPanelProps {
  form: SceneEditorForm
  onFormChange: (form: SceneEditorForm) => void
  onSave: () => void
  onGenerate: () => void
  onDelete: () => void
  onClose: () => void
  confirmingGenerate: boolean
  setConfirmingGenerate: (v: boolean) => void
}

const panelInputStyle: React.CSSProperties = {
  display: "block",
  width: "100%",
  marginTop: 4,
  padding: "7px 10px",
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius-sm)",
  color: "var(--text)",
  fontSize: 12,
  fontFamily: "var(--font-body)",
  boxSizing: "border-box",
  boxShadow: "var(--shadow-inner)",
  transition: "border-color var(--transition-fast), box-shadow var(--transition-fast)",
}

const panelBtnStyle: React.CSSProperties = {
  flex: 1,
  padding: "8px 14px",
  background: "var(--success)",
  color: "#1a1512",
  border: "none",
  borderRadius: "var(--radius-md)",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 12,
  letterSpacing: "0.02em",
  boxShadow: "0 1px 2px rgba(0,0,0,0.2)",
  transition: "background var(--transition-base), box-shadow var(--transition-base), transform var(--transition-fast)",
}

const closeSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M4 4l8 8M12 4l-8 8" />
  </svg>
)

const trashSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M2 4h12M5 4V2.5A.5.5 0 015.5 2h5a.5.5 0 01.5.5V4M13 4v9.5a1.5 1.5 0 01-1.5 1.5h-7A1.5 1.5 0 013 13.5V4" />
  </svg>
)

export default function SceneEditorPanel({
  form, onFormChange, onSave, onGenerate, onDelete, onClose,
  confirmingGenerate, setConfirmingGenerate,
}: SceneEditorPanelProps) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div>
        <label style={labelStyle}>Beat Intent</label>
        <input
          value={form.beat_intent}
          onChange={(e) => onFormChange({ ...form, beat_intent: e.target.value })}
          style={panelInputStyle}
          placeholder="Narrative purpose of this scene"
        />
      </div>

      <div>
        <label style={labelStyle}>POV</label>
        <select
          value={form.pov}
          onChange={(e) => onFormChange({ ...form, pov: e.target.value })}
          style={panelInputStyle}
        >
          <option value="first-person">First person</option>
          <option value="third-person">Third person</option>
          <option value="omniscient">Omniscient</option>
        </select>
      </div>

      <div>
        <label style={labelStyle}>Tone</label>
        <select
          value={form.tone}
          onChange={(e) => onFormChange({ ...form, tone: e.target.value })}
          style={panelInputStyle}
        >
          <option value="neutral">Neutral</option>
          <option value="tense">Tense</option>
          <option value="melancholy">Melancholy</option>
          <option value="humorous">Humorous</option>
          <option value="dramatic">Dramatic</option>
        </select>
      </div>

      <div>
        <label style={labelStyle}>Target Words</label>
        <input
          type="number"
          value={form.target_words}
          onChange={(e) => onFormChange({ ...form, target_words: +e.target.value })}
          style={panelInputStyle}
          min={50}
          max={5000}
        />
      </div>

      <div style={{
        display: "flex", gap: 8, marginTop: 4,
        paddingTop: 12, borderTop: "1px solid var(--border)",
      }}>
        <button
          onClick={onSave}
          className="btn-press"
          style={{ ...panelBtnStyle, flex: 2 }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = "#7ba06c"
            e.currentTarget.style.boxShadow = "0 2px 6px rgba(107,143,94,0.3)"
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = "var(--success)"
            e.currentTarget.style.boxShadow = "0 1px 2px rgba(0,0,0,0.2)"
          }}
        >
          Save
        </button>
        {confirmingGenerate ? (
          <div style={{ display: "flex", gap: 4, flex: 3 }}>
            <button
              onClick={onGenerate}
              className="btn-press"
              style={{ ...panelBtnStyle, background: "var(--error)", flex: 1 }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = "#c06c60"
                e.currentTarget.style.boxShadow = "0 2px 6px rgba(176,92,80,0.3)"
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "var(--error)"
                e.currentTarget.style.boxShadow = "0 1px 2px rgba(0,0,0,0.2)"
              }}
            >
              Confirm
            </button>
            <button
              onClick={() => setConfirmingGenerate(false)}
              className="btn-press"
              style={{ ...panelBtnStyle, background: "var(--surface)", color: "var(--text-dim)", flex: 1 }}
              onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-hover)"}
              onMouseLeave={(e) => e.currentTarget.style.background = "var(--surface)"}
            >
              Cancel
            </button>
          </div>
        ) : (
          <button
            onClick={() => setConfirmingGenerate(true)}
            className="btn-press"
            style={{ ...panelBtnStyle, background: "var(--warn)", flex: 3 }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = "#d9865f"
              e.currentTarget.style.boxShadow = "0 2px 6px rgba(201,115,74,0.3)"
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "var(--warn)"
              e.currentTarget.style.boxShadow = "0 1px 2px rgba(0,0,0,0.2)"
            }}
          >
            Generate
          </button>
        )}
      </div>

      <div style={{ display: "flex", gap: 6, marginTop: 2 }}>
        <button
          onClick={onClose}
          className="btn-press"
          style={{
            ...ghostBtnStyle, flex: 1, justifyContent: "center",
            border: "1px solid var(--border)", fontSize: 11,
            padding: "6px 8px",
          }}
        >
          {closeSvg} Close
        </button>
        <button
          onClick={onDelete}
          className="btn-press"
          style={{ ...destructiveBtnStyle, flex: 1, justifyContent: "center", fontSize: 11 }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
          title="Delete scene (Delete key)"
        >
          {trashSvg} Delete
        </button>
      </div>
    </div>
  )
}
