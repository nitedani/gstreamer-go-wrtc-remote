package stream

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func remove(s []MediaChannel, i int) []MediaChannel {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

type MediaChannel struct {
	Channel chan *gst.Buffer
	Context context.Context
	Cancel  context.CancelFunc
}

var video_channels = make([]MediaChannel, 0)
var audio_channels = make([]MediaChannel, 0)

func CreateVideoCapture() func() chan *gst.Buffer {

	bitrate, hasEnv := os.LookupEnv("BITRATE")
	if !hasEnv {
		log.Info().Msg("No bitrate specified, defaulting to 5Mbps")
		bitrate = "5242880"
	}

	resolution, hasEnv := os.LookupEnv("RESOLUTION")
	if !hasEnv {
		log.Info().Msg("No resolution specified, defaulting to 1280x720")
		resolution = "1280x720"
	}
	sizes := strings.Split(resolution, "x")

	framerate, hasEnv := os.LookupEnv("FRAMERATE")
	if !hasEnv {
		log.Info().Msg("No framerate specified, defaulting to 30fps")
		framerate = "30"
	}

	threads, hasEnv := os.LookupEnv("THREADS")
	if !hasEnv {
		log.Info().Msg("No threads specified, defaulting to 2")
		threads = "2"
	}

	frames_ch := make(chan *gst.Buffer)
	pipelinearr := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		"show-cursor=0",
		"!",

		"d3d11convert",
		"!",

		"d3d11download",
		"!",

		"queue",
		"!",

		//Optimize for framerate
		"vp8enc",
		"threads=" + threads,
		"deadline=100",
		"max-quantizer=40",
		"min-quantizer=10",
		"max-intra-bitrate=" + bitrate,
		"target-bitrate=" + bitrate,
		//"keyframe-max-dist=10",
		"!",

		fmt.Sprintf("video/x-vp8,framerate=%s/1,width=%s,height=%s", framerate, sizes[0], sizes[1]),
		"!",

		"appsink",
		"name=appsink",
	}

	pipelineString := strings.Join(pipelinearr, " ")

	gst.Init(nil)
	pipeline, err := gst.NewPipelineFromString(pipelineString)
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

			if pipeline.GetState() != gst.StatePlaying {
				time.Sleep(time.Second)
				frames = 0
				buffer_len = 0
				continue
			}

			time.Sleep(time.Second)
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

			select {
			case frames_ch <- buffer:
				break
			default:
				break
			}

			return gst.FlowOK
		},
	})

	go func() {
		for {
			val := <-frames_ch
			for i := 0; i < len(video_channels); i++ {
				go func(i int) {
					select {
					case <-video_channels[i].Context.Done():
					case video_channels[i].Channel <- val:
					case <-time.After(time.Millisecond * 100):
						video_channels[i].Cancel()
						<-video_channels[i].Context.Done()
						close(video_channels[i].Channel)
						video_channels = remove(video_channels, i)
						log.Info().Int("viewers", len(video_channels)).Msg("Closing video channel")
						if len(video_channels) == 0 {
							log.Info().Msg("Pausing screen capture")
							pipeline.SetState(gst.StatePaused)
						}
					}
				}(i)
			}
		}
	}()

	return func() chan *gst.Buffer {
		ctx, cancel := context.WithCancel(context.Background())
		mediaChannel := MediaChannel{
			Channel: make(chan *gst.Buffer),
			Context: ctx,
			Cancel:  cancel,
		}

		video_channels = append(video_channels, mediaChannel)
		if condition := pipeline.GetState() != gst.StatePlaying; condition {
			log.Info().Int("viewers", len(video_channels)).Msg("Starting video capture")
			pipeline.SetState(gst.StatePlaying)
		}
		return mediaChannel.Channel
	}
}

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
		fmt.Println(err)
		os.Exit(2)
	}

	sink_el, _ := pipeline.GetElementByName("appsink")

	sink := app.SinkFromElement(sink_el)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var samples = 0
	var buffer_len = int64(0)

	go func() {
		for {

			if pipeline.GetState() != gst.StatePlaying {
				time.Sleep(time.Second)
				samples = 0
				buffer_len = 0
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
			for i := 0; i < len(audio_channels); i++ {
				go func(i int) {

					select {
					case <-audio_channels[i].Context.Done():
						return
					case audio_channels[i].Channel <- val:
					case <-time.After(time.Millisecond * 100):
						audio_channels[i].Cancel()
						<-audio_channels[i].Context.Done()
						close(audio_channels[i].Channel)
						audio_channels = remove(audio_channels, i)
						log.Info().Int("viewers", len(audio_channels)).Msg("Closing audio channel")
						if len(audio_channels) == 0 {
							log.Info().Msg("Pausing audio capture")
							pipeline.SetState(gst.StatePaused)
						}
					}
				}(i)
			}
		}
	}()

	return func() chan *gst.Buffer {
		ctx, cancel := context.WithCancel(context.Background())
		mediaChannel := MediaChannel{
			Channel: make(chan *gst.Buffer),
			Context: ctx,
			Cancel:  cancel,
		}

		audio_channels = append(audio_channels, mediaChannel)
		if pipeline.GetState() != gst.StatePlaying {
			log.Info().Int("viewers", len(audio_channels)).Msg("Starting audio capture")
			pipeline.SetState(gst.StatePlaying)
		}
		return mediaChannel.Channel
	}

}
