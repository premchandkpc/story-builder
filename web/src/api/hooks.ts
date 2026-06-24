// ---- React Query Hooks ----
//
// "Hooks" are React functions that let you use state and other React features
// inside function components. Custom hooks (like these) let you reuse logic.
//
// These hooks use TanStack React Query (formerly "React Query") library for:
//   - Caching API responses
//   - Auto-refetching when data is stale
//   - Loading/error states
//   - Mutations (create/update/delete) with automatic cache invalidation

// useQuery: fetches and caches data (GET requests)
// useMutation: sends changes (POST/PUT/DELETE) and can invalidate caches
// useQueryClient: gives access to the query cache for invalidation
import { useEffect, useRef, useMemo } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
// useNavigate: React Router hook to programmatically navigate between pages
import { useNavigate } from "react-router-dom"
import { api } from "./client"
import type { Character, CriticScoreData, Story, StoryStats, SceneTurn, AgentRun, LlmMetrics, StoryBible, TimelineEvent, Generation, GraphNode, GraphEdge, Topology, StoryRun, RunStep, NarrativeEvent, PromptSnapshot, CostSummary, RunStats, ScenePlan, GenDiff, CreateNodePayload, CreateEdgePayload } from "./types"

// ---- useStories() ----
// Custom hook that fetches the list of all stories.
// Returns { data, isLoading, error, ... } from useQuery.
export function useStories() {
  return useQuery<Story[]>({
    queryKey: ["stories"],        // unique cache key — identifies this query
    queryFn: () => api.stories.list(), // the function that fetches data
  })
}

// ---- useStoryNodeStats(storyId) ----
// Fetches node counts (total, generated, accepted, stale) for a single story.
// Instead of a dedicated API endpoint, it fetches all nodes and computes stats client-side.
export function useStoryNodeStats(storyId: string) {
  return useQuery<StoryStats>({
    queryKey: ["storyStats", storyId],
    queryFn: async () => {
      const nodes = await api.nodes.list(storyId)
      const total = nodes.length
      const generated = nodes.filter((n) => n.status === "generated").length
      const accepted = nodes.filter((n) => n.status === "accepted").length
      const stale = nodes.filter((n) => n.status === "stale").length
      return { total, generated, accepted, stale, pending: total - accepted }
    },
  })
}

// ---- useAllStoryStats(stories) ----
// Fetches node stats for ALL stories in one go (used by sidebar).
// Batches requests with a concurrency limit to avoid N simultaneous API calls.
// staleTime: data is considered fresh for 10 seconds.
export function useAllStoryStats(stories: Story[]) {
  return useQuery<Record<string, StoryStats>>({
    queryKey: ["allStoryStats", stories.map((s) => s.id).sort()],
    queryFn: async () => {
      const concurrency = 6
      const results: (readonly [string, StoryStats])[] = []
      for (let i = 0; i < stories.length; i += concurrency) {
        const batch = stories.slice(i, i + concurrency)
        const entries = await Promise.all(
          batch.map(async (s) => {
            try {
              const nodes = await api.nodes.list(s.id)
              const total = nodes.length
              const generated = nodes.filter((n) => n.status === "generated").length
              const accepted = nodes.filter((n) => n.status === "accepted").length
              const stale = nodes.filter((n) => n.status === "stale").length
              return [s.id, { total, generated, accepted, stale, pending: total - accepted }] as const
            } catch {
              return [s.id, { total: 0, generated: 0, accepted: 0, stale: 0, pending: 0 }] as const
            }
          }),
        )
        results.push(...entries)
      }
      return Object.fromEntries(results)
    },
    enabled: stories.length > 0,
    staleTime: 10_000,
  })
}

// ---- useCreateStory() ----
// Mutation: creates a new story, then navigates to its detail page.
// Uses optimistic update: adds placeholder story to cache immediately.
export function useCreateStory() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { success, error: showError } = useToastExternal()

  return useMutation({
    mutationFn: (title: string) => api.stories.create({ title }),
    onMutate: async (title) => {
      await queryClient.cancelQueries({ queryKey: ["stories"] })
      const prev = queryClient.getQueryData<Story[]>(["stories"])
      const placeholder: Story = {
        id: `new-${Date.now()}`,
        title,
        canon_pins: {},
        createdAt: new Date().toISOString(),
      }
      queryClient.setQueryData<Story[]>(["stories"], (old) => [...(old || []), placeholder])
      return { prev }
    },
    onSuccess: (story) => {
      // Replace placeholder with real story
      queryClient.setQueryData<Story[]>(["stories"], (old) =>
        (old || []).map((s) => s.id.startsWith("new-") ? story : s),
      )
      success("Story created")
      navigate(`/stories/${story.id}`)
    },
    onError: (_err, _title, context) => {
      if (context?.prev) queryClient.setQueryData(["stories"], context.prev)
      showError("Failed to create story")
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["stories"] })
    },
  })
}

/** Access toast from outside a component context — used inside mutations. */
let toastFns: { success: (m: string) => void; error: (m: string) => void } | null = null
export function setToastFns(fns: typeof toastFns) { toastFns = fns }
function useToastExternal() {
  return toastFns || { success: () => {}, error: () => {} }
}

// ---- useDeleteStory() ----
// Mutation: deletes a story with optimistic removal + rollback on error.
export function useDeleteStory() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { error: showError } = useToastExternal()

  return useMutation({
    mutationFn: (storyId: string) => api.stories.delete(storyId),
    onMutate: async (storyId) => {
      await queryClient.cancelQueries({ queryKey: ["stories"] })
      const prev = queryClient.getQueryData<Story[]>(["stories"])
      queryClient.setQueryData<Story[]>(["stories"], (old) =>
        (old || []).filter((s) => s.id !== storyId),
      )
      return { prev }
    },
    onSuccess: (_data, storyId) => {
      // If user is viewing the deleted story, navigate home
      const currentPath = window.location.pathname
      if (currentPath.includes(storyId)) navigate("/")
    },
    onError: (_err, _storyId, context) => {
      if (context?.prev) queryClient.setQueryData(["stories"], context.prev)
      showError("Failed to delete story")
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["stories"] })
    },
  })
}

// ---- useGenerateTitle() ----
// Mutation: sends a synopsis to the LLM and gets back a suggested title.
// Used on the home page when user wants AI to generate a title.
export function useGenerateTitle() {
  return useMutation({
    mutationFn: (synopsis: string) => api.stories.generateTitle({ synopsis }),
  })
}

// ---- useGenerateStory() ----
// Mutation: kicks off full story generation from a synopsis.
// Navigates to the new story page on success.
export function useGenerateStory() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: (synopsis: string) => api.stories.generate({ synopsis }),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["stories"] })
      if (result?.story_id) {
        navigate(`/stories/${result.story_id}`)
      }
    },
  })
}

// ---- Topology Hooks ----
const TOPOLOGY_KEY = (storyId: string) => ["topology", storyId] as const

export function useTopology(storyId: string) {
  return useQuery<Topology>({
    queryKey: TOPOLOGY_KEY(storyId),
    queryFn: () => api.topology.get(storyId),
    enabled: !!storyId,
  })
}

export function useCreateNode(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: (data: CreateNodePayload) => api.nodes.create(storyId, data),
    onMutate: async () => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      const tempNode: GraphNode = {
        id: `temp-${Date.now()}`,
        story_id: storyId,
        title: "",
        status: "draft" as const,
        beat_intent: "New scene",
        character_refs: [],
        location_ref: null,
        pov: "third-person",
        tone: "neutral",
        target_words: 300,
        position_x: 100,
        position_y: 100,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      qc.setQueryData<Topology>(key, (old) => old ? { ...old, nodes: [...old.nodes, tempNode] } : old)
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to add scene")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export function useUpdateNode(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: ({ nodeId, data }: { nodeId: string; data: Record<string, unknown> }) =>
      api.nodes.update(storyId, nodeId, data),
    onMutate: async ({ nodeId, data }) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      qc.setQueryData<Topology>(key, (old) => {
        if (!old) return old
        return {
          ...old,
          nodes: old.nodes.map((n) =>
            n.id === nodeId ? { ...n, ...data } as GraphNode : n,
          ),
        }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to save scene")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export function useDeleteNode(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: (nodeId: string) => api.nodes.delete(storyId, nodeId),
    onMutate: async (nodeId) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      qc.setQueryData<Topology>(key, (old) => {
        if (!old) return old
        return {
          ...old,
          nodes: old.nodes.filter((n) => n.id !== nodeId),
          edges: old.edges.filter((e) => e.from_node !== nodeId && e.to_node !== nodeId),
        }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to delete scene")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export function useCreateEdge(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: (data: CreateEdgePayload) =>
      api.edges.create(storyId, data),
    onMutate: async (data) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      const tempEdge: GraphEdge = {
        id: `temp-edge-${Date.now()}`,
        from_node: data.from_node,
        to_node: data.to_node,
        edge_type: data.edge_type as import("./types").EdgeType,
        story_id: storyId,
      }
      qc.setQueryData<Topology>(key, (old) => {
        if (!old) return old
        return { ...old, edges: [...old.edges, tempEdge] }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to create edge")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export function useDeleteEdge(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: (edgeId: string) => api.edges.deleteById(storyId, edgeId),
    onMutate: async (edgeId) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      qc.setQueryData<Topology>(key, (old) => {
        if (!old) return old
        return { ...old, edges: old.edges.filter((e) => e.id !== edgeId) }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to delete edge")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

export function useUpdateNodePosition(storyId: string) {
  const qc = useQueryClient()
  const key = TOPOLOGY_KEY(storyId)
  const { error: showError } = useToastExternal()
  return useMutation({
    mutationFn: ({ nodeId, x, y }: { nodeId: string; x: number; y: number }) =>
      api.nodes.updatePosition(storyId, nodeId, x, y),
    onMutate: async ({ nodeId, x, y }) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Topology>(key)
      if (!prev) return { prev: null }
      qc.setQueryData<Topology>(key, (old) => {
        if (!old) return old
        return {
          ...old,
          nodes: old.nodes.map((n) =>
            n.id === nodeId ? { ...n, position_x: x, position_y: y } : n,
          ),
        }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      showError("Failed to save position")
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
}

// ---- Character Hooks ----
export function useCharacters(storyId: string) {
  return useQuery<Character[]>({
    queryKey: ["characters", storyId],
    queryFn: () => api.characters.listByStory(storyId),
    enabled: !!storyId,
  })
}

export function useMigrateCharacter(storyId: string) {
  const qc = useQueryClient()
  const { success, error: showError } = useToastExternal()
  return useMutation({
    mutationFn: (charId: string) => api.characters.migrate(storyId, charId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["characters", storyId] })
      success("Character migrated to this story")
    },
    onError: () => {
      showError("Failed to migrate character")
    },
  })
}

// ---- useTurns(storyId, nodeId) ----
// Fetches all turns for a given scene/node.
export function useTurns(storyId: string, nodeId: string | null) {
  return useQuery<SceneTurn[]>({
    queryKey: ["turns", storyId, nodeId],
    queryFn: () => api.turns.list(storyId, nodeId!),
    enabled: !!nodeId,
  })
}

// ---- useTurn() ----
// Fetches a single turn by ID.
export function useTurn(storyId: string, nodeId: string, turnId: string) {
  return useQuery<SceneTurn>({
    queryKey: ["turn", storyId, nodeId, turnId],
    queryFn: () => api.turns.get(storyId, nodeId, turnId),
    enabled: !!turnId,
  })
}

// ---- useAgentRuns(storyId, nodeId) ----
// Fetches agent run logs for a given scene/node.
export function useAgentRuns(storyId: string, nodeId: string | null) {
  return useQuery<AgentRun[]>({
    queryKey: ["agentRuns", storyId, nodeId],
    queryFn: () => api.agentRuns.list(),
    enabled: !!nodeId,
  })
}

// ---- useLlmMetrics(storyId) ----
// Fetches aggregated LLM token usage metrics for a story.
export function useLlmMetrics(storyId: string) {
  return useQuery<LlmMetrics>({
    queryKey: ["llmMetrics", storyId],
    queryFn: () => api.metrics.llm(storyId),
    enabled: !!storyId,
    refetchInterval: 30_000,
  })
}

// ---- useGenerationStatusPolling(storyId, nodeId) ----
// Polls generations every 2s while enabled, skips invalidation when no pending gens.
export function useGenerationStatusPolling(storyId: string, nodeId: string | null, enabled = false) {
  const queryClient = useQueryClient()
  const queryKey = useMemo(() => ["generations", storyId, nodeId], [storyId, nodeId])

  const { data: generations, isLoading, isError, refetch } = useQuery<Generation[]>({
    queryKey,
    queryFn: () => api.generations.list(storyId, nodeId!),
    enabled: !!nodeId,
    staleTime: 1000,
  })

  const hasPending = (generations || []).some((g) =>
    g.status === "pending" || g.status === "running" || g.status === "queued"
  )

  const hasPendingRef = useRef(hasPending)

  useEffect(() => {
    hasPendingRef.current = hasPending
  }, [hasPending])

  useEffect(() => {
    if (!enabled || !nodeId) return
    const interval = setInterval(async () => {
      if (!hasPendingRef.current) return
      await queryClient.invalidateQueries({ queryKey })
    }, 2000)
    return () => clearInterval(interval)
  }, [enabled, nodeId, queryClient, queryKey])

  return { generations: generations || [], isLoading, isError, refetch, hasPending }
}

// ---- useCriticScores(storyId) ----
// Fetches critic evaluation scores for all agent-generated scenes in a story.
export function useCriticScores(storyId: string) {
  return useQuery<CriticScoreData[]>({
    queryKey: ["criticScores", storyId],
    queryFn: () => api.critic.list(storyId),
    enabled: !!storyId,
    refetchInterval: 30_000,
  })
}

// ---- Timeline Hooks ----
export function useTimeline(storyId: string) {
  return useQuery<TimelineEvent[]>({
    queryKey: ["timeline", storyId],
    queryFn: () => api.timeline.list(storyId),
    enabled: !!storyId,
  })
}

export function useCrossStoryTimeline(storyId: string) {
  return useQuery<TimelineEvent[]>({
    queryKey: ["crossStoryTimeline", storyId],
    queryFn: () => api.timeline.crossStoryList(storyId),
    enabled: !!storyId,
  })
}

// ---- Bible Hooks ----
export function useBible(storyId: string) {
  return useQuery<StoryBible | null>({
    queryKey: ["bible", storyId],
    queryFn: () => api.bible.get(storyId),
    enabled: !!storyId,
  })
}

export function useGenerateBible(storyId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.bible.generate(storyId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bible", storyId] }),
  })
}

export function useUpdateBible(storyId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (bible: Partial<StoryBible>) => api.bible.update(storyId, bible),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bible", storyId] }),
  })
}

export function useReferencingBibles(storyId: string) {
  return useQuery<StoryBible[]>({
    queryKey: ["referencingBibles", storyId],
    queryFn: () => api.bible.listReferencing(storyId),
    enabled: !!storyId,
  })
}

export function useLinkBible(storyId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (bibleId: string) => api.bible.link(storyId, bibleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["bible", storyId] })
      qc.invalidateQueries({ queryKey: ["referencingBibles", storyId] })
    },
  })
}

export function useUnlinkBible(storyId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (bibleId: string) => api.bible.unlink(storyId, bibleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["bible", storyId] })
      qc.invalidateQueries({ queryKey: ["referencingBibles", storyId] })
    },
  })
}

// ---- Plan & Diff Hooks ----
export function useScenePlan(storyId: string, nodeId: string | null) {
  return useQuery<ScenePlan>({
    queryKey: ["scenePlan", storyId, nodeId],
    queryFn: () => api.nodes.plan(storyId, nodeId!),
    enabled: !!storyId && !!nodeId,
  })
}

export function useGenDiff(storyId: string, nodeId: string, genA: string | null, genB: string | null) {
  return useQuery<GenDiff>({
    queryKey: ["genDiff", storyId, nodeId, genA, genB],
    queryFn: () => api.nodes.diff(storyId, nodeId, genA!, genB!),
    enabled: !!storyId && !!nodeId && !!genA && !!genB,
  })
}

// ---- Run Hooks ----
export function useStoryRuns(storyId: string, limit?: number) {
  return useQuery<StoryRun[]>({
    queryKey: ["runs", storyId],
    queryFn: () => api.runs.listByStory(storyId, limit),
    enabled: !!storyId,
  })
}

export function useRunDetails(runId: string | null) {
  return useQuery<StoryRun>({
    queryKey: ["run", runId],
    queryFn: () => api.runs.get(runId!),
    enabled: !!runId,
  })
}

export function useRunSteps(runId: string | null) {
  return useQuery<RunStep[]>({
    queryKey: ["runSteps", runId],
    queryFn: () => api.runs.steps(runId!),
    enabled: !!runId,
  })
}

export function useCancelRun() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (runId: string) => api.runs.cancel(runId),
    onSuccess: (_data, runId) => {
      qc.invalidateQueries({ queryKey: ["run", runId] })
      qc.invalidateQueries({ queryKey: ["runs"] })
    },
  })
}

export function useRunPromptSections(runId: string | null) {
  return useQuery<PromptSnapshot>({
    queryKey: ["runPromptSections", runId],
    queryFn: () => api.runs.promptSections(runId!),
    enabled: !!runId,
  })
}

export function useRunEvents(runId: string | null, limit = 100) {
  return useQuery<NarrativeEvent[]>({
    queryKey: ["runEvents", runId],
    queryFn: () => api.runs.events(runId!, limit),
    enabled: !!runId,
  })
}

export function useRunCost(runId: string | null) {
  return useQuery<CostSummary>({
    queryKey: ["runCost", runId],
    queryFn: () => api.runs.cost(runId!),
    enabled: !!runId,
  })
}

export function useRunStats(storyId: string) {
  return useQuery<RunStats>({
    queryKey: ["runStats", storyId],
    queryFn: () => api.runs.stats(storyId),
    enabled: !!storyId,
  })
}

// ---- Narrative Event Hooks ----
export function useNarrativeEvents(storyId: string, limit?: number) {
  return useQuery<NarrativeEvent[]>({
    queryKey: ["narrativeEvents", storyId],
    queryFn: () => api.narrativeEvents.listByStory(storyId, limit),
    enabled: !!storyId,
  })
}

export function useSceneNarrativeEvents(storyId: string, nodeId: string | null, limit?: number) {
  return useQuery<NarrativeEvent[]>({
    queryKey: ["narrativeEvents", storyId, nodeId],
    queryFn: () => api.narrativeEvents.listByScene(storyId, nodeId!, limit),
    enabled: !!storyId && !!nodeId,
  })
}
