package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zackerydev/goth-template/internal/contact"
)

type apiContact struct {
	ID    int64  `json:"id"`
	First string `json:"first"`
	Last  string `json:"last"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type apiListResponse struct {
	Contacts []apiContact `json:"contacts"`
	HasMore  bool         `json:"has_more"`
	Page     int          `json:"page"`
}

type apiError struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func registerAPI(mux *http.ServeMux, store *contact.Store) {
	mux.Handle("GET /api/v1/contacts", apiList(store))
	mux.Handle("POST /api/v1/contacts", apiCreate(store))
	mux.Handle("GET /api/v1/contacts/{id}", apiGet(store))
	mux.Handle("PUT /api/v1/contacts/{id}", apiUpdate(store))
	mux.Handle("PATCH /api/v1/contacts/{id}", apiUpdate(store))
	mux.Handle("DELETE /api/v1/contacts/{id}", apiDelete(store))
}

func apiList(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query().Get("q")
		page := parsePage(request.URL.Query().Get("page"))
		result, err := store.List(request.Context(), query, page)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, apiError{Error: "list contacts failed"})
			return
		}
		writeJSON(writer, http.StatusOK, apiListResponse{
			Contacts: toAPIContacts(result.Contacts),
			HasMore:  result.HasMore,
			Page:     page,
		})
	})
}

func apiCreate(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, err := decodeAPIContact(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, apiError{Error: "invalid json"})
			return
		}
		created, err := store.Create(request.Context(), item)
		if writeAPIError(writer, err) {
			return
		}
		writeJSON(writer, http.StatusCreated, toAPIContact(created))
	})
}

func apiGet(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, ok := loadAPIContact(writer, request, store)
		if !ok {
			return
		}
		writeJSON(writer, http.StatusOK, toAPIContact(item))
	})
}

func apiUpdate(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, ok := loadAPIContact(writer, request, store)
		if !ok {
			return
		}
		form, err := decodeAPIContact(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, apiError{Error: "invalid json"})
			return
		}
		item.First = form.First
		item.Last = form.Last
		item.Phone = form.Phone
		item.Email = form.Email
		saved, err := store.Update(request.Context(), item)
		if writeAPIError(writer, err) {
			return
		}
		writeJSON(writer, http.StatusOK, toAPIContact(saved))
	})
}

func apiDelete(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseID(request)
		if err != nil {
			writeJSON(writer, http.StatusNotFound, apiError{Error: "not found"})
			return
		}
		if err := store.Delete(request.Context(), id); writeAPIError(writer, err) {
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}

func loadAPIContact(writer http.ResponseWriter, request *http.Request, store *contact.Store) (contact.Contact, bool) {
	id, err := parseID(request)
	if err != nil {
		writeJSON(writer, http.StatusNotFound, apiError{Error: "not found"})
		return contact.Contact{}, false
	}
	item, err := store.Find(request.Context(), id)
	if writeAPIError(writer, err) {
		return contact.Contact{}, false
	}
	return item, true
}

func writeAPIError(writer http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var invalid *contact.InvalidError
	if errors.As(err, &invalid) {
		writeJSON(writer, http.StatusUnprocessableEntity, apiError{
			Error:  "invalid contact",
			Fields: invalid.Contact.Errors,
		})
		return true
	}
	if notFound(err) {
		writeJSON(writer, http.StatusNotFound, apiError{Error: "not found"})
		return true
	}
	writeJSON(writer, http.StatusInternalServerError, apiError{Error: "contact store failed"})
	return true
}

func decodeAPIContact(request *http.Request) (contact.Contact, error) {
	var body apiContact
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		return contact.Contact{}, err
	}
	return contact.Contact{
		First: body.First,
		Last:  body.Last,
		Phone: body.Phone,
		Email: body.Email,
	}, nil
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func toAPIContact(item contact.Contact) apiContact {
	return apiContact{ID: item.ID, First: item.First, Last: item.Last, Phone: item.Phone, Email: item.Email}
}

func toAPIContacts(items []contact.Contact) []apiContact {
	listed := make([]apiContact, 0, len(items))
	for _, item := range items {
		listed = append(listed, toAPIContact(item))
	}
	return listed
}
