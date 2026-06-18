import { useState, useMemo } from "react"

interface Finding {
  id: string
  title: string
  severity: "critical" | "high" | "medium" | "low"
  status: "fixed" | "wontfix"
  desc: string
  fix: string
  file: string
}

const findings: Finding[] = [
  { id: "F-01", title: "genInFlight leak on panic", severity: "critical", status: "fixed", desc: "Generate() goroutine had no deferred genInFlight.Delete(). If runPipeline panicked, the sceneID was permanently locked.", fix: "Added defer s.genInFlight.Delete(sceneID) as first line inside goroutine.", file: "internal/service/generation.go:87" },
  { id: "F-02", title: "CharState only holds last participant", severity: "critical", status: "fixed", desc: "params.CharState initialization was inside the character loop, resetting on each iteration.", fix: "Moved map initialization before the character loop.", file: "internal/service/generation.go:128" },
  { id: "F-03", title: "AcceptGeneration race condition", severity: "critical", status: "fixed", desc: "No concurrency guard on AcceptGeneration — two concurrent calls could interleave read-modify-write cycles.", fix: "Added acceptInFlight sync.Map guard; atomic pass marks all false then sets target true.", file: "internal/service/generation.go:33,180" },
  { id: "F-04", title: "GenerateStory missing character fields", severity: "high", status: "fixed", desc: "Character records created with only Name and Persona; MoralAlignment, Personality, etc. silently dropped.", fix: "Added explicit field mapping for all StoryOutlineCharacter fields.", file: "internal/api/stories.go:120-154" },
  { id: "F-05", title: "GenerateStory skips Location creation", severity: "high", status: "fixed", desc: "Beat LocationName field ignored; no Location records created, scenes had no LocationRef.", fix: "Created LocationRepository/Service/types; GenerateStory populates LocationRef.", file: "internal/api/stories.go, internal/domain/location.go" },
  { id: "F-06", title: "GenerateStory missing TimelinePosition", severity: "high", status: "fixed", desc: "Scenes created with TimelinePosition=0; topology ordering unreliable.", fix: "Set scene.TimelinePosition = i+1 during scene creation.", file: "internal/api/stories.go:154" },
  { id: "F-07", title: "Entire Location system absent", severity: "high", status: "fixed", desc: "No Location domain type, repository, service, or API handlers.", fix: "Created domain.Location, LocationRepo, LocationService, CRUD handlers, wiring.", file: "internal/domain/location.go, internal/repository/mongo/locations.go" },
  { id: "F-08", title: "Topology sorts by insertion order", severity: "high", status: "fixed", desc: "V2Topology returned nodes in MongoDB insertion order, not story order.", fix: "Sort by TimelinePosition before returning.", file: "internal/api/nodes.go:198" },
  { id: "F-09", title: "Dual topology endpoint confusion", severity: "medium", status: "fixed", desc: "Unused Topology() handler with different response shape from real endpoint.", fix: "Removed dead Topology() handler.", file: "internal/api/scenes.go:79-96" },
  { id: "F-10", title: "ExtractStateWorker ignores location/mood", severity: "high", status: "fixed", desc: "CharacterState.Location and .Mood left empty despite delta having them.", fix: "Set state fields directly from extraction deltas.", file: "internal/worker/extract.go:53-58" },
  { id: "F-11", title: "14-param constructor", severity: "medium", status: "fixed", desc: "NewGenerationService took 14 positional args; transposed args compile but corrupt data.", fix: "Refactored to GenerationServiceConfig struct with named fields.", file: "internal/service/generation.go:16-58" },
  { id: "F-12", title: "Missing pipeline step observability", severity: "medium", status: "fixed", desc: "runPipeline ran 6 steps but persisted no status; operators couldn't diagnose failures.", fix: "Added StepStatus map to Generation; pipeline updates running/done/failed per step.", file: "internal/domain/scene.go:46" },
  { id: "F-13", title: "go.mod minimum Go version", severity: "low", status: "wontfix", desc: "Already at go 1.26.4 (context.WithoutCancel needs 1.21+).", fix: "No change needed.", file: "go.mod:3" },
  { id: "F-14", title: "docs/schema.md Props type", severity: "low", status: "fixed", desc: "Props documented as map, actually []string.", fix: "Corrected type in docs.", file: "docs/schema.md:261" },
  { id: "F-15", title: "Generate called with empty context", severity: "critical", status: "fixed", desc: "runPipeline received only beat_intent/pov/tone/target_words — no character or location data.", fix: "Added buildPromptParams() fetching character cards, states, location, summary.", file: "internal/service/generation.go:94-178" },
]

const severityColors: Record<string, string> = {
  critical: "var(--error)",
  high: "var(--warn, #c9734a)",
  medium: "var(--accent)",
  low: "var(--text-muted)",
}

const severityOrder: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 }

export default function AuditDashboard() {
  const [filter, setFilter] = useState("all")

  const filtered = useMemo(() => {
    let f = [...findings]
    if (filter === "fixed") f = f.filter(x => x.status === "fixed")
    else if (filter === "wontfix") f = f.filter(x => x.status === "wontfix")
    else if (filter !== "all") f = f.filter(x => x.severity === filter)
    return f.sort((a, b) => severityOrder[a.severity] - severityOrder[b.severity])
  }, [filter])

  const counts = useMemo(() => ({
    all: findings.length,
    critical: findings.filter(f => f.severity === "critical").length,
    high: findings.filter(f => f.severity === "high").length,
    medium: findings.filter(f => f.severity === "medium").length,
    low: findings.filter(f => f.severity === "low").length,
    fixed: findings.filter(f => f.status === "fixed").length,
    wontfix: findings.filter(f => f.status === "wontfix").length,
  }), [])

  const filters = ["all", "critical", "high", "medium", "low", "fixed", "wontfix"]

  return (
    <div style={{ padding: 32, maxWidth: 900, margin: "0 auto" }}>
      <h1 style={{
        fontFamily: "var(--font-heading)",
        fontSize: 28,
        color: "var(--accent)",
        marginBottom: 4,
      }}>
        Code Audit
      </h1>
      <p style={{ color: "var(--text-muted)", fontSize: 14, marginBottom: 28 }}>
        Story Builder — internal code review · {findings.length} findings
      </p>

      <div style={{ display: "flex", gap: 12, marginBottom: 28, flexWrap: "wrap" }}>
        {[
          { key: "critical", label: "Critical", n: counts.critical, c: "var(--error)" },
          { key: "high", label: "High", n: counts.high, c: "#c9734a" },
          { key: "medium", label: "Medium", n: counts.medium, c: "var(--accent)" },
          { key: "low", label: "Low", n: counts.low, c: "var(--text-muted)" },
          { key: "fixed", label: "Fixed", n: counts.fixed, c: "var(--success)" },
          { key: "wontfix", label: "Won't Fix", n: counts.wontfix, c: "var(--text-muted)" },
        ].map(s => (
          <div key={s.key} style={{
            background: "var(--surface)", border: `1px solid var(--border)`, borderRadius: 8,
            padding: "14px 18px", minWidth: 100, flex: 1,
          }}>
            <div style={{ fontSize: 26, fontWeight: 700, fontFamily: "var(--font-heading)", color: s.c }}>{s.n}</div>
            <div style={{ fontSize: 11, color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: "0.06em" }}>{s.label}</div>
          </div>
        ))}
      </div>

      <div style={{ display: "flex", gap: 8, marginBottom: 20, flexWrap: "wrap" }}>
        {filters.map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            style={{
              background: filter === f ? "rgba(212,168,83,0.12)" : "var(--surface)",
              border: `1px solid ${filter === f ? "var(--accent)" : "var(--border)"}`,
              borderRadius: 6, padding: "6px 14px",
              color: filter === f ? "var(--accent)" : "var(--text)",
              cursor: "pointer", fontSize: 13,
              textTransform: "capitalize",
            }}
          >
            {f}
          </button>
        ))}
      </div>

      {filtered.map(f => (
        <div key={f.id} style={{
          background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8,
          padding: "14px 18px", marginBottom: 8, display: "flex", gap: 12,
        }}>
          <span style={{
            width: 10, height: 10, borderRadius: "50%", flexShrink: 0, marginTop: 4,
            background: f.status === "fixed" ? "var(--success)" : "var(--text-muted)",
            boxShadow: f.status === "fixed" ? "0 0 6px var(--success)" : "none",
          }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 3 }}>
              <span style={{ fontWeight: 600, fontSize: 14 }}>{f.id}: {f.title}</span>
              <span style={{
                fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em",
                padding: "1px 7px", borderRadius: 4,
                background: `${severityColors[f.severity]}18`,
                color: severityColors[f.severity],
                border: `1px solid ${severityColors[f.severity]}`,
              }}>
                {f.severity}
              </span>
            </div>
            <p style={{ fontSize: 13, color: "var(--text-muted)", lineHeight: 1.5, margin: 0 }}>{f.desc}</p>
            <div style={{
              marginTop: 6, fontSize: 12, color: "var(--text)",
              background: "rgba(123,184,123,0.08)", borderLeft: "2px solid var(--success)",
              padding: "5px 10px", borderRadius: "0 4px 4px 0",
            }}>
              {f.fix}
            </div>
            <div style={{ marginTop: 6, fontSize: 11, color: "var(--text-muted)", display: "flex", gap: 12 }}>
              <span>File: <code style={{ background: "var(--bg)", padding: "0 5px", borderRadius: 3, color: "var(--accent)" }}>{f.file}</code></span>
              <span>Status: {f.status}</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}