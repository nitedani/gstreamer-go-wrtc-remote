package capture

import (
	"client/utils"
	"fmt"
	"os"
	"time"

	"github.com/olebedev/emitter"
	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

func NewVideoCapture() *ControlledCapture {
	config := utils.GetMediaConfig()
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)

	gst.Init(nil)
	pipeline, err := gst.NewPipelineFromString(config.VideoPipeline)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	sink_el, _ := pipeline.GetElementByName("appsink")

	sink := app.SinkFromElement(sink_el)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var frames = 0
	var buffer_len = int64(0)

	go func() {
		for {

			if pipeline.GetState() != gst.StatePlaying || frames == 0 {
				time.Sleep(time.Second)
				continue
			}

			per_buffer := buffer_len / int64(frames)

			log.Info().
				Int("video_framerate", frames).
				Int("video_frame_size_kb", int(per_buffer/1024)).
				Int("video_bitrate_kb", int(buffer_len/1024)).
				Send()

			frames = 0
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

			frames++
			buffer_len += len

			e.Emit("data", buffer)

			return gst.FlowOK
		},
	})

	return &ControlledCapture{
		Emitter:  e,
		counter:  0,
		pipeline: pipeline,
	}
}
