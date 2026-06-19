import { Component, type ReactNode, type ErrorInfo } from "react"
import { btnStyle } from "../api/types"

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ErrorBoundary caught:", error, info)
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          minHeight: 200,
          padding: 40,
          gap: 12,
          textAlign: "center",
        }}>
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--error)" strokeWidth="1.5" strokeLinecap="round">
            <circle cx="12" cy="12" r="10" />
            <path d="M12 8v4M12 16h.01" />
          </svg>
          <div style={{ fontFamily: "var(--font-heading)", fontSize: 18, color: "var(--text)" }}>
            Something went wrong
          </div>
          <div style={{ fontSize: 13, color: "var(--text-muted)", maxWidth: 400, lineHeight: 1.5 }}>
            {this.state.error?.message || "An unexpected error occurred"}
          </div>
          <button
            onClick={() => this.setState({ hasError: false, error: null })}
            style={btnStyle("var(--accent)")}
          >
            Try again
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
