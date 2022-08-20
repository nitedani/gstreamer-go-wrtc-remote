package capture

import (
	"client/utils"
	"os"
	"time"

	"github.com/olebedev/emitter"
	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

func NewAudioCapture() *ControlledCapture {
	counter := 0
	config := utils.GetMediaConfig()
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	gst.Init(nil)
	pipeline, err := gst.NewPipelineFromString(config.AudioPipeline)
	if err != nil {
		log.Err(err).Msg("Failed to create pipeline")
		os.Exit(2)
	}

	sink_el, _ := pipeline.GetElementByName("appsink")

	sink := app.SinkFromElement(sink_el)
	if err != nil {
		log.Err(err).Msg("Failed to create sink")
		os.Exit(2)
	}

	var samples = 0
	var buffer_len = int64(0)

	go func() {
		for {
			if pipeline.GetState() != gst.StatePlaying || samples == 0 {
				time.Sleep(time.Second)
				continue
			}

			per_buffer := buffer_len / int64(samples)

			log.Info().
				Int("audio_samplerate", samples).
				Int("audio_samples_size_kb", int(per_buffer/1024)).
				Int("audio_bitrate_kb", int(buffer_len/1024)).
				Send()

			samples = 0
			buffer_len = 0
		}
	}()

	sink.SetCallbacks(&app.SinkCallbacks{
		NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {
			sample := sink.PullSample()
			if sample == nil {
				return gst.FlowEOS
			}

			buffer := sample.GetBuffer()
			if buffer == nil {
				return gst.FlowError
			}

			len := buffer.GetSize()

			samples++
			buffer_len += len

			e.Emit("data", buffer)

			return gst.FlowOK
		},
	})

	start := func() {
		pipeline.SetState(gst.StatePlaying)
	}

	stop := func() {
		pipeline.SetState(gst.StatePaused)
	}

	return &ControlledCapture{
		Emitter: e,
		Start:   start,
		Stop:    stop,
		GetChannel: func() (chan *gst.Buffer, func()) {
			counter++
			channel := make(chan *gst.Buffer, 2)
			writing := false
			subscription := e.On("data", func(e *emitter.Event) {
				if writing {
					return
				}
				writing = true
				channel <- e.Args[0].(*gst.Buffer)
				writing = false
			})

			cleanup := func() {
				e.Off("data", subscription)
				close(channel)
				counter--
				if counter == 0 {
					stop()
				}
			}

			if counter == 1 {
				start()
			}

			return channel, cleanup
		},
	}
}
