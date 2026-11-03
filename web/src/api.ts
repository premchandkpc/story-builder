export type Position = { x: number; y: number };
export type Story = { id: string; title: string; createdAt: string };
export type Scene = { id: string; storyId: string; title: string; brief: string; position: Position };
export type Edge = { id: string; storyId: string; fromSceneId: string; toSceneId: string };
export type Graph = { story: Story; scenes: Scene[]; edges: Edge[] };

type ApiError = { code: string; message: string };
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { headers: { "Content-Type": "application/json", ...(options?.headers ?? {}) }, ...options });
  if (!response.ok) { const body = await response.json().catch(() => ({ message: "The request failed." })) as ApiError; throw new Error(body.message); }
  return response.json() as Promise<T>;
}

export const api = {
  listStories: () => request<Story[]>("/stories"),
  createStory: (title: string) => request<Story>("/stories", { method: "POST", body: JSON.stringify({ title }) }),
  graph: (storyId: string) => request<Graph>(`/stories/${storyId}/graph`),
  createScene: (storyId: string, title: string, position: Position) => request<Scene>(`/stories/${storyId}/scenes`, { method: "POST", body: JSON.stringify({ title, position }) }),
  moveScene: (sceneId: string, position: Position) => request<Scene>(`/scenes/${sceneId}`, { method: "PATCH", body: JSON.stringify({ position }) }),
  updateScene: (sceneId: string, title: string, brief: string) => request<Scene>(`/scenes/${sceneId}`, { method: "PATCH", body: JSON.stringify({ title, brief }) }),
  createEdge: (storyId: string, fromSceneId: string, toSceneId: string) => request<Edge>(`/stories/${storyId}/edges`, { method: "POST", body: JSON.stringify({ from_scene_id: fromSceneId, to_scene_id: toSceneId }) }),
};
