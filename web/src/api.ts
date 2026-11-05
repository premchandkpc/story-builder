export type Position = { x: number; y: number };
export type Story = { id: string; title: string; createdAt: string };
export type Scene = { id: string; storyId: string; title: string; brief: string; position: Position; acceptedDraftId?: string; prose?: string };
export type Draft = { id: string; status: "GENERATING" | "READY" | "FAILED" | "ACCEPTED"; prose: string };
export type Edge = { id: string; storyId: string; fromSceneId: string; toSceneId: string };
export type Graph = { story: Story; scenes: Scene[]; edges: Edge[] };

type ApiError = { code: string; message: string };
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { headers: { "Content-Type": "application/json", ...(options?.headers ?? {}) }, ...options });
  if (!response.ok) { const body = await response.json().catch(() => ({ message: "The request failed." })) as ApiError; throw new Error(body.message); }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const api = {
  generation: () => request<{mode: string; minimumWords: number; configured: boolean; model: string}>("/generation"),
  listStories: () => request<Story[]>("/stories"),
  createStory: (title: string) => request<Story>("/stories", { method: "POST", body: JSON.stringify({ title }) }),
  graph: (storyId: string) => request<Graph>(`/stories/${storyId}/graph`),
  createScene: (storyId: string, title: string, position: Position) => request<Scene>(`/stories/${storyId}/scenes`, { method: "POST", body: JSON.stringify({ title, position }) }),
  moveScene: (sceneId: string, position: Position) => request<Scene>(`/scenes/${sceneId}`, { method: "PATCH", body: JSON.stringify({ position }) }),
  updateScene: (sceneId: string, title: string, brief: string) => request<Scene>(`/scenes/${sceneId}`, { method: "PATCH", body: JSON.stringify({ title, brief }) }),
  createEdge: (storyId: string, fromSceneId: string, toSceneId: string) => request<Edge>(`/stories/${storyId}/edges`, { method: "POST", body: JSON.stringify({ from_scene_id: fromSceneId, to_scene_id: toSceneId }) }),
  acceptDraft: (draftId: string) => request<Scene>(`/drafts/${draftId}/accept`, { method: "POST" }),
  rejectDraft: (draftId: string) => request<void>(`/drafts/${draftId}/reject`, { method: "POST" }),
  generate: async (sceneId: string, onChunk: (text: string) => void, signal?: AbortSignal): Promise<Draft> => {
    const response = await fetch(`/api/v1/scenes/${sceneId}/generate`, { method: "POST", signal });
    if (!response.ok || !response.body) throw new Error("Generation could not start.");
    const reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = ""; let complete: Draft | undefined;
    while (true) { const { done, value } = await reader.read(); if (done) break; buffer += decoder.decode(value, { stream: true }); const messages = buffer.split("\n\n"); buffer = messages.pop() ?? ""; for (const message of messages) { const event = message.match(/^event: (.+)$/m)?.[1]; const data = message.match(/^data: (.+)$/m)?.[1]; if (!event || !data) continue; const parsed = JSON.parse(data); if (event === "chunk") onChunk(parsed.text); if (event === "complete") complete = parsed; if (event === "error") throw new Error(parsed.message); } }
    if (!complete) throw new Error("Generation ended without a draft."); return complete;
  },
};
