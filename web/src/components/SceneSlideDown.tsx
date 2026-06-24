import { useState, useCallback, type ReactNode } from "react"

interface SceneSlideDownProps {
  label: string
  defaultOpen?: boolean
  children: ReactNode
  badge?: string | number
}

const chevronStyle: React.CSSProperties = {
  display: "inline-block", transition: "transform 0.15s", marginRight: 6, fontSize: 10,
}

export default function SceneSlideDown({ label, defaultOpen = false, children, badge }: SceneSlideDownProps) {
  const [open, setOpen] = useState(defaultOpen)
  const toggle = useCallback(() => setOpen((v) => !v), [])

  return (
    <div style={{ borderTop: "1px solid var(--border)", marginTop: 0 }}>
      <button
        onClick={toggle}
        className="btn-press"
        style={{
          width: "100%", display: "flex", alignItems: "center", gap: 6,
          padding: "8px 0", background: "none", border: "none",
          cursor: "pointer", color: "var(--text-dim)", fontSize: 11,
          letterSpacing: "0.03em", textTransform: "uppercase",
          fontFamily: "var(--font-heading)",
        }}
      >
        <span style={{ ...chevronStyle, transform: open ? "rotate(90deg)" : "rotate(0deg)" }}>
          ▸
        </span>
        {label}
        {badge != null && (
          <span style={{
            marginLeft: "auto", background: "var(--accent-dim)", color: "var(--accent)",
            padding: "1px 6px", borderRadius: 8, fontSize: 9, fontWeight: 600,
          }}>
            {badge}
          </span>
        )}
      </button>
      {open && (
        <div style={{ padding: "4px 0 8px 16px", fontSize: 12, lineHeight: 1.6 }}>
          {children}
        </div>
      )}
    </div>
  )
}
