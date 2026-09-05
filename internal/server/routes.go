package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zackerydev/goth-template/assets"
)

func registerRoutes(e *echo.Echo, home, greeting http.Handler) {
	e.StaticFS("/assets/", assets.FS())
	e.GET("/", echo.WrapHandler(home))
	e.GET("/greeting", echo.WrapHandler(greeting))
}
