# Structure the application around small Go packages

## Status

Accepted

## Context

A reusable Go, templ, and HTMX base needs clear ownership without speculative
layers. Keeping the executable, framework wiring, handlers, templates, and
assets separate makes the dependency direction visible while leaving future
domain architecture open.

## Decision

- `cmd/app` owns process startup and HTTP server settings.
- `internal/server` is the composition root and the only package that constructs
  Echo or registers routes.
- `internal/handler` adapts HTTP requests to templ responses.
- `templates` owns page and fragment markup.
- `assets` embeds local CSS, JavaScript, HTMX, and its upstream license.
- Generated `*_templ.go` files are committed and never edited directly.
- Service and model packages are added only when a real workflow needs them.
- `.config/architecture.yml` enforces package and third-party dependency edges.

The initial dependency graph is:

```text
cmd/app -> internal/server -> internal/handler -> templates
                          \-> assets
```

## Consequences

The base remains small, builds without a generation step, and exposes a standard
`http.Handler` seam for tests. New packages and dependencies require an explicit
architecture policy change rather than silently expanding the graph.
