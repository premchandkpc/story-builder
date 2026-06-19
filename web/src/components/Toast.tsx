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

const toastColors: Record<ToastType, { bg: string; color: string }> = {
  success: { bg: "var(--success)", color: "#fff" },
  error: { bg: "var(--error)", color: "#fff" },
  info: { bg: "var(--accent)", color: "#1a1a24" },
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
        bottom: 16,
        right: 16,
        zIndex: 9999,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        pointerEvents: "none",
      }}>
        {toasts.map((t) => {
          const c = toastColors[t.type]
          return (
            <div
              key={t.id}
              style={{
                animation: "slideUp 0.25s var(--ease-out)",
                padding: "10px 16px",
                borderRadius: 8,
                background: c.bg,
                color: c.color,
                fontSize: 13,
                fontWeight: 500,
                boxShadow: "0 4px 16px rgba(0,0,0,0.3)",
                maxWidth: 360,
                pointerEvents: "auto",
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: 8,
              }}
              onClick={() => removeToast(t.id)}
              title="Dismiss"
            >
              {t.message}
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}
