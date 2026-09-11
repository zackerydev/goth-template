# contacts.app

Hypermedia Systems Contact.app on this GoTH template: Go `net/http`,
`html/template`, vendored htmx 4, and SQLite through `database/sql`. One module
dependency — `modernc.org/sqlite` — registers the driver. The rest is standard
library.

This file is the contract for agents building on the example. Follow it. Do
not re-litigate the stack.

## Do not introduce

- Routers or frameworks (Echo, Chi, Gin, Fiber)
- HTML codegen (`templ`, gomponents) or a JS bundler
- Runtime CDNs for htmx, CSS, or JS
- ORMs, sqlx, pgx, Postgres, Goose, golang-migrate, sqlc
- Implicit htmx inheritance (`htmx.config.implicitInheritance`)
- A boosted-request render path (always return the full page for navigation)
- New Go dependencies unless the feature cannot be done in the standard library

Load `.agents/skills/htmx-guidance` when writing or changing HTML/htmx.
Load `.agents/skills/htmx-debugging` when a request or swap misbehaves.

## Run

```sh
mise run setup
mise run dev
```

Open the printed `APP_URL` (worktrees get a derived port; `APP_PORT`/`PORT`
override). Production: `mise run run`. Database path: `APP_DATABASE`, default
`tmp/contacts.db`. Empty databases are seeded with the demo address book.

Air restarts on Go changes. `--dev` reparses HTML from disk per request;
refresh the browser. Embedded CSS/JS need a restart.

## Routes

| Method | Path | Behavior |
| --- | --- | --- |
| GET | `/` | Redirect to `/contacts` |
| GET | `/contacts` | Searchable list. `q` filters; `page` is 10-row windows |
| POST | `/contacts/new` | Create; PRG to `/contacts` or 422 with field errors |
| GET | `/contacts/{id}` | Detail |
| POST | `/contacts/{id}/edit` | Update; PRG to the detail page or 422 |
| POST | `/contacts/{id}/delete` | Delete without JavaScript; PRG to the list |
| DELETE | `/contacts/{id}` | Delete via htmx; `HX-Redirect` when not boosted |
| GET | `/contacts/{id}/email` | Inline unique-email fragment for the edit field |
| GET | `/contacts/count` | Lazy `(N total Contacts)` fragment |

List search is a real GET form. htmx enhances the box: `hx-select` extracts
`#contacts-rows` from the full page. The sentinel row loads the next page when
revealed. Email uniqueness is checked on keyup. Delete keeps the POST form for
no-JS and `hx-delete` when htmx is present.

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
cmd/app            process: flags, port, sqlite file, seed
internal/server    mux, /assets/, / → /contacts
internal/handler   HTTP adapters
internal/contact   SQLite store via database/sql
assets             embed css/, js/
templates          html/template renderer
```

Dependency direction (enforced):

```text
cmd/app → server → handler → templates
        ↘ contact ↗
               ↘ assets
```

`internal/contact` is the only package that imports the SQLite driver.

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
