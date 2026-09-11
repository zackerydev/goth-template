package handler

import (
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/zackerydev/goth-template/internal/contact"
	"github.com/zackerydev/goth-template/templates"
)

type listPage struct {
	Title    string
	Flash    string
	Query    string
	Contacts []contact.Contact
	HasMore  bool
	NextPage int
}

type contactPage struct {
	Title   string
	Flash   string
	Contact contact.Contact
}

// Register attaches Contact.app routes to mux.
func Register(mux *http.ServeMux, renderer *templates.Renderer, store *contact.Store) {
	mux.Handle("GET /contacts", list(renderer, store))
	mux.Handle("GET /contacts/new", newContact(renderer))
	mux.Handle("POST /contacts/new", createContact(renderer, store))
	mux.Handle("GET /contacts/count", count(store))
	mux.Handle("GET /contacts/{id}", show(renderer, store))
	mux.Handle("GET /contacts/{id}/edit", edit(renderer, store))
	mux.Handle("POST /contacts/{id}/edit", update(renderer, store))
	mux.Handle("GET /contacts/{id}/email", email(store))
	mux.Handle("POST /contacts/{id}/delete", destroy(store))
	mux.Handle("DELETE /contacts/{id}", destroy(store))
	registerAPI(mux, store)
}

func list(renderer *templates.Renderer, store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query().Get("q")
		page := parsePage(request.URL.Query().Get("page"))
		result, err := store.List(request.Context(), query, page)
		if err != nil {
			http.Error(writer, "list contacts failed", http.StatusInternalServerError)
			return
		}
		writePage(writer, pageWrite{
			renderer: renderer,
			name:     "contacts",
			data: listPage{
				Title:    "Contacts",
				Flash:    takeFlash(writer, request),
				Query:    query,
				Contacts: result.Contacts,
				HasMore:  result.HasMore,
				NextPage: page + 1,
			},
		})
	})
}

func newContact(renderer *templates.Renderer) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writePage(writer, pageWrite{
			renderer: renderer,
			name:     "contacts-new",
			data:     contactPage{Title: "New Contact"},
		})
	})
}

func createContact(renderer *templates.Renderer, store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, err := store.Create(request.Context(), formContact(request)); writeFormError(writer, formView{
			renderer: renderer,
			name:     "contacts-new",
			title:    "New Contact",
		}, err) {
			return
		}
		setFlash(writer, "Created New Contact!")
		seeOther(writer, request, "/contacts")
	})
}

func show(renderer *templates.Renderer, store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, ok := loadContact(writer, request, store)
		if !ok {
			return
		}
		writePage(writer, pageWrite{
			renderer: renderer,
			name:     "contacts-show",
			data: contactPage{
				Title:   item.First + " " + item.Last,
				Flash:   takeFlash(writer, request),
				Contact: item,
			},
		})
	})
}

func edit(renderer *templates.Renderer, store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, ok := loadContact(writer, request, store)
		if !ok {
			return
		}
		writePage(writer, pageWrite{
			renderer: renderer,
			name:     "contacts-edit",
			data:     contactPage{Title: "Edit Contact", Contact: item},
		})
	})
}

func update(renderer *templates.Renderer, store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		item, ok := loadContact(writer, request, store)
		if !ok {
			return
		}
		form := formContact(request)
		item.First = form.First
		item.Last = form.Last
		item.Phone = form.Phone
		item.Email = form.Email
		saved, err := store.Update(request.Context(), item)
		if writeFormError(writer, formView{renderer: renderer, name: "contacts-edit", title: "Edit Contact"}, err) {
			return
		}
		setFlash(writer, "Updated Contact!")
		seeOther(writer, request, "/contacts/"+strconv.FormatInt(saved.ID, 10))
	})
}

func email(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseID(request)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		message, err := store.EmailError(request.Context(), id, request.FormValue("email"))
		if notFound(err) {
			http.NotFound(writer, request)
			return
		}
		if err != nil {
			http.Error(writer, "validate email failed", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(html.EscapeString(message)))
	})
}

func count(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		total, err := store.Count(request.Context())
		if err != nil {
			http.Error(writer, "count contacts failed", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte("(" + strconv.Itoa(total) + " total Contacts)"))
	})
}

func destroy(store *contact.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseID(request)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		if err := store.Delete(request.Context(), id); notFound(err) {
			http.NotFound(writer, request)
			return
		} else if err != nil {
			http.Error(writer, "delete contact failed", http.StatusInternalServerError)
			return
		}
		setFlash(writer, "Deleted Contact!")
		seeOther(writer, request, "/contacts")
	})
}

func loadContact(writer http.ResponseWriter, request *http.Request, store *contact.Store) (contact.Contact, bool) {
	id, err := parseID(request)
	if err != nil {
		http.NotFound(writer, request)
		return contact.Contact{}, false
	}
	item, err := store.Find(request.Context(), id)
	if notFound(err) {
		http.NotFound(writer, request)
		return contact.Contact{}, false
	}
	if err != nil {
		http.Error(writer, "load contact failed", http.StatusInternalServerError)
		return contact.Contact{}, false
	}
	return item, true
}

type formView struct {
	renderer *templates.Renderer
	name     string
	title    string
}

func writeFormError(writer http.ResponseWriter, view formView, err error) bool {
	if err == nil {
		return false
	}
	var invalid *contact.InvalidError
	if errors.As(err, &invalid) {
		writePage(writer, pageWrite{
			renderer: view.renderer,
			name:     view.name,
			status:   http.StatusUnprocessableEntity,
			data:     contactPage{Title: view.title, Contact: invalid.Contact},
		})
		return true
	}
	http.Error(writer, "save contact failed", http.StatusInternalServerError)
	return true
}

func parseID(request *http.Request) (int64, error) {
	id, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &contact.NotFoundError{}
	}
	return id, nil
}

func notFound(err error) bool {
	var missing *contact.NotFoundError
	return errors.As(err, &missing)
}
