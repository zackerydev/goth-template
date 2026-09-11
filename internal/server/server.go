package server

import (
	"net/http"
)

// New composes the application from its HTTP handlers.
func New(register func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mountPublic(mux)
	if register != nil {
		register(mux)
	}
	return mux
}
