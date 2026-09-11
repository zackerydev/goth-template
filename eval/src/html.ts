export type DiscoveredAction = {
  kind: "link" | "form" | "hx";
  method: string;
  url: string;
  text: string;
  fields?: Record<string, string>;
};

export function extractMain(html: string): string {
  const main = html.match(/<main\b[^>]*id="main-content"[^>]*>([\s\S]*?)<\/main>/i);
  if (main?.[1]) {
    return main[1];
  }
  const root = html.match(/<div id="root"[^>]*>[\s\S]*?<\/div>/i);
  if (root?.[0]) {
    return root[0];
  }
  const body = html.match(/<body\b[^>]*>([\s\S]*?)<\/body>/i);
  return body?.[1] ?? html;
}

export type CrawlerPage = {
  html: string;
  links: { text: string; url: string }[];
  forms: { method: string; url: string; fields: Record<string, string> }[];
};

export function parseActions(html: string, origin: string): DiscoveredAction[] {
  const fragment = extractMain(html);
  const actions: DiscoveredAction[] = [];
  actions.push(...parseLinks(fragment, origin));
  actions.push(...parseForms(fragment, origin));
  actions.push(...parseHx(fragment, origin));
  return uniqueActions(actions);
}

export function crawlerPage(html: string, origin: string): CrawlerPage {
  const fragment = extractMain(html);
  const links = parseLinks(fragment, origin).map((item) => ({ text: item.text, url: item.url }));
  const forms = parseForms(fragment, origin).map((item) => ({
    method: item.method,
    url: item.url,
    fields: item.fields ?? {},
  }));
  for (const item of parseHx(fragment, origin)) {
    if (item.method === "GET") {
      continue;
    }
    if (forms.some((form) => form.method === item.method && form.url === item.url)) {
      continue;
    }
    forms.push({ method: item.method, url: item.url, fields: {} });
  }
  return { html: fragment, links, forms };
}

export function parseTableRows(html: string): { text: string; hrefs: string[] }[] {
  const fragment = extractMain(html);
  const rows: { text: string; hrefs: string[] }[] = [];
  const rowRe = /<tr\b[^>]*>([\s\S]*?)<\/tr>/gi;
  let match: RegExpExecArray | null;
  while ((match = rowRe.exec(fragment))) {
    const body = match[1] ?? "";
    const hrefs: string[] = [];
    const hrefRe = /href="([^"]+)"/gi;
    let href: RegExpExecArray | null;
    while ((href = hrefRe.exec(body))) {
      hrefs.push(decode(href[1] ?? ""));
    }
    rows.push({ text: decode(stripTags(body)), hrefs });
  }
  return rows;
}

export function countNamedRows(html: string, name: string): number {
  const needle = name.toLowerCase();
  return parseTableRows(html).filter((row) => {
    const text = row.text.toLowerCase();
    return text.includes(needle) && !text.includes("load more") && !text.includes("no contacts");
  }).length;
}

function parseLinks(html: string, origin: string): DiscoveredAction[] {
  const actions: DiscoveredAction[] = [];
  const re = /<a\b([^>]*)>([\s\S]*?)<\/a>/gi;
  let match: RegExpExecArray | null;
  while ((match = re.exec(html))) {
    const href = attr(match[1] ?? "", "href");
    if (!href || href.startsWith("#") || href.startsWith("mailto:")) {
      continue;
    }
    actions.push({
      kind: "link",
      method: "GET",
      url: resolve(origin, href),
      text: decode(stripTags(match[2] ?? "")).trim(),
    });
  }
  return actions;
}

function parseForms(html: string, origin: string): DiscoveredAction[] {
  const actions: DiscoveredAction[] = [];
  const re = /<form\b([^>]*)>([\s\S]*?)<\/form>/gi;
  let match: RegExpExecArray | null;
  while ((match = re.exec(html))) {
    const action = attr(match[1] ?? "", "action") || "/";
    const method = (attr(match[1] ?? "", "method") || "GET").toUpperCase();
    actions.push({
      kind: "form",
      method,
      url: resolve(origin, action),
      text: method === "GET" ? "Search" : "Save",
      fields: parseFields(match[2] ?? ""),
    });
  }
  return actions;
}

function parseHx(html: string, origin: string): DiscoveredAction[] {
  const actions: DiscoveredAction[] = [];
  for (const name of ["get", "post", "put", "patch", "delete"] as const) {
    const re = new RegExp(`hx-${name}="([^"]+)"`, "gi");
    let match: RegExpExecArray | null;
    while ((match = re.exec(html))) {
      actions.push({
        kind: "hx",
        method: name.toUpperCase(),
        url: resolve(origin, decode(match[1] ?? "")),
        text: `hx-${name}`,
      });
    }
  }
  return actions;
}

function parseFields(html: string): Record<string, string> {
  const fields: Record<string, string> = {};
  const re = /<input\b([^>]*)>/gi;
  let match: RegExpExecArray | null;
  while ((match = re.exec(html))) {
    const name = attr(match[1] ?? "", "name");
    if (!name) {
      continue;
    }
    fields[name] = decode(attr(match[1] ?? "", "value"));
  }
  return fields;
}

function attr(tag: string, name: string): string {
  const match = tag.match(new RegExp(`${name}\\s*=\\s*["']([^"']*)["']`, "i"));
  return match?.[1] ? decode(match[1]) : "";
}

export function resolve(origin: string, href: string): string {
  try {
    return new URL(href, origin).toString();
  } catch {
    return origin;
  }
}

export function pathAndQuery(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.pathname + parsed.search;
  } catch {
    const parsed = new URL(url, "http://localhost");
    return parsed.pathname + parsed.search;
  }
}

function stripTags(html: string): string {
  return html.replace(/<[^>]+>/g, " ").replace(/\s+/g, " ");
}

function decode(value: string): string {
  return value
    .replaceAll("&amp;", "&")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replaceAll("&quot;", '"')
    .replaceAll("&#39;", "'");
}

function uniqueActions(actions: DiscoveredAction[]): DiscoveredAction[] {
  const byKey = new Map<string, DiscoveredAction>();
  for (const action of actions) {
    const key = `${action.method} ${action.url}`;
    const existing = byKey.get(key);
    if (!existing || fieldCount(action) > fieldCount(existing)) {
      byKey.set(key, action);
    }
  }
  return [...byKey.values()];
}

function fieldCount(action: DiscoveredAction): number {
  return Object.keys(action.fields ?? {}).length;
}
