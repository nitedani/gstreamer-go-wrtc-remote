package capture

import (
	"context"
	"os"
	"server/main/utils"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

var audio_channels = make(map[string]*MediaChannel, 0)

func CreateAudioCapture() func() chan *gst.Buffer {

	samples_ch := make(chan *gst.Buffer)
	pipelinearr := []string{
		"wasapisrc",
		"low-latency=true",
		"loopback=true",
		"!",
		"audioconvert",
		"!",
		"queue2",
		"max-size-buffers=0",
		"!",
		"opusenc",
		"max-payload-size=1500",
		"bitrate=128000",
		"!",
		"appsink",
		"name=appsink",
	}

	pipelineString := strings.Join(pipelinearr, " ")

	gst.Init(nil)
	pipeline, err := gst.NewPipelineFromString(pipelineString)
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

			time.Sleep(time.Second)
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

			select {
			case samples_ch <- buffer:
				break
			default:
				break
			}

			return gst.FlowOK
		},
	})

	go func() {
		for {
			val := <-samples_ch
			for _, channel := range audio_channels {
				if channel.IsDone || channel.Writing {
					continue
				}
				channel.Writing = true
				go func(channel *MediaChannel) {
					select {
					case <-channel.Context.Done():
						return
					case channel.Channel <- val:
						channel.Writing = false
						return
					case <-time.After(time.Millisecond * 100):
						channel.IsDone = true
						channel.Cancel()
					}
				}(channel)
			}
		}
	}()

	return func() chan *gst.Buffer {
		ctx, cancel := context.WithCancel(context.Background())

		id := utils.RandomStr()

		mediaChannel := MediaChannel{
			ID:      id,
			Channel: make(chan *gst.Buffer),
			Context: ctx,
			Cancel:  cancel,
			IsDone:  false,
			Writing: false,
		}

		audio_channels[id] = &mediaChannel

		if pipeline.GetState() != gst.StatePlaying {
			log.Info().Int("viewers", len(audio_channels)).Msg("Starting audio capture")
			pipeline.SetState(gst.StatePlaying)
		}

		go func() {
			<-mediaChannel.Context.Done()
			delete(audio_channels, id)
			log.Info().Int("viewers", len(audio_channels)).Msg("Closing audio channel")
			if len(audio_channels) == 0 {
				log.Info().Msg("Pausing audio capture")
				pipeline.SetState(gst.StatePaused)
			}
		}()

		return mediaChannel.Channel
	}

}
