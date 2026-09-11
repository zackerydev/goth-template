import { appendFile, mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { startApp } from "./app.ts";
import { runScriptedBrowser } from "./browser-agent.ts";
import { HttpClient, summarizeHTML } from "./http.ts";
import { runScriptedMCP } from "./mcp-agent.ts";
import { checkOracle } from "./oracle.ts";
import { costUSD, emptyUsage } from "./prices.ts";
import { runChatLoop } from "./provider.ts";
import { parseJSONL, writeReport } from "./report.ts";
import { runScriptedHTTP } from "./scripted-http.ts";
import { allTasks, getTask } from "./tasks.ts";
import type { AgentKind, AppKind, Capability, Cell, InterfaceKind, Task, TaskID, TrialResult } from "./types.ts";
import { callContactTool, toolCatalog, type ToolName } from "./mcp-tools.ts";
import { pathAndQuery } from "./html.ts";

const matrixCells: Cell[] = [
  { app: "htmx", iface: "http", startPath: "/contacts" },
  { app: "spa", iface: "http", startPath: "/app/" },
  { app: "spa", iface: "mcp", startPath: "/app/" },
  { app: "spa", iface: "browser", startPath: "/app/" },
];

type Options = {
  app?: AppKind;
  iface?: InterfaceKind;
  task?: TaskID;
  matrix: boolean;
  agent: AgentKind;
  repeats: number;
  model: string;
  includeBrowser: boolean;
  out: string;
  from?: string;
};

async function main(): Promise<void> {
  const options = parseArgs(process.argv.slice(2));
  if (options.from) {
    const results = parseJSONL(await readFile(options.from, "utf8"));
    const htmlPath = await writeReport(results, options.from);
    console.log(`wrote ${htmlPath}`);
    return;
  }
  const repoRoot = path.resolve(import.meta.dirname, "../..");
  const binary = await buildApp(repoRoot);
  const cells = cellsFor(options);
  const tasks = options.task ? [getTask(options.task)] : allTasks();
  await mkdir(path.dirname(options.out), { recursive: true });
  await writeFile(options.out, "");

  const results: TrialResult[] = [];
  for (const cell of cells) {
    for (const task of tasks) {
      for (let repeat = 0; repeat < options.repeats; repeat += 1) {
        const result = await runTrial(repoRoot, binary, cell, task, options);
        results.push(result);
        await appendFile(options.out, `${JSON.stringify(result)}\n`);
        const mark = result.success ? "pass" : "fail";
        console.log(
          `${mark} ${cell.app}×${cell.iface} ${task.id} capability=${result.capability} tokens=${result.tokens.input + result.tokens.output} cost=${result.cost_usd.toFixed(4)}`,
        );
      }
    }
  }
  printSummary(results);
  const htmlPath = await writeReport(results, options.out);
  console.log(`wrote ${htmlPath}`);
  if (options.agent === "scripted" && !scriptedMatrixOK(results, options.includeBrowser)) {
    process.exitCode = 1;
  }
}

async function runTrial(
  repoRoot: string,
  binary: string,
  cell: Cell,
  task: Task,
  options: Options,
): Promise<TrialResult> {
  const app = await startApp(repoRoot, binary);
  try {
    if (cell.iface === "http") {
      return await runHTTPTrial(app.origin, app.database, cell, task, options);
    }
    if (cell.iface === "mcp") {
      return await runMCPTrial(app.origin, app.database, cell, task, options);
    }
    return await runBrowserTrial(app.origin, app.database, cell, task, options);
  } catch (error) {
    const message = error instanceof Error ? `${error.name}: ${error.message}` : String(error);
    return failedTrial(cell, task, options, message);
  } finally {
    await app.stop();
  }
}

async function runHTTPTrial(
  origin: string,
  database: string,
  cell: Cell,
  task: Task,
  options: Options,
): Promise<TrialResult> {
  const client = new HttpClient(origin, cell.startPath);
  let answer: number | null = null;
  let turns = 0;
  let trace: string[] = [];
  let usage = emptyUsage();
  let bytes = 0;
  if (options.agent === "scripted") {
    const outcome = await runScriptedHTTP(client, cell.startPath, task);
    answer = outcome.answer;
    turns = outcome.turns;
    trace = outcome.trace;
    bytes = client.bytesIn;
  } else {
    const first = await client.request("GET", cell.startPath);
    const start = summarizeHTML(first.body, origin);
    const outcome = await runChatLoop({
      model: options.model,
      system: httpSystem(cell.app),
      prompt: `${task.prompt}\nStart URL: ${origin}${cell.startPath}\nFirst page: ${JSON.stringify({
        status: first.status,
        url: pathAndQuery(first.url),
        html: start.html,
        links: start.links,
        forms: start.forms,
      })}`,
      tools: [
        {
          name: "http_request",
          description:
            "Crawl this origin. GET a link, or POST/PUT/PATCH a form from the HTML. fields is the application/x-www-form-urlencoded body: one entry per input name. Omitting fields on a form with named inputs sends an empty body. This is not a JSON API client.",
          parameters: {
            type: "object",
            properties: {
              method: { type: "string" },
              url: { type: "string" },
              fields: { type: "object", additionalProperties: { type: "string" } },
            },
            required: ["method", "url"],
          },
        },
      ],
      callTool: async (name, args) => {
        if (name !== "http_request") {
          return JSON.stringify({ error: "unknown tool" });
        }
        try {
          const method = String(args.method ?? "GET");
          const url = String(args.url ?? cell.startPath);
          const fields = (args.fields ?? undefined) as Record<string, string> | undefined;
          const response = await client.request(method, url, fields);
          const summary = summarizeHTML(response.body, origin);
          return JSON.stringify({
            status: response.status,
            url: pathAndQuery(response.url),
            html: summary.html,
            links: summary.links,
            forms: summary.forms,
          });
        } catch (error) {
          return JSON.stringify({ error: error instanceof Error ? error.message : String(error) });
        }
      },
    });
    answer = outcome.answer;
    turns = outcome.turns;
    trace = outcome.trace;
    usage = outcome.usage;
    bytes = outcome.bytesIn + client.bytesIn;
  }
  return finishTrial(database, cell, task, options, {
    answer,
    turns,
    trace,
    usage,
    bytes,
    usedSearch: client.usedSearch(),
    work: { http_requests: client.requests.length, mcp_calls: 0, playwright_actions: 0, screenshots: 0 },
  });
}

async function runMCPTrial(
  origin: string,
  database: string,
  cell: Cell,
  task: Task,
  options: Options,
): Promise<TrialResult> {
  if (options.agent === "scripted") {
    const outcome = await runScriptedMCP(origin, task);
    return finishTrial(database, cell, task, options, {
      answer: outcome.answer,
      turns: outcome.turns,
      trace: outcome.trace,
      usage: emptyUsage(),
      bytes: outcome.bytesIn,
      usedSearch: outcome.usedSearch,
      work: { http_requests: 0, mcp_calls: outcome.mcpCalls, playwright_actions: 0, screenshots: 0 },
    });
  }
  const outcome = await runChatLoop({
    model: options.model,
    system:
      "You manage contacts through MCP tools only. Use search_contacts for name lookup and counting. Call finish when done.",
    prompt: task.prompt,
    tools: toolCatalog().map((tool) => ({
      name: tool.name,
      description: tool.description,
      parameters: tool.input,
    })),
    callTool: async (name, args) => {
      try {
        const result = await callContactTool(origin, { name: name as ToolName, arguments: args });
        return JSON.stringify(result.body);
      } catch (error) {
        return JSON.stringify({ error: error instanceof Error ? error.message : String(error) });
      }
    },
  });
  return finishTrial(database, cell, task, options, {
    answer: outcome.answer,
    turns: outcome.turns,
    trace: outcome.trace,
    usage: outcome.usage,
    bytes: outcome.bytesIn,
    usedSearch: outcome.trace.some((line) => line.includes("search_contacts")),
    work: { http_requests: 0, mcp_calls: outcome.turns, playwright_actions: 0, screenshots: 0 },
  });
}

async function runBrowserTrial(
  origin: string,
  database: string,
  cell: Cell,
  task: Task,
  options: Options,
): Promise<TrialResult> {
  const outcome = await runScriptedBrowser(origin, task);
  if (outcome.error === "playwright not installed") {
    return {
      ...failedTrial(cell, task, options, outcome.error),
      capability: "none",
    };
  }
  return finishTrial(database, cell, task, options, {
    answer: outcome.answer,
    turns: outcome.turns,
    trace: outcome.trace,
    usage: emptyUsage(),
    bytes: outcome.trace.reduce((sum, line) => sum + line.length, 0),
    usedSearch: outcome.usedSearch,
    work: { http_requests: 0, mcp_calls: 0, playwright_actions: outcome.actions, screenshots: 0 },
    error: outcome.error,
  });
}

function finishTrial(
  database: string,
  cell: Cell,
  task: Task,
  options: Options,
  parts: {
    answer: number | null;
    turns: number;
    trace: string[];
    usage: ReturnType<typeof emptyUsage>;
    bytes: number;
    usedSearch: boolean;
    work: TrialResult["work"];
    error?: string;
  },
): TrialResult {
  const oracle = checkOracle(database, task.id, parts.answer, task.id === "count-ryu" ? parts.usedSearch : true);
  const success = oracle.ok && !parts.error;
  return {
    task: task.id,
    app: cell.app,
    interface: cell.iface,
    success,
    capability: success ? cell.iface : "none",
    tokens: parts.usage,
    cost_usd: Number(costUSD(options.agent === "scripted" ? "scripted" : options.model, parts.usage).toFixed(6)),
    work: parts.work,
    bytes_in_context: parts.bytes,
    used_search: parts.usedSearch,
    answer: parts.answer,
    agent: options.agent,
    model: options.agent === "scripted" ? "scripted" : options.model,
    turns: parts.turns,
    error: parts.error ?? (success ? undefined : oracle.reason),
    trace: parts.trace,
  };
}

function failedTrial(cell: Cell, task: Task, options: Options, error: string): TrialResult {
  return {
    task: task.id,
    app: cell.app,
    interface: cell.iface,
    success: false,
    capability: "none",
    tokens: emptyUsage(),
    cost_usd: 0,
    work: { http_requests: 0, mcp_calls: 0, playwright_actions: 0, screenshots: 0 },
    bytes_in_context: 0,
    used_search: false,
    answer: null,
    agent: options.agent,
    model: options.agent === "scripted" ? "scripted" : options.model,
    turns: 0,
    error,
    trace: [error],
  };
}

function httpSystem(app: AppKind): string {
  const shell =
    app === "spa"
      ? "This start document may be an empty client-rendered shell with no links or forms. If so, you cannot finish over HTTP."
      : "This app is hypermedia: the HTML is the interface. Follow <a href> and submit <form> controls.";
  return [
    "You crawl a website with GET and form POST. No JavaScript.",
    "The page is the API: named inputs on a form are the fields you must send when you POST that form.",
    "Copy existing values on edit forms and change what the task requires. On an empty create form, fill the names from the task (choose any valid email).",
    "Do not invent JSON API paths unless that URL appears as href, form action, or hx-* in HTML you fetched.",
    "Stay on origin. Script tags are not next actions.",
    shell,
  ].join(" ");
}

function cellsFor(options: Options): Cell[] {
  let cells = matrixCells.filter((cell) => options.includeBrowser || cell.iface !== "browser");
  if (options.app) {
    cells = cells.filter((cell) => cell.app === options.app);
  }
  if (options.iface) {
    cells = cells.filter((cell) => cell.iface === options.iface);
  }
  if (!options.matrix && (options.app || options.iface || options.task)) {
    return cells;
  }
  return cells;
}

function parseArgs(argv: string[]): Options {
  const agent = (flag(argv, "--agent") as AgentKind) || "scripted";
  const app = flag(argv, "--app") as AppKind | undefined;
  const iface = flag(argv, "--interface") as InterfaceKind | undefined;
  const task = flag(argv, "--task") as TaskID | undefined;
  const includeBrowser = argv.includes("--browser") || iface === "browser";
  const explicit = Boolean(app || iface || task);
  return {
    app,
    iface,
    task,
    matrix: argv.includes("--matrix") || !explicit,
    agent,
    repeats: Number(flag(argv, "--repeats") ?? 1),
    model: flag(argv, "--model") ?? (agent === "llm" ? "kimi-k3" : "gpt-4o-mini"),
    includeBrowser,
    out: flag(argv, "--out") ?? defaultOut(),
    from: flag(argv, "--from"),
  };
}

function flag(argv: string[], name: string): string | undefined {
  const index = argv.indexOf(name);
  if (index === -1) {
    return undefined;
  }
  return argv[index + 1];
}

function defaultOut(): string {
  const stamp = new Date().toISOString().replaceAll(":", "-");
  return path.resolve(import.meta.dirname, `../results/${stamp}.jsonl`);
}

async function buildApp(repoRoot: string): Promise<string> {
  await mkdir(path.join(repoRoot, "tmp"), { recursive: true });
  const binary = path.join(repoRoot, "tmp", "eval-app");
  const built = spawnSync("go", ["build", "-o", binary, "./cmd/app"], { cwd: repoRoot, encoding: "utf8" });
  if (built.status !== 0) {
    throw new Error(built.stderr || built.stdout || "go build failed");
  }
  return binary;
}

function printSummary(results: TrialResult[]): void {
  console.log("\ncapability matrix");
  for (const result of results) {
    const cap: Capability = result.capability;
    console.log(`- ${result.app} ${result.interface} ${result.task}: success=${result.success} capability=${cap} http=${result.work.http_requests} mcp=${result.work.mcp_calls} browser=${result.work.playwright_actions} tokens=${result.tokens.input + result.tokens.output} cost_usd=${result.cost_usd}`);
  }
}

function scriptedMatrixOK(results: TrialResult[], includeBrowser: boolean): boolean {
  const want: Record<string, boolean> = {
    "htmx/http": true,
    "spa/http": false,
    "spa/mcp": true,
  };
  if (includeBrowser) {
    want["spa/browser"] = true;
  }
  for (const result of results) {
    const key = `${result.app}/${result.interface}`;
    if (!(key in want)) {
      continue;
    }
    if (key === "spa/browser" && result.error?.includes("playwright not installed")) {
      continue;
    }
    if (result.success !== want[key]) {
      console.error(`expected ${key} ${result.task} success=${want[key]}, got ${result.success} (${result.error ?? ""})`);
      return false;
    }
  }
  return true;
}

await main();
