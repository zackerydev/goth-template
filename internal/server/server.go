package server

import (
	"net/http"

	"github.com/zackerydev/goth-template/assets"
)

// New composes the application from its HTTP handlers.
func New(register func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.RedirectHandler("/contacts", http.StatusSeeOther))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets.FS())))
	if register != nil {
		register(mux)
	}
	return mux
}
