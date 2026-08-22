package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zackerydev/goth-template/internal/handler"
)

func TestHandlersRenderHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.Handler
		want    string
	}{
		{name: "home", handler: handler.Home(), want: "Start building."},
		{name: "greeting", handler: handler.Greeting(), want: "HTMX is connected."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			test.handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
				t.Fatalf("content type = %q, want text/html", contentType)
			}
			if !strings.Contains(response.Body.String(), test.want) {
				t.Errorf("body does not contain %q", test.want)
			}
		})
	}
}
