package handler_test

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/templates"
)

type labClient struct{ app http.Handler }

func newLabClient(t *testing.T) labClient {
	t.Helper()
	renderer, err := templates.New()
	if err != nil {
		t.Fatal(err)
	}
	return labClient{app: handler.NewLab(renderer)}
}

func (client labClient) request(t *testing.T, method, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	request.Header.Set("HX-Request", "true")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	client.app.ServeHTTP(response, request)
	return response
}

func taskForm(title, status string) url.Values {
	return url.Values{"title": {title}, "status": {status}, "priority": {"High"}, "notes": {"Keep the platform in charge."}}
}

func TestLabDocumentAndFragments(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	for _, headers := range []http.Header{{}, {"Hx-Request": {"true"}, "Hx-History-Restore-Request": {"true"}}, {"Hx-Request": {"true"}, "Hx-Request-Type": {"full"}}} {
		request := httptest.NewRequest(http.MethodGet, "/lab?view=list&id=1", nil)
		request.Header = headers
		response := httptest.NewRecorder()
		client.app.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "<!doctype html>") || !strings.Contains(response.Body.String(), `id="editor"`) {
			t.Fatalf("deep link must render full document and editor: %d %s", response.Code, response.Body.String())
		}
	}
	response := client.request(t, http.MethodGet, "/lab?view=list", nil)
	if strings.Contains(response.Body.String(), "<!doctype html>") || !strings.Contains(response.Body.String(), "Complete selected") {
		t.Fatal("enhanced list navigation must return only the workspace")
	}
	if response.Header().Get("Vary") != "HX-Request" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("representations must not be mixed or cached")
	}
}

func TestLabFiltersAndLessonRoutes(t *testing.T) {
	t.Parallel()
	cases := []struct{ path, want, absent string }{
		{"/lab?q=DESIGN&status=Backlog", "Design a calmer", "Make the first five"},
		{"/lab?q=no-match", "No matching tasks", "HL–1"},
		{"/lab?view=list&q=no-match", "Nothing matches", "HL–1"},
		{"/lab?view=unknown", "Studio workspace", "pattern-layout"},
		{"/lab?id=-1", "Create task", "Delete task"},
		{"/lab?view=patterns&lesson=validation", "Editing &#43; server validation", "task-card"},
		{"/lab?view=patterns&lesson=unknown", "URL-driven navigation", "task-card"},
		{"/lab?view=activity", "Run preview", `hx-trigger="every 1s"`},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			response := newLabClient(t).request(t, http.MethodGet, tc.path, nil)
			body := response.Body.String()
			if response.Code != http.StatusOK || !strings.Contains(body, tc.want) || strings.Contains(body, tc.absent) {
				t.Fatalf("unexpected representation: %d %s", response.Code, body)
			}
		})
	}
}

func TestLabEditValidationAndMultiRegionResponse(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	bad := client.request(t, http.MethodPost, "/lab/task?id=1", taskForm("x", "Done"))
	if bad.Code != http.StatusUnprocessableEntity || !strings.Contains(bad.Body.String(), `value="x"`) || strings.Contains(bad.Body.String(), `id="workspace"`) {
		t.Fatalf("validation must return only editor and preserve draft: %d %s", bad.Code, bad.Body.String())
	}
	unchanged := client.request(t, http.MethodGet, "/lab?status=In+progress", nil)
	if !strings.Contains(unchanged.Body.String(), "Make the first five") {
		t.Fatal("invalid edit mutated the task")
	}
	saved := client.request(t, http.MethodPost, "/lab/task?id=1&view=list&q=Renamed", taskForm("Renamed <script>alert(1)</script>", "Done"))
	if saved.Code != http.StatusOK || !strings.Contains(saved.Body.String(), `<hx-partial hx-target="#notice"`) || strings.Contains(saved.Body.String(), "<script>alert(1)</script>") {
		t.Fatalf("save must escape content and update multiple regions: %d %s", saved.Code, saved.Body.String())
	}
	if saved.Header().Get("HX-Replace-Url") != "/lab?q=Renamed&view=list" {
		t.Fatalf("save lost filters or retained editor URL: %s", saved.Header())
	}
	done := client.request(t, http.MethodGet, "/lab?status=Done", nil)
	if !strings.Contains(done.Body.String(), "Renamed &lt;script&gt;") {
		t.Fatal("saved task was not moved to Done")
	}
}

func TestLabRejectsInvalidFieldsWithoutMutation(t *testing.T) {
	t.Parallel()
	cases := []struct{ field, value, message string }{
		{"title", strings.Repeat("x", 101), "between 3 and 100"},
		{"status", "Imaginary", "valid workflow status"},
		{"priority", "Urgent", "valid priority"},
		{"notes", strings.Repeat("x", 2001), "under 2,000"},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()
			values := taskForm("Valid title", "Backlog")
			values.Set(tc.field, tc.value)
			client := newLabClient(t)
			response := client.request(t, http.MethodPost, "/lab/task?id=-1", values)
			if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), tc.message) {
				t.Fatalf("invalid %s accepted: %d %s", tc.field, response.Code, response.Body.String())
			}
			if strings.Contains(client.request(t, http.MethodGet, "/lab", nil).Body.String(), "Valid title") {
				t.Fatal("invalid creation was persisted")
			}
		})
	}
}

func TestLabCreateBulkDeleteAndReset(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	created := client.request(t, http.MethodPost, "/lab/task?id=-1", taskForm("A fresh idea", "Backlog"))
	if created.Code != http.StatusOK || !strings.Contains(created.Body.String(), "HL–10") {
		t.Fatal("new task must receive next ID")
	}
	empty := client.request(t, http.MethodPost, "/lab/bulk?view=list", nil)
	if !strings.Contains(empty.Body.String(), "Select at least one") {
		t.Fatal("empty bulk submission must explain how to proceed")
	}
	bulk := client.request(t, http.MethodPost, "/lab/bulk?view=list", url.Values{"task": {"1", "10", "999"}})
	if bulk.Code != http.StatusOK {
		t.Fatalf("bulk status: %d", bulk.Code)
	}
	done := client.request(t, http.MethodGet, "/lab?status=Done", nil).Body.String()
	if !strings.Contains(done, "A fresh idea") || !strings.Contains(done, "Make the first five") {
		t.Fatal("bulk action did not complete both selected tasks")
	}
	removed := client.request(t, http.MethodPost, "/lab/remove?id=10", nil)
	if removed.Code != http.StatusOK || strings.Contains(removed.Body.String(), "A fresh idea") {
		t.Fatal("deleted task remains in workspace")
	}
	reset := client.request(t, http.MethodPost, "/lab/reset", nil)
	if !strings.Contains(reset.Body.String(), "Sandbox reset") {
		t.Fatal("reset must announce result")
	}
	if strings.Contains(client.request(t, http.MethodGet, "/lab?status=Done", nil).Body.String(), "Make the first five") {
		t.Fatal("reset did not restore seed state")
	}
}

func TestLabOrdinaryFormsRedirectAndPreserveInvalidDraft(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	for _, title := range []string{"x", "Plain form submission"} {
		request := httptest.NewRequest(http.MethodPost, "/lab/task?id=1&view=list", strings.NewReader(taskForm(title, "Done").Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		client.app.ServeHTTP(response, request)
		if title == "x" {
			if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "<!doctype html>") {
				t.Fatal("plain invalid submission must return full document")
			}
		} else if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/lab?view=list" {
			t.Fatalf("plain successful POST must redirect: %d %s", response.Code, response.Header())
		}
	}
}

func TestLabActivityPagination(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	for i := range 7 {
		client.request(t, http.MethodPost, "/lab/task?id=1", taskForm("Revision "+string(rune('A'+i)), "Backlog"))
	}
	first := client.request(t, http.MethodGet, "/lab?view=activity", nil).Body.String()
	if !strings.Contains(first, "Revision G") || strings.Contains(first, "Revision A") || !strings.Contains(first, "offset=5") {
		t.Fatal("first activity page must show latest five changes")
	}
	older := client.request(t, http.MethodGet, "/lab/events?offset=5", nil).Body.String()
	if !strings.Contains(older, "Revision A") || strings.Contains(older, "Load older activity") {
		t.Fatal("last activity page must show remaining changes without continuation")
	}
	if !strings.Contains(client.request(t, http.MethodGet, "/lab/events?offset=999", nil).Body.String(), "No activity yet") {
		t.Fatal("out-of-range pagination must be safe")
	}
	if !strings.Contains(client.request(t, http.MethodGet, "/lab/events?offset=-1", nil).Body.String(), "Revision G") {
		t.Fatal("negative offset must clamp to start")
	}
}

func TestLabPollingStopsAtCompletion(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	started := client.request(t, http.MethodPost, "/lab/build?view=activity", nil)
	if started.Code != http.StatusOK || !strings.Contains(started.Body.String(), `hx-trigger="every 1s"`) {
		t.Fatal("starting a build must render polling component")
	}
	pulse := client.request(t, http.MethodGet, "/lab/pulse", nil)
	if !strings.Contains(pulse.Body.String(), "Bringing it together") {
		t.Fatal("poll must show running job")
	}
	time.Sleep(10100 * time.Millisecond)
	finished := client.request(t, http.MethodGet, "/lab/pulse", nil).Body.String()
	if !strings.Contains(finished, `value="100"`) || strings.Contains(finished, "hx-trigger") || !strings.Contains(finished, "Run another preview") {
		t.Fatal("completed job must stop polling and allow restart")
	}
}

func TestLabFilterLinksKeepQueryDataOutOfURLStructure(t *testing.T) {
	t.Parallel()
	query := "a&status=Done#fragment"
	response := newLabClient(t).request(t, http.MethodGet, "/lab?view=list&id=1&q="+url.QueryEscape(query), nil)
	links := regexp.MustCompile(`hx-(?:get|post)="([^"]+)"`).FindAllStringSubmatch(response.Body.String(), -1)
	checked := 0
	for _, link := range links {
		target, err := url.Parse(html.UnescapeString(link[1]))
		if err != nil {
			t.Fatal(err)
		}
		if target.Query().Has("q") {
			checked++
			if target.Query().Get("q") != query || target.Query().Get("status") != "" || target.Fragment != "" {
				t.Fatalf("filter escaped into URL structure: %s", target)
			}
		}
	}
	if checked < 3 {
		t.Fatal("expected filter links and form actions in list editor")
	}
}

func TestLabRequestBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method, path string
		values       url.Values
		status       int
	}{
		{http.MethodGet, "/lab/missing", nil, http.StatusNotFound},
		{http.MethodPut, "/lab", nil, http.StatusMethodNotAllowed},
		{http.MethodPost, "/lab/missing", nil, http.StatusNotFound},
		{http.MethodPost, "/lab/task?id=999", taskForm("Missing task", "Backlog"), http.StatusNotFound},
		{http.MethodPost, "/lab/remove?id=999", nil, http.StatusNotFound},
		{http.MethodPost, "/lab/task?id=1", url.Values{"notes": {strings.Repeat("x", 17000)}}, http.StatusBadRequest},
		{http.MethodPost, "/lab/bulk", url.Values{"task": {"999"}}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			t.Parallel()
			response := newLabClient(t).request(t, tc.method, tc.path, tc.values)
			if response.Code != tc.status {
				t.Fatalf("status=%d, want %d", response.Code, tc.status)
			}
			if tc.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, POST" {
				t.Fatal("method rejection must advertise allowed methods")
			}
		})
	}
}

func TestLabCrossOriginMutationIsRejected(t *testing.T) {
	t.Parallel()
	client := newLabClient(t)
	request := httptest.NewRequest(http.MethodPost, "/lab/reset", nil)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response := httptest.NewRecorder()
	client.app.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin reset accepted: %d", response.Code)
	}
}

func TestLabRenderFailureDoesNotLeakPartialOutput(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	handler.NewLab(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/lab", nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != "Unable to render the lab\n" {
		t.Fatalf("render failure response: %d %s", response.Code, response.Body.String())
	}
	client := newLabClient(t)
	writer := failingResponseWriter{header: make(http.Header)}
	client.app.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/lab", nil))
	if writer.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatal("write failure changed representation headers")
	}
}
