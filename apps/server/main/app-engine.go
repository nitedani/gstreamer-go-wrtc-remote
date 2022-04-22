//go:build appengine
// +build appengine

package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo/v5"
	"net/http"
)

func createMux() *echo.Echo {
	e := echo.New()
	// note: we don't need to provide the middleware or static handlers, that's taken care of by the platform
	// app engine has it's own "main" wrapper - we just need to hook echo into the default handler
	http.Handle("/", e)
	return e
}

func init() {
	var e = createMux()
	g := e.Group("")
	stream.StartSignalingServer(g)
}
