export const TASK_IDS = ["update-dan", "create-ken", "delete-chun-li", "count-ryu"] as const;

export type TaskID = (typeof TASK_IDS)[number];
export type AppKind = "htmx" | "spa";
export type InterfaceKind = "http" | "mcp" | "browser";
export type Capability = InterfaceKind | "computer_use" | "none";
export type AgentKind = "scripted" | "llm";

export type Task = {
  id: TaskID;
  prompt: string;
};

export type Usage = {
  input: number;
  output: number;
  cache_read: number;
};

export type Work = {
  http_requests: number;
  mcp_calls: number;
  playwright_actions: number;
  screenshots: number;
};

export type TrialResult = {
  task: TaskID;
  app: AppKind;
  interface: InterfaceKind;
  success: boolean;
  capability: Capability;
  tokens: Usage;
  cost_usd: number;
  work: Work;
  bytes_in_context: number;
  used_search: boolean;
  answer: number | null;
  agent: AgentKind;
  model: string;
  turns: number;
  error?: string;
  trace: string[];
};

export type Cell = {
  app: AppKind;
  iface: InterfaceKind;
  startPath: string;
};
