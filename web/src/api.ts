export type Position = { x: number; y: number };
export type Story = { id: string; title: string; createdAt: string };
export type Scene = {
  id: string;
  storyId: string;
  title: string;
  brief: string;
  position: Position;
  acceptedDraftId?: string;
  prose?: string;
  revision: number;
};
export type Draft = {
  id: string;
  sceneId: string;
  storyId: string;
  status: "GENERATING" | "READY" | "FAILED" | "ACCEPTED";
  prose: string;
};
export type Edge = {
  id: string;
  storyId: string;
  fromSceneId: string;
  toSceneId: string;
};
export type Graph = { story: Story; scenes: Scene[]; edges: Edge[] };

let token = sessionStorage.getItem("story-builder-token") ?? "";
export function setAuthToken(value: string) {
  token = value.trim();
  if (token) sessionStorage.setItem("story-builder-token", token);
  else sessionStorage.removeItem("story-builder-token");
}
function headers(extra?: HeadersInit) {
  const result = new Headers(extra);
  result.set("Content-Type", "application/json");
  if (token) result.set("Authorization", `Bearer ${token}`);
  return result;
}
export type Relationship = {
  characterId: string;
  kind: string;
  description: string;
};
export type CharacterDefinition = {
  role: string;
  pronouns: string;
  description: string;
  backstory: string;
  motivation: string;
  goals: string[];
  fears: string[];
  flaws: string[];
  values: string[];
  voice: string;
  constraints: string[];
  secrets: string[];
  initialKnowledge: string[];
  relationships: Relationship[];
};
export type Character = CharacterDefinition & {
  characterId: string;
  version: number;
  storyId: string;
  name: string;
  traits: string[];
};
export type CharacterInput = CharacterDefinition & {
  name: string;
  traits: string[];
  expectedVersion?: number;
};
type ApiError = { code: string; message: string };
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers: headers(options?.headers),
  });
  if (!response.ok) {
    const body = (await response
      .json()
      .catch(() => ({ message: "The request failed." }))) as ApiError;
    throw new Error(body.message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const api = {
 generation: () => request<{mode: string; minimumWords: number; configured: boolean; model: string}>("/generation"),
  characters: (storyId: string) =>
    request<Character[]>(`/stories/${storyId}/characters`),
  saveCharacter: (
    storyId: string,
    input: CharacterInput,
    characterId?: string,
  ) =>
    request<Character>(
      `/stories/${storyId}/characters${characterId ? `/${characterId}` : ""}`,
      { method: characterId ? "PUT" : "POST", body: JSON.stringify(input) },
    ),
  listStories: () => request<Story[]>("/stories"),
  createStory: (title: string) =>
    request<Story>("/stories", {
      method: "POST",
      body: JSON.stringify({ title }),
    }),
  graph: (storyId: string) => request<Graph>(`/stories/${storyId}/graph`),
  createScene: (storyId: string, title: string, position: Position) =>
    request<Scene>(`/stories/${storyId}/scenes`, {
      method: "POST",
      body: JSON.stringify({ title, position }),
    }),
  moveScene: (sceneId: string, position: Position, expectedRevision?: number) =>
    request<Scene>(`/scenes/${sceneId}`, {
      method: "PATCH",
      body: JSON.stringify({ position, expectedRevision }),
    }),
  updateScene: (
    sceneId: string,
    title: string,
    brief: string,
    expectedRevision?: number,
  ) =>
    request<Scene>(`/scenes/${sceneId}`, {
      method: "PATCH",
      body: JSON.stringify({ title, brief, expectedRevision }),
    }),
  createEdge: (storyId: string, fromSceneId: string, toSceneId: string) =>
    request<Edge>(`/stories/${storyId}/edges`, {
      method: "POST",
      body: JSON.stringify({
        from_scene_id: fromSceneId,
        to_scene_id: toSceneId,
      }),
    }),
  acceptDraft: (draftId: string) =>
    request<Scene>(`/drafts/${draftId}/accept`, { method: "POST" }),
  rejectDraft: (draftId: string) =>
    request<void>(`/drafts/${draftId}/reject`, { method: "POST" }),
  generate: async (
    sceneId: string,
    onChunk: (text: string) => void,
    signal?: AbortSignal,
  ): Promise<Draft> => {
    const response = await fetch(`/api/v1/scenes/${sceneId}/generate`, {
      method: "POST",
      headers: headers(),
      signal,
    });
    if (!response.ok || !response.body)
      throw new Error("Generation could not start.");
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let complete: Draft | undefined;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const messages = buffer.split("\n\n");
        buffer = messages.pop() ?? "";
        for (const message of messages) {
          const event = message.match(/^event: (.+)$/m)?.[1];
          const data = message.match(/^data: (.+)$/m)?.[1];
          if (!event || !data) continue;
          const parsed = JSON.parse(data);
          if (event === "chunk") onChunk(parsed.text);
          if (event === "complete") complete = parsed;
          if (event === "error") throw new Error(parsed.message);
        }
      }
      if (!complete) throw new Error("Generation ended without a draft.");
      return complete;
    } finally {
      await reader.cancel().catch(() => {});
      reader.releaseLock();
    }
  },
};
