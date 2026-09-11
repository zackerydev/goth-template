import type { Usage } from "./types.ts";

const prices: Record<string, { input: number; output: number; cache: number }> = {
  "gpt-4o-mini": { input: 0.15, output: 0.6, cache: 0.075 },
  "gpt-4o": { input: 2.5, output: 10, cache: 1.25 },
  "claude-haiku-4-5": { input: 1, output: 5, cache: 0.1 },
  "claude-sonnet-4-5": { input: 3, output: 15, cache: 0.3 },
  "kimi-k3": { input: 3, output: 15, cache: 0.3 },
  scripted: { input: 0, output: 0, cache: 0 },
};

export function costUSD(model: string, usage: Usage): number {
  const table = prices[priceKey(model)] ?? prices["kimi-k3"] ?? { input: 3, output: 15, cache: 0.3 };
  return (usage.input * table.input + usage.output * table.output + usage.cache_read * table.cache) / 1_000_000;
}

function priceKey(model: string): string {
  return model.replace(/^opencode-go\//, "").replace(/:.*$/, "");
}

export function emptyUsage(): Usage {
  return { input: 0, output: 0, cache_read: 0 };
}

export function addUsage(total: Usage, next: Usage): Usage {
  return {
    input: total.input + next.input,
    output: total.output + next.output,
    cache_read: total.cache_read + next.cache_read,
  };
}
