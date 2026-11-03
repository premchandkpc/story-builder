// ---- Route Configuration ----
// This file defines all the URL routes for the application using React Router v7.

// createBrowserRouter: creates a router that uses the HTML5 History API
// (normal URLs like /stories/abc, no hash fragments).
import { createBrowserRouter } from "react-router-dom"
import Layout from "./components/Layout"
import HomeView from "./components/HomeView"
import StoryWorkspace from "./components/StoryWorkspace"
import AuditDashboard from "./components/AuditDashboard"

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <HomeView /> },
      {
        path: "stories/:storyId",
        element: <StoryWorkspace />,
      },
      {
        path: "stories/:storyId/:viewMode?",
        element: <StoryWorkspace />,
      },
      {
        path: "audit",
        element: <AuditDashboard />,
      },
    ],
  },
])
