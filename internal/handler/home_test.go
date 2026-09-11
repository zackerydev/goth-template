package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/templates"
)

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

	response := httptest.NewRecorder()
	handler.Home(renderer).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
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
