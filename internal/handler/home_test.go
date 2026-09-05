package handler_test

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/templates"
)

func TestHomeRendersNativeAndBoostedResponses(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	home := handler.Home(renderer)
	for _, test := range []homeResponseTest{
		{
			name:       "native",
			want:       []string{"<!doctype html>", "<html", "<main", "GoTH Template", "Start building."},
			wantStatus: http.StatusOK,
		},
		{
			name:       "boosted",
			header:     map[string]string{"HX-Request": "true", "HX-Boosted": "true"},
			want:       []string{"<title>GoTH Template</title>", "Start building."},
			notWant:    []string{"<!doctype html>", "<html", "<main", "site-header"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "ordinary HTMX",
			header:     map[string]string{"HX-Request": "true"},
			want:       []string{"<!doctype html>", "<main", "GoTH Template"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "history restoration",
			header:     map[string]string{"HX-Request": "true", "HX-Boosted": "true", "HX-History-Restore-Request": "true"},
			want:       []string{"<!doctype html>", "<main", "GoTH Template"},
			wantStatus: http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertHomeResponse(t, home, test)
		})
	}
}

type homeResponseTest struct {
	name       string
	header     map[string]string
	want       []string
	notWant    []string
	wantStatus int
}

func assertHomeResponse(t *testing.T, home http.Handler, test homeResponseTest) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	for name, value := range test.header {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()

	home.ServeHTTP(response, request)

	if response.Code != test.wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("content type = %q", contentType)
	}
	for _, want := range test.want {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body does not contain %q: %s", want, response.Body.String())
		}
	}
	for _, notWant := range test.notWant {
		if strings.Contains(response.Body.String(), notWant) {
			t.Errorf("body contains %q: %s", notWant, response.Body.String())
		}
	}
	assertVary(t, response.Header(), "HX-Boosted")
	assertVary(t, response.Header(), "HX-History-Restore-Request")
}

func TestBoostedHomeLoadsDevelopmentSourceOnce(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"layouts/layout.html":          &fstest.MapFile{Data: []byte(`{{ define "layout" }}{{ template "content" . }}{{ end }}`)},
		"partials/document-title.html": &fstest.MapFile{Data: []byte(`{{ define "document-title" }}<title>{{ .Title }}</title>{{ end }}`)},
		"partials/greeting.html":       &fstest.MapFile{Data: []byte(`{{ define "greeting" }}greeting{{ end }}`)},
		"pages/home.html":              &fstest.MapFile{Data: []byte(`{{ define "home" }}{{ template "layout" . }}{{ end }}{{ define "content" }}content{{ end }}`)},
	}
	renderer, err := templates.NewDevelopmentFS(&singleOpenFS{FS: source, opened: make(map[string]bool)})
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("HX-Boosted", "true")
	response := httptest.NewRecorder()
	handler.Home(renderer).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Body.String(); !strings.Contains(got, "<title>GoTH Template</title>") || !strings.Contains(got, "content") {
		t.Errorf("body = %q", got)
	}
}

type singleOpenFS struct {
	fs.FS
	opened map[string]bool
}

func (source *singleOpenFS) Open(name string) (fs.File, error) {
	if source.opened[name] {
		return nil, errors.New("template source opened more than once")
	}
	source.opened[name] = true
	return source.FS.Open(name)
}

func (source *singleOpenFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(source.FS, name)
}

func TestHomePreservesExistingVaryHeaders(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	response.Header().Set("Vary", "Accept-Encoding, HX-Boosted")

	handler.Home(renderer).ServeHTTP(response, request)

	assertVary(t, response.Header(), "Accept-Encoding")
	assertVary(t, response.Header(), "HX-Boosted")
	assertVary(t, response.Header(), "HX-History-Restore-Request")
}

func TestGreetingRendersExplicitPartial(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/greeting", nil)
	response := httptest.NewRecorder()

	handler.Greeting(renderer).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if strings.Contains(response.Body.String(), "<html") || !strings.Contains(response.Body.String(), "HTMX is connected.") {
		t.Errorf("greeting body = %q", response.Body.String())
	}
	if response.Header().Get("Vary") != "" {
		t.Errorf("greeting Vary = %q, want empty", response.Header().Get("Vary"))
	}
}

func TestHandlersReturnCleanErrors(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"layouts/layout.html":          &fstest.MapFile{Data: []byte(`{{ define "layout" }}{{ template "content" . }}{{ end }}`)},
		"partials/document-title.html": &fstest.MapFile{Data: []byte(`{{ define "document-title" }}<title>{{ .Title }}</title>{{ end }}`)},
		"partials/greeting.html":       &fstest.MapFile{Data: []byte(`{{ define "greeting" }}greeting{{ end }}`)},
		"pages/home.html":              &fstest.MapFile{Data: []byte(`{{ define "home" }}{{ template "layout" . }}{{ end }}{{ define "content" }}valid{{ end }}`)},
	}
	renderer, err := templates.NewDevelopmentFS(source)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	source["pages/home.html"].Data = []byte(`{{ define "home" }}`)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.Home(renderer).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Body.String(); got != "template rendering failed\n" {
		t.Errorf("error body = %q", got)
	}
	if strings.Contains(response.Body.String(), "valid") {
		t.Error("error response contains stale successful output")
	}
}

func TestHandlersAcceptNilRenderer(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.Home(nil).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Errorf("home nil renderer status = %d", response.Code)
	}

	response = httptest.NewRecorder()
	handler.Greeting(nil).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Errorf("greeting nil renderer status = %d", response.Code)
	}
}

func TestHandlersIgnoreResponseWriterErrors(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	handler.Greeting(renderer).ServeHTTP(failingResponseWriter{header: make(http.Header)}, httptest.NewRequest(http.MethodGet, "/greeting", nil))
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

func assertVary(t *testing.T, header http.Header, want string) {
	t.Helper()
	for _, value := range header.Values("Vary") {
		for item := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), want) {
				return
			}
		}
	}
	t.Errorf("Vary = %q, missing %q", header.Values("Vary"), want)
}
