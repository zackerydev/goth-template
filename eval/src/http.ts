import { crawlerPage, parseActions, pathAndQuery, resolve } from "./html.ts";

export type HttpExchange = {
  method: string;
  url: string;
  status: number;
  body: string;
};

export class HttpClient {
  origin: string;
  cookies = new Map<string, string>();
  allowed = new Set<string>();
  requests: HttpExchange[] = [];
  bytesIn = 0;
  jsBytes = 0;
  private formInputs = new Map<string, string[]>();

  constructor(origin: string, startPath: string) {
    this.origin = origin.replace(/\/$/, "");
    this.allow(this.origin + startPath);
  }

  async request(method: string, url: string, fields?: Record<string, string>): Promise<HttpExchange> {
    const absolute = resolve(this.origin, url);
    if (!this.sameOrigin(absolute)) {
      throw new Error(`refusing off-origin request ${absolute}`);
    }
    if (!this.isAllowed(absolute)) {
      throw new Error(`path not discovered from HTML: ${pathAndQuery(absolute)}`);
    }
    this.requireFormFields(method, absolute, fields);
    const response = await this.exchange(method, absolute, fields);
    this.ingest(response);
    return response;
  }

  usedSearch(): boolean {
    return this.requests.some((item) => /[?&]q=/i.test(item.url));
  }

  private async exchange(method: string, url: string, fields?: Record<string, string>): Promise<HttpExchange> {
    const target = method === "GET" && fields ? withQuery(url, fields) : url;
    const init: RequestInit = {
      method,
      redirect: "manual",
      headers: this.headers(method, fields),
    };
    if (fields && method !== "GET") {
      init.body = new URLSearchParams(fields);
    }
    const response = await fetch(target, init);
    const body = await response.text();
    this.storeCookies(response.headers.getSetCookie?.() ?? []);
    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get("location") ?? response.headers.get("hx-redirect");
      if (location) {
        const next = resolve(this.origin, location);
        this.allow(next);
        return this.exchange("GET", next);
      }
    }
    const exchange = { method, url: target, status: response.status, body };
    this.requests.push(exchange);
    this.bytesIn += body.length;
    if (/\.js(\?|$)/i.test(target) || response.headers.get("content-type")?.includes("javascript")) {
      this.jsBytes += body.length;
    }
    return exchange;
  }

  private ingest(exchange: HttpExchange): void {
    for (const action of parseActions(exchange.body, this.origin)) {
      this.allow(action.url);
      const names = Object.keys(action.fields ?? {});
      if (action.kind === "form" && names.length > 0) {
        this.formInputs.set(`${action.method} ${normalize(action.url)}`, names);
      }
    }
  }

  private requireFormFields(method: string, url: string, fields?: Record<string, string>): void {
    if (!["POST", "PUT", "PATCH"].includes(method.toUpperCase())) {
      return;
    }
    const expected = this.formInputs.get(`${method.toUpperCase()} ${normalize(url)}`);
    if (!expected?.length) {
      return;
    }
    if (fields && Object.keys(fields).length > 0) {
      return;
    }
    throw new Error(
      `form at ${pathAndQuery(url)} has named inputs (${expected.join(", ")}); POST without fields submits an empty body`,
    );
  }

  private allow(url: string): void {
    this.allowed.add(normalize(url));
  }

  private isAllowed(url: string): boolean {
    if (this.allowed.has(normalize(url))) {
      return true;
    }
    const target = new URL(url);
    for (const item of this.allowed) {
      const allowed = new URL(item);
      if (allowed.pathname === target.pathname) {
        return true;
      }
    }
    return false;
  }

  private sameOrigin(url: string): boolean {
    return new URL(url).origin === new URL(this.origin).origin;
  }

  private headers(method: string, fields?: Record<string, string>): HeadersInit {
    const headers: Record<string, string> = { Accept: "text/html,application/xhtml+xml" };
    const cookie = [...this.cookies.entries()].map(([name, value]) => `${name}=${value}`).join("; ");
    if (cookie) {
      headers.Cookie = cookie;
    }
    if (fields && method !== "GET") {
      headers["Content-Type"] = "application/x-www-form-urlencoded";
    }
    return headers;
  }

  private storeCookies(setCookie: string[]): void {
    for (const header of setCookie) {
      const pair = header.split(";")[0];
      const eq = pair?.indexOf("=") ?? -1;
      if (!pair || eq < 1) {
        continue;
      }
      this.cookies.set(pair.slice(0, eq), pair.slice(eq + 1));
    }
  }
}

export function summarizeHTML(html: string, origin: string) {
  return crawlerPage(html, origin);
}

function withQuery(url: string, fields: Record<string, string>): string {
  const parsed = new URL(url);
  for (const [name, value] of Object.entries(fields)) {
    parsed.searchParams.set(name, value);
  }
  return parsed.toString();
}

function normalize(url: string): string {
  const parsed = new URL(url);
  if (parsed.pathname.endsWith("/") && parsed.pathname !== "/") {
    parsed.pathname = parsed.pathname.slice(0, -1);
  }
  return parsed.origin + parsed.pathname + parsed.search;
}
