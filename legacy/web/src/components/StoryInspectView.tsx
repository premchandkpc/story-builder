import { useState } from "react"
import TimelineView from "./TimelineView"
import BiblePanel from "./BiblePanel"
import CharacterListPanel from "./CharacterListPanel"
import LlmMetricsDashboard from "./LlmMetricsDashboard"
import CriticScoreDashboard from "./CriticScoreDashboard"

interface StoryInspectViewProps {
  storyId: string
}

const tabs = [
  { key: "timeline", label: "Timeline" },
  { key: "bible", label: "Bible" },
  { key: "characters", label: "Characters" },
  { key: "metrics", label: "Metrics" },
  { key: "critic", label: "Critic" },
] as const

type InspectTab = (typeof tabs)[number]["key"]

const tabBtnStyle = (active: boolean): React.CSSProperties => ({
  flex: 1, padding: "8px 0", background: "none", border: "none",
  borderBottom: active ? "1.5px solid var(--accent)" : "1.5px solid transparent",
  color: active ? "var(--accent)" : "var(--text-dim)",
  cursor: "pointer", fontWeight: active ? 600 : 400,
  fontSize: 10, textTransform: "uppercase", letterSpacing: "0.06em",
  transition: "color 0.15s, border-color 0.15s",
})

export default function StoryInspectView({ storyId }: StoryInspectViewProps) {
  const [activeTab, setActiveTab] = useState<InspectTab>("timeline")

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <div style={{ display: "flex", borderBottom: "1px solid var(--border)", padding: "0 20px" }}>
        {tabs.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className="btn-press"
            style={tabBtnStyle(activeTab === key)}
          >
            {label}
          </button>
        ))}
      </div>
      <div style={{ flex: 1, overflowY: "auto", padding: 20 }}>
        {activeTab === "timeline" && <TimelineView storyId={storyId} />}
        {activeTab === "bible" && <BiblePanel storyId={storyId} />}
        {activeTab === "characters" && <CharacterListPanel storyId={storyId} />}
        {activeTab === "metrics" && <LlmMetricsDashboard storyId={storyId} />}
        {activeTab === "critic" && <CriticScoreDashboard storyId={storyId} />}
      </div>
    </div>
  )
}
