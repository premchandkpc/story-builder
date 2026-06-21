import type { SimulationResult } from "headroom-ai"

const HEADROOM_BASE_URL = import.meta.env.VITE_HEADROOM_BASE_URL || ""

export interface CompressStats {
  enabled: boolean
  tokensBefore: number
  tokensAfter: number
  tokensSaved: number
  compressionRatio: number
  transforms: string[]
  wasteSignals: Record<string, number>
}

function parseSimulationResult(sim: SimulationResult): CompressStats {
  return {
    enabled: true,
    tokensBefore: sim.tokensBefore,
    tokensAfter: sim.tokensAfter,
    tokensSaved: sim.tokensSaved,
    compressionRatio: sim.tokensAfter / sim.tokensBefore,
    transforms: sim.transforms,
    wasteSignals: sim.wasteSignals,
  }
}

export async function simulateCompression(
  system: string,
  userMessage: string,
  model: string,
): Promise<CompressStats | null> {
  if (!HEADROOM_BASE_URL) return null

  try {
    const { simulate } = await import("headroom-ai")
    const messages: { role: string; content: string }[] = []
    if (system) messages.push({ role: "system", content: system })
    messages.push({ role: "user", content: userMessage })

    const result = await simulate(messages, {
      model,
      baseUrl: HEADROOM_BASE_URL,
    })
    return parseSimulationResult(result)
  } catch {
    return null
  }
}
