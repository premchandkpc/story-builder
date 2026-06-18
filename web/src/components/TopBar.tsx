import { useNavigate } from "react-router-dom"
import { inputStyle, btnStyle } from "../api/types"

interface TopBarProps {
  searchQuery: string
  onSearchChange: (q: string) => void
  hasActiveStory: boolean
}

const quillSvg = (
  <svg width="20" height="20" viewBox="0 0 48 48" fill="none" style={{ verticalAlign: "middle", marginRight: 8 }}>
    <path fill="#d4a853" d="M29.5 4.5c-2.5 2-5.5 6.5-6 10-.5 3.5 1 6 3 7.5s5 2 7 .5c2-1.5 3.5-5 3.5-8s-1.5-6-3.5-8c-1-.75-2.5-2-4-2z"/>
    <path fill="#c9734a" d="M23.5 11.5c-2 2.5-3.5 6-3.5 9 0 3 1.5 5.5 3 6.5s3 1 4.5-.5c1.5-1.5 2.5-4.5 2.5-7s-1-4.5-2.5-6c-.75-.75-2.25-2.5-4-2z"/>
    <path fill="#e8e4d8" d="M17 22c-1.5 2-2.5 5-2.5 7.5 0 2.5 1 4.5 2.5 5.5s3 .5 4-1c1-1.5 1.5-4 1.5-6s-.5-4-1.5-5c-.5-.5-2.25-2-4-1z"/>
    <path fill="#d4a853" d="M12 31c-2 1.5-3.5 4.5-3.5 7s1.5 4.5 3.5 5 4-.5 5-3c1-2.5 1-5.5 0-7.5s-2.5-2.5-5-1.5z"/>
    <path fill="#1a1a24" d="M37.5 10.5c-1 .75-1.5 2-1.5 3s.5 2 1.5 2.5c1 .5 2 0 2.5-1s.5-2.5-.5-3.5c-.5-.5-1.25-1.5-2-1z"/>
  </svg>
)

export default function TopBar({ searchQuery, onSearchChange, hasActiveStory }: TopBarProps) {
  const navigate = useNavigate()

  return (
    <div style={{
      display: "flex",
      alignItems: "center",
      padding: "10px 20px",
      background: "var(--surface)",
      borderBottom: "1px solid var(--border)",
      gap: 16,
      flexShrink: 0,
    }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          cursor: "pointer",
          userSelect: "none",
        }}
        onClick={() => navigate("/")}
      >
        {quillSvg}
        <span style={{
          fontFamily: "var(--font-heading)",
          fontSize: 20,
          fontWeight: 700,
          color: "var(--accent)",
          letterSpacing: "-0.02em",
        }}>
          Story Builder
        </span>
      </div>

      <div style={{ position: "relative", flex: "0 1 300px" }}>
        <input
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search stories..."
          style={{ ...inputStyle, paddingLeft: 32, fontSize: 13 }}
        />
        <span style={{
          position: "absolute",
          left: 10,
          top: "50%",
          transform: "translateY(-50%)",
          color: "var(--text-muted)",
          fontSize: 13,
        }}>
          &#128269;
        </span>
      </div>

      <div style={{ flex: 1 }} />

      <button onClick={() => navigate("/audit")} style={btnStyle("var(--surface-hover)")} title="Code Audit">&#128214;</button>
      {hasActiveStory && (
        <button onClick={() => navigate("/")} style={btnStyle("var(--surface-hover)")}>Home</button>
      )}
    </div>
  )
}