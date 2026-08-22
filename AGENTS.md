# AGENTS.md — Repository contract

This repository is a deliberately small Go, Echo, templ, and HTMX application.
Keep it simple. Add code, packages, dependencies, and infrastructure only when a
current requirement needs them.

## Before changing the tree

1. Read [README.md](README.md), the relevant files in [docs/adr](docs/adr/),
   and this contract.
2. Inspect `git status --short --branch`. Preserve unrelated work and never
   modify sibling worktrees.
3. In a fresh clone, run `mise trust`, then `mise run setup`.
4. Use focused tests for the area being changed. Run the complete gates before
   committing.

Treat the current checkout as the assigned worktree. Use `.worktrees/` for
concurrent branches; it is intentionally ignored and excluded from architecture
scans. Use `mise run dev` or `mise run run` so linked worktrees receive isolated
ports. Do not leave development servers running after verification.

## Architecture boundaries

The initial dependency graph is:

```text
cmd/app -> internal/server -> internal/handler -> templates
                          \-> assets
```

Ownership is strict:

- `cmd/app` owns process startup and HTTP server settings. Keep it thin.
- `internal/server` is the composition root. It constructs Echo, assembles
  dependencies, registers routes, and returns the application's `http.Handler`.
- `internal/handler` owns transport behavior and converts requests into templ
  responses. Keep Echo-specific routing out of handlers.
- `templates` owns page and fragment markup.
- `assets` owns embedded browser assets and upstream licenses.

Do not add speculative layers, generic repositories, empty marker packages, or
interfaces with one hypothetical consumer. Add `internal/service/<domain>` only
when a real workflow exists. Add `internal/model` only when persistence exists.
When they do exist, handlers call services rather than database queries, and
services own workflow invariants.

A consuming package defines the smallest interface it actually needs.
Constructors return concrete types by default. Prefer standard-library seams,
especially `http.Handler`, over framework-shaped interfaces.

`.config/architecture.yml` is executable policy. Any new package or third-party
import must be classified there in the same atomic change that introduces it;
otherwise the fail-closed commit hook will reject the intermediate tree. Do not
weaken architecture rules merely to make a check pass.

Keep one root Go module. Do not add `go.work`, nested modules, or `replace`
directives without a documented requirement.

## Go conventions

- Prefer straightforward standard-library Go over helpers and abstractions.
- Accept contexts from callers; do not store request contexts in long-lived
  structs.
- Keep process-wide resources open for the process lifetime and close them at
  the composition boundary.
- Set explicit HTTP server timeouts.
- Make time-dependent behavior deterministic by injecting or passing the clock
  value used by the workflow.
- Validate at the boundary that owns the rule. Preserve typed errors when
  callers need to distinguish outcomes.
- Do not export a symbol until another package needs it.

If persistence is introduced, choose it from actual deployment and concurrency
requirements. Do not add Docker or a network service when an embedded database
meets the need. Keep migrations and query sources authoritative; generated data
access is an adapter, not the domain layer. For SQLite, migrate and serve with
the same open handle—especially for `:memory:` databases—and use isolated
temporary databases in tests.

## templ and HTMX conventions

Server-rendered HTML is the baseline. HTMX is progressive enhancement, not a
client-side application architecture.

- Every form must work as normal HTML with a real `action` and `method`.
- Native success should use Post/Redirect/Get. Native validation failures render
  a useful full page.
- HTMX requests may return a stable, replaceable fragment. If the representation
  varies by `HX-Request`, include `Vary: HX-Request`.
- Validation fragments may use status `422`; `assets/js/app.js` is configured to
  swap them. Successful HTMX mutations may use `HX-Redirect` when navigation is
  required.
- Avoid global boosting, SPA routing, and client-side state unless a concrete
  requirement justifies them.
- `Page` owns the document boundary and shared metadata. Fragment components do
  not emit document markup.

HTML must remain semantic and accessible: preserve landmarks, heading order,
skip-link targets, visible labels, control names, keyboard focus, and explicit
error relationships. A button performs an action; a link navigates. Do not make
color, motion, or JavaScript the only way to understand state.

Edit `*.templ` files, then run `mise run generate`. Generated `*_templ.go` files
are committed and never edited directly.

## Assets and design

Keep browser dependencies local and deliberate. HTMX is committed with its
upstream license; update the file, license, and asset tests together. Do not add
a runtime CDN dependency by default.

Prefer one small handwritten stylesheet, semantic class names, and design
tokens. Do not add a CSS build, component library, icon library, or frontend
framework until repeated real code demonstrates the need. Reuse a component
only when it represents a real shared contract, not merely similar markup.

The starter styling is not product design. When an application develops a
visual language, document it as durable design guidance and test only meaningful
contracts. Avoid tests coupled to exact class strings unless styling itself is
the requirement.

## Testing

Test through the narrowest observable seam that proves the behavior:

- Use `httptest` against the composed `http.Handler` for routes, status codes,
  headers, redirects, assets, and response bodies.
- Use goquery for semantic template contracts such as metadata, landmarks,
  labels, values, and HTMX attributes. Avoid snapshots and generated HTML
  serialization checks.
- Exercise both native and HTMX paths for enhanced forms.
- Use real disposable persistence for database behavior; use narrow fakes for
  service consumers only when substitution is useful.
- Add a regression test with every bug fix.
- Run tests in parallel only when their state is isolated.

Eligible handwritten code maintains 100% file, package, and total coverage.
Generated templ code and the thin executable are intentionally excluded. Do not
exclude new handwritten code or add empty tests to satisfy the metric; prove its
behavior or simplify it away.

Use focused tests while iterating, then run the race-tested, shuffled canonical
suite through `mise run check`.

## Tooling and commits

`.config/mise.toml` is the command source of truth:

```sh
mise run dev       # live reload
mise run generate  # regenerate templ Go
mise run fix       # format, generate, import-sort, and tidy
mise run check     # all read-only quality gates
mise run build     # build bin/app
mise run run       # run the application
```

`mise run fix` intentionally mutates the whole repository. Lefthook then stages
the complete result before verification. Review the full diff; do not assume
only previously staged hunks will remain staged.

Before committing:

1. Run `mise run fix`.
2. Inspect `git diff`, `git diff --check`, and `git status`.
3. Run `mise run check`.
4. If startup, routes, assets, or runtime configuration changed, perform a real
   HTTP smoke test against the built application.
5. Commit normally and let Lefthook run. Never use `--no-verify`.

Independent checks run concurrently; preserve that structure while keeping
true dependencies ordered, such as tests before coverage. Do not duplicate the
same gate in mise and Lefthook merely to increase apparent rigor.

Commits use `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, or `chore:` and
contain one logical change. Never report a clean review without running the
narrowest executable test for each materially changed behavior. If a required
check cannot run, report the exact blocker.

## Documentation

README documents current setup, commands, and repository shape. ADRs record
meaningful architectural choices and their consequences. Supersede an accepted
ADR with a new numbered ADR instead of rewriting history.

Do not commit temporary plans, implementation journals, generated reports, or
completed checklists. Documentation should describe durable current truth.

When creating an application from this template, replace the module path and
starter identity before product work. Remove the `/greeting` demonstration once
the first real HTMX interaction exists.
