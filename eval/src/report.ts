import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { TASK_IDS, type TrialResult } from "./types.ts";

export async function writeReport(results: TrialResult[], jsonlPath: string): Promise<string> {
  const htmlPath = jsonlPath.replace(/\.jsonl$/i, ".html");
  const html = renderReport(results, {
    generatedAt: new Date().toISOString(),
    source: path.basename(jsonlPath),
  });
  await mkdir(path.dirname(htmlPath), { recursive: true });
  await writeFile(htmlPath, html);
  await writeFile(path.join(path.dirname(htmlPath), "index.html"), html);
  return htmlPath;
}

export function parseJSONL(text: string): TrialResult[] {
  const rows: TrialResult[] = [];
  for (const line of text.split("\n")) {
    if (!line.trim()) {
      continue;
    }
    rows.push(JSON.parse(line) as TrialResult);
  }
  return rows;
}

export function renderReport(results: TrialResult[], meta: { generatedAt: string; source: string }): string {
  const cells = cellKeys(results);
  const agent = results[0]?.agent ?? "unknown";
  const model = results[0]?.model ?? "unknown";
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Agent eval readout</title>
    <style>${css()}</style>
  </head>
  <body>
    <main>
      <p class="eyebrow">contacts.app agent eval</p>
      <h1>HTMX vs SPA vs SPA+MCP</h1>
      <p class="lede">
        A hypermedia UI is already a tool surface. Capability is the score:
        the cheapest interface that finished the task.
      </p>
      <p class="meta">
        ${escape(results.length)} trials · ${escape(agent)} / ${escape(model)} ·
        ${escape(meta.generatedAt)} · ${escape(meta.source)}
      </p>
      <h2>Capability matrix</h2>
      ${capabilityTable(results, cells)}
      <h2>Work and cost</h2>
      ${workTable(results, cells)}
      <h2>Trials</h2>
      ${trialTable(results)}
    </main>
  </body>
</html>
`;
}

function capabilityTable(results: TrialResult[], cells: string[]): string {
  const rows = TASK_IDS.map((task) => {
    const cellsHtml = cells
      .map((cell) => {
        const trials = results.filter((item) => key(item) === cell && item.task === task);
        if (trials.length === 0) {
          return `<td class="empty">—</td>`;
        }
        const passed = trials.filter((item) => item.success).length;
        const cap = trials[0]?.capability ?? "none";
        const klass = passed === trials.length ? "pass" : passed === 0 ? "fail" : "mixed";
        return `<td class="${klass}"><strong>${passed === trials.length ? "pass" : "fail"}</strong><span>${escape(cap)}</span></td>`;
      })
      .join("");
    return `<tr><th>${escape(task)}</th>${cellsHtml}</tr>`;
  }).join("");
  return `<table class="matrix">
    <thead><tr><th>Task</th>${cells.map((cell) => `<th>${escape(cell)}</th>`).join("")}</tr></thead>
    <tbody>${rows}</tbody>
  </table>`;
}

function workTable(results: TrialResult[], cells: string[]): string {
  const rows = cells
    .map((cell) => {
      const trials = results.filter((item) => key(item) === cell);
      const http = sum(trials, (item) => item.work.http_requests);
      const mcp = sum(trials, (item) => item.work.mcp_calls);
      const browser = sum(trials, (item) => item.work.playwright_actions);
      const tokens = sum(trials, (item) => item.tokens.input + item.tokens.output);
      const cost = trials.reduce((total, item) => total + item.cost_usd, 0);
      const bytes = sum(trials, (item) => item.bytes_in_context);
      const passed = trials.filter((item) => item.success).length;
      return `<tr>
        <th>${escape(cell)}</th>
        <td>${passed}/${trials.length}</td>
        <td>${http}</td>
        <td>${mcp}</td>
        <td>${browser}</td>
        <td>${tokens}</td>
        <td>${cost.toFixed(4)}</td>
        <td>${bytes}</td>
      </tr>`;
    })
    .join("");
  return `<table>
    <thead>
      <tr>
        <th>Cell</th><th>Pass</th><th>HTTP</th><th>MCP</th><th>Browser</th>
        <th>Tokens</th><th>USD</th><th>Bytes in context</th>
      </tr>
    </thead>
    <tbody>${rows}</tbody>
  </table>`;
}

function trialTable(results: TrialResult[]): string {
  const rows = results
    .map((item) => {
      const mark = item.success ? "pass" : "fail";
      const trace = item.trace.map((line) => `<li>${escape(line)}</li>`).join("");
      const error = item.error ? `<p class="error">${escape(item.error)}</p>` : "";
      return `<tr class="${mark}">
        <td>${escape(item.app)}×${escape(item.interface)}</td>
        <td>${escape(item.task)}</td>
        <td>${mark}</td>
        <td>${escape(item.capability)}</td>
        <td>${item.work.http_requests}/${item.work.mcp_calls}/${item.work.playwright_actions}</td>
        <td>${item.cost_usd.toFixed(4)}</td>
        <td>${item.tokens.input + item.tokens.output}</td>
        <td>
          ${error}
          <details><summary>${item.turns} turns</summary><ol>${trace}</ol></details>
        </td>
      </tr>`;
    })
    .join("");
  return `<table>
    <thead>
      <tr>
        <th>Cell</th><th>Task</th><th>Success</th><th>Capability</th>
        <th>HTTP/MCP/Browser</th><th>USD</th><th>Tokens</th><th>Trace</th>
      </tr>
    </thead>
    <tbody>${rows}</tbody>
  </table>`;
}

function cellKeys(results: TrialResult[]): string[] {
  const seen = new Set<string>();
  const order: string[] = [];
  for (const item of results) {
    const name = key(item);
    if (seen.has(name)) {
      continue;
    }
    seen.add(name);
    order.push(name);
  }
  return order;
}

function key(item: TrialResult): string {
  return `${item.app}×${item.interface}`;
}

function sum(results: TrialResult[], value: (item: TrialResult) => number): number {
  return results.reduce((total, item) => total + value(item), 0);
}

function escape(value: string | number): string {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function css(): string {
  return `
    :root { color-scheme: light; --bg:#f5f5f2; --fg:#1d211f; --muted:#5d6661; --accent:#214e3b; --border:#c9cecb; --pass:#e7efe9; --fail:#f6e8e8; }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--bg); color: var(--fg); font: 1rem/1.5 system-ui, sans-serif; }
    main { inline-size: min(100% - 2rem, 70rem); margin: 0 auto; padding: 2rem 0 4rem; }
    h1, h2 { line-height: 1.2; }
    h1 { font-size: 1.75rem; }
    .eyebrow { text-transform: uppercase; letter-spacing: 0.08em; color: var(--accent); font-size: 0.8rem; }
    .lede { max-inline-size: 42rem; }
    .meta { color: var(--muted); }
    table { width: 100%; border-collapse: collapse; background: #fff; margin: 1rem 0 2rem; }
    th, td { border: 1px solid var(--border); padding: 0.5rem 0.75rem; text-align: left; vertical-align: top; }
    thead th { background: #eef2ef; }
    td.pass { background: var(--pass); }
    td.fail, tr.fail td:nth-child(3) { background: var(--fail); }
    td.mixed { background: #f7f1e3; }
    td.empty { color: var(--muted); }
    td.pass span, td.fail span, td.mixed span { display: block; color: var(--muted); font-size: 0.85rem; }
    .error { color: #8b2e2e; margin: 0 0 0.5rem; }
    details { font-size: 0.9rem; }
    ol { margin: 0.5rem 0 0; padding-inline-start: 1.2rem; }
  `;
}
