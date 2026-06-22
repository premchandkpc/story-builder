import { memo } from "react"

interface StatCardProps {
  label: string
  value: string
  color?: string
  delay?: number
}

const StatCard = memo(function StatCard({ label, value, color, delay = 0 }: StatCardProps) {
  return (
    <div
      className="card-hover"
      style={{
        background: "var(--surface)",
        border: "1px solid var(--border)",
        borderRadius: "var(--radius-md)",
        padding: "12px 14px",
        animation: `slideUp 0.3s var(--ease-out) ${delay}s both`,
      }}
    >
      <div style={{
        fontSize: 10, color: "var(--text-dim)",
        textTransform: "uppercase", letterSpacing: "0.05em",
        fontWeight: 500,
      }}>
        {label}
      </div>
      <div style={{
        fontSize: color ? 20 : 18,
        fontWeight: 700,
        fontFamily: "var(--font-heading)",
        color: color || "var(--accent)",
        marginTop: 4,
        animation: value ? "fadeIn 0.4s var(--ease-out)" : undefined,
      }}>
        {value}
      </div>
    </div>
  )
})

export default StatCard
