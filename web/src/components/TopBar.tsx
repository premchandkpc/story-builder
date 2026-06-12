import { useNavigate } from "react-router-dom"
import { inputStyle, btnStyle } from "../api/types"

interface TopBarProps {
  searchQuery: string
  onSearchChange: (q: string) => void
  hasActiveStory: boolean
}

export default function TopBar({ searchQuery, onSearchChange, hasActiveStory }: TopBarProps) {
  const navigate = useNavigate()

  return (
    <div style={{ display: "flex", alignItems: "center", padding: "8px 16px", background: "#1e293b", borderBottom: "1px solid #334155", gap: 12, flexShrink: 0 }}>
      <span style={{ fontSize: 18, fontWeight: 700, color: "#f8fafc", marginRight: 8, cursor: "pointer" }} onClick={() => navigate("/")}>
        Story Builder
      </span>
      <div style={{ position: "relative", flex: "0 1 300px" }}>
        <input
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search stories..."
          style={{ ...inputStyle, paddingLeft: 32, fontSize: 13 }}
        />
        <span style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "#64748b", fontSize: 13 }}>🔍</span>
      </div>
      <div style={{ flex: 1 }} />
      {hasActiveStory && (
        <button onClick={() => navigate("/")} style={btnStyle("#475569")}>Home</button>
      )}
    </div>
  )
}
