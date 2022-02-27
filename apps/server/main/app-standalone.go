//go:build !appengine && !appenginevm
// +build !appengine,!appenginevm

package main

import (
	"os"
	"os/signal"
	"server/main/stream"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func createMux() *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	return e
}

func main() {

	if os.Getenv("GO_ENV") == "release" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	if len(os.Args) > 1 {
		envFilePath := os.Args[1]
		godotenv.Load(envFilePath)
	}

	frameBuffers := stream.CreateCapture()
	stream.StartWrtcServer(frameBuffers)
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
}
