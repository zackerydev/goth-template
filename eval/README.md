# Agent eval: HTMX vs SPA vs SPA+MCP

The claim is that a hypermedia-first page is already crawlable: an agent with
ordinary HTTP can finish CRUD and search from `a` / `form` / `hx-*` in the
HTML. It should not need MCP. A production SPA hides those moves in JavaScript,
so the same agent stalls until you bolt on tools.

Part of the score is **what the agent had to do**. Capability is a first-class
outcome, not a footnote.

## Capability ladder

| Interface | Agent may | Expected HTMX | Expected SPA | Expected SPA+MCP |
| --- | --- | --- | --- | --- |
| `http` | GET/POST HTML, follow `a`/`form`/`hx-*`, no JS | pass | fail | fail without tools |
| `mcp` | `list`/`search`/`get`/`create`/`update`/`delete` | skip | n/a | pass |
| `browser` | Playwright accessibility tree, no screenshots | pass, wasteful | maybe, brittle | pass, wasteful |
| `computer_use` | screenshots and clicks | pass, most wasteful | SPA-without-MCP path | pass, wasteful |

This slice runs `htmx×http`, `spa×http`, `spa×mcp`, and optional `spa×browser`.
Computer-use is a follow-on cell.

## Fairness rules

- Production SPA: Vite CSR, `minify: true`, sourcemaps off, hashed assets.
- Same model and temperature for every cell when using `--agent llm`.
- Same user prompt; the system prompt differs only by available interface.
- No JSON-LD, hidden agent manifests, `data-agent-*`, or `llms.txt`.
- The SPA agent is not given `/api/v1` unless that URL appears in HTML it
  fetched. MCP is how we *explicitly* grant the JSON API.
- The HTTP adapter is a crawler, not an MCP: it gets HTML, extracts links and
  forms from the document, and POSTs `application/x-www-form-urlencoded`
  fields. Allowlist is HATEOAS-strict (start URL plus discovered `a` / `form` /
  `hx-*`). Script tags are not next actions. POST to a form with named inputs
  and no `fields` is rejected as an empty body.
- Eval seed is isolated from the talk demo: Dan (not Hibiki), Chun-Li, three
  contacts named Ryu, no Ken Masters, 16 rows so the list paginates.

## Tasks and oracles

| ID | Prompt | Oracle |
| --- | --- | --- |
| `update-dan` | Update contact Dan's last name to Hibiki | row `first=Dan` has `last=Hibiki` |
| `create-ken` | Create a contact for Ken Masters | row exists (email may be agent-chosen) |
| `delete-chun-li` | Delete Chun-Li's contact | no row with that name |
| `count-ryu` | How many contacts named Ryu do I have | integer equals `COUNT` on first or last matching `Ryu`; fail if the list is paginated and the agent did not search |

Oracles read SQLite. They never trust the model's self-report.

## Run

```sh
mise run spa-build
mise run eval
```

`mise run eval` boots a fresh eval-seeded SQLite copy per trial and writes
JSONL plus an HTML readout under `eval/results/`. Open
`eval/results/index.html` (or the timestamped `.html` next to the JSONL).
Default agent is `scripted` (no API keys): HTMX HTTP must pass, SPA HTTP
must fail, SPA MCP must pass.

Rebuild the HTML from an existing JSONL file:

```sh
npx --prefix eval tsx src/run.ts --from eval/results/run.jsonl
```

Live model loop via `pi` (kimi-k3 on opencode-go). This is what fills in tokens
and USD; the scripted agent records zeros because it never calls a model.

```sh
mise run eval-llm
```

Or:

```sh
npx --prefix eval tsx src/run.ts --matrix --agent llm --model kimi-k3 --repeats 1
```

Optional Playwright cell (install the package and Chromium first):

```sh
npm --prefix eval install -D playwright
npx --prefix eval playwright install chromium
npx --prefix eval tsx src/run.ts --matrix --agent scripted --browser
```

MCP stdio server wrapping the JSON API (not SQL):

```sh
BASE_URL=http://127.0.0.1:PORT npx --prefix eval tsx src/mcp-server.ts --stdio
```

JSONL fields: `task`, `app`, `interface`, `success`, `capability`, `tokens`,
`cost_usd`, `work`, `bytes_in_context`, `used_search`, `answer`, `trace`.

`APP_SEED=eval` selects the eval fixture. The demo book names stay the default
seed for the talk app.
