import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  ReactFlow,
  useNodesState,
  Background,
  Controls,
  MiniMap,
  type Connection,
  type Edge as FlowEdge,
  type Node,
  type NodeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { CharacterPanel } from "./CharacterPanel";
import { api, setAuthToken, type Draft, type Scene, type Story } from "./api";
import "./styles.css";

const queryClient = new QueryClient();

function App() {
  const cache = useQueryClient();
  const [authToken, updateToken] = useState(
    sessionStorage.getItem("story-builder-token") ?? "",
  );
  const generation = useRef<AbortController | null>(null);
  const [panel, setPanel] = useState<"scene" | "characters">("scene");
  const [selectedStoryID, setSelectedStoryID] = useState("");
  const [selectedScene, setSelectedScene] = useState<Scene | null>(null);
  const [newTitle, setNewTitle] = useState("");
  const [notice, setNotice] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [draftText, setDraftText] = useState("");
  const [generating, setGenerating] = useState(false);
  const stories = useQuery({ queryKey: ["stories"], queryFn: api.listStories });
  const graph = useQuery({
    queryKey: ["graph", selectedStoryID],
    queryFn: () => api.graph(selectedStoryID),
    enabled: Boolean(selectedStoryID),
  });

  useEffect(() => {
    if (!selectedStoryID && stories.data?.[0])
      setSelectedStoryID(stories.data[0].id);
  }, [selectedStoryID, stories.data]);
  useEffect(() => {
    if (selectedScene && graph.data)
      setSelectedScene(
        graph.data.scenes.find((scene) => scene.id === selectedScene.id) ??
          null,
      );
  }, [graph.data, selectedScene?.id]);
  useEffect(() => {
    generation.current?.abort();
    generation.current = null;
    setGenerating(false);
    setDraft(null);
    setDraftText("");
  }, [selectedStoryID, selectedScene?.id]);
  useEffect(() => () => generation.current?.abort(), []);
  const selection = useRef({
    storyID: selectedStoryID,
    sceneID: selectedScene?.id,
  });
  selection.current = { storyID: selectedStoryID, sceneID: selectedScene?.id };
  const refreshGraph = () =>
    cache.invalidateQueries({ queryKey: ["graph", selectedStoryID] });
  const refreshSceneGraph = (scene: Scene) =>
    cache.invalidateQueries({ queryKey: ["graph", scene.storyId] });
  const createStory = useMutation({
    mutationFn: api.createStory,
    onSuccess: (story) => {
      cache.invalidateQueries({ queryKey: ["stories"] });
      setSelectedStoryID(story.id);
      setSelectedScene(null);
      setPanel("scene");
      setNotice("New map opened.");
    },
    onError: (error) => setNotice(error.message),
  });
  const createScene = useMutation({
    mutationFn: ({
      title,
      position,
    }: {
      title: string;
      position: { x: number; y: number };
    }) => api.createScene(selectedStoryID, title, position),
    onSuccess: refreshSceneGraph,
    onError: (error) => setNotice(error.message),
  });
  const moveScene = useMutation({
    mutationFn: ({
      id,
      position,
      revision,
    }: {
      id: string;
      position: { x: number; y: number };
      revision: number;
    }) => api.moveScene(id, position, revision),
    onSuccess: refreshSceneGraph,
    onError: (error) => setNotice(error.message),
  });
  const connect = useMutation({
    mutationFn: (connection: Connection) =>
      api.createEdge(
        selectedStoryID,
        connection.source ?? "",
        connection.target ?? "",
      ),
    onSuccess: refreshGraph,
    onError: (error) => setNotice(error.message),
  });
  const saveScene = useMutation({
    mutationFn: ({
      id,
      title,
      brief,
      revision,
    }: {
      id: string;
      title: string;
      brief: string;
      revision: number;
    }) => api.updateScene(id, title, brief, revision),
    onSuccess: (scene) => {
      refreshSceneGraph(scene);
      if (selection.current.sceneID === scene.id) {
        generation.current?.abort();
        generation.current = null;
        setGenerating(false);
        setDraft(null);
        setDraftText("");
        setSelectedScene(scene);
        setNotice("Scene notes saved.");
      }
    },
    onError: (error) => setNotice(error.message),
  });
  const generateDraft = async () => {
    if (!selectedScene) return;
    const controller = new AbortController();
    generation.current?.abort();
    generation.current = controller;
    setGenerating(true);
    setDraft(null);
    setDraftText("");
    try {
      const value = await api.generate(
        selectedScene.id,
        (chunk) => {
          if (generation.current === controller && !controller.signal.aborted)
            setDraftText((text) => text + chunk);
        },
        controller.signal,
      );
      if (generation.current === controller && !controller.signal.aborted)
        setDraft(value);
    } catch (error) {
      if (!controller.signal.aborted)
        setNotice(
          error instanceof Error ? error.message : "Generation failed.",
        );
    } finally {
      if (generation.current === controller) setGenerating(false);
    }
  };
  const acceptDraft = async () => {
    if (
      !draft ||
      draft.sceneId !== selectedScene?.id ||
      draft.storyId !== selectedStoryID
    )
      return;
    const current = draft;
    try {
      await api.acceptDraft(current.id);
      cache.invalidateQueries({ queryKey: ["graph", current.storyId] });
      if (
        selection.current.sceneID === current.sceneId &&
        selection.current.storyID === current.storyId
      ) {
        setNotice("Draft accepted into the story.");
        setDraft(null);
        setDraftText("");
      }
    } catch (error) {
      setNotice(
        error instanceof Error ? error.message : "Draft could not be accepted.",
      );
    }
  };
  const rejectDraft = async () => {
    if (
      !draft ||
      draft.sceneId !== selectedScene?.id ||
      draft.storyId !== selectedStoryID
    )
      return;
    const current = draft;
    try {
      await api.rejectDraft(current.id);
      if (
        selection.current.sceneID === current.sceneId &&
        selection.current.storyID === current.storyId
      ) {
        setDraft(null);
        setDraftText("");
        setNotice("Draft rejected. The story is unchanged.");
      }
    } catch (error) {
      setNotice(
        error instanceof Error ? error.message : "Draft could not be rejected.",
      );
    }
  };

  const nodes = useMemo<Node[]>(
    () =>
      (graph.data?.scenes ?? []).map((scene) => ({
        id: scene.id,
        position: scene.position,
        data: {
          label: (
            <>
              <span className="node-kicker">SCENE</span>
              <strong>{scene.title}</strong>
              <small>{scene.brief || "Add a scene brief"}</small>
            </>
          ),
        },
        className:
          selectedScene?.id === scene.id
            ? "scene-node is-selected"
            : "scene-node",
      })),
    [graph.data, selectedScene?.id],
  );
  const edges = useMemo<FlowEdge[]>(
    () =>
      (graph.data?.edges ?? []).map((edge) => ({
        id: edge.id,
        source: edge.fromSceneId,
        target: edge.toSceneId,
        type: "smoothstep",
        animated: false,
      })),
    [graph.data],
  );
  const [flowNodes, setFlowNodes, onNodesChange] = useNodesState(nodes);
  useEffect(() => setFlowNodes(nodes), [nodes, setFlowNodes]);
  const onNodeClick: NodeMouseHandler = useCallback(
    (_, node) => {
      const scene = graph.data?.scenes.find((item) => item.id === node.id);
      setSelectedScene(scene ?? null);
      setPanel("scene");
    },
    [graph.data],
  );
  const addScene = () => {
    const count = graph.data?.scenes.length ?? 0;
    createScene.mutate({
      title: `Untitled scene ${count + 1}`,
      position: {
        x: 80 + (count % 3) * 240,
        y: 80 + Math.floor(count / 3) * 180,
      },
    });
  };
  const startStory = (event: React.FormEvent) => {
    event.preventDefault();
    if (newTitle.trim()) {
      createStory.mutate(newTitle.trim());
      setNewTitle("");
    }
  };
  const activeStory: Story | undefined = stories.data?.find(
    (story) => story.id === selectedStoryID,
  );

  return (
    <main className="app-shell">
      <aside className="story-rail">
        <div className="brand">
          <span className="brand-mark">✦</span>
          <span>
            Story
            <br />
            Builder
          </span>
        </div>
        <form
          className="story-form"
          onSubmit={(event) => {
            event.preventDefault();
            setAuthToken(authToken);
            cache.resetQueries();
            setSelectedStoryID("");
            setSelectedScene(null);
          }}
        >
          <input
            type="password"
            autoComplete="off"
            aria-label="API bearer token"
            placeholder="API token (if required)"
            value={authToken}
            onChange={(event) => updateToken(event.target.value)}
          />
          <button type="submit">Connect</button>
        </form>
        <div className="rail-label">Story maps</div>
        <div className="story-list">
          {stories.data?.map((story) => (
            <button
              key={story.id}
              className={
                story.id === selectedStoryID ? "story-tab active" : "story-tab"
              }
              onClick={() => {
                setSelectedStoryID(story.id);
                setSelectedScene(null);
              }}
            >
              {story.title}
            </button>
          ))}
        </div>
        <form className="story-form" onSubmit={startStory}>
          <input
            aria-label="New story title"
            value={newTitle}
            onChange={(event) => setNewTitle(event.target.value)}
            placeholder="Name a new story"
          />
          <button type="submit">Create map</button>
        </form>
      </aside>
      <section className="workspace">
        <header className="workspace-header">
          <div>
            <span className="eyebrow">Narrative topology</span>
            <h1>{activeStory?.title ?? "Choose or create a story"}</h1>
          </div>
          <div className="header-actions">
            <button
              className="primary-action"
              onClick={() =>
                setPanel(panel === "characters" ? "scene" : "characters")
              }
              disabled={!selectedStoryID}
            >
              Characters
            </button>
            <span className="cycle-note">
              Connections mean “must happen before”
            </span>
            <button
              className="primary-action"
              disabled={!selectedStoryID}
              onClick={addScene}
            >
              + Add scene
            </button>
          </div>
        </header>
        {notice && (
          <div className="notice" role="status">
            {notice}
            <button onClick={() => setNotice("")}>×</button>
          </div>
        )}
        {selectedStoryID ? (
          <div className="map-frame">
            <ReactFlow
              nodes={flowNodes}
              onNodesChange={onNodesChange}
              edges={edges}
              fitView
              fitViewOptions={{ maxZoom: 1, padding: 0.2 }}
              minZoom={0.25}
              onNodeClick={onNodeClick}
              onConnect={(connection) => connect.mutate(connection)}
              onNodeDragStop={(_, node) =>
                moveScene.mutate({
                  id: node.id,
                  position: node.position,
                  revision:
                    graph.data?.scenes.find((scene) => scene.id === node.id)
                      ?.revision ?? 0,
                })
              }
              proOptions={{ hideAttribution: true }}
            >
              <Background gap={24} size={1} />
              <Controls showInteractive={false} />
              <MiniMap pannable zoomable />
            </ReactFlow>
            <div className="map-hint">
              Drag from a scene handle to draw a new path.
            </div>
          </div>
        ) : (
          <section className="empty-map">
            <span>✦</span>
            <h2>Begin with a story map</h2>
            <p>Name the story in the left rail, then place its first scene.</p>
          </section>
        )}
      </section>
      <aside className="scene-inspector">
        {panel === "characters" && selectedStoryID ? (
          <CharacterPanel key={selectedStoryID} storyID={selectedStoryID} />
        ) : selectedScene ? (
          <SceneInspector
            scene={selectedScene}
            saving={saveScene.isPending}
            onSave={(title, brief) =>
              saveScene.mutate({
                id: selectedScene.id,
                title,
                brief,
                revision: selectedScene.revision,
              })
            }
            onGenerate={generateDraft}
            generating={generating}
            draft={draft?.sceneId === selectedScene.id ? draft : null}
            draftText={
              draft && draft.sceneId !== selectedScene.id ? "" : draftText
            }
            onAccept={acceptDraft}
            onReject={rejectDraft}
          />
        ) : (
          <div className="inspector-empty">
            <span>⌁</span>
            <h2>Scene details</h2>
            <p>
              Select a waypoint to give it a title and a concise dramatic
              purpose.
            </p>
          </div>
        )}
      </aside>
    </main>
  );
}

function SceneInspector({
  scene,
  saving,
  onSave,
  onGenerate,
  generating,
  draft,
  draftText,
  onAccept,
  onReject,
}: {
  scene: Scene;
  saving: boolean;
  onSave: (title: string, brief: string) => void;
  onGenerate: () => void;
  generating: boolean;
  draft: Draft | null;
  draftText: string;
  onAccept: () => void;
  onReject: () => void;
}) {
  const [title, setTitle] = useState(scene.title);
  const [brief, setBrief] = useState(scene.brief);
  useEffect(() => {
    setTitle(scene.title);
    setBrief(scene.brief);
  }, [scene]);
  return (
    <form
      className="inspector-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSave(title, brief);
      }}
    >
      <span className="eyebrow">Selected waypoint</span>
      <h2>Shape this scene</h2>
      <label>
        Title
        <input
          aria-label="Title"
          disabled={Boolean(scene.acceptedDraftId)}
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />
      </label>
      <label>
        Scene brief
        <textarea
          aria-label="Scene brief"
          disabled={Boolean(scene.acceptedDraftId)}
          value={brief}
          onChange={(event) => setBrief(event.target.value)}
          placeholder="What must change by the end of this scene?"
          rows={5}
        />
      </label>
      <button
        className="save-action"
        disabled={saving || Boolean(scene.acceptedDraftId)}
        type="submit"
      >
        {saving ? "Saving…" : "Save scene notes"}
      </button>
      <button
        className="generate-action"
        type="button"
        onClick={onGenerate}
        disabled={
          generating ||
          saving ||
          title !== scene.title ||
          brief !== scene.brief ||
          Boolean(scene.acceptedDraftId)
        }
      >
        {generating ? "Drafting…" : "Generate draft"}
      </button>
      {scene.acceptedDraftId && (
        <section className="draft-panel">
          <span className="eyebrow">Accepted prose</span>
          <p>{scene.prose}</p>
        </section>
      )}
      {(draftText || draft) && (
        <section className="draft-panel">
          <span className="eyebrow">Unaccepted draft</span>
          <p>{draftText}</p>
          {draft && (
            <div className="draft-actions">
              <button type="button" onClick={onReject}>
                Reject
              </button>
              <button
                type="button"
                disabled={
                  saving || title !== scene.title || brief !== scene.brief
                }
                onClick={onAccept}
              >
                Accept into story
              </button>
            </div>
          )}
        </section>
      )}
      <div className="scene-id">{scene.id.slice(0, 8)}</div>
    </form>
  );
}

createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <App />
  </QueryClientProvider>,
);
