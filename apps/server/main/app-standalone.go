//go:build !appengine && !appenginevm
// +build !appengine,!appenginevm

package main

import (
	"os"
	"signaling/main/stream"

	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func createMux() *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())

	return e
}

func main() {

	if len(os.Args) > 1 {
		envFilePath := os.Args[1]
		godotenv.Load(envFilePath)
	}

	if os.Getenv("GO_ENV") != "release" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	}
	e := createMux()

	g := e.Group("/api")

	server := socketio.NewServer(nil)
	go server.Serve()
	defer server.Close()

	stream.StartSignalingServer(g, server)

	e.Any("/api/socket/", func(context echo.Context) error {
		server.ServeHTTP(context.Response(), context.Request())
		return nil
	})

	if os.Getenv("GO_ENV") == "release" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:  "webapp",
			Index: "index.html",
			HTML5: true,
		}))
		e.Start(":" + port)

	} else {
		e.Start(":4000")
	}

}
