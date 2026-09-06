# GoTH Template

Base for a Go + html/template + [htmx 4](https://four.htmx.org/) app. Stdlib
only: `net/http`, `html/template`, `embed`. Vendored htmx. No Node, no codegen,
no Go module dependencies.

This file is the contract for agents building on the template. Follow it. Do
not re-litigate the stack.

## Do not introduce

- Routers or frameworks (Echo, Chi, Gin, Fiber)
- HTML codegen (`templ`, gomponents) or a JS bundler
- Runtime CDNs for htmx, CSS, or JS
- ORMs, sqlx, pgx, Postgres, Goose
- Implicit htmx inheritance (`htmx.config.implicitInheritance`)
- A boosted-request render path (always return the full page for navigation)
- New Go dependencies unless the feature cannot be done in the standard library

Load `.agents/skills/htmx-guidance` when writing or changing HTML/htmx.
Load `.agents/skills/htmx-debugging` when a request or swap misbehaves.

## First steps

1. Replace the module path and rename `cmd/app` if needed. Linux: drop the
   `''` after `sed -i`.

   ```sh
   old='github.com/zackerydev/goth-template'
   new='github.com/you/your-app'
   grep -RIl --exclude-dir=.git "$old" . | xargs sed -i '' "s|$old|$new|g"
   ```

2. `mise run setup` (pins Go/tools, installs Lefthook).
3. `mise run dev` → <http://localhost:8888> (`APP_PORT`/`PORT` override;
   worktrees get a derived port).
4. Delete the demo fragment when you start real work: `GET /greeting`,
   `handler.Greeting`, `templates/partials/greeting.html`, and tests that
   mention it.

Air restarts on Go changes. `--dev` reparses HTML from disk per request;
refresh the browser. Embedded CSS/JS need a restart. `mise run run` is
production (embedded templates).

## Add a page

Three edits. Templates do not register routes.

1. `templates/pages/about.html` — named page + `content`. Do not copy the
   document shell.

   ```html
   {{ define "about" }}{{ template "layout" . }}{{ end }}

   {{ define "content" }}
   <section aria-labelledby="about-title">
     <h1 id="about-title">About</h1>
   </section>
   {{ end }}
   ```

2. Handler in `internal/handler`: buffer, `renderer.Render`, `PageData.Title`,
   `text/html` on success, `500` + `"template rendering failed"` on parse/execute
   errors. Copy `Home`.
3. Construct the handler in `cmd/app`, register `GET /about` on the mux in
   `internal/server`.

Link with a normal `<a href="/about">`. Boosting is inherited from `<body>`.

## Add a fragment

Put markup in `templates/partials/<name>.html` as `{{ define "<name>" }}`.
Register a dedicated route. Return only the fragment (no layout). Target a
stable id with `hx-get`/`hx-post` and an explicit `hx-target` / `hx-swap` on
that control — those attributes are not inherited unless you add `:inherited`.

## HTTP and htmx

- Mux: `http.NewServeMux` in `internal/server`. Handlers are `http.Handler`.
- Navigation pages: always `Render` the page name (full document).
- Boost config lives on the layout body:

  ```html
  <body hx-boost:inherited="swap:outerSync select:#main-content target:#main-content">
  ```

  htmx 4 inheritance is explicit. Parent `hx-*` without `:inherited` does not
  apply to children.
- `assets/js/app.js`: `htmx.config.noSwap = [204, 304, "5xx"]`. `422` swaps
  (validation). Do not restore htmx 2 `responseHandling`.
- Forms: real `action`/`method`, server validation, PRG on success. htmx is
  enhancement, not a client router.
- Vendor htmx under `assets/js/`. Bump the file, license, and asset test
  together.

## Packages

```text
cmd/app            process: flags, port, http.Server
internal/server    mux, /assets/
internal/handler   HTTP adapters
assets             embed css/, js/
templates          html/template renderer
.config            mise, Lefthook, linters
.agents/skills     htmx 4 skills
```

Dependency direction (enforced):

```text
cmd/app → server → handler → templates
               ↘ assets
```

Add `internal/service/<domain>` or `internal/model` only when a feature needs
it, then declare the package in `.config/architecture.yml`. Do not put Echo or
other HTTP libraries in `server`.

## Persistence

Skip until the first real schema. Then: SQLite, `database/sql`, sqlc,
golang-migrate, all introduced together (mise tools, generate task, check).

## Tests

`httptest` + status/body substrings. No HTML parsers, no browser driver, no
Node. Cover the handler/mux behavior you added. Keep 100% coverage on eligible
packages (`cmd/app` is excluded). Race + shuffle is already in `mise run test`.

## Quality gates

Commits are blocked until hooks pass. Do not `--no-verify`.

```sh
mise run fix     # format, tidy — hook runs this and git add -A
mise run check   # config, format, mod, arch, lint, prose, tests, coverage,
                 # build, vulns, history secrets
```

Commit types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore` (Conventional
Commits). Lefthook also scans the staged diff with gitleaks.
