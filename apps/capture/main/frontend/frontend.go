package frontend

import (
	"os"

	"github.com/webview/webview"
)

func OpenWindow() {

	debug := true
	w := webview.New(debug)

	w.SetTitle("Streamer")
	w.SetSize(800, 600, 0)

	if os.Getenv("GO_ENV") != "release" {
		w.Navigate("http://localhost:3050")
	} else {
		w.Navigate("https://en.m.wikipedia.org/wiki/Main_Page")
	}

	w.Run()

}
