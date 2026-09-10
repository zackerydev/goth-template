# Hypermedia Lab

**HTML has range.** A hands-on htmx 4 learning site built on the GoTH template.

A studio workspace with a board, searchable list, deep-linked task editor, activity feed,
and a simulated release preview. The interface is rendered with Go's `html/template`.
No client router, application state store, or frontend build step.

## Run

```bash
mise trust
mise run setup
mise run dev
```

Open the URL printed by the task. `/` redirects to `/lab`.
The existing worktree environment script assigns this worktree its own port.
Use `APP_PORT=8889 mise run dev` to choose a port explicitly.

## Explore

1. **Workspace:** switch Board/List, search, combine status filters, and open a task.
   Each view has a real URL that works in a fresh tab.
2. **Task editor:** change a title or status. A one-character title demonstrates a
   server-side 422 response that preserves the draft and replaces only the editor.
3. **Bulk actions:** select tasks in List view and complete them with one form submission.
4. **Activity & jobs:** run a ten-second simulated preview. The server returns progress
   HTML every second, then removes the polling trigger at completion. Nothing is deployed.
5. **Pagination:** make more than five changes, then load older activity without replacing
   existing entries.
6. **Pattern library:** eight annotated patterns explain the markup, response contract,
   and an experiment to try in the working application.
7. **The wire:** expand the inspector to see actual methods, URLs, status codes, timing,
   response headers, and HTML. It retains twelve requests; displayed bodies cap at 20 KB.
8. **Progressive enhancement:** disable JavaScript. Navigation, search submission, editing,
   and bulk actions still use normal links, forms, and POST/Redirect/GET.

## Architecture

- `internal/handler/lab.go`: shared sandbox, representation selection, filtering, rendering.
- `internal/handler/lab_mutations.go`: bounded form submissions, validation, mutations,
  redirects, and multi-region responses.
- `internal/handler/lab_content.go`: initial tasks and pattern explanations.
- `templates/pages/lab.html`: document, workspace, editor, activity, and partial templates.
- `assets/css/lab.css`: responsive visual design, reduced-motion support, and focus styles.
- `assets/js/lab.js`: observation and request-failure feedback only. No application state.
- `internal/server/server.go`: routes and embedded assets.

Full page requests return a document. Enhanced navigation returns the workspace.
Validation returns the editor with HTTP 422. Mutations return the updated workspace plus
an explicit htmx 4 `<hx-partial>` notification. Search uses `outerMorph` to retain focus;
normal navigation and successful submissions use replacement swaps.

This is a **shared, in-memory demo**, not an authenticated application. All visitors to a
process see the same data. A mutex serializes access, forms are limited to 16 KiB, and Go's
cross-origin protection rejects cross-site mutations. Activity retains the latest 100
entries. Restarting the server or using Reset sandbox restores the initial data.
Google Fonts is optional; system fonts are the fallback. Everything else is served locally.

The examples cover core hypermedia interaction patterns, not every htmx attribute or
extension. SSE, uploads, drag-and-drop, and authentication are intentionally outside this
first demo. Native workflow selects provide an accessible alternative to dragging cards.

## Validation

All validation runs through the existing commit hooks: formatting, architecture, static
analysis, race-tested Go tests, 100% eligible-code coverage, security checks, and builds.
Behavioral tests cover representation negotiation, filtering, edits and escaping, validation,
ordinary forms, bulk actions, pagination, polling completion, and request boundaries.
The polling lifecycle test deliberately takes ten seconds to exercise the real server clock.
