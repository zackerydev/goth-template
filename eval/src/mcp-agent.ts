import { callContactTool, type ToolCall, type ToolName } from "./mcp-tools.ts";
import type { Task } from "./types.ts";

export type MCPOutcome = {
  answer: number | null;
  turns: number;
  usedSearch: boolean;
  mcpCalls: number;
  bytesIn: number;
  trace: string[];
};

export async function runScriptedMCP(baseURL: string, task: Task): Promise<MCPOutcome> {
  const calls: ToolCall[] = [];
  const trace: string[] = [];
  let usedSearch = false;
  let answer: number | null = null;

  switch (task.id) {
    case "update-dan": {
      usedSearch = true;
      const found = await invoke(baseURL, { name: "search_contacts", arguments: { query: "Dan" } }, calls, trace);
      const dan = firstHit(found, "Dan");
      if (!dan) {
        break;
      }
      await invoke(
        baseURL,
        {
          name: "update_contact",
          arguments: { id: dan.id, first: dan.first, last: "Hibiki", phone: dan.phone, email: dan.email },
        },
        calls,
        trace,
      );
      break;
    }
    case "create-ken":
      await invoke(
        baseURL,
        {
          name: "create_contact",
          arguments: { first: "Ken", last: "Masters", phone: "555-0199", email: "ken.masters@eval.test" },
        },
        calls,
        trace,
      );
      break;
    case "delete-chun-li": {
      usedSearch = true;
      const found = await invoke(baseURL, { name: "search_contacts", arguments: { query: "Chun-Li" } }, calls, trace);
      const chun = firstHit(found, "Chun-Li");
      if (chun) {
        await invoke(baseURL, { name: "delete_contact", arguments: { id: chun.id } }, calls, trace);
      }
      break;
    }
    case "count-ryu": {
      usedSearch = true;
      const found = await invoke(baseURL, { name: "search_contacts", arguments: { query: "Ryu" } }, calls, trace);
      answer = countNamed(found, "Ryu");
      break;
    }
  }

  const bytesIn = trace.reduce((sum, line) => sum + line.length, 0);
  return { answer, turns: calls.length, usedSearch, mcpCalls: calls.length, bytesIn, trace };
}

async function invoke(baseURL: string, call: ToolCall, calls: ToolCall[], trace: string[]): Promise<unknown> {
  calls.push(call);
  const result = await callContactTool(baseURL, call);
  trace.push(`${call.name} ${JSON.stringify(call.arguments)} -> ${JSON.stringify(result.body)}`);
  return result.body;
}

function firstHit(body: unknown, name: string): { id: number; first: string; last: string; phone: string; email: string } | undefined {
  const contacts = contactsOf(body);
  const needle = name.toLowerCase();
  return contacts.find((item) => `${item.first} ${item.last}`.toLowerCase().includes(needle));
}

function countNamed(body: unknown, name: string): number {
  const needle = name.toLowerCase();
  return contactsOf(body).filter((item) => item.first.toLowerCase().includes(needle) || item.last.toLowerCase().includes(needle)).length;
}

function contactsOf(body: unknown): { id: number; first: string; last: string; phone: string; email: string }[] {
  if (body && typeof body === "object" && "contacts" in body && Array.isArray(body.contacts)) {
    return body.contacts as { id: number; first: string; last: string; phone: string; email: string }[];
  }
  return [];
}

export async function runMCPLLM(
  baseURL: string,
  task: Task,
  complete: (input: LLMToolLoop) => Promise<LLMToolLoopResult>,
): Promise<MCPOutcome> {
  const tools = (await import("./mcp-tools.ts")).toolCatalog();
  const result = await complete({
    system:
      "You manage an address book through MCP tools only: list_contacts, search_contacts, get_contact, create_contact, update_contact, delete_contact. Use search_contacts for name questions. When finished, call finish.",
    prompt: task.prompt,
    tools: tools.map((tool) => ({ name: tool.name, description: tool.description, parameters: tool.input })),
    callTool: async (name, args) => {
      const output = await callContactTool(baseURL, { name: name as ToolName, arguments: args });
      return JSON.stringify(output.body);
    },
  });
  return {
    answer: result.answer,
    turns: result.turns,
    usedSearch: result.trace.some((line) => line.includes("search_contacts")),
    mcpCalls: result.turns,
    bytesIn: result.bytesIn,
    trace: result.trace,
  };
}

export type LLMToolLoop = {
  system: string;
  prompt: string;
  tools: { name: string; description: string; parameters: Record<string, unknown> }[];
  callTool: (name: string, args: Record<string, unknown>) => Promise<string>;
};

export type LLMToolLoopResult = {
  answer: number | null;
  turns: number;
  bytesIn: number;
  trace: string[];
  usage: { input: number; output: number; cache_read: number };
};
