import { createBrowserRouter } from "react-router-dom"
import Layout from "./components/Layout"
import HomeView from "./components/HomeView"
import StoryView from "./components/StoryView"

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <HomeView /> },
      { path: "stories/:storyId", element: <StoryView /> },
    ],
  },
])
