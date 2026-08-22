package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// New composes the application.
func New() http.Handler {
	e := echo.New()
	registerRoutes(e)
	return e
}
