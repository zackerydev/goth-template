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

All validation is performed in precommit by lefthook.
These are meant to be restrictive - the more validation performed by deterministic automation the better.

- `go build`
- `go test` (with coverage)
- `gofumpt` strict go formatting
- `rumdl` markdown formatter/typo checker
- `goimports` import cleaner
- `go mod tidy` tidy dependencies
- `cog` validate conventional commit messages
- `config-check` `mise`, `lefthook`, `golangci`
- `gitleaks` secret scanning
- `go-arch-lint` validating architectural boundaries between `cmd`, `server`, `handler`, `assets`, and `templates`

The intent of all the validators is to tell the agent: "do one thing: commit".

## Layout

- `cmd/app/` — process entry: flags, port, `http.Server`.
- `internal/server/` — mux, routes, `/assets/`; only place that wires handlers.
- `internal/handler/` — HTTP adapters; request → renderer response.
- `templates/` — stdlib `html/template` files.
- `assets/` — embedded static: `css/`, `js/` (vendored htmx + `app.js` config).
- `.config/` — mise tasks, lefthook, linters, coverage/architecture policy.

## Development

```bash
mise run dev
```
