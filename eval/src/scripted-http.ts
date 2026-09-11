import { countNamedRows, parseActions, parseTableRows, pathAndQuery } from "./html.ts";
import { HttpClient } from "./http.ts";
import type { Task, TaskID } from "./types.ts";

export type ScriptedOutcome = {
  answer: number | null;
  turns: number;
  trace: string[];
};

export async function runScriptedHTTP(client: HttpClient, startPath: string, task: Task): Promise<ScriptedOutcome> {
  const first = await client.request("GET", startPath);
  const trace = [`GET ${startPath} -> ${first.status}`];
  if (task.id !== "count-ryu" && !parseActions(first.body, client.origin).some((action) => action.kind === "form" || action.text)) {
    if (!hasHypermedia(first.body)) {
      return { answer: null, turns: 1, trace: [...trace, "no HTML forms or links to follow"] };
    }
  }
  if (!hasHypermedia(first.body) && task.id !== "count-ryu") {
    return { answer: null, turns: 1, trace: [...trace, "SPA shell has no hypermedia controls"] };
  }
  if (!hasHypermedia(first.body) && task.id === "count-ryu") {
    return { answer: null, turns: 1, trace: [...trace, "SPA shell has no search form"] };
  }
  return runHTMXTask(client, task.id, trace);
}

function hasHypermedia(html: string): boolean {
  return /<form\b/i.test(html) || /<a\b[^>]*href="\/contacts/i.test(html);
}

async function runHTMXTask(client: HttpClient, task: TaskID, trace: string[]): Promise<ScriptedOutcome> {
  switch (task) {
    case "update-dan":
      return updateDan(client, trace);
    case "create-ken":
      return createKen(client, trace);
    case "delete-chun-li":
      return deleteChunLi(client, trace);
    case "count-ryu":
      return countRyu(client, trace);
  }
}

async function updateDan(client: HttpClient, trace: string[]): Promise<ScriptedOutcome> {
  const search = await client.request("GET", "/contacts", { q: "Dan" });
  trace.push(`GET /contacts?q=Dan -> ${search.status}`);
  const edit = rowAction(search.body, "Dan", "/edit");
  if (!edit) {
    return { answer: null, turns: trace.length, trace: [...trace, "Dan edit link missing"] };
  }
  const page = await client.request("GET", edit);
  trace.push(`GET ${pathAndQuery(edit)} -> ${page.status}`);
  const form = parseActions(page.body, client.origin).find((action) => action.method === "POST" && action.url.includes("/edit"));
  if (!form?.fields) {
    return { answer: null, turns: trace.length, trace: [...trace, "edit form missing"] };
  }
  const saved = await client.request("POST", form.url, { ...form.fields, last_name: "Hibiki" });
  trace.push(`POST ${pathAndQuery(form.url)} -> ${saved.status}`);
  return { answer: null, turns: trace.length, trace };
}

async function createKen(client: HttpClient, trace: string[]): Promise<ScriptedOutcome> {
  const list = await client.request("GET", "/contacts");
  const add = parseActions(list.body, client.origin).find((action) => /add contact/i.test(action.text));
  if (!add) {
    return { answer: null, turns: trace.length, trace: [...trace, "Add Contact missing"] };
  }
  const page = await client.request("GET", add.url);
  trace.push(`GET ${pathAndQuery(add.url)} -> ${page.status}`);
  const form = parseActions(page.body, client.origin).find((action) => action.method === "POST");
  if (!form) {
    return { answer: null, turns: trace.length, trace: [...trace, "create form missing"] };
  }
  const saved = await client.request("POST", form.url, {
    first_name: "Ken",
    last_name: "Masters",
    phone: "555-0199",
    email: "ken.masters@eval.test",
  });
  trace.push(`POST ${pathAndQuery(form.url)} -> ${saved.status}`);
  return { answer: null, turns: trace.length, trace };
}

async function deleteChunLi(client: HttpClient, trace: string[]): Promise<ScriptedOutcome> {
  const search = await client.request("GET", "/contacts", { q: "Chun-Li" });
  trace.push(`GET /contacts?q=Chun-Li -> ${search.status}`);
  const edit = rowAction(search.body, "Chun-Li", "/edit") ?? rowAction(search.body, "Chun-Li", "");
  if (!edit) {
    return { answer: null, turns: trace.length, trace: [...trace, "Chun-Li link missing"] };
  }
  const page = await client.request("GET", edit);
  trace.push(`GET ${pathAndQuery(edit)} -> ${page.status}`);
  const destroy = parseActions(page.body, client.origin).find(
    (action) => action.method === "POST" && action.url.includes("/delete"),
  );
  const hx = parseActions(page.body, client.origin).find((action) => action.method === "DELETE");
  const target = destroy ?? hx;
  if (!target) {
    return { answer: null, turns: trace.length, trace: [...trace, "delete control missing"] };
  }
  const deleted = await client.request(target.method, target.url, target.fields);
  trace.push(`${target.method} ${pathAndQuery(target.url)} -> ${deleted.status}`);
  return { answer: null, turns: trace.length, trace };
}

async function countRyu(client: HttpClient, trace: string[]): Promise<ScriptedOutcome> {
  const search = await client.request("GET", "/contacts", { q: "Ryu" });
  trace.push(`GET /contacts?q=Ryu -> ${search.status}`);
  const answer = countNamedRows(search.body, "Ryu");
  trace.push(`counted ${answer} Ryu rows`);
  return { answer, turns: trace.length, trace };
}

function rowAction(html: string, name: string, suffix: string): string | undefined {
  const needle = name.toLowerCase();
  for (const row of parseTableRows(html)) {
    if (!row.text.toLowerCase().includes(needle)) {
      continue;
    }
    const href = row.hrefs.find((item) => (suffix ? item.includes(suffix) : true));
    if (href) {
      return href;
    }
  }
  return undefined;
}
