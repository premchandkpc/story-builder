import type { GraphEdge, GraphNode } from "../api/types"

interface SceneConnectionsProps {
  sceneId: string
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export default function SceneConnections({ sceneId, nodes, edges }: SceneConnectionsProps) {
  const incoming = edges.filter((e) => e.to_node === sceneId)
  const outgoing = edges.filter((e) => e.from_node === sceneId)

  if (incoming.length === 0 && outgoing.length === 0) {
    return <span style={{ color: "var(--text-faint)", fontStyle: "italic" }}>No connections</span>
  }

  const edgeLabel = (et: string) => {
    switch (et) {
      case "seq": return "→"
      case "fork": return "↗"
      case "join": return "↙"
      case "choice": return "◇"
      default: return "→"
    }
  }

  const nodeTitle = (id: string) => nodes.find((n) => n.id === id)?.title || id.slice(0, 8)

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {incoming.length > 0 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 4, alignItems: "center" }}>
          <span style={{ color: "var(--text-faint)", fontSize: 10 }}>In:</span>
          {incoming.map((e) => (
            <span key={e.id} style={{
              display: "inline-flex", alignItems: "center", gap: 3,
              padding: "2px 6px", background: "var(--bg-warm)",
              border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
              fontSize: 10, color: "var(--text-dim)",
            }}>
              <span style={{ color: "var(--accent)", fontWeight: 600 }}>{edgeLabel(e.edge_type)}</span>
              {nodeTitle(e.from_node)}
            </span>
          ))}
        </div>
      )}
      {outgoing.length > 0 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 4, alignItems: "center" }}>
          <span style={{ color: "var(--text-faint)", fontSize: 10 }}>Out:</span>
          {outgoing.map((e) => (
            <span key={e.id} style={{
              display: "inline-flex", alignItems: "center", gap: 3,
              padding: "2px 6px", background: "var(--bg-warm)",
              border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
              fontSize: 10, color: "var(--text-dim)",
            }}>
              <span style={{ color: "var(--accent)", fontWeight: 600 }}>{edgeLabel(e.edge_type)}</span>
              {nodeTitle(e.to_node)}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
