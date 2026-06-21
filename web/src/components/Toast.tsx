/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useCallback, useRef, type ReactNode } from "react"

type ToastType = "success" | "error" | "info"

interface Toast {
  id: number
  message: string
  type: ToastType
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType) => void
  error: (message: string) => void
  success: (message: string) => void
}

const ToastContext = createContext<ToastContextValue>({
  toast: () => {},
  error: () => {},
  success: () => {},
})

export function useToast() {
  return useContext(ToastContext)
}

const toastConfig: Record<ToastType, { bg: string; color: string; icon: string }> = {
  success: { bg: "var(--success)", color: "#f5f0e8", icon: "✓" },
  error: { bg: "var(--error)", color: "#f5f0e8", icon: "✕" },
  info: { bg: "var(--accent)", color: "#1a1512", icon: "●" },
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextId = useRef(0)

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const addToast = useCallback((message: string, type: ToastType = "info") => {
    const id = nextId.current++
    setToasts((prev) => [...prev, { id, message, type }])
    setTimeout(() => removeToast(id), 3500)
  }, [removeToast])

  const toast = useCallback((message: string, type: ToastType = "info") => addToast(message, type), [addToast])
  const error = useCallback((message: string) => addToast(message, "error"), [addToast])
  const success = useCallback((message: string) => addToast(message, "success"), [addToast])

  return (
    <ToastContext.Provider value={{ toast, error, success }}>
      {children}
      <div style={{
        position: "fixed",
        bottom: 20,
        right: 20,
        zIndex: 9999,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        pointerEvents: "none",
      }}>
        {toasts.map((t) => {
          const c = toastConfig[t.type]
          return (
            <div
              key={t.id}
              style={{
                animation: "slideUp 0.25s var(--ease-out)",
                padding: "10px 16px",
                borderRadius: "var(--radius-lg)",
                background: c.bg,
                color: c.color,
                fontSize: 13,
                fontWeight: 500,
                boxShadow: "0 4px 20px rgba(0,0,0,0.35)",
                maxWidth: 380,
                pointerEvents: "auto",
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: 10,
                border: "1px solid rgba(255,255,255,0.08)",
              }}
              onClick={() => removeToast(t.id)}
              title="Dismiss"
            >
              <span style={{
                width: 18, height: 18, borderRadius: "50%",
                background: "rgba(255,255,255,0.15)",
                display: "flex", alignItems: "center", justifyContent: "center",
                fontSize: 10, fontWeight: 700, flexShrink: 0,
              }}>
                {c.icon}
              </span>
              {t.message}
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}
