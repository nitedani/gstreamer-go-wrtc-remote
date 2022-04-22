package capture

import (
	"github.com/olebedev/emitter"
	"github.com/tinyzimmer/go-gst/gst"
)

type ControlledCapture struct {
	*emitter.Emitter
	GetChannel func() (channel chan *gst.Buffer, cleanup func())
	Start      func()
	Stop       func()
}
