# Structure the application around small Go packages

## Status

Accepted

## Context

A reusable Go, Echo, `html/template`, and HTMX base needs clear ownership
without speculative layers. Keeping the executable, framework wiring, handlers,
templates, and assets separate makes the dependency direction visible while
leaving future domain architecture open.

## Decision

- `cmd/app` owns process startup, the `--dev` flag, and HTTP server settings. It
  constructs the renderer and injects it into the handlers.
- `internal/server` is the composition root and the only package that
  constructs Echo or registers routes.
- `internal/handler` adapts HTTP requests to buffered standard-library template
  responses.
- `templates` owns page and fragment markup plus the embedded/development
  `html/template` renderer.
- `assets` embeds local CSS, JavaScript, HTMX, and its upstream license.
- Service and model packages are added only when a real workflow needs them.
- `.config/architecture.yml` enforces package and third-party dependency edges.

The initial dependency graph is:

```text
cmd/app -> internal/server
       \-> internal/handler -> templates
       \-> templates
internal/server -> internal/handler
                \-> assets
```

## Consequences

The base remains small, builds without a generation step, and exposes standard
`http.Handler` seams for tests. Production startup is independent of the
working directory because templates are embedded. New packages and dependencies
require an explicit architecture policy change rather than silently expanding
the graph.

This supersedes the earlier templ-specific response and generated-file
conventions. The page, layout, and partial contract is recorded in [ADR
0006](0006-html-template-rendering.md).
