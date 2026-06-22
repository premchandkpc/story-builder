import { memo, useEffect, useState } from "react"
import { simulateCompression, type CompressStats } from "../api/compress"

interface Props {
  system: string
  userMessage: string
  model: string
}

export default memo(function CompressionStats({ system, userMessage, model }: Props) {
  const [stats, setStats] = useState<CompressStats | null | "loading">(null)

  useEffect(() => {
    if (!system && !userMessage) return
    let cancelled = false
    setStats("loading")

    simulateCompression(system, userMessage, model).then((result) => {
      if (!cancelled) setStats(result)
    })

    return () => { cancelled = true }
  }, [system, userMessage, model])

  if (stats === null) return null
  if (stats === "loading") return <div style={{ fontSize: 10, color: "var(--text-muted)", fontStyle: "italic" }}>Compression stats...</div>

  const pct = ((1 - stats.compressionRatio) * 100).toFixed(0)

  return (
    <div style={{
      display: "flex", gap: 12, fontSize: 10,
      color: "var(--text-muted)", padding: "4px 0",
    }}>
      <span title="Tokens before compression">{stats.tokensBefore} → {stats.tokensAfter} tok</span>
      <span style={{ color: "var(--success)", fontWeight: 600 }}>-{pct}%</span>
      {stats.transforms.length > 0 && (
        <span title={stats.transforms.join(", ")}>
          {stats.transforms.length} transforms
        </span>
      )}
    </div>
  )
})
