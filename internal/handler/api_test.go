package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/zackerydev/goth-template/internal/contact"
)

func TestAPIListCreateGetUpdateDelete(t *testing.T) {
	t.Parallel()

	_, application := openApp(t)
	listed := send(t, application, httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil))
	assertStatus(t, listed, http.StatusOK)
	assertBody(t, listed, `"contacts":[]`, `"has_more":false`)

	created := send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", map[string]string{
		"first": "Ada", "last": "Lovelace", "phone": "1", "email": "ada@example.com",
	}))
	assertStatus(t, created, http.StatusCreated)
	assertBody(t, created, `"first":"Ada"`)
	path := "/api/v1/contacts/" + createdID(t, created)
	assertBody(t, send(t, application, httptest.NewRequest(http.MethodGet, path, nil)), "ada@example.com")

	updated := send(t, application, jsonBody(t, http.MethodPut, path, map[string]string{
		"first": "Ada", "last": "Byron", "phone": "2", "email": "ada@example.com",
	}))
	assertStatus(t, updated, http.StatusOK)
	assertBody(t, updated, `"last":"Byron"`)
	patched := send(t, application, jsonBody(t, http.MethodPatch, path, map[string]string{
		"first": "Ada", "last": "Byron", "phone": "3", "email": "ada@example.com",
	}))
	assertBody(t, patched, `"phone":"3"`)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodDelete, path, nil)), http.StatusNoContent)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodGet, path, nil)), http.StatusNotFound)
}

func TestAPIValidationAndNotFound(t *testing.T) {
	t.Parallel()

	_, application := openApp(t)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", nil)), http.StatusBadRequest)
	assertStatus(t, send(t, application, jsonRawBody(http.MethodPost, "/api/v1/contacts", "{")), http.StatusBadRequest)
	missing := send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", map[string]string{"email": ""}))
	assertStatus(t, missing, http.StatusUnprocessableEntity)
	assertBody(t, missing, "Email Required")

	created := send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", map[string]string{
		"first": "Ada", "email": "ada@example.com",
	}))
	path := "/api/v1/contacts/" + createdID(t, created)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", map[string]string{
		"email": "ADA@example.com",
	})), http.StatusUnprocessableEntity)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPut, path, map[string]string{"email": ""})), http.StatusUnprocessableEntity)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPut, path, nil)), http.StatusBadRequest)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodGet, "/api/v1/contacts/nope", nil)), http.StatusNotFound)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPut, "/api/v1/contacts/nope", map[string]string{
		"email": "x@y.z",
	})), http.StatusNotFound)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/999", nil)), http.StatusNotFound)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/nope", nil)), http.StatusNotFound)
}

func TestAPIClosedStoreAndSearch(t *testing.T) {
	t.Parallel()

	store, application := openApp(t)
	item := createPerson(t, store, contact.Contact{First: "Ada", Email: "ada@example.com"})
	createPerson(t, store, contact.Contact{First: "Kenji", Last: "Ryu", Email: "kenji@example.com"})
	assertBody(t, send(t, application, httptest.NewRequest(http.MethodGet, "/api/v1/contacts?q=Ryu", nil)), "Kenji")
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodGet, "/api/v1/contacts?page=2", nil)), http.StatusOK)
	path := "/api/v1/contacts/" + strconv.FormatInt(item.ID, 10)
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)), http.StatusInternalServerError)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodGet, path, nil)), http.StatusInternalServerError)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPost, "/api/v1/contacts", map[string]string{
		"email": "new@example.com",
	})), http.StatusInternalServerError)
	assertStatus(t, send(t, application, jsonBody(t, http.MethodPut, path, map[string]string{
		"email": "ada@example.com",
	})), http.StatusInternalServerError)
	assertStatus(t, send(t, application, httptest.NewRequest(http.MethodDelete, path, nil)), http.StatusInternalServerError)
}

func TestAPIWriteIgnoresResponseWriterErrors(t *testing.T) {
	t.Parallel()

	_, application := openApp(t)
	application.ServeHTTP(failingResponseWriter{header: make(http.Header)}, httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil))
}

func jsonBody(t *testing.T, method, path string, body map[string]string) *http.Request {
	t.Helper()
	if body == nil {
		return jsonRawBody(method, path, "")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return jsonRawBody(method, path, string(payload))
}

func jsonRawBody(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func createdID(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var item struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return strconv.FormatInt(item.ID, 10)
}
