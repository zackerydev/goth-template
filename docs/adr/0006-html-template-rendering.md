# Render HTML with the standard library and request-time development reloads

## Status

Accepted

## Context

The base application needs server-rendered HTML, progressive HTMX navigation,
and a fast local development loop without generated Go, a frontend build, or a
browser reload proxy. The previous templ and proxy arrangement made HTML edits
part of a generation/restart pipeline and kept boosted navigation disabled even
though the page shell already had a stable `main` target.

Production must be independent of the working directory and must fail before
listening if a template is malformed. Development should use the current
worktree, show HTML edits on the next request, and never share mutable parsed
templates between requests.

## Decision

Use the standard-library `html/template` package. HTML sources are organized
as:

```text
templates/layouts/    shared document shell
templates/pages/     canonical named pages
templates/partials/  standalone fragments and document-title markup
```

Each canonical page defines a named page that calls `layout`, plus a `content`
definition. The layout owns the doctype, document metadata, assets, navigation
shell, and `main#main-content`. Partials are parsed independently and can be
rendered directly for explicit fragment endpoints. Normal `html/template`
escaping is always used for data.

`cmd/app` constructs the renderer and HTTP handlers, then injects those handlers
into `internal/server`. Echo construction and route registration remain confined
to `internal/server`; HTTP adaptation remains in `internal/handler`.

Production embeds all layouts, pages, and partials. The renderer parses every
page in an isolated template set and every standalone partial during
initialization, then reuses those immutable parsed templates. Initialization
errors are returned to startup and the server does not listen. The production
renderer performs no source filesystem reads during rendering.

The `--dev` flag uses the current worktree's `templates/` directory. Each page
or partial render discovers and parses fresh source files, so edits to a page,
layout, or partial are visible on the next request. Parse errors return a clean
HTTP 500 and do not replace a prior successful response; a subsequent fixed
request can recover. Air watches Go files only, builds the application into
`tmp/`, and runs it with `--dev`. HTML changes do not rebuild or restart the Go
process. `mise run run` remains production mode.

The shared layout enables inherited HTMX boosting with
`hx-boost="true"`, `hx-target="#main-content"`, and `hx-swap="innerHTML"`.
Native requests render the complete named page. A request with
`HX-Boosted: true` renders the escaped document title and `content` definition
only; it excludes the doctype, document wrappers, navigation shell, and outer
`main`. Title and content execute from the same parsed page set, including in
development, rather than independently reloading sources for each definition.
`HX-Request: true` alone and history-restoration requests render the
complete page. Page responses preserve existing `Vary` values and add
`HX-Boosted` and `HX-History-Restore-Request`. Explicit partial responses do not
render document-title markup.

This decision supersedes the templ source/generation and startup-wiring
conventions in [ADR 0001](0001-application-structure.md), the template-generation
checks in [ADR 0002](0002-local-quality-gates.md), the proxy-port and
proxy-lifecycle portions of [ADR 0003](0003-worktree-isolated-development.md),
and the no-global-boost restriction in
[ADR 0004](0004-progressive-htmx-and-local-assets.md). It does not supersede the
local asset, progressive-enhancement, worktree application-port, or persistence
decisions. Historical ADRs remain unchanged.

The application port retains its primary-checkout default of `8888`, stable
worktree derivation, `APP_PORT`/`PORT` overrides, and occupancy validation. There
is no longer a proxy port or `PROXY_PORT` override.

`mise run fix` formats Go and Markdown and tidies the module. Template generation
and generated-output checks are removed; the 100% eligible Go coverage gate
remains unchanged. The future sqlc generation and verification tasks described
in [ADR 0005](0005-sqlite-persistence-tooling.md) are introduced with the first
persistence feature, not retained as placeholder tasks.

## Consequences

Production binaries are self-contained and startup validation is deterministic.
Development is simple: Air handles Go lifecycle changes and the renderer handles
HTML request-time reloads. There is no template generator, generated template
output, proxy port, browser reload proxy, or frontend build system in the base.

Pages must maintain distinct named definitions and tests must cover native,
ordinary HTMX, boosted, history-restoration, and explicit partial responses. A
future generator, including sqlc for the persistence decision in ADR 0005, must
be introduced with the feature that needs it and must not be added as a generic
HTML generation step.
