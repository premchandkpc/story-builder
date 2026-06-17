// ---- Route Configuration ----
// This file defines all the URL routes for the application using React Router v7.

// createBrowserRouter: creates a router that uses the HTML5 History API
// (normal URLs like /stories/abc, no hash fragments).
import { createBrowserRouter } from "react-router-dom"

import Layout from "./components/Layout"     // main app shell (sidebar + content area)
import HomeView from "./components/HomeView" // the home/landing page
import StoryView from "./components/StoryView" // story detail page with graph

// ---- Route Tree ----
// createBrowserRouter takes an array of route objects (a "route tree").
// Nested routes inherit their parent's UI.
export const router = createBrowserRouter([
  {
    path: "/",          // the root URL path
    element: <Layout />, // Layout renders the sidebar + <Outlet/>
    children: [         // child routes render INSIDE Layout via <Outlet/>
      {
        index: true,    // "index: true" means this matches the parent path exactly ("/")
        element: <HomeView />,
      },
      {
        path: "stories/:storyId", // "/stories/<some-id>" — :storyId is a URL parameter
        element: <StoryView />,   // StoryView reads :storyId via useParams()
      },
    ],
  },
])
