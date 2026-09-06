package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	application := server.New(handler.Home(renderer), handler.Greeting(renderer))

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantType   string
		wantBody   string
	}{
		{name: "home", path: "/", wantStatus: http.StatusOK, wantType: "text/html", wantBody: "Start building."},
		{name: "greeting", path: "/greeting", wantStatus: http.StatusOK, wantType: "text/html", wantBody: "HTMX is connected."},
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
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Errorf("body does not contain %q", test.wantBody)
			}
		})
	}
}
