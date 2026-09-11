package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewApplication(t *testing.T) {
	t.Parallel()

	database := filepath.Join(t.TempDir(), "contacts.db")
	application, err := newApplicationWithTemplateDirectory(false, "templates", database)
	if err != nil {
		t.Fatalf("create production application: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "joe@blow.com") {
		t.Fatalf("application response = %d %q", response.Code, response.Body.String())
	}
}

func TestNewApplicationEvalSeed(t *testing.T) {
	database := filepath.Join(t.TempDir(), "contacts.db")
	t.Setenv("APP_SEED", "eval")
	application, err := newApplicationWithTemplateDirectory(false, "templates", database)
	if err != nil {
		t.Fatalf("create eval application: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "dan@eval.test") {
		t.Fatalf("eval application response = %d %q", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "joe@blow.com") {
		t.Fatal("eval seed included demo contacts")
	}
	api := httptest.NewRecorder()
	application.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/v1/contacts?q=Ryu", nil))
	if api.Code != http.StatusOK || !strings.Contains(api.Body.String(), "Kenji") {
		t.Fatalf("eval API = %d %q", api.Code, api.Body.String())
	}
}

func TestDevelopmentStartupPropagatesTemplateDirectoryError(t *testing.T) {
	t.Parallel()

	database := filepath.Join(t.TempDir(), "contacts.db")
	if _, err := newApplicationWithTemplateDirectory(true, filepath.Join(t.TempDir(), "templates"), database); err == nil {
		t.Fatal("development application succeeded without templates directory")
	}
}

func TestNewApplicationDatabaseError(t *testing.T) {
	t.Parallel()

	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write blocked path: %v", err)
	}
	if _, err := newApplicationWithTemplateDirectory(false, "templates", filepath.Join(blocked, "contacts.db")); err == nil {
		t.Fatal("application succeeded with blocked database path")
	}
}
