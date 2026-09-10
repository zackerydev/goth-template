# Hypermedia Lab

A hands-on htmx 4 learning site built on the GoTH template.

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
- `assets/js/lab.js`: native dialog focus, selection feedback, HTTP inspection, and errors.
  No client-side task model.
- `internal/server/server.go`: routes and embedded assets.

Full page requests return a document. Enhanced navigation returns the workspace.
Validation returns the editor with HTTP 422. Mutations return the updated workspace plus
an explicit htmx 4 `<hx-partial>` notification. Search uses `outerMorph` to retain focus;
normal navigation and successful submissions use replacement swaps.

This is a **shared, in-memory demo**, not an authenticated application. All visitors to a
process see the same data. A mutex serializes access, forms are limited to 16 KiB, and Go's
cross-origin protection rejects cross-site mutations. Activity retains the latest 100
entries. Restarting the server or using Reset sandbox restores the initial data.
System fonts and locally served assets keep the interface independent of third-party requests.

The examples cover core hypermedia interaction patterns, not every htmx attribute or
extension. SSE, uploads, drag-and-drop, and authentication are intentionally outside this
first demo. Native workflow selects provide an accessible alternative to dragging cards.

## Validation

All validation runs through the existing commit hooks: formatting, architecture, static
analysis, race-tested Go tests, 100% eligible-code coverage, security checks, and builds.
Behavioral tests cover representation negotiation, filtering, edits and escaping, validation,
ordinary forms, bulk actions, pagination, polling completion, and request boundaries.
The polling lifecycle test deliberately takes ten seconds to exercise the real server clock.

### htmx skill alignment

The implementation follows `.agents/skills/htmx-guidance` and `htmx-debugging`:

- Explicit `:inherited` targeting, swaps, history, and shared loading feedback.
- Explicit GET form inclusion; debounced search uses `hx-sync` to cancel obsolete requests.
- `outerMorph` preserves search input; replacement swaps reset saved forms and selections.
- HTTP 422 returns editor HTML through `hx-status:422`; transport failures preserve drafts.
- `<hx-partial>` updates the notification alongside the authoritative workspace.
- Full documents for direct/history requests, fragments for targeted requests, and 303 after
  ordinary POSTs. `Vary` lists all representation headers; the demo is not cached.
- Native links, forms, dialog behavior, and htmx 4 colon-separated lifecycle events.
- Paired markup and response contracts in every walkthrough; no client-side task model.

The extension-authoring and upgrade skills were also reviewed. This example neither authors
an extension nor migrates htmx 2, so no compatibility layer or extension API is needed.
Behavioral browser tests exercise these contracts rather than asserting source-code spellings.

### Native view transitions

htmx's `transitions` config delegates swaps to `document.startViewTransition()`.
CSS does the animation; there is no animation library or manual position tracking.

- Stable `view-transition-name` values match task IDs across Board/List and status changes.
- Workspace content crossfades; the header and inspector stay visually anchored.
- The native editor sheet slides from the right on desktop and rises slightly on mobile.
- Search, filter resets, validation, and polling use `transition:false` to avoid distraction.
- Reduced-motion preferences disable transitions, including changes while the page is open.
- Browsers without the API keep ordinary htmx swaps and native dialog behavior.

The editor initializes on `htmx:after:settle`, inside the swap, so the browser captures
its modal state before animating. The normal `htmx:after:swap` callback still handles
final UI synchronization. No application data is owned by these callbacks.

### Browser and visual regression

The pre-commit hook also runs Playwright against a **separate server on port 19169**.
It never resets the development server's sandbox. Node and Playwright are test-only;
the application still has no frontend build step. `mise run setup` installs both the
locked npm dependencies and Chromium.

The browser suite covers desktop (1440 × 1000) and mobile (390 × 844):

- Board, list, native editor sheet, pattern library, and activity screenshots.
- Inline validation, empty search, request inspector, and scrolled board screenshots.
- Search focus, combined filters, clearing, bulk feedback, and real response inspection.
- Stable board geometry, modal keyboard containment, Escape, and restored focus.
- Back/Forward, direct links, reloads, no-JavaScript forms, and horizontal overflow.
- Last-card visibility when the inspector is expanded.
- Inherited loading indicators, request cancellation, disabled submit buttons, and recovery
  from failed saves or tasks deleted by another visitor.

Run validation through the hooks, with changes staged:

```bash
git add -A
mise exec -- lefthook run pre-commit
```

Missing or changed snapshots fail by default. To intentionally refresh baselines:

```bash
UPDATE_VISUALS=1 mise exec -- lefthook run pre-commit
```

**Review the generated images before staging them**, then rerun the hook without
`UPDATE_VISUALS` to verify the comparison. Baselines live in
`tests/browser/snapshots/<platform>/`; the checked-in reference is macOS Chromium.
Different operating systems need their own reviewed baselines because system fonts differ.
Only event clock prefixes and request timings are normalized or masked, not whole panels.
Static layout tests use reduced motion. Separate motion tests observe the real native API,
reject failed captures, check shared task animations and editor entry/exit, and snapshot
paused transition frames. They also cover quiet updates, live preference changes, and
browsers without the API.

Failures write expected/actual/diff images, traces, and an HTML report under
`tmp/browser-results/` and `tmp/browser-report/`. A snapshot is a regression guard,
not proof of good UX: review the actual screens and interaction assertions together.
