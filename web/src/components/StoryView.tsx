// ---- StoryView ----
// A thin wrapper component for the story detail page.
// It reads the :storyId URL parameter and renders the StoryGraph.
// If storyId is missing (shouldn't happen), it renders nothing.

// useParams: React Router hook that extracts URL parameters from the current route.
// The parameter name :storyId matches the route path "stories/:storyId" in routes.tsx.
import { useParams } from "react-router-dom"
import StoryGraph from "./StoryGraph"

export default function StoryView() {
  // Destructure storyId from the params object, typed as { storyId: string }
  const { storyId } = useParams<{ storyId: string }>()

  // Guard clause: if for some reason storyId is undefined, show nothing
  if (!storyId) return null

  // Pass storyId to the graph component
  return <StoryGraph storyId={storyId} />
}
