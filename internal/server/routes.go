package server

import (
	"github.com/labstack/echo/v5"
	"github.com/zackerydev/goth-template/assets"
	"github.com/zackerydev/goth-template/internal/handler"
)

func registerRoutes(e *echo.Echo) {
	e.StaticFS("/assets/", assets.FS())
	e.GET("/", echo.WrapHandler(handler.Home()))
	e.GET("/greeting", echo.WrapHandler(handler.Greeting()))
}
