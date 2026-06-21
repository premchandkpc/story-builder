import { useNavigate } from "react-router-dom"
import { inputStyle } from "../api/types"

interface TopBarProps {
  searchQuery: string
  onSearchChange: (q: string) => void
  hasActiveStory: boolean
  onToggleSidebar: () => void
  sidebarOpen: boolean
}

function btnStyle(bg: string): Record<string, string | number> {
  return {
    padding: "8px 12px",
    background: bg,
    color: "var(--text-muted)",
    border: "1px solid var(--border)",
    borderRadius: 6,
    cursor: "pointer",
    fontSize: 13,
    transition: "color 0.15s, border-color 0.15s, background 0.15s",
    display: "flex",
    alignItems: "center",
    gap: 4,
    whiteSpace: "nowrap",
  }
}

const quillSvg = (
  <svg width="20" height="20" viewBox="0 0 48 48" fill="none" style={{ verticalAlign: "middle" }}>
    <path fill="#d4a853" d="M29.5 4.5c-2.5 2-5.5 6.5-6 10-.5 3.5 1 6 3 7.5s5 2 7 .5c2-1.5 3.5-5 3.5-8s-1.5-6-3.5-8c-1-.75-2.5-2-4-2z"/>
    <path fill="#c9734a" d="M23.5 11.5c-2 2.5-3.5 6-3.5 9 0 3 1.5 5.5 3 6.5s3 1 4.5-.5c1.5-1.5 2.5-4.5 2.5-7s-1-4.5-2.5-6c-.75-.75-2.25-2.5-4-2z"/>
    <path fill="#e8e4d8" d="M17 22c-1.5 2-2.5 5-2.5 7.5 0 2.5 1 4.5 2.5 5.5s3 .5 4-1c1-1.5 1.5-4 1.5-6s-.5-4-1.5-5c-.5-.5-2.25-2-4-1z"/>
    <path fill="#d4a853" d="M12 31c-2 1.5-3.5 4.5-3.5 7s1.5 4.5 3.5 5 4-.5 5-3c1-2.5 1-5.5 0-7.5s-2.5-2.5-5-1.5z"/>
    <path fill="#1a1a24" d="M37.5 10.5c-1 .75-1.5 2-1.5 3s.5 2 1.5 2.5c1 .5 2 0 2.5-1s.5-2.5-.5-3.5c-.5-.5-1.25-1.5-2-1z"/>
  </svg>
)

const searchSvg = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <circle cx="11" cy="11" r="8" />
    <path d="M21 21l-4.35-4.35" />
  </svg>
)

const clearSvg = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M18 6L6 18M6 6l12 12" />
  </svg>
)

const homeSvg = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
    <path d="M9 22V12h6v10" />
  </svg>
)

const auditSvg = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
    <path d="M14 2v6h6" />
    <path d="M16 13H8M16 17H8M10 9H8" />
  </svg>
)

const sidebarSvg = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M9 3v18" />
  </svg>
)

export default function TopBar({ searchQuery, onSearchChange, hasActiveStory, onToggleSidebar, sidebarOpen }: TopBarProps) {
  const navigate = useNavigate()

  return (
    <div style={{
      display: "flex",
      alignItems: "center",
      padding: "10px 20px",
      background: "var(--surface)",
      borderBottom: "1px solid var(--border)",
      gap: 12,
      flexShrink: 0,
    }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          cursor: "pointer",
          userSelect: "none",
          transition: "opacity 0.15s",
          paddingRight: 4,
        }}
        onClick={() => navigate("/")}
        onMouseEnter={(e) => e.currentTarget.style.opacity = "0.85"}
        onMouseLeave={(e) => e.currentTarget.style.opacity = "1"}
      >
        {quillSvg}
        <div style={{ display: "flex", flexDirection: "column" }}>
          <span style={{
            fontFamily: "var(--font-heading)",
            fontSize: 18,
            fontWeight: 700,
            color: "var(--accent)",
            letterSpacing: "-0.02em",
            lineHeight: 1.2,
          }}>
            Story Builder
          </span>
          <span style={{ fontSize: 9, color: "var(--text-dim)", letterSpacing: "0.06em", textTransform: "uppercase" }}>
            Narrative Editor
          </span>
        </div>
      </div>

      <button
        onClick={onToggleSidebar}
        style={btnStyle("transparent")}
        title={sidebarOpen ? "Collapse sidebar" : "Expand sidebar"}
        onMouseEnter={(e) => { e.currentTarget.style.color = "var(--accent)"; e.currentTarget.style.borderColor = "var(--accent)" }}
        onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-muted)"; e.currentTarget.style.borderColor = "var(--border)" }}
      >
        {sidebarSvg}
      </button>

      <div style={{ position: "relative", flex: "0 1 260px" }}>
        <span style={{
          position: "absolute",
          left: 10,
          top: "50%",
          transform: "translateY(-50%)",
          color: "var(--text-dim)",
          pointerEvents: "none",
          display: "flex",
        }}>
          {searchSvg}
        </span>
        <input
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search stories..."
          style={{ ...inputStyle, paddingLeft: 30, paddingRight: searchQuery ? 30 : 10, fontSize: 13 }}
        />
        {searchQuery && (
          <button
            onClick={() => onSearchChange("")}
            style={{
              position: "absolute",
              right: 6,
              top: "50%",
              transform: "translateY(-50%)",
              background: "none",
              border: "none",
              color: "var(--text-dim)",
              cursor: "pointer",
              padding: 2,
              display: "flex",
              borderRadius: 4,
              transition: "color 0.15s",
            }}
            onMouseEnter={(e) => e.currentTarget.style.color = "var(--text)"}
            onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-dim)"}
          >
            {clearSvg}
          </button>
        )}
      </div>

      <div style={{ flex: 1 }} />

      {hasActiveStory && (
        <button
          onClick={() => navigate("/")}
          style={btnStyle("transparent")}
          onMouseEnter={(e) => { e.currentTarget.style.color = "var(--accent)"; e.currentTarget.style.borderColor = "var(--accent)" }}
          onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-muted)"; e.currentTarget.style.borderColor = "var(--border)" }}
        >
          {homeSvg}
          <span style={{ fontSize: 12 }}>Home</span>
        </button>
      )}
      <button
        onClick={() => navigate("/audit")}
        style={btnStyle("transparent")}
        title="Code Audit"
        onMouseEnter={(e) => { e.currentTarget.style.color = "var(--accent)"; e.currentTarget.style.borderColor = "var(--accent)" }}
        onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-muted)"; e.currentTarget.style.borderColor = "var(--border)" }}
      >
        {auditSvg}
        <span style={{ fontSize: 12 }}>Audit</span>
      </button>
    </div>
  )
}
