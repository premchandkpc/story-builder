import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { api } from "./client"
import type { Story, StoryStats } from "./types"

export function useStories() {
  return useQuery<Story[]>({
    queryKey: ["stories"],
    queryFn: () => api.stories.list(),
  })
}

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

export function useAllStoryStats(stories: Story[]) {
  return useQuery<Record<string, StoryStats>>({
    queryKey: ["allStoryStats", stories.map((s) => s.id).sort()],
    queryFn: async () => {
      const entries = await Promise.all(
        stories.map(async (s) => {
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
      return Object.fromEntries(entries)
    },
    enabled: stories.length > 0,
    staleTime: 10_000,
  })
}

export function useCreateStory() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: (title: string) => api.stories.create({ title }),
    onSuccess: (story) => {
      queryClient.invalidateQueries({ queryKey: ["stories"] })
      navigate(`/stories/${story.id}`)
    },
  })
}

export function useGenerateTitle() {
  return useMutation({
    mutationFn: (synopsis: string) => api.stories.generateTitle({ synopsis }),
  })
}

export function useGenerateStory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (synopsis: string) => api.stories.generate({ synopsis }),
    onSuccess: () => {
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ["stories"] }), 3000)
    },
  })
}
