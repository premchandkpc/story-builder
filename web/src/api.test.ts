import { beforeEach, describe, expect, it, vi } from "vitest";

const storage = new Map<string, string>();
vi.stubGlobal("sessionStorage", {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => storage.set(key, value),
  removeItem: (key: string) => storage.delete(key),
});
const { api, setAuthToken } = await import("./api");
beforeEach(() => {
  vi.restoreAllMocks();
  setAuthToken("");
});
describe("API transport", () => {
  it("treats successful no-content rejection as success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    await expect(api.rejectDraft("draft")).resolves.toBeUndefined();
  });
  it("authenticates both JSON and streaming requests", async () => {
    setAuthToken("test-token");
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(new Response("[]", { status: 200 }))
      .mockResolvedValueOnce(
        new Response(
          'event: chunk\ndata: {"text":"Hello"}\n\nevent: complete\ndata: {"id":"d","sceneId":"s","storyId":"story","status":"READY","prose":"Hello"}\n\n',
        ),
      );
    vi.stubGlobal("fetch", fetch);
    await api.listStories();
    const controller = new AbortController();
    const chunk = vi.fn();
    await expect(
      api.generate("s", chunk, controller.signal),
    ).resolves.toMatchObject({ id: "d", sceneId: "s" });
    expect(chunk).toHaveBeenCalledWith("Hello");
    for (const [, options] of fetch.mock.calls)
      expect(options.headers.get("Authorization")).toBe("Bearer test-token");
    expect(fetch.mock.calls[1][1].signal).toBe(controller.signal);
  });
  it("handles SSE frames split across network chunks", async () => {
    const chunks = [
      "event: chu",
      'nk\ndata: {"text":"A"}\n\nevent: complete\ndata: {"id":',
      '"d","sceneId":"s","storyId":"story","status":"READY","prose":"A"}\n\n',
    ];
    const encoder = new TextEncoder();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          new ReadableStream({
            start(controller) {
              for (const chunk of chunks)
                controller.enqueue(encoder.encode(chunk));
              controller.close();
            },
          }),
        ),
      ),
    );
    const onChunk = vi.fn();
    await expect(api.generate("s", onChunk)).resolves.toMatchObject({
      id: "d",
    });
    expect(onChunk).toHaveBeenCalledWith("A");
  });
  it("surfaces streaming errors instead of returning an incomplete draft", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response('event: error\ndata: {"message":"Scene is locked"}\n\n'),
        ),
    );
    await expect(api.generate("s", vi.fn())).rejects.toThrow("Scene is locked");
  });
});

it("reassembles long fragmented UTF-8 streams without truncating prose", async () => {
  const prose = "மீரா 🌙 她选择回家。\n".repeat(4000);
  const draft = { id: "d", sceneId: "s", storyId: "story", status: "READY", prose };
  const bytes = new TextEncoder().encode(`event: chunk\ndata: ${JSON.stringify({ text: prose })}\n\nevent: complete\ndata: ${JSON.stringify(draft)}\n\n`);
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(new ReadableStream({
    start(controller) {
      for (let i = 0; i < bytes.length; i += 37) controller.enqueue(bytes.slice(i, i + 37));
      controller.close();
    },
  }))));
  let text = "";
  await expect(api.generate("s", chunk => { text += chunk; })).resolves.toEqual(draft);
  expect(text).toBe(prose);
});
