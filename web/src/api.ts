export type Contact = {
  id: number;
  first: string;
  last: string;
  phone: string;
  email: string;
};

export type ListResponse = {
  contacts: Contact[];
  has_more: boolean;
  page: number;
};

export type ContactInput = Omit<Contact, "id">;

type APIError = {
  error?: string;
  fields?: Record<string, string>;
};

const root = "/api/v1/contacts";

export class ContactAPIError extends Error {
  fields: Record<string, string>;

  constructor(message: string, fields: Record<string, string> = {}) {
    super(message);
    this.fields = fields;
  }
}

export async function listContacts(query = "", page = 1): Promise<ListResponse> {
  const params = new URLSearchParams({ page: String(page) });
  if (query) {
    params.set("q", query);
  }
  return request<ListResponse>(`${root}?${params.toString()}`);
}

export async function getContact(id: number): Promise<Contact> {
  return request<Contact>(`${root}/${id}`);
}

export async function createContact(input: ContactInput): Promise<Contact> {
  return request<Contact>(root, { method: "POST", body: JSON.stringify(input) });
}

export async function updateContact(id: number, input: ContactInput): Promise<Contact> {
  return request<Contact>(`${root}/${id}`, { method: "PUT", body: JSON.stringify(input) });
}

export async function deleteContact(id: number): Promise<void> {
  await request<null>(`${root}/${id}`, { method: "DELETE" });
}

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(url, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
  });
  if (response.status === 204) {
    return null as T;
  }
  const payload = (await response.json()) as T & APIError;
  if (!response.ok) {
    throw new ContactAPIError(payload.error ?? "request failed", payload.fields ?? {});
  }
  return payload;
}
