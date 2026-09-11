package server_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zackerydev/goth-template/assets"
	"github.com/zackerydev/goth-template/internal/contact"
	"github.com/zackerydev/goth-template/internal/handler"
	"github.com/zackerydev/goth-template/internal/server"
	"github.com/zackerydev/goth-template/templates"
)

func TestRoutes(t *testing.T) {
	t.Parallel()

	renderer, err := templates.New()
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	store, err := contact.Open(filepath.Join(t.TempDir(), "contacts.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	application := server.New(func(mux *http.ServeMux) {
		handler.Register(mux, renderer, store)
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantType   string
		wantBody   string
	}{
		{name: "root redirect", path: "/", wantStatus: http.StatusSeeOther, wantType: "text/html", wantBody: "/contacts"},
		{name: "contacts", path: "/contacts", wantStatus: http.StatusOK, wantType: "text/html", wantBody: "contacts.app"},
		{name: "stylesheet", path: "/assets/css/app.css", wantStatus: http.StatusOK, wantType: "text/css"},
		{name: "spa redirect", path: "/app", wantStatus: http.StatusSeeOther, wantType: "text/html", wantBody: "/app/"},
		{name: "spa shell", path: "/app/", wantStatus: http.StatusOK, wantType: "text/html", wantBody: `id="root"`},
		{name: "spa index", path: "/app/index.html", wantStatus: http.StatusOK, wantType: "text/html", wantBody: `id="root"`},
		{name: "spa client route", path: "/app/contacts/new", wantStatus: http.StatusOK, wantType: "text/html", wantBody: `id="root"`},
		{name: "spa missing asset", path: "/app/missing.js", wantStatus: http.StatusNotFound, wantType: "text/plain", wantBody: "not found"},
		{name: "spa asset dir", path: "/app/assets/", wantStatus: http.StatusNotFound, wantType: "text/plain", wantBody: "not found"},
		{name: "assets hide spa", path: "/assets/app/bundle.js", wantStatus: http.StatusNotFound, wantType: "text/plain", wantBody: "not found"},
		{name: "not found", path: "/missing", wantStatus: http.StatusNotFound, wantType: "text/plain"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			application.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, test.wantType) {
				t.Errorf("content type = %q, want to contain %q", contentType, test.wantType)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) && response.Header().Get("Location") != test.wantBody {
				t.Errorf("body %q location %q does not contain %q", response.Body.String(), response.Header().Get("Location"), test.wantBody)
			}
			if test.name == "spa shell" && strings.Contains(response.Body.String(), "/api/v1") {
				t.Error("SPA document leaked the JSON API path")
			}
		})
	}
}

func TestNilRegisterStillServesAssets(t *testing.T) {
	t.Parallel()

	application := server.New(nil)
	request := httptest.NewRequest(http.MethodGet, "/assets/css/app.css", nil)
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestSPAServesHashedBundle(t *testing.T) {
	t.Parallel()

	bundle := hashedBundle(t)
	application := server.New(nil)
	request := httptest.NewRequest(http.MethodGet, "/app/"+bundle, nil)
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "/api/v1/contacts") {
		t.Fatal("minified bundle missing JSON API path")
	}
}

func hashedBundle(t *testing.T) string {
	t.Helper()
	entries, err := fs.ReadDir(assets.SPA(), "assets")
	if err != nil {
		t.Fatalf("read spa assets: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".js") {
			return "assets/" + entry.Name()
		}
	}
	t.Fatal("no hashed javascript bundle")
	return ""
}
