//go:build !appengine && !appenginevm
// +build !appengine,!appenginevm

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	//go frontend.OpenWindow()

	if os.Getenv("GO_ENV") != "release" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	StartWrtcServer()
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
}
