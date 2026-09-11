package server_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
