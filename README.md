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

note this section of the readme will self destruct.

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
