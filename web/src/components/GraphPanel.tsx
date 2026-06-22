import type { Node, Edge } from "@xyflow/react"
import type { Generation, EdgeType, SceneNodeData } from "../api/types"
import { slideUpStyle } from "../api/types"
import SceneEditorPanel from "./SceneEditorPanel"
import NodeInfoPanel from "./NodeInfoPanel"
import EdgeInfoPanel from "./EdgeInfoPanel"
import GenerationList from "./GenerationList"
import TurnTimeline from "./TurnTimeline"
import AgentRunPanel from "./AgentRunPanel"
import LlmMetricsDashboard from "./LlmMetricsDashboard"
import CriticScoreDashboard from "./CriticScoreDashboard"
import BiblePanel from "./BiblePanel"
import TimelineView from "./TimelineView"

interface GraphPanelProps {
  storyId: string
  nodes: Node<SceneNodeData>[]
  edges: Edge[]
  selectedNode: Node<SceneNodeData> | null
  selectedEdge: Edge | null
  onClose: () => void
  onAddNode: () => void
  activeTab: "edit" | "info" | "generations" | "turns" | "agents" | "critic"
  setActiveTab: (tab: "edit" | "info" | "generations" | "turns" | "agents" | "critic") => void
  form: { beat_intent: string; pov: string; tone: string; target_words: number }
  onFormChange: (form: { beat_intent: string; pov: string; tone: string; target_words: number }) => void
  confirmingGenerate: boolean
  setConfirmingGenerate: (v: boolean) => void
  onSave: () => void
  onGenerate: () => void
  onDeleteNode: () => void
  onDeleteEdge: () => void
  generations: Generation[]
  gensLoading: boolean
  onAcceptGeneration: (nodeId: string, genId: string) => void
  pendingEdgeType: EdgeType
  setPendingEdgeType: (t: EdgeType) => void
}

const addSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M8 3v10M3 8h10" />
  </svg>
)

const closeSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M4 4l8 8M12 4l-8 8" />
  </svg>
)

const edgeTypesConfig: { value: EdgeType; label: string; desc: string }[] = [
  { value: "seq", label: "Seq", desc: "Default progression" },
  { value: "fork", label: "Fork", desc: "Branch to multiple paths" },
  { value: "join", label: "Join", desc: "Converge branches" },
  { value: "choice", label: "Choice", desc: "Decision point" },
]

const tabBtnStyle = (active: boolean): React.CSSProperties => ({
  flex: 1,
  padding: "8px 0",
  background: "none",
  border: "none",
  borderBottom: active ? "1.5px solid var(--accent)" : "1.5px solid transparent",
  color: active ? "var(--accent)" : "var(--text-dim)",
  cursor: "pointer",
  fontWeight: active ? 600 : 400,
  fontSize: 10,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
  transition: "color 0.15s, border-color 0.15s",
})

export default function GraphPanel({
  storyId, nodes, edges,
  selectedNode, selectedEdge,
  onClose, onAddNode,
  activeTab, setActiveTab,
  form, onFormChange,
  confirmingGenerate, setConfirmingGenerate,
  onSave, onGenerate, onDeleteNode, onDeleteEdge,
  generations, gensLoading, onAcceptGeneration,
  pendingEdgeType, setPendingEdgeType,
}: GraphPanelProps) {
  const selectedNodeData = selectedNode?.data as SceneNodeData | undefined

  const renderPanelContent = () => {
    if (selectedNode && activeTab === "edit") {
      return (
        <SceneEditorPanel
          form={form}
          onFormChange={onFormChange}
          onSave={onSave}
          onGenerate={onGenerate}
          onDelete={onDeleteNode}
          onClose={onClose}
          confirmingGenerate={confirmingGenerate}
          setConfirmingGenerate={setConfirmingGenerate}
        />
      )
    }

    if (selectedNode && activeTab === "generations") {
      return (
        <GenerationList
          generations={generations}
          gensLoading={gensLoading}
          selectedNodeId={selectedNode.id}
          onAccept={onAcceptGeneration}
        />
      )
    }

    if (selectedNode && activeTab === "info") {
      return (
        <NodeInfoPanel
          edges={edges}
          selectedNodeId={selectedNode.id}
          status={selectedNodeData?.status || "Unknown"}
          beatIntent={selectedNodeData?.beatIntent || ""}
          onDelete={onDeleteNode}
        />
      )
    }

    if (selectedNode && activeTab === "turns") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Turn Timeline</div>
          <TurnTimeline storyId={storyId} nodeId={selectedNode.id} compact />
        </div>
      )
    }

    if (selectedNode && activeTab === "agents") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Agent Runs</div>
          <AgentRunPanel storyId={storyId} nodeId={selectedNode.id} />
        </div>
      )
    }

    if (selectedNode && activeTab === "critic") {
      return <CriticScoreDashboard storyId={storyId} />
    }

    if (selectedEdge) {
      return <EdgeInfoPanel selectedEdge={selectedEdge} onDelete={onDeleteEdge} />
    }

    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 4, flex: 1 }}>
        <div style={{ flex: 1, overflowY: "auto", display: "flex", flexDirection: "column", gap: 20 }}>
          <TimelineView storyId={storyId} />
          <BiblePanel storyId={storyId} />
          <LlmMetricsDashboard storyId={storyId} />
          <CriticScoreDashboard storyId={storyId} />
        </div>
        <div style={{
          padding: "14px 16px", textAlign: "center",
          borderTop: "1px solid var(--border)",
          lineHeight: 1.7,
        }}>
          <span style={{
            fontSize: 12, color: "var(--text-dim)", fontStyle: "italic",
            fontFamily: "var(--font-heading)",
          }}>
            Select a node to begin editing
          </span>
          <br />
          <span style={{ fontSize: 10, color: "var(--text-faint)", letterSpacing: "0.03em" }}>
            Esc to deselect  ·  Delete to remove
          </span>
        </div>
      </div>
    )
  }

  return (
    <div
      style={{
        ...slideUpStyle,
        width: 300,
        background: "var(--bg-warm)",
        borderLeft: "1px solid var(--border)",
        boxShadow: "inset 1px 0 0 rgba(212,168,83,0.04)",
        display: "flex",
        flexDirection: "column",
        color: "var(--text)",
        fontFamily: "var(--font-body)",
        fontSize: 13,
        flexShrink: 0,
      }}
    >
      <div style={{ padding: "16px 16px 0", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <h3 style={{
          margin: 0,
          fontSize: 15,
          fontFamily: "var(--font-heading)",
          color: "var(--accent)",
          fontWeight: 600,
          letterSpacing: "0.02em",
        }}>
          {selectedNode ? "Scene Editor" : selectedEdge ? "Edge Details" : "Story Graph"}
        </h3>
        {(selectedNode || selectedEdge) && (
          <button
            onClick={onClose}
            className="btn-press"
            style={{
              background: "none", border: "none", color: "var(--text-faint)",
              cursor: "pointer", padding: 4, display: "flex",
              transition: "color 0.15s",
            }}
            onMouseEnter={(e) => e.currentTarget.style.color = "var(--text-dim)"}
            onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-faint)"}
          >
            {closeSvg}
          </button>
        )}
      </div>
      <div style={{
        margin: "10px 16px 12px",
        height: 1,
        background: "linear-gradient(90deg, var(--border) 0%, rgba(212,168,83,0.18) 50%, var(--border) 100%)",
      }} />

      <div style={{ padding: "0 16px", marginBottom: 12 }}>
        <button
          onClick={onAddNode}
          className="btn-press"
          style={{
            padding: "8px 16px",
            background: "var(--accent)",
            color: "#1a1512",
            border: "none",
            borderRadius: "var(--radius-md)",
            cursor: "pointer",
            fontWeight: 600,
            fontSize: 13,
            transition: "background var(--transition-base), box-shadow var(--transition-base), transform var(--transition-fast)",
            display: "flex",
            alignItems: "center",
            gap: 6,
            width: "100%",
            justifyContent: "center",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = "var(--accent-hover)"
            e.currentTarget.style.boxShadow = "0 2px 8px rgba(212,168,83,0.3)"
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = "var(--accent)"
            e.currentTarget.style.boxShadow = "none"
          }}
        >
          {addSvg}
          Add Scene
        </button>
      </div>

      {(selectedNode || selectedEdge) && (
        <div style={{ display: "flex", borderBottom: "1px solid var(--border)", padding: "0 16px", gap: 0 }}>
          {selectedNode && (["edit", "info", "generations", "turns", "agents", "critic"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              style={tabBtnStyle(activeTab === tab)}
              className="btn-press"
              onMouseEnter={(e) => {
                if (activeTab !== tab) e.currentTarget.style.color = "var(--text)"
              }}
              onMouseLeave={(e) => {
                if (activeTab !== tab) e.currentTarget.style.color = "var(--text-dim)"
              }}
            >
              {tab === "edit" ? "Edit" : tab === "info" ? "Info" : tab === "generations" ? "Gen" : tab === "turns" ? "Turns" : tab === "agents" ? "Agents" : "Critic"}
            </button>
          ))}
        </div>
      )}

      <div style={{
        flex: 1, overflowY: "auto", padding: "14px 16px",
        display: "flex", flexDirection: "column", gap: 8,
      }}>
        {renderPanelContent()}
      </div>

      {nodes.length > 0 && (
        <div style={{
          borderTop: "1px solid var(--border)", padding: "10px 16px",
          fontSize: 10, color: "var(--text-dim)",
          letterSpacing: "0.02em",
        }}>
          <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
            <span style={{ color: "var(--text-faint)", fontWeight: 500 }}>{nodes.length} nodes · {edges.length} edges</span>
            <span style={{ display: "flex", gap: 4 }}>
              {edgeTypesConfig.map((et) => (
                <button
                  key={et.value}
                  onClick={() => setPendingEdgeType(et.value)}
                  className="btn-press"
                  style={{
                    padding: "3px 8px",
                    background: pendingEdgeType === et.value ? "var(--accent-dim)" : "transparent",
                    border: `1px solid ${pendingEdgeType === et.value ? "var(--accent)" : "var(--border)"}`,
                    borderRadius: "var(--radius-sm)",
                    color: pendingEdgeType === et.value ? "var(--accent)" : "var(--text-dim)",
                    cursor: "pointer",
                    fontSize: 10,
                    fontWeight: pendingEdgeType === et.value ? 600 : 400,
                    transition: "all 0.1s",
                    textTransform: "uppercase",
                    letterSpacing: "0.03em",
                  }}
                  title={et.desc}
                >
                  {et.label}
                </button>
              ))}
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
