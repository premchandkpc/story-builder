// ---- TopBar ----
// The top navigation bar showing the app name, search input, and Home button.

// useNavigate: React Router hook for programmatic navigation
import { useNavigate } from "react-router-dom"
import { inputStyle, btnStyle } from "../api/types"

// ---- Props interface ----
interface TopBarProps {
  searchQuery: string          // current search text (controlled by parent)
  onSearchChange: (q: string) => void  // called when user types in the search box
  hasActiveStory: boolean      // whether a story is currently being viewed
}

// ---- Component ----
export default function TopBar({ searchQuery, onSearchChange, hasActiveStory }: TopBarProps) {
  const navigate = useNavigate()

  return (
    <div style={{
      display: "flex",
      alignItems: "center",
      padding: "8px 16px",
      background: "#1e293b",        // slightly lighter than page background
      borderBottom: "1px solid #334155",
      gap: 12,
      flexShrink: 0,                // don't shrink when space is tight
    }}>
      {/* App title — clicking navigates to home */}
      <span
        style={{
          fontSize: 18,
          fontWeight: 700,
          color: "#f8fafc",
          marginRight: 8,
          cursor: "pointer",
        }}
        onClick={() => navigate("/")}
      >
        Story Builder
      </span>

      {/* Search input — wrapped in a relative div for the search icon positioning */}
      <div style={{ position: "relative", flex: "0 1 300px" }}>
        <input
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)} // controlled input
          placeholder="Search stories..."
          style={{ ...inputStyle, paddingLeft: 32, fontSize: 13 }}
          // spread inputStyle, then override paddingLeft for the icon
        />
        {/* Search icon (magnifying glass emoji) — absolutely positioned inside input */}
        <span style={{
          position: "absolute",
          left: 10,
          top: "50%",
          transform: "translateY(-50%)", // vertical centering trick
          color: "#64748b",
          fontSize: 13,
        }}>
          🔍
        </span>
      </div>

      {/* Spacer — pushes the Home button to the right */}
      <div style={{ flex: 1 }} />

      {/* Home button — only shown when viewing a story (not on home page) */}
      {hasActiveStory && (
        <button onClick={() => navigate("/")} style={btnStyle("#475569")}>Home</button>
      )}
    </div>
  )
}
