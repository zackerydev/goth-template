package server

import (
	"net/http"

	"github.com/zackerydev/goth-template/assets"
)

// New composes the application from its HTTP handlers.
func New(home, greeting, lab http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", home)
	mux.Handle("GET /greeting", greeting)
	mux.Handle("/lab", lab)
	mux.Handle("/lab/", lab)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets.FS())))
	return mux
}
