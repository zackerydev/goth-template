# Use progressive HTMX with local browser assets

## Status

Accepted

## Context

The application needs lightweight interactivity without a client-rendered
application, frontend build pipeline, or runtime CDN dependency. Server-only
HTML keeps the system simple but cannot update a focused page region. A SPA
would duplicate routing and state across Go and JavaScript and add substantial
tooling.

HTMX can enhance server-rendered HTML, but its document, fragment, error, and
asset-delivery contracts need to be explicit.

## Decision

Go and templ remain authoritative for application state and HTML. Echo routes
return either complete templ pages or stable templ fragments. HTMX provides
progressive enhancement only; it does not introduce client-side routing or
application state.

HTMX is committed under `assets/js/` with its upstream license, embedded in the
Go binary, and served under `/assets/`. Pages load the local HTMX file and the
small local `app.js` configuration. Runtime CDNs, global `hx-boost`, and a
frontend package or build system are not part of the baseline.

`app.js` defines response handling deliberately: `204` does not swap, successful
and redirect responses may swap, `422` swaps validation content while remaining
an error, and other client or server errors do not replace the current region.
When a response representation varies by the `HX-Request` header, the server
must include `Vary: HX-Request`.

Forms added to the application must remain complete native HTML submissions
with real actions, methods, labels, and server-side validation. Native success
uses Post/Redirect/Get and native validation failure returns a useful complete
page. HTMX may replace only a stable fragment and may use `HX-Redirect` when
navigation is required.

The initial `/greeting` route and `hx-get` button are a disposable wiring proof.
They demonstrate an `outerHTML` fragment swap and are removed when the first
real interaction replaces them.

Tests verify embedded files and licenses, document metadata and script sources,
HTMX attributes, fragment boundaries, and HTTP route behavior. Generated templ
Go remains committed so normal builds do not require generation first.

## Consequences

The binary is self-contained, interactive behavior remains server-owned, and
future forms preserve a useful no-JavaScript path. HTMX upgrades are deliberate
repository changes that update the vendored file, license when necessary, and
asset tests together.

Enhanced endpoints may have both page and fragment representations, so tests
must cover native and HTMX requests. The progressive-enhancement constraint
rules out interactions that exist only in client-side state unless a later ADR
introduces and justifies that capability.
