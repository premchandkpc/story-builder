import { useParams } from "react-router-dom"
import StoryGraph from "./StoryGraph"

export default function StoryView() {
  const { storyId } = useParams<{ storyId: string }>()
  if (!storyId) return null
  return <StoryGraph storyId={storyId} />
}
