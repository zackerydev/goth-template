package handler

func seedTasks() []Task {
	return []Task{
		{ID: 1, Title: "Make the first five seconds unforgettable", Category: "Experience", Status: "In progress", Priority: "High", Notes: "A confident headline. A clear next step. No onboarding carousel."},
		{ID: 2, Title: "Command center, not another dashboard", Category: "Design", Status: "In progress", Priority: "High", Notes: "Bring the important work forward. Let the rest recede."},
		{ID: 3, Title: "Give empty states a little personality", Category: "Experience", Status: "Backlog", Priority: "Low", Notes: "Helpful, not cute. Always show a way forward."},
		{ID: 4, Title: "Ship keyboard-friendly navigation", Category: "Engineering", Status: "Backlog", Priority: "High", Notes: "Real links, visible focus, labeled forms. Start with the platform."},
		{ID: 5, Title: "Design a calmer notification system", Category: "Design", Status: "Backlog", Priority: "Medium", Notes: "One useful message beats five distracting ones."},
		{ID: 6, Title: "Let URLs tell the whole story", Category: "Engineering", Status: "Done", Priority: "Medium", Notes: "Copy a filtered view. Open a task in a new tab. Back should just work."},
		{ID: 7, Title: "Build the release preview pipeline", Category: "Engineering", Status: "In progress", Priority: "Medium", Notes: "Run the preview in the activity view to see conditional polling."},
		{ID: 8, Title: "Write like a person, not a system", Category: "Content", Status: "Done", Priority: "Low", Notes: "Short sentences. Specific verbs. No unnecessary ceremony."},
		{ID: 9, Title: "Make every interaction explain itself", Category: "Content", Status: "Backlog", Priority: "Medium", Notes: "The response is the interface. Open the wire inspector and follow along."},
	}
}

func lessons() []Lesson {
	return []Lesson{
		{ID: "navigation", Name: "URL-driven navigation", Summary: "An application shell without a client router. Every view and task is a real URL. Direct visits get a document; htmx visits get the workspace.", Markup: `<a href="/lab?view=list" hx-get="/lab?view=list"
   hx-target="#workspace" hx-swap="outerHTML transition:true"
   hx-push-url="true">List view</a>`, Response: "GET /lab?view=list → 200 text/html\n\n<section id=\"workspace\">…list view…</section>\n\nWithout HX-Request: true, return the complete document.\nVary: HX-Request prevents mixing the two representations.", Try: "Switch Board → List. Open a task. Copy the URL into a new tab, then use Back."},
		{ID: "search", Name: "Live search + request sync", Summary: "Debounce input, include the whole filter form, and replace older requests. Search state lives in the URL, not a client-side store.", Markup: `<input name="q" hx-get="/lab"
   hx-trigger="input changed delay:300ms"
   hx-include="closest form" hx-sync="closest form:replace"
   hx-target="#workspace" hx-swap="outerMorph"
   hx-replace-url="true" />`, Response: "GET /lab?q=design&view=board → 200 text/html\n\nThe server filters tasks and returns the workspace.\nouterMorph keeps the focused search input and caret in place.\nReplace history for keystrokes; push history for navigation.", Try: "Type ‘design’ into the workspace search. Combine it with a status filter. Clear both to recover all cards."},
		{ID: "validation", Name: "Editing + server validation", Summary: "A deep-linked detail panel is just another representation of the workspace. Invalid forms return 422 HTML, keeping the draft and explaining the error.", Markup: `<form method="post" action="/lab/task?id=1"
   hx-post="/lab/task?id=1" hx-target="#workspace"
   hx-swap="outerHTML"
   hx-status:422="target:#editor swap:outerHTML"
   hx-disable="find button">…</form>`, Response: "POST /lab/task?id=1 → 422 text/html\n\n<aside id=\"editor\">\n  <p role=\"alert\">Give this task a title…</p>\n  <form>…your submitted values…</form>\n</aside>\n\nValid submission → 200 workspace + notification partial.", Try: "Open a card, enter a one-character title, and save. Then fix it and change the workflow status."},
		{ID: "partials", Name: "One response, multiple regions", Summary: "Update the board and announce the result in the same round trip. Explicit htmx 4 partials keep distant regions in sync without a global state store.", Markup: `<section id="workspace">…updated cards and counts…</section>
<hx-partial hx-target="#notice" hx-swap="innerHTML">
  Saved: Ship keyboard-friendly navigation
</hx-partial>`, Response: "POST /lab/task?id=4 → 200 text/html\nHX-Replace-Url: /lab?view=board\n\nMain swap: #workspace\nAdditional partial: #notice (persistent live region)\nCounts derive from the same server state as the cards.", Try: "Move a task to Done using its detail panel. Watch the card, metrics, URL, and bottom notification update together."},
		{ID: "bulk", Name: "Bulk actions + native forms", Summary: "Checkboxes already know how to submit a collection. The server applies the operation, then renders the authoritative result. No selection store required.", Markup: `<form method="post" action="/lab/bulk?view=list"
   hx-post="/lab/bulk?view=list" hx-target="#workspace"
   hx-swap="outerHTML" hx-disable="find button">
  <input type="checkbox" name="task" value="1" />
  <input type="checkbox" name="task" value="2" />
  <button>Complete selected</button>
</form>`, Response: "POST /lab/bulk?view=list\nContent-Type: application/x-www-form-urlencoded\n\ntask=1&task=2\n\n→ 200 workspace + notification partial\nNo JavaScript data model, reducer, or optimistic rollback.", Try: "Open List view. Select two tasks and complete them together. Try submitting an empty selection too."},
		{ID: "polling", Name: "A self-stopping live job", Summary: "The server decides whether polling continues by including—or omitting—the trigger in its next response. This demo measures a ten-second preview with a server clock; it does not deploy anything.", Markup: `<section id="pulse" hx-get="/lab/pulse"
   hx-trigger="every 1s" hx-swap="outerHTML">
  <progress value="40" max="100">40%</progress>
</section>`, Response: "GET /lab/pulse → 200 text/html\n\nWhile running: return #pulse with hx-trigger=\"every 1s\".\nAt 100%: return #pulse without polling attributes.\nThe loop ends because the new HTML says it should.", Try: "Open Activity and run a preview. Watch progress and the wire inspector. Requests stop at 100%."},
		{ID: "pagination", Name: "Append without rebuilding", Summary: "Older activity arrives as HTML plus its own continuation control. The server owns pagination, including the point where there is no next page.", Markup: `<button hx-get="/lab/events?offset=5"
   hx-target="this" hx-swap="outerHTML">
  Load older activity
</button>`, Response: "GET /lab/events?offset=5 → 200 text/html\n\n<div class=\"event\">…older event…</div>\n<div class=\"event\">…older event…</div>\n<button hx-get=\"/lab/events?offset=10\" …>Load older</button>\n\nThe button replaces itself with items + the next button.", Try: "Make six or more edits, then open Activity and load older entries. Existing entries stay untouched."},
		{ID: "platform", Name: "The platform is the framework", Summary: "Native links, forms, details, focus styles, and HTTP redirects provide the foundation. htmx enhances the transport. A little JavaScript powers observation, not application state.", Markup: `<form method="post" action="/lab/reset"
   hx-post="/lab/reset" hx-target="#workspace"
   hx-swap="outerHTML" hx-confirm="Reset the shared sandbox?">
  <button>Reset sandbox</button>
</form>`, Response: "Enhanced POST → HTML fragments\nOrdinary POST → 303 See Other → full document\n\nSearch, navigation, editing, and bulk actions work without JS.\nLive polling, partial swaps, and this inspector require JS.\nThis is a shared demo, not an authenticated production app.", Try: "Disable JavaScript and reload. Navigate, submit search, edit a task. Notice what still works—and what enhancement adds."},
	}
}
