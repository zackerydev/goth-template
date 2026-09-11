package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zackerydev/goth-template/internal/contact"
	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/templates"
)

func TestContactListAndCreate(t *testing.T) {
	t.Parallel()

	store, application := openApp(t)
	createPerson(t, store, contact.Contact{First: "Ada", Last: "Lovelace", Email: "ada@example.com"})

	list := get(t, application, "/contacts")
	assertStatus(t, list, http.StatusOK)
	assertBody(t, list, "Ada", "Add Contact", `hx-get="/contacts"`)
	assertBody(t, get(t, application, "/contacts?q=no-such-person"), "No contacts found.")
	assertStatus(t, get(t, application, "/contacts?page=0"), http.StatusOK)
	assertStatus(t, get(t, application, "/contacts?page=2"), http.StatusOK)
	assertBody(t, get(t, application, "/contacts/new"), `action="/contacts/new"`)

	created := post(t, application, "/contacts/new", url.Values{
		"first_name": {"Carson"},
		"last_name":  {"Gross"},
		"phone":      {"555-0100"},
		"email":      {"carson@htmx.org"},
	})
	assertStatus(t, created, http.StatusSeeOther)
	if created.Header().Get("Location") != "/contacts" {
		t.Fatalf("create location = %q", created.Header().Get("Location"))
	}
	assertBody(t, doCookie(t, application, created), "Created New Contact!", "Carson")

	missing := post(t, application, "/contacts/new", url.Values{"email": {""}})
	assertStatus(t, missing, http.StatusUnprocessableEntity)
	assertBody(t, missing, "Email Required")
}

func TestContactShowEditAndDelete(t *testing.T) {
	t.Parallel()

	store, application := openApp(t)
	item := createPerson(t, store, contact.Contact{First: "Ada", Last: "Lovelace", Email: "ada@example.com"})
	createPerson(t, store, contact.Contact{First: "Carson", Email: "carson@htmx.org"})
	path := "/contacts/" + strconv.FormatInt(item.ID, 10)

	assertBody(t, get(t, application, path), "ada@example.com", "Edit")
	assertBody(t, get(t, application, path+"/edit"), `hx-delete="`+path+`"`)

	updated := post(t, application, path+"/edit", url.Values{
		"first_name": {"Ada"},
		"last_name":  {"Byron"},
		"phone":      {"555-0112"},
		"email":      {"ada@example.com"},
	})
	assertStatus(t, updated, http.StatusSeeOther)
	assertBody(t, doCookie(t, application, updated), "Updated Contact!", "Byron")

	conflict := post(t, application, path+"/edit", url.Values{"email": {"carson@htmx.org"}})
	assertStatus(t, conflict, http.StatusUnprocessableEntity)
	assertBody(t, conflict, "Email Must Be Unique")

	deleted := post(t, application, path+"/delete", url.Values{})
	assertStatus(t, deleted, http.StatusSeeOther)
	assertBody(t, doCookie(t, application, deleted), "Deleted Contact!")
}

func TestEmailCountAndHTMXRedirects(t *testing.T) {
	t.Parallel()

	store, application := openApp(t)
	item := createPerson(t, store, contact.Contact{First: "Ken", Email: "ken@bell-labs.com"})
	createPerson(t, store, contact.Contact{First: "Dennis", Email: "dmr@bell-labs.com"})
	path := "/contacts/" + strconv.FormatInt(item.ID, 10)

	okEmail := get(t, application, path+"/email?email=ken@bell-labs.com")
	if okEmail.Code != http.StatusOK || okEmail.Body.String() != "" {
		t.Fatalf("valid email = %d %q", okEmail.Code, okEmail.Body.String())
	}
	taken := get(t, application, path+"/email?email=dmr@bell-labs.com")
	if taken.Body.String() != "Email Must Be Unique" {
		t.Fatalf("taken email = %q", taken.Body.String())
	}
	required := get(t, application, path+"/email?email=")
	if required.Body.String() != "Email Required" {
		t.Fatalf("required email = %q", required.Body.String())
	}
	assertBody(t, get(t, application, "/contacts/count"), "2 total Contacts")

	other := createPerson(t, store, contact.Contact{First: "Rob", Email: "rob@swtch.com"})
	htmxDelete := httptest.NewRequest(http.MethodDelete, "/contacts/"+strconv.FormatInt(other.ID, 10), nil)
	htmxDelete.Header.Set("HX-Request", "true")
	htmxResponse := httptest.NewRecorder()
	application.ServeHTTP(htmxResponse, htmxDelete)
	if htmxResponse.Code != http.StatusSeeOther || htmxResponse.Header().Get("HX-Redirect") != "/contacts" {
		t.Fatalf("htmx delete = %d %v", htmxResponse.Code, htmxResponse.Header())
	}

	boosted := httptest.NewRequest(http.MethodPost, "/contacts/new", strings.NewReader(url.Values{
		"first_name": {"Joe"},
		"email":      {"joe@blow.com"},
	}.Encode()))
	boosted.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	boosted.Header.Set("HX-Request", "true")
	boosted.Header.Set("HX-Boosted", "true")
	boostedResponse := httptest.NewRecorder()
	application.ServeHTTP(boostedResponse, boosted)
	if boostedResponse.Header().Get("Location") != "/contacts" || boostedResponse.Header().Get("HX-Redirect") != "" {
		t.Fatalf("boosted create = %v", boostedResponse.Header())
	}
}

func TestMissingContacts(t *testing.T) {
	t.Parallel()

	_, application := openApp(t)
	assertStatus(t, get(t, application, "/contacts/999"), http.StatusNotFound)
	assertStatus(t, get(t, application, "/contacts/nope"), http.StatusNotFound)
	assertStatus(t, get(t, application, "/contacts/nope/edit"), http.StatusNotFound)
	assertStatus(t, get(t, application, "/contacts/nope/email"), http.StatusNotFound)
	assertStatus(t, post(t, application, "/contacts/nope/delete", url.Values{}), http.StatusNotFound)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodDelete, "/contacts/999", nil)), http.StatusNotFound)
	assertStatus(t, get(t, application, "/contacts/999/email"), http.StatusNotFound)
}

func TestClosedStoreErrors(t *testing.T) {
	t.Parallel()

	store, application := openApp(t)
	item := createPerson(t, store, contact.Contact{First: "Grace", Email: "grace@cobol.dev"})
	path := "/contacts/" + strconv.FormatInt(item.ID, 10)
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	assertStatus(t, get(t, application, "/contacts"), http.StatusInternalServerError)
	assertStatus(t, get(t, application, "/contacts/count"), http.StatusInternalServerError)
	assertStatus(t, get(t, application, path), http.StatusInternalServerError)
	assertStatus(t, get(t, application, path+"/edit"), http.StatusInternalServerError)
	assertStatus(t, post(t, application, path+"/edit", url.Values{"email": {"a@b.c"}}), http.StatusInternalServerError)
	assertStatus(t, post(t, application, "/contacts/new", url.Values{"email": {"a@b.c"}}), http.StatusInternalServerError)
	assertStatus(t, get(t, application, path+"/email?email=a@b.c"), http.StatusInternalServerError)
	assertStatus(t, post(t, application, path+"/delete", url.Values{}), http.StatusInternalServerError)
}

func TestBrokenTemplateAndFlash(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"layouts/layout.html":          &fstest.MapFile{Data: []byte(`{{ define "layout" }}{{ template "content" . }}{{ end }}`)},
		"partials/document-title.html": &fstest.MapFile{Data: []byte(`{{ define "document-title" }}{{ end }}`)},
		"partials/flash.html":          &fstest.MapFile{Data: []byte(`{{ define "flash" }}{{ end }}`)},
		"pages/contacts.html":          &fstest.MapFile{Data: []byte(`{{ define "contacts" }}{{ template "layout" . }}{{ end }}{{ define "content" }}ok{{ end }}`)},
		"pages/contacts-new.html":      &fstest.MapFile{Data: []byte(`{{ define "contacts-new" }}ok{{ end }}`)},
		"pages/contacts-show.html":     &fstest.MapFile{Data: []byte(`{{ define "contacts-show" }}ok{{ end }}`)},
		"pages/contacts-edit.html":     &fstest.MapFile{Data: []byte(`{{ define "contacts-edit" }}ok{{ end }}`)},
	}
	renderer, err := templates.NewDevelopmentFS(source)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	store, err := contact.Open(filepath.Join(t.TempDir(), "contacts.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	mux := http.NewServeMux()
	handler.Register(mux, renderer, store)

	source["pages/contacts.html"].Data = []byte(`{{ define "contacts" }}`)
	response := get(t, mux, "/contacts")
	if response.Code != http.StatusInternalServerError || response.Body.String() != "template rendering failed\n" {
		t.Fatalf("broken list = %d %q", response.Code, response.Body.String())
	}

	source["pages/contacts.html"].Data = []byte(`{{ define "contacts" }}{{ template "layout" . }}{{ end }}{{ define "content" }}ok{{ end }}`)
	request := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	request.AddCookie(&http.Cookie{ //nolint:gosec // G124: malformed-value fixture, not a browser cookie.
		Name: "flash", Value: "%",
	})
	badFlash := httptest.NewRecorder()
	mux.ServeHTTP(badFlash, request)
	if badFlash.Code != http.StatusOK {
		t.Fatalf("invalid flash status = %d", badFlash.Code)
	}
}

func TestWriteIgnoresResponseWriterErrors(t *testing.T) {
	t.Parallel()

	_, application := openApp(t)
	application.ServeHTTP(failingResponseWriter{header: make(http.Header)}, httptest.NewRequest(http.MethodGet, "/contacts/new", nil))
	application.ServeHTTP(failingResponseWriter{header: make(http.Header)}, httptest.NewRequest(http.MethodGet, "/contacts/count", nil))
}

func openApp(t *testing.T) (*contact.Store, http.Handler) {
	t.Helper()
	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	store := openStore(t)
	mux := http.NewServeMux()
	handler.Register(mux, renderer, store)
	return store, mux
}

func openStore(t *testing.T) *contact.Store {
	t.Helper()
	store, err := contact.Open(filepath.Join(t.TempDir(), "contacts.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func createPerson(t *testing.T, store *contact.Store, item contact.Contact) contact.Contact {
	t.Helper()
	created, err := store.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	return created
}

func get(t *testing.T, application http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, application, httptest.NewRequest(http.MethodGet, path, nil))
}

func post(t *testing.T, application http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return send(t, application, request)
}

func send(t *testing.T, application http.Handler, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	return response
}

func doCookie(t *testing.T, application http.Handler, previous *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	location := previous.Header().Get("Location")
	if location == "" {
		t.Fatal("missing redirect location")
	}
	request := httptest.NewRequest(http.MethodGet, location, nil)
	for _, cookie := range previous.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d body=%q", response.Code, want, response.Body.String())
	}
}

func assertBody(t *testing.T, response *httptest.ResponseRecorder, want ...string) {
	t.Helper()
	for _, snippet := range want {
		if !strings.Contains(response.Body.String(), snippet) {
			t.Fatalf("body does not contain %q: %s", snippet, response.Body.String())
		}
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (writer failingResponseWriter) Header() http.Header {
	return writer.header
}

func (failingResponseWriter) WriteHeader(int) {}

func (failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("response writer failed")
}
