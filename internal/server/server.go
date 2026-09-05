package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// New composes the application from its HTTP handlers.
func New(home, greeting http.Handler) http.Handler {
	e := echo.New()
	registerRoutes(e, home, greeting)
	return e
}
