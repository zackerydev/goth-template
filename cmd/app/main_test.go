package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewApplication(t *testing.T) {
	t.Parallel()

	application, err := newApplication(false)
	if err != nil {
		t.Fatalf("create production application: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/lab", nil)
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Studio workspace") {
		t.Fatalf("application response = %d %q", response.Code, response.Body.String())
	}
}

func TestDevelopmentStartupPropagatesTemplateDirectoryError(t *testing.T) {
	t.Parallel()

	if _, err := newApplicationWithTemplateDirectory(true, filepath.Join(t.TempDir(), "templates")); err == nil {
		t.Fatal("development application succeeded without templates directory")
	}
}
