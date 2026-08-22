package handler

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/zackerydev/goth-template/templates"
)

// Home returns the home page handler.
func Home() http.Handler {
	return templ.Handler(templates.Home(), templ.WithStatus(http.StatusOK))
}

// Greeting returns the example HTMX fragment handler.
func Greeting() http.Handler {
	return templ.Handler(templates.Greeting(), templ.WithStatus(http.StatusOK))
}
