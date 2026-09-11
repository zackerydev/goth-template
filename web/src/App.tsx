import { FormEvent, ReactNode, useEffect, useState } from "react";
import {
  Contact,
  ContactAPIError,
  ContactInput,
  createContact,
  deleteContact,
  getContact,
  listContacts,
  updateContact,
} from "./api";

type View =
  | { name: "list" }
  | { name: "new" }
  | { name: "show"; id: number }
  | { name: "edit"; id: number };

const emptyForm: ContactInput = { first: "", last: "", phone: "", email: "" };

export function App() {
  const [view, setView] = useState<View>({ name: "list" });

  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <header className="site-header">
        <a className="site-header__brand" href="#/" onClick={(event) => {
          event.preventDefault();
          setView({ name: "list" });
        }}>
          <span className="site-header__name">contacts.app</span>
          <span className="site-header__tag">A Demo Contacts Application</span>
        </a>
      </header>
      <main id="main-content">
        {view.name === "list" ? <ContactList onOpen={setView} /> : null}
        {view.name === "new" ? (
          <ContactForm
            title="New Contact"
            initial={emptyForm}
            onSave={async (input) => {
              await createContact(input);
              setView({ name: "list" });
            }}
            onBack={() => setView({ name: "list" })}
          />
        ) : null}
        {view.name === "show" ? (
          <ContactShow
            id={view.id}
            onEdit={() => setView({ name: "edit", id: view.id })}
            onDeleted={() => setView({ name: "list" })}
            onBack={() => setView({ name: "list" })}
          />
        ) : null}
        {view.name === "edit" ? (
          <ContactEditor
            id={view.id}
            onSaved={() => setView({ name: "show", id: view.id })}
            onDeleted={() => setView({ name: "list" })}
            onBack={() => setView({ name: "list" })}
          />
        ) : null}
      </main>
    </>
  );
}

function ContactList({ onOpen }: { onOpen: (view: View) => void }) {
  const [query, setQuery] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    listContacts(submitted, page)
      .then((result) => {
        if (cancelled) {
          return;
        }
        setContacts((current) => (page === 1 ? result.contacts : [...current, ...result.contacts]));
        setHasMore(result.has_more);
        setError("");
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "list failed");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [submitted, page]);

  function search(event: FormEvent) {
    event.preventDefault();
    setPage(1);
    setSubmitted(query);
  }

  return (
    <section className="workspace" aria-labelledby="contacts-title">
      <div className="workspace__header">
        <h1 id="contacts-title">Contacts</h1>
        <p>
          <button type="button" onClick={() => onOpen({ name: "new" })}>
            Add Contact
          </button>
        </p>
      </div>
      <form className="tool-bar" role="search" onSubmit={search}>
        <label htmlFor="search">Search Term</label>
        <input id="search" type="search" name="q" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button type="submit">Search</button>
      </form>
      {error ? <p className="error">{error}</p> : null}
      <table>
        <thead>
          <tr>
            <th>First</th>
            <th>Last</th>
            <th>Phone</th>
            <th>Email</th>
            <th>
              <span className="visually-hidden">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {contacts.length === 0 ? (
            <tr>
              <td colSpan={5}>No contacts found.</td>
            </tr>
          ) : (
            contacts.map((item) => (
              <tr key={item.id}>
                <td>{item.first}</td>
                <td>{item.last}</td>
                <td>{item.phone}</td>
                <td>{item.email}</td>
                <td>
                  <button type="button" onClick={() => onOpen({ name: "show", id: item.id })}>
                    View
                  </button>{" "}
                  <button type="button" onClick={() => onOpen({ name: "edit", id: item.id })}>
                    Edit
                  </button>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
      {hasMore ? (
        <p>
          <button type="button" onClick={() => setPage((current) => current + 1)}>
            Load More
          </button>
        </p>
      ) : null}
    </section>
  );
}

function ContactShow({
  id,
  onEdit,
  onDeleted,
  onBack,
}: {
  id: number;
  onEdit: () => void;
  onDeleted: () => void;
  onBack: () => void;
}) {
  const [item, setItem] = useState<Contact | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    getContact(id)
      .then(setItem)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "load failed"));
  }, [id]);

  if (error) {
    return (
      <section className="workspace">
        <p className="error">{error}</p>
        <p>
          <button type="button" onClick={onBack}>
            Back
          </button>
        </p>
      </section>
    );
  }
  if (!item) {
    return <p>Loading…</p>;
  }
  return (
    <section className="workspace" aria-labelledby="contact-title">
      <h1 id="contact-title">
        {item.first} {item.last}
      </h1>
      <p>{item.phone}</p>
      <p>
        <a href={`mailto:${item.email}`}>{item.email}</a>
      </p>
      <p>
        <button type="button" onClick={onEdit}>
          Edit
        </button>{" "}
        <DeleteButton id={item.id} onDeleted={onDeleted} />
      </p>
      <p>
        <button type="button" onClick={onBack}>
          Back
        </button>
      </p>
    </section>
  );
}

function ContactEditor({
  id,
  onSaved,
  onDeleted,
  onBack,
}: {
  id: number;
  onSaved: () => void;
  onDeleted: () => void;
  onBack: () => void;
}) {
  const [item, setItem] = useState<Contact | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    getContact(id)
      .then(setItem)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "load failed"));
  }, [id]);

  if (error) {
    return (
      <section className="workspace">
        <p className="error">{error}</p>
        <button type="button" onClick={onBack}>
          Back
        </button>
      </section>
    );
  }
  if (!item) {
    return <p>Loading…</p>;
  }
  return (
    <ContactForm
      title="Edit Contact"
      initial={item}
      onSave={async (input) => {
        await updateContact(id, input);
        onSaved();
      }}
      onBack={onBack}
      extra={<DeleteButton id={id} onDeleted={onDeleted} />}
    />
  );
}

function ContactForm({
  title,
  initial,
  onSave,
  onBack,
  extra,
}: {
  title: string;
  initial: ContactInput;
  onSave: (input: ContactInput) => Promise<void>;
  onBack: () => void;
  extra?: ReactNode;
}) {
  const [form, setForm] = useState(initial);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      await onSave(form);
    } catch (err: unknown) {
      if (err instanceof ContactAPIError) {
        setFields(err.fields);
        setError(err.message);
        return;
      }
      setError(err instanceof Error ? err.message : "save failed");
    }
  }

  return (
    <section className="workspace" aria-labelledby="contact-form-title">
      <h1 id="contact-form-title">{title}</h1>
      {error ? <p className="error">{error}</p> : null}
      <form onSubmit={submit}>
        <fieldset>
          <legend>Contact Values</legend>
          <Field label="Email" name="email" type="email" value={form.email} error={fields.email} onChange={(email) => setForm({ ...form, email })} />
          <Field label="First Name" name="first_name" value={form.first} onChange={(first) => setForm({ ...form, first })} />
          <Field label="Last Name" name="last_name" value={form.last} onChange={(last) => setForm({ ...form, last })} />
          <Field label="Phone" name="phone" value={form.phone} onChange={(phone) => setForm({ ...form, phone })} />
          <button type="submit">Save</button>
        </fieldset>
      </form>
      {extra}
      <p>
        <button type="button" onClick={onBack}>
          Back
        </button>
      </p>
    </section>
  );
}

function Field({
  label,
  name,
  value,
  error,
  type = "text",
  onChange,
}: {
  label: string;
  name: string;
  value: string;
  error?: string;
  type?: string;
  onChange: (value: string) => void;
}) {
  const id = name;
  return (
    <p>
      <label htmlFor={id}>{label}</label>
      <input id={id} name={name} type={type} value={value} onChange={(event) => onChange(event.target.value)} />
      {error ? <span className="error">{error}</span> : null}
    </p>
  );
}

function DeleteButton({ id, onDeleted }: { id: number; onDeleted: () => void }) {
  return (
    <button
      type="button"
      onClick={async () => {
        if (!window.confirm("Are you sure you want to delete this contact?")) {
          return;
        }
        await deleteContact(id);
        onDeleted();
      }}
    >
      Delete Contact
    </button>
  );
}
