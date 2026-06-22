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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
// useNavigate: React Router hook to programmatically navigate between pages
import { useNavigate } from "react-router-dom"
import { api } from "./client"
import type { CriticScoreData, Story, StoryStats, SceneTurn, AgentRun, LlmMetrics, StoryBible } from "./types"

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
      return { total, generated, accepted, stale }
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
              return [s.id, { total, generated, accepted, stale }] as const
            } catch {
              return [s.id, { total: 0, generated: 0, accepted: 0, stale: 0 }] as const
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
// onSuccess:
//   1. Invalidates the "stories" cache so the sidebar refreshes
//   2. Navigates to the new story's page using React Router
export function useCreateStory() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()       // hook for programmatic navigation
  return useMutation({
    mutationFn: (title: string) => api.stories.create({ title }),
    onSuccess: (story) => {
      queryClient.invalidateQueries({ queryKey: ["stories"] })
      navigate(`/stories/${story.id}`)  // redirect to the new story
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
    queryFn: () => api.agentRuns.list(storyId, nodeId!),
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
