import { callContactTool, toolCatalog, type ToolCall, type ToolName } from "./mcp-tools.ts";

type RPC = {
  jsonrpc: "2.0";
  id?: number | string | null;
  method?: string;
  params?: Record<string, unknown>;
};

export async function serveMCP(baseURL: string): Promise<void> {
  let buffer = "";
  process.stdin.setEncoding("utf8");
  for await (const chunk of process.stdin) {
    buffer += chunk;
    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";
    for (const line of lines) {
      if (!line.trim()) {
        continue;
      }
      const message = JSON.parse(line) as RPC;
      const response = await handle(baseURL, message);
      if (response) {
        process.stdout.write(`${JSON.stringify(response)}\n`);
      }
    }
  }
}

async function handle(baseURL: string, message: RPC): Promise<unknown> {
  switch (message.method) {
    case "initialize":
      return result(message.id, {
        protocolVersion: "2024-11-05",
        capabilities: { tools: {} },
        serverInfo: { name: "contacts", version: "1.0.0" },
      });
    case "notifications/initialized":
      return null;
    case "tools/list":
      return result(message.id, {
        tools: toolCatalog().map((tool) => ({
          name: tool.name,
          description: tool.description,
          inputSchema: tool.input,
        })),
      });
    case "tools/call":
      return result(message.id, await callTool(baseURL, message.params ?? {}));
    default:
      return {
        jsonrpc: "2.0",
        id: message.id ?? null,
        error: { code: -32601, message: `unknown method ${message.method}` },
      };
  }
}

async function callTool(baseURL: string, params: Record<string, unknown>): Promise<unknown> {
  const name = String(params.name ?? "") as ToolName;
  const args = (params.arguments ?? {}) as ToolCall["arguments"];
  const output = await callContactTool(baseURL, { name, arguments: args });
  return {
    content: [{ type: "text", text: JSON.stringify(output.body) }],
    isError: !output.ok,
  };
}

function result(id: RPC["id"], value: unknown): unknown {
  return { jsonrpc: "2.0", id: id ?? null, result: value };
}

const baseURL = process.env.BASE_URL ?? process.env.APP_URL;
if (process.argv.includes("--stdio")) {
  if (!baseURL) {
    throw new Error("BASE_URL is required");
  }
  void serveMCP(baseURL);
}
