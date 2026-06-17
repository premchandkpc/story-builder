// ---- React 19 Entry Point ----
// This file is the FIRST JavaScript that runs when the page loads.
// It is referenced by index.html via <script type="module" src="/src/main.tsx">.

// StrictMode: a development-only wrapper that:
//   - Double-invokes effects to detect side-effect bugs
//   - Highlights potential problems
//   - Does nothing in production builds
import { StrictMode } from "react"

// createRoot: React 19's way to mount a React app into the DOM.
// This replaced ReactDOM.render() in React 18+.
import { createRoot } from "react-dom/client"

// QueryClient: the central cache manager for TanStack React Query.
// QueryClientProvider: makes the QueryClient available to all components via React context.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

// RouterProvider: renders the active route based on the router configuration.
// We pass in our `router` (defined in routes.tsx) and it handles URL matching.
import { RouterProvider } from "react-router-dom"
import { router } from "./routes"

// Import global CSS — Vite processes this and injects it into the page.
import "./index.css"

// ---- Create a QueryClient instance ----
// This configures the default behavior for all queries in the app.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5_000,  // data is fresh for 5 seconds before refetch is allowed
      retry: 1,           // if a query fails, retry once before showing error
    },
  },
})

// ---- Mount the React app ----
// `document.getElementById("root")!` — finds the <div id="root"> in index.html.
// The `!` (non-null assertion) tells TypeScript "trust me, this exists."

createRoot(document.getElementById("root")!).render(
  // StrictMode enables dev-only checks
  <StrictMode>
    {/*
      QueryClientProvider wraps the entire app so any component can use useQuery/useMutation
      client={queryClient} passes our configured QueryClient instance
    */}
    <QueryClientProvider client={queryClient}>
      {/*
        RouterProvider renders the correct component based on the current URL.
        router={router} is our route config from routes.tsx
      */}
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)
