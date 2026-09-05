# Use progressive HTMX with local browser assets

## Status

Accepted

## Context

The application needs lightweight interactivity without a client-rendered
application, frontend build pipeline, or runtime CDN dependency. Server-only
HTML keeps the system simple while HTMX can update a focused page region.

HTMX needs explicit document, boosted-navigation, fragment, error, and
asset-delivery contracts. The renderer and handler must preserve a useful
native path rather than requiring JavaScript for navigation.

## Decision

Go and `html/template` remain authoritative for application state and HTML.
Echo routes return either complete pages or stable fragments. HTMX provides
progressive enhancement only; it does not introduce client-side application
state or a frontend build system.

HTMX is committed under `assets/js/` with its upstream license, embedded in the
Go binary, and served under `/assets/`. Pages load the local HTMX file and the
small local `app.js` configuration. Runtime CDNs and frontend package/build
systems are not part of the baseline.

The shared layout enables inherited boosted navigation with
`hx-boost="true"`, targeting `#main-content` and swapping `innerHTML`. Explicit
fragment attributes override that inheritance: the greeting button targets
`#greeting` and swaps its `outerHTML`.

The response contract is:

- native requests render the named page and receive the complete document;
- a request with `HX-Boosted: true` renders an escaped `<title>` followed by the
  page content, without the doctype, document wrappers, navigation shell, or
  outer `main` element;
- `HX-Request: true` without `HX-Boosted: true` still receives the complete
  document;
- history-restoration requests receive the complete document even when they
  also include `HX-Boosted: true`;
- page responses vary on `HX-Boosted` and
  `HX-History-Restore-Request`, preserving existing `Vary` values; and
- explicit partial responses, such as `/greeting`, never include document-title
  markup.

`app.js` defines response handling deliberately: `204` does not swap, successful
and redirect responses may swap, `422` swaps validation content while remaining
an error, and other client or server errors do not replace the current region.
When a response representation varies by an HTMX request header, the server
must include the corresponding `Vary` header.

Forms added to the application must remain complete native HTML submissions
with real actions, methods, labels, and server-side validation. Native success
uses Post/Redirect/Get and native validation failure returns a useful complete
page. HTMX may replace only a stable fragment and may use `HX-Redirect` when
navigation is required.

The initial `/greeting` route and `hx-get` button are a disposable wiring proof.
They demonstrate an `outerHTML` fragment swap and are removed when the first
real interaction replaces them.

Tests verify embedded files and licenses, document metadata and script sources,
page/fragment boundaries, boosted and native response contracts, Vary headers,
and HTTP route behavior.

## Consequences

The binary is self-contained, interactive behavior remains server-owned, and
future forms preserve a useful no-JavaScript path. HTML production templates
are embedded and parsed before startup; development templates are read from the
current worktree on each request. HTMX upgrades are deliberate repository
changes that update the vendored file, license when necessary, and asset tests
together.

Enhanced endpoints may have both page and fragment representations, so tests
must cover native, ordinary HTMX, boosted, and history-restoration requests. The
baseline now permits inherited boosted navigation; this supersedes the earlier
ADR restriction against global boosting without changing the progressive-
enhancement constraint.
