type Contact = {
  id: number;
  first: string;
  last: string;
  phone: string;
  email: string;
};

type ListResponse = {
  contacts: Contact[];
  has_more: boolean;
  page: number;
};

type ContactInput = Omit<Contact, "id">;

export type ToolName =
  | "list_contacts"
  | "search_contacts"
  | "get_contact"
  | "create_contact"
  | "update_contact"
  | "delete_contact";

export type ToolCall = {
  name: ToolName;
  arguments: Record<string, unknown>;
};

export type ToolResult = {
  name: ToolName;
  ok: boolean;
  body: unknown;
};

const catalog: { name: ToolName; description: string; input: Record<string, unknown> }[] = [
  {
    name: "list_contacts",
    description: "List one page of contacts. Page size is 10.",
    input: { type: "object", properties: { page: { type: "integer", minimum: 1 } } },
  },
  {
    name: "search_contacts",
    description: "Search contacts by name, email, or phone.",
    input: {
      type: "object",
      properties: { query: { type: "string" }, page: { type: "integer", minimum: 1 } },
      required: ["query"],
    },
  },
  {
    name: "get_contact",
    description: "Get one contact by id.",
    input: { type: "object", properties: { id: { type: "integer" } }, required: ["id"] },
  },
  {
    name: "create_contact",
    description: "Create a contact. Email must be unique.",
    input: {
      type: "object",
      properties: {
        first: { type: "string" },
        last: { type: "string" },
        phone: { type: "string" },
        email: { type: "string" },
      },
      required: ["email"],
    },
  },
  {
    name: "update_contact",
    description: "Update a contact by id.",
    input: {
      type: "object",
      properties: {
        id: { type: "integer" },
        first: { type: "string" },
        last: { type: "string" },
        phone: { type: "string" },
        email: { type: "string" },
      },
      required: ["id", "email"],
    },
  },
  {
    name: "delete_contact",
    description: "Delete a contact by id.",
    input: { type: "object", properties: { id: { type: "integer" } }, required: ["id"] },
  },
];

export function toolCatalog(): typeof catalog {
  return catalog;
}

export async function callContactTool(baseURL: string, call: ToolCall): Promise<ToolResult> {
  const root = `${baseURL.replace(/\/$/, "")}/api/v1/contacts`;
  try {
    const body = await dispatch(root, call);
    return { name: call.name, ok: true, body };
  } catch (error) {
    return { name: call.name, ok: false, body: { error: error instanceof Error ? error.message : String(error) } };
  }
}

async function dispatch(root: string, call: ToolCall): Promise<unknown> {
  const args = call.arguments;
  switch (call.name) {
    case "list_contacts":
      return getJSON(`${root}?page=${num(args.page, 1)}`);
    case "search_contacts":
      return getJSON(`${root}?q=${encodeURIComponent(str(args.query))}&page=${num(args.page, 1)}`);
    case "get_contact":
      return getJSON(`${root}/${num(args.id)}`);
    case "create_contact":
      return sendJSON(root, "POST", fields(args));
    case "update_contact":
      return sendJSON(`${root}/${num(args.id)}`, "PUT", fields(args));
    case "delete_contact":
      return sendJSON(`${root}/${num(args.id)}`, "DELETE");
  }
}

async function getJSON(url: string): Promise<ListResponse | Contact> {
  const response = await fetch(url, { headers: { Accept: "application/json" } });
  return readJSON(response);
}

async function sendJSON(url: string, method: string, body?: ContactInput): Promise<unknown> {
  const response = await fetch(url, {
    method,
    headers: body ? { Accept: "application/json", "Content-Type": "application/json" } : { Accept: "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (response.status === 204) {
    return { deleted: true };
  }
  return readJSON(response);
}

async function readJSON(response: Response): Promise<never | unknown> {
  const payload: unknown = await response.json().catch(() => ({ error: response.statusText }));
  if (!response.ok) {
    throw new Error(JSON.stringify(payload));
  }
  return payload;
}

function fields(args: Record<string, unknown>): ContactInput {
  return {
    first: str(args.first),
    last: str(args.last),
    phone: str(args.phone),
    email: str(args.email),
  };
}

function str(value: unknown): string {
  return typeof value === "string" ? value : value == null ? "" : String(value);
}

function num(value: unknown, fallback = 0): number {
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}
