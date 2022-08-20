package capture

import (
	"github.com/olebedev/emitter"
	"github.com/tinyzimmer/go-gst/gst"
)

type ControlledCapture struct {
	*emitter.Emitter
	pipeline *gst.Pipeline
	counter  int
}

func (c *ControlledCapture) Start() {
	c.pipeline.SetState(gst.StatePlaying)
}

func (c *ControlledCapture) Stop() {
	c.pipeline.SetState(gst.StateNull)
}

func (c *ControlledCapture) GetChannel() (channel chan *gst.Buffer, cleanup func()) {
	c.counter++
	channel = make(chan *gst.Buffer, 2)
	writing := false

	subscription := c.On("data", func(e *emitter.Event) {
		if writing {
			return
		}
		writing = true
		channel <- e.Args[0].(*gst.Buffer)
		writing = false
	})

	cleanup = func() {
		c.Off("data", subscription)
		close(channel)
		c.counter--
		if c.counter <= 0 {
			c.Stop()
		}
	}

	if c.counter >= 1 {
		c.Start()
	}

	return channel, cleanup
}
