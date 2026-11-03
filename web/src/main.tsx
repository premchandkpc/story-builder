import { useCallback, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ReactFlow, Background, Controls, MiniMap, type Connection, type Edge as FlowEdge, type Node, type NodeMouseHandler } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { api, type Graph, type Scene, type Story } from "./api";
import "./styles.css";

const queryClient = new QueryClient();

function App() {
  const cache = useQueryClient();
  const [selectedStoryID, setSelectedStoryID] = useState("");
  const [selectedScene, setSelectedScene] = useState<Scene | null>(null);
  const [newTitle, setNewTitle] = useState("");
  const [notice, setNotice] = useState("");
  const stories = useQuery({ queryKey: ["stories"], queryFn: api.listStories });
  const graph = useQuery({ queryKey: ["graph", selectedStoryID], queryFn: () => api.graph(selectedStoryID), enabled: Boolean(selectedStoryID) });

  useEffect(() => { if (!selectedStoryID && stories.data?.[0]) setSelectedStoryID(stories.data[0].id); }, [selectedStoryID, stories.data]);
  useEffect(() => { if (selectedScene && graph.data) setSelectedScene(graph.data.scenes.find((scene) => scene.id === selectedScene.id) ?? null); }, [graph.data, selectedScene?.id]);
  const refreshGraph = () => cache.invalidateQueries({ queryKey: ["graph", selectedStoryID] });
  const createStory = useMutation({ mutationFn: api.createStory, onSuccess: (story) => { cache.invalidateQueries({ queryKey: ["stories"] }); setSelectedStoryID(story.id); setNotice("New map opened."); }, onError: (error) => setNotice(error.message) });
  const createScene = useMutation({ mutationFn: ({ title, position }: { title: string; position: { x:number; y:number } }) => api.createScene(selectedStoryID, title, position), onSuccess: refreshGraph, onError: (error) => setNotice(error.message) });
  const moveScene = useMutation({ mutationFn: ({ id, position }: { id:string; position:{x:number;y:number} }) => api.moveScene(id, position), onError: (error) => setNotice(error.message) });
  const connect = useMutation({ mutationFn: (connection: Connection) => api.createEdge(selectedStoryID, connection.source ?? "", connection.target ?? ""), onSuccess: refreshGraph, onError: (error) => setNotice(error.message) });
  const saveScene = useMutation({ mutationFn: ({ id, title, brief }: { id:string; title:string; brief:string }) => api.updateScene(id,title,brief), onSuccess: (scene) => { refreshGraph(); setSelectedScene(scene); setNotice("Scene notes saved."); }, onError: (error) => setNotice(error.message) });

  const nodes = useMemo<Node[]>(() => (graph.data?.scenes ?? []).map((scene) => ({ id: scene.id, position: scene.position, data: { label: <><span className="node-kicker">SCENE</span><strong>{scene.title}</strong><small>{scene.brief || "Add a scene brief"}</small></> }, className: selectedScene?.id === scene.id ? "scene-node is-selected" : "scene-node" })), [graph.data, selectedScene?.id]);
  const edges = useMemo<FlowEdge[]>(() => (graph.data?.edges ?? []).map((edge) => ({ id: edge.id, source: edge.fromSceneId, target: edge.toSceneId, type: "smoothstep", animated: false })), [graph.data]);
  const onNodeClick: NodeMouseHandler = useCallback((_, node) => { const scene = graph.data?.scenes.find((item) => item.id === node.id); setSelectedScene(scene ?? null); }, [graph.data]);
  const addScene = () => { const count = graph.data?.scenes.length ?? 0; createScene.mutate({ title: `Untitled scene ${count + 1}`, position: { x: 120 + count * 44, y: 100 + count * 38 } }); };
  const startStory = (event: React.FormEvent) => { event.preventDefault(); if (newTitle.trim()) { createStory.mutate(newTitle.trim()); setNewTitle(""); } };
  const activeStory: Story | undefined = stories.data?.find((story) => story.id === selectedStoryID);

  return <main className="app-shell">
    <aside className="story-rail"><div className="brand"><span className="brand-mark">✦</span><span>Story<br />Builder</span></div><div className="rail-label">Story maps</div><div className="story-list">{stories.data?.map((story) => <button key={story.id} className={story.id === selectedStoryID ? "story-tab active" : "story-tab"} onClick={() => { setSelectedStoryID(story.id); setSelectedScene(null); }}>{story.title}</button>)}</div><form className="story-form" onSubmit={startStory}><input aria-label="New story title" value={newTitle} onChange={(event) => setNewTitle(event.target.value)} placeholder="Name a new story" /><button type="submit">Create map</button></form></aside>
    <section className="workspace"><header className="workspace-header"><div><span className="eyebrow">Narrative topology</span><h1>{activeStory?.title ?? "Choose or create a story"}</h1></div><div className="header-actions"><span className="cycle-note">Connections cannot loop back</span><button className="primary-action" disabled={!selectedStoryID} onClick={addScene}>+ Add scene</button></div></header>
      {notice && <div className="notice" role="status">{notice}<button onClick={() => setNotice("")}>×</button></div>}
      {selectedStoryID ? <div className="map-frame"><ReactFlow nodes={nodes} edges={edges} fitView minZoom={0.25} onNodeClick={onNodeClick} onConnect={(connection) => connect.mutate(connection)} onNodeDragStop={(_, node) => moveScene.mutate({ id: node.id, position: node.position })} proOptions={{ hideAttribution: true }}><Background gap={24} size={1} /><Controls showInteractive={false} /><MiniMap pannable zoomable /></ReactFlow><div className="map-hint">Drag from a scene handle to draw a new path.</div></div> : <section className="empty-map"><span>✦</span><h2>Begin with a story map</h2><p>Name the story in the left rail, then place its first scene.</p></section>}
    </section>
    <aside className="scene-inspector">{selectedScene ? <SceneInspector scene={selectedScene} saving={saveScene.isPending} onSave={(title,brief) => saveScene.mutate({id:selectedScene.id,title,brief})} /> : <div className="inspector-empty"><span>⌁</span><h2>Scene details</h2><p>Select a waypoint to give it a title and a concise dramatic purpose.</p></div>}</aside>
  </main>;
}

function SceneInspector({ scene, saving, onSave }: { scene: Scene; saving: boolean; onSave: (title:string, brief:string) => void }) { const [title, setTitle] = useState(scene.title); const [brief, setBrief] = useState(scene.brief); useEffect(() => { setTitle(scene.title); setBrief(scene.brief); }, [scene]); return <form className="inspector-form" onSubmit={(event) => { event.preventDefault(); onSave(title,brief); }}><span className="eyebrow">Selected waypoint</span><h2>Shape this scene</h2><label>Title<input value={title} onChange={(event) => setTitle(event.target.value)} /></label><label>Scene brief<textarea value={brief} onChange={(event) => setBrief(event.target.value)} placeholder="What must change by the end of this scene?" rows={8} /></label><button className="save-action" disabled={saving} type="submit">{saving ? "Saving…" : "Save scene notes"}</button><div className="scene-id">{scene.id.slice(0, 8)}</div></form> }

createRoot(document.getElementById("root")!).render(<QueryClientProvider client={queryClient}><App /></QueryClientProvider>);
