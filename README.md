# GoTH Template

> Go Templating HTML

This repo serves as a base for building simple, 0 dependency full-stack applications with Go and HTMX 4.

<!-- SETUP BEGIN -->

## Template Setup

Run these steps to remove the template setup guide and set the Go imports to your repo.

```bash
git clone https://github.com/zackerydev/goth-template
mise install
mise run init
```

Note: this section of the README will self-destruct during template setup.

<!-- EXAMPLES BEGIN -->

## Examples

Complete applications built from this template live on example branches:

- [Contacts](https://github.com/zackerydev/goth-template/tree/examples/contacts) — a progressively enhanced contact manager.
- [Agent eval](https://github.com/zackerydev/goth-template/tree/examples/agent-eval) — the contact manager with an agent evaluation harness and React comparison.
- [Hypermedia Lab](https://github.com/zackerydev/goth-template/tree/examples/hypermedia-lab) — an interactive htmx 4 pattern lab.

<!-- EXAMPLES END -->

<!-- SETUP END -->

## Precommit

Lefthook runs deterministic checks that prevent defective commits:

- race-tested Go tests
- Go, Markdown, and template formatting
- focused static analysis
- tidy dependencies
- configuration validation
- staged-diff secret scanning
- architectural boundary validation
- conventional commit validation

Hooks never modify or stage files. Run `mise run fix`, review the result, and stage it before committing.

## Layout

- `cmd/app/` — process entry: flags, port, `http.Server`.
- `internal/server/` — mux, routes, `/assets/`; only place that wires handlers.
- `internal/handler/` — HTTP adapters; request → renderer response.
- `templates/` — stdlib `html/template` files.
- `assets/` — embedded static: `css/`, `js/` (vendored htmx + `app.js` config).
- `.config/` — mise tasks, lefthook, linters, and architecture policy.

## Development

```bash
mise run dev
```
