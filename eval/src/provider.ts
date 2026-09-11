import { spawnSync } from "node:child_process";
import { addUsage, emptyUsage } from "./prices.ts";
import type { Usage } from "./types.ts";

export type ChatTool = {
  name: string;
  description: string;
  parameters: Record<string, unknown>;
};

export type ChatLoop = {
  system: string;
  prompt: string;
  tools: ChatTool[];
  model: string;
  maxTurns?: number;
  callTool: (name: string, args: Record<string, unknown>) => Promise<string>;
};

export type ChatLoopResult = {
  text: string;
  answer: number | null;
  turns: number;
  bytesIn: number;
  trace: string[];
  usage: Usage;
};

type ToolCall = { id: string; name: string; arguments: Record<string, unknown> };

type Endpoint = {
  url: string;
  key: string;
  model: string;
  session: string;
};

export async function runChatLoop(input: ChatLoop): Promise<ChatLoopResult> {
  const endpoint = resolveEndpoint(input.model);
  const messages: Record<string, unknown>[] = [
    { role: "system", content: input.system },
    { role: "user", content: input.prompt },
  ];
  const tools = [
    ...input.tools.map((tool) => ({
      type: "function",
      function: { name: tool.name, description: tool.description, parameters: tool.parameters },
    })),
    {
      type: "function",
      function: {
        name: "finish",
        description: "End the task. For count questions, set answer to the integer.",
        parameters: {
          type: "object",
          properties: {
            success: { type: "boolean" },
            answer: { type: "integer" },
          },
        },
      },
    },
  ];
  const trace: string[] = [];
  let usage = emptyUsage();
  let bytesIn = input.system.length + input.prompt.length;
  const maxTurns = input.maxTurns ?? 20;

  for (let turn = 1; turn <= maxTurns; turn += 1) {
    const completion = await chatCompletions(endpoint, messages, tools);
    usage = addUsage(usage, completion.usage);
    bytesIn += JSON.stringify(completion.message).length;
    const calls = completion.toolCalls;
    if (calls.length === 0) {
      const text = String(completion.message.content ?? "");
      trace.push(text);
      return { text, answer: parseAnswer(text), turns: turn, bytesIn, trace, usage };
    }
    messages.push(completion.message);
    for (const call of calls) {
      trace.push(`${call.name} ${JSON.stringify(call.arguments)}`);
      if (call.name === "finish") {
        const answer =
          typeof call.arguments.answer === "number"
            ? call.arguments.answer
            : parseAnswer(String(call.arguments.answer ?? ""));
        return { text: "finish", answer, turns: turn, bytesIn, trace, usage };
      }
      const output = await input.callTool(call.name, call.arguments);
      bytesIn += output.length;
      messages.push({ role: "tool", tool_call_id: call.id, content: output });
    }
  }
  return { text: "budget exceeded", answer: null, turns: maxTurns, bytesIn, trace, usage };
}

function resolveEndpoint(model: string): Endpoint {
  const modelId = model.replace(/^opencode-go\//, "").replace(/:.*$/, "");
  if (usesPi(model)) {
    return {
      url: "https://opencode.ai/zen/go/v1/chat/completions",
      key: piAPIKey("opencode-go"),
      model: modelId,
      session: crypto.randomUUID(),
    };
  }
  const key = process.env.OPENAI_API_KEY;
  if (!key) {
    throw new Error("OPENAI_API_KEY is not set (or use --model kimi-k3 with pi)");
  }
  return {
    url: "https://api.openai.com/v1/chat/completions",
    key,
    model: modelId,
    session: crypto.randomUUID(),
  };
}

function usesPi(model: string): boolean {
  return model.includes("kimi") || model.startsWith("opencode-go/") || process.env.EVAL_PROVIDER === "pi";
}

function piAPIKey(provider: string): string {
  const result = spawnSync("pi", ["auth", "print-api-key", "--provider", provider], {
    encoding: "utf8",
  });
  const key = result.stdout.trim();
  if (result.status !== 0 || !key) {
    throw new Error(result.stderr.trim() || `pi auth print-api-key failed for ${provider}`);
  }
  return key;
}

async function chatCompletions(
  endpoint: Endpoint,
  messages: Record<string, unknown>[],
  tools: unknown[],
): Promise<{ message: Record<string, unknown>; toolCalls: ToolCall[]; usage: Usage }> {
  const response = await fetch(endpoint.url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${endpoint.key}`,
      "Content-Type": "application/json",
      "User-Agent": "contacts-agent-eval/1.0",
      "x-opencode-session": endpoint.session,
    },
    body: JSON.stringify({
      model: endpoint.model,
      temperature: 0,
      max_tokens: 4096,
      messages,
      tools,
      tool_choice: "auto",
    }),
  });
  const payload = (await response.json()) as {
    error?: { message?: string };
    usage?: {
      prompt_tokens?: number;
      completion_tokens?: number;
      input_tokens?: number;
      output_tokens?: number;
      prompt_tokens_details?: { cached_tokens?: number };
      cache_read_input_tokens?: number;
    };
    choices?: {
      message?: {
        content?: string | null;
        tool_calls?: { id?: string; function?: { name?: string; arguments?: string } }[];
      };
    }[];
  };
  if (!response.ok) {
    throw new Error(payload.error?.message ?? `chat completions ${response.status}`);
  }
  const message = payload.choices?.[0]?.message ?? {};
  const toolCalls: ToolCall[] = (message.tool_calls ?? []).map((call, index) => ({
    id: call.id ?? `call_${index}`,
    name: call.function?.name ?? "",
    arguments: parseArgs(call.function?.arguments),
  }));
  return {
    message,
    toolCalls,
    usage: {
      input: payload.usage?.prompt_tokens ?? payload.usage?.input_tokens ?? 0,
      output: payload.usage?.completion_tokens ?? payload.usage?.output_tokens ?? 0,
      cache_read: payload.usage?.prompt_tokens_details?.cached_tokens ?? payload.usage?.cache_read_input_tokens ?? 0,
    },
  };
}

function parseArgs(raw: string | undefined): Record<string, unknown> {
  try {
    return JSON.parse(raw || "{}") as Record<string, unknown>;
  } catch {
    return {};
  }
}

function parseAnswer(text: string): number | null {
  const match = text.match(/-?\d+/);
  return match ? Number(match[0]) : null;
}
